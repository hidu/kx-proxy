package handler

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xanygo/anygo/xattr"
	"github.com/xanygo/anygo/xhttp"

	"github.com/hidu/kx-proxy/internal"
	"github.com/hidu/kx-proxy/internal/links"
)

var doProxy = &DoProxy{}

func (k *KxProxy) handlerProxy(w http.ResponseWriter, r *http.Request) {
	doProxy.ServeHTTP(w, r)
}

var _ http.Handler = (*DoProxy)(nil)

type DoProxy struct {
}

func (d *DoProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logData := make(internal.LogData)
	reqStart := time.Now()
	defer func() {
		used := time.Since(reqStart)
		log.Println("remote:", r.RemoteAddr, "path:", r.URL.Path, "cost:", used, logData)
	}()

	r.Header.Del("Connection")
	encodedURL := r.URL.Path[len("/p/"):]

	pu, err := links.DecodeProxyURL(encodedURL)
	if err != nil {
		logData["emsg"] = "decode_url_failed:" + err.Error()
		http.Error(w, "decode_url_failed:"+err.Error(), http.StatusInternalServerError)
		return
	}
	urlString := pu.GetURLStr()
	logData["ID"] = pu.ID

	// 检查url是否过期
	if pu.IsExpire() {
		xhttp.Error(w, r, http.StatusBadRequest, "", "url expired")
		return
	}

	if !pu.CheckSign(r) {
		xhttp.Error(w, r, http.StatusBadRequest, "", "invalid sign")
		return
	}

	logData["origin"] = urlString

	if r.URL.Query().Get("cache") == "no" {
		pu.SetNoCache()
	}

	ctx, cancel := pu.Context(r.Context())
	defer cancel()
	ctx = context.WithValue(ctx, ctxKeyLogData, logData)

	d.do(ctx, w, r, pu)
}

type ctxKeyType uint8

const (
	ctxKeyLogData ctxKeyType = iota
)

func getLogData(ctx context.Context) internal.LogData {
	return ctx.Value(ctxKeyLogData).(internal.LogData)
}

func (d *DoProxy) do(ctx context.Context, w http.ResponseWriter, r *http.Request, pu *links.ProxyURL) {
	defer func() {
		if re := recover(); re != nil {
			xhttp.Error(w, r, http.StatusInternalServerError, "", fmt.Sprintf("panic: %v", re))
		}
	}()
	urlString := pu.GetURLStr()
	logData := getLogData(ctx)

	var resp *internal.Response

	if pu.CacheAble() {
		resp = d.fromCache(urlString)
	}

	fromCache := resp != nil
	logData["from_cache"] = fromCache

	if resp == nil {
		var err error
		resp, err = d.directGet(ctx, r, pu)
		if err != nil {
			logData["emsg"] = "fetch_failed:" + err.Error()
			xhttp.Error(w, r, http.StatusBadGateway, "", "Fetching "+urlString+": "+err.Error())
			return
		}
		logData["ct"] = resp.ContentType
	}

	logData["resp_status"] = resp.StatusCode

	rg, err := d.redirect(resp, pu)
	if err != nil {
		logData["emsg"] = "redirect_failed:" + err.Error()
		xhttp.Error(w, r, http.StatusBadGateway, "", "Redirect "+urlString+": "+err.Error())
		return
	}

	if rg != "" {
		http.Redirect(w, r, rg, http.StatusFound)
		return
	}

	// 是否允许 cache
	canCache := pu.CacheAble() && resp.ContentType.CanCache()

	if canCache {
		if resp.StatusCode == http.StatusOK {
			// canCache
		} else if resp.StatusCode == http.StatusNotFound && pu.IsStaticURL() {
			// canCache
		} else {
			canCache = false
		}
	}

	logData["canCache"] = canCache

	if canCache && !fromCache {
		cached := d.trySetCache(pu, resp)
		logData["save_cache"] = cached
	}

	if pu.NoCache() {
		fileCache.Del(pu.GetURLStr())
	}

	inm := r.Header.Get("If-None-Match")

	var wrote int64
	if inm != "" && inm == resp.HeaderGet("Etag") {
		w.WriteHeader(304)
	} else {
		d.reWriteResp(r, resp, pu)
		wrote, err = resp.WriteTo(w)
	}

	logData["resp_wrote_size"] = wrote
	logData["resp_wrote_err"] = err
}

func (d *DoProxy) fromCache(urlString string) *internal.Response {
	cd := fileCache.Get(urlString)
	if cd != nil {
		return &internal.Response{
			StatusCode:  http.StatusOK,
			ContentType: cd.ContentType(),
			Body:        cd.Body,
			HeaderMap:   cd.Header,
		}
	}
	return nil
}

func (d *DoProxy) directGet(ctx context.Context, r *http.Request, pu *links.ProxyURL) (*internal.Response, error) {
	urlString := pu.GetURLStr()
	req, err := http.NewRequestWithContext(ctx, r.Method, urlString, r.Body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", r.Header.Get("Content-Type"))
	req.Header.Set("User-Agent", r.Header.Get("User-Agent"))
	var resp *http.Response
	var err1 error
	for try := 0; try < pu.Extension.TryTotal(); try++ {
		resp, err1 = internal.GetClient(pu.Extension.SkipVerify()).Do(req)
		if err1 == nil {
			break
		}
	}
	if err1 != nil {
		return nil, err1
	}
	contentType := internal.ContentType(resp.Header.Get("Content-Type"))

	if contentType.CanCache() {
		defer resp.Body.Close()
		body, err2 := io.ReadAll(io.LimitReader(resp.Body, internal.CacheMaxSize))
		if err2 != nil {
			return nil, err2
		}
		return &internal.Response{
			StatusCode:  resp.StatusCode,
			ContentType: contentType,
			Header:      resp.Header,
			Body:        body,
			Raw:         resp,
		}, nil
	}

	return &internal.Response{
		StatusCode:  resp.StatusCode,
		ContentType: contentType,
		Header:      resp.Header,
		Raw:         resp,
	}, nil
}

func (d *DoProxy) redirect(resp *internal.Response, pu *links.ProxyURL) (string, error) {
	if resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusFound {
		location := resp.Header.Get("Location")
		if location != "" {
			pu.SwitchURL(location)
			encodedURL, err := pu.Encode()
			if err != nil {
				return "", err
			}
			return "/p/" + encodedURL, nil
		}
	}
	return "", nil
}

func (d *DoProxy) trySetCache(pu *links.ProxyURL, resp *internal.Response) bool {
	life := staticCacheTime(resp)
	if life <= 0 {
		return false
	}

	if len(resp.Body) == 0 {
		return false
	}

	ncd := &internal.CacheData{
		Header: map[string]string{
			"Content-Type": string(resp.ContentType),
			"Etag":         `"` + fmt.Sprintf("%x", md5.Sum(resp.Body)) + `"`,
		},
		Body: resp.Body,
	}

	// 404 资源可以cache，但是特殊对待
	if resp.StatusCode == 404 {
		if life > time.Hour {
			life = time.Hour
		}
	}

	fileCache.SetWithTTL(pu.GetURLStr(), ncd, life)
	return true
}

func (d *DoProxy) reWriteResp(r *http.Request, resp *internal.Response, pu *links.ProxyURL) *internal.Response {
	contentType := resp.ContentType

	if contentType.IsHTML() {
		return d.reWriteHTML(r, resp, pu)
	}

	if contentType.IsCSS() {
		return d.reWriteCSS(r, resp, pu)
	}

	return resp
}

func (d *DoProxy) reWriteHTML(r *http.Request, resp *internal.Response, pu *links.ProxyURL) *internal.Response {
	urlString := pu.GetURLStr()
	body := pu.Extension.Rewrite(resp.Body)
	if pu.Extension.PreloadingNext() {
		go preLoader.PreLoad(pu, body, internal.PreLoadTypeNext)
	} else if pu.Extension.PreloadingSameDir() {
		go preLoader.PreLoad(pu, body, internal.PreLoadTypeSameDir)
	} else if pu.Extension.Preloading() {
		go preLoader.PreLoad(pu, body, internal.PreLoadTypeAll)
	}

	encodedBody := links.HTMLURLReplace(body, urlString, pu, r)
	encodedBody = links.CSSURLReplace(encodedBody, urlString, pu, r)

	var hBuf bytes.Buffer
	hdCode := pu.HeadHTML()
	if len(hdCode) > 0 {
		hBuf.Write(hdCode)
	}

	ucss := d.userCSSLink(pu)
	hBuf.WriteString(ucss)

	if ujs := d.userJSLink(pu); len(ujs) > 0 {
		hBuf.WriteString(ujs)
	}

	hBuf.Write(encodedBody)

	hBuf.WriteString(ucss)

	if uc := d.userCSS(pu); uc != "" {
		hBuf.WriteString(uc)
	}

	if pu.Extension.InverseColor() {
		hBuf.WriteString(inverseColor)
	}

	resp.Body = hBuf.Bytes()
	return resp
}

const inverseColor = `
<style>
html {
  background: white !important;
  filter: invert(1) hue-rotate(180deg) !important;
}
</style>
`

func (d *DoProxy) userCSSLink(pu *links.ProxyURL) string {
	if !pu.Extension.Has("ucss") {
		return ""
	}
	hostName := pu.Hostname()
	ucss := []string{
		"/ucss/all.css",
		"/ucss/" + hostName + ".css",
	}
	var buf strings.Builder
	for _, ufile := range ucss {
		if version := hasUserFile(ufile); version != "" {
			buf.WriteString(`<link rel="stylesheet" href="`)
			buf.WriteString(ufile)
			buf.WriteString("?")
			buf.WriteString(version)
			buf.WriteString(`" />`)
			buf.WriteString("\n")
		}
	}
	return buf.String()
}

func (d *DoProxy) userCSS(pu *links.ProxyURL) string {
	if pu.UserCss == "" {
		return ""
	}
	return `<style type="text/css">` + html.EscapeString(pu.UserCss) + "</style>\n"
}

func (d *DoProxy) userJSLink(pu *links.ProxyURL) string {
	if !pu.Extension.Has("ucss") {
		return ""
	}
	hostName := pu.Hostname()
	ujss := []string{
		"/ucss/jquery.min.js",
		"/ucss/all.js",
		"/ucss/" + hostName + ".js",
	}
	var buf strings.Builder
	for _, ufile := range ujss {
		if version := hasUserFile(ufile); version != "" {
			buf.WriteString(`<script src="`)
			buf.WriteString(ufile)
			buf.WriteString("?")
			buf.WriteString(version)
			buf.WriteString(`" defer></script>`)
			buf.WriteString("\n")
		}
	}
	return buf.String()
}

func (d *DoProxy) reWriteCSS(r *http.Request, resp *internal.Response, pu *links.ProxyURL) *internal.Response {
	urlString := pu.GetURLStr()
	encodedBody := links.CSSURLReplace(resp.Body, urlString, pu, r)
	resp.Body = encodedBody
	return resp
}

func staticCacheTime(resp *internal.Response) time.Duration {
	raw := resp.Raw
	if raw == nil {
		return 0
	}
	if raw.Header.Get("ETag") != "" &&
		raw.ContentLength > 0 &&
		raw.ContentLength < internal.CacheMaxSize {
		return 12 * time.Hour
	}

	if raw.Header.Get("X-Cache-Status") == "HIT" {
		return time.Hour
	}

	ct := internal.ContentType(raw.Header.Get("Content-Type"))
	if ct.IsStaticFile() {
		return 12 * time.Hour
	}

	if links.IsStaticPath(raw.Request.URL.Path) {
		return 12 * time.Hour
	}
	return 0
}

// hasUserFile 是否存在用户自定义的 js 、css
//
//	eg: /ucss/all.css, /ucss/all.js
func hasUserFile(name string) string {
	fp := filepath.Join(xattr.ConfDir(), name)
	info, err := os.Stat(fp)
	if err != nil {
		return ""
	}
	if info.IsDir() || info.Size() == 0 {
		return ""
	}
	return info.ModTime().Format("02150405")
}
