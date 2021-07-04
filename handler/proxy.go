package handler

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/hidu/kx-proxy/internal"
	"github.com/hidu/kx-proxy/util"
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
		used := time.Now().Sub(reqStart)
		log.Println("remote:", r.RemoteAddr, "path:", r.URL.Path, "used:", used, logData)
	}()

	r.Header.Del("Connection")
	encodedURL := r.URL.Path[len("/p/"):]

	pu, err := util.DecodeProxyUrl(encodedURL)
	if err != nil {
		logData["emsg"] = "decode_url_failed:" + err.Error()
		http.Error(w, "decode_url_failed:"+err.Error(), http.StatusInternalServerError)
		return
	}
	urlString := pu.GetUrlStr()

	// 检查url是否过期
	if pu.IsExpire() {
		http.Error(w, "expired", http.StatusBadRequest)
		return
	}

	if !pu.CheckSign(r) {
		http.Error(w, "sign not match", http.StatusBadRequest)
		return
	}

	logData["visit_url"] = urlString

	if r.URL.Query().Get("cache") == "no" {
		fileCache.Del(urlString)
	}

	ctx := r.Context()
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

func (d *DoProxy) do(ctx context.Context, w http.ResponseWriter, r *http.Request, pu *util.ProxyUrl) {
	urlString := pu.GetUrlStr()
	logData := getLogData(ctx)

	var resp *internal.Response

	if pu.Extension.Cache() && !pu.Extension.NoCache() {
		resp = d.fromCache(urlString)
	}

	fromCache := resp != nil

	logData["from_cache"] = fromCache

	if resp == nil {
		var err error
		resp, err = d.directGet(r, pu)
		if err != nil {
			logData["emsg"] = "fetch_failed:" + err.Error()
			http.Error(w, "Error Fetching "+urlString+"\n"+err.Error(), http.StatusBadGateway)
			return
		}
	}

	logData["resp_status"] = resp.StatusCode

	rg, err := d.redirect(resp, pu)
	if err != nil {
		logData["emsg"] = "rediect_failed:" + err.Error()
		http.Error(w, "Error Redirect "+urlString+"\n"+err.Error(), http.StatusBadGateway)
		return
	}

	if rg != "" {
		http.Redirect(w, r, rg, 302)
		return
	}

	// 是否允许 cache
	canCache := pu.Extension.Cache() && !fromCache && resp.ContentType.CanCache()

	canCache = canCache && r.URL.Query().Get("no_cache") == ""

	logData["canCache"] = canCache

	if canCache {
		cached := d.trySetCache(pu, resp)
		logData["save_cache"] = cached
	}

	d.reWriteResp(r, resp, pu)

	wrote, err := resp.WriteTo(w)
	logData["resp_wrote_size"] = wrote
	logData["resp_wrote_err"] = err
}

func (d *DoProxy) fromCache(urlString string) *internal.Response {
	cd := fileCache.Get(urlString)
	if cd != nil {
		return &internal.Response{
			StatusCode:  200,
			ContentType: cd.ContentType(),
			Body:        cd.Body,
			HeaderMap:   cd.Header,
		}
	}
	return nil
}

func (d *DoProxy) directGet(r *http.Request, pu *util.ProxyUrl) (*internal.Response, error) {
	urlString := pu.GetUrlStr()
	req, err := http.NewRequestWithContext(r.Context(), r.Method, urlString, r.Body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", r.Header.Get("Content-Type"))
	req.Header.Set("User-Agent", r.Header.Get("User-Agent"))
	resp, err1 := internal.Client.Do(req)
	if err1 != nil {
		return nil, err1
	}
	contentType := internal.ContentType(resp.Header.Get("Content-Type"))
	if contentType.IsText() {
		defer resp.Body.Close()
		body, err2 := ioutil.ReadAll(io.LimitReader(resp.Body, internal.CacheMaxSize))
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

func (d *DoProxy) redirect(resp *internal.Response, pu *util.ProxyUrl) (string, error) {
	if resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusFound {
		location := resp.Header.Get("Location")
		if location != "" {
			pu.SwitchUrl(location)
			encodedURL, err := pu.Encode()
			if err != nil {
				return "", err
			}
			return "/p/" + encodedURL, nil
		}
	}
	return "", nil
}

func (d *DoProxy) trySetCache(pu *util.ProxyUrl, resp *internal.Response) bool {
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
		},
		Body: resp.Body,
	}
	fileCache.SetWithTTL(pu.GetUrlStr(), ncd, life)
	return true
}

func (d *DoProxy) reWriteResp(r *http.Request, resp *internal.Response, pu *util.ProxyUrl) *internal.Response {
	contentType := resp.ContentType

	if contentType.IsHTML() {
		return d.reWriteHTML(r, resp, pu)
	}

	if contentType.IsCss() {
		return d.reWriteCSS(r, resp, pu)
	}

	return resp
}

func (d *DoProxy) reWriteHTML(r *http.Request, resp *internal.Response, pu *util.ProxyUrl) *internal.Response {
	urlString := pu.GetUrlStr()
	body := pu.Extension.Rewrite(resp.Body)
	if pu.Extension.PreloadingNext() {
		go preLoader.PreLoad(pu, body, internal.PreLoadTypeNext)
	} else if pu.Extension.PreloadingSameDir() {
		go preLoader.PreLoad(pu, body, internal.PreLoadTypeSameDir)

	} else if pu.Extension.Preloading() {
		go preLoader.PreLoad(pu, body, internal.PreLoadTypeAll)
	}

	encodedBody := util.HTMLURLReplace(body, urlString, pu, r)
	encodedBody = util.CSSURLReplace(encodedBody, urlString, pu, r)

	var hBuf bytes.Buffer
	var needBR bool
	if pu.Extension.Has("raw_url") {
		hBuf.WriteString(`<a href="/?`)
		raw := pu.URLValues().Encode()
		hBuf.WriteString(raw)
		hBuf.WriteString(`">`)
		hBuf.WriteString(urlString)
		hBuf.WriteString("</a>")
		needBR = true
	}

	if pu.Extension.Cache() {
		hBuf.WriteString(`&nbsp;&nbsp;<a href="?no_cache=1">no_cache</a>`)
		needBR = true
	}
	if needBR {
		hBuf.WriteString("<br/>\n")
	}

	hBuf.Write(encodedBody)

	if pu.Extension.Has("ucss") {
		ru, erru := url.Parse(urlString)
		var hostName string
		if erru == nil {
			hostName = ru.Hostname()
		}
		ucss := []string{
			"/ucss/all.css",
			"/ucss/" + hostName + ".css",
		}
		for _, ufile := range ucss {
			if hasFile(ufile) {
				hBuf.WriteString(`<link rel="stylesheet" href="`)
				hBuf.WriteString(ufile)
				hBuf.WriteString(`" />`)
				hBuf.WriteString("\n")
			}
		}
		ujss := []string{
			"/ucss/jquery.min.js",
			"/ucss/all.js",
			"/ucss/" + hostName + ".js",
		}
		for _, ufile := range ujss {
			if hasFile(ufile) {
				hBuf.WriteString(`<script src="`)
				hBuf.WriteString(ufile)
				hBuf.WriteString(`" defer></script>`)
				hBuf.WriteString("\n")
			}
		}
	}

	resp.Body = hBuf.Bytes()
	return resp
}

func (d *DoProxy) reWriteCSS(r *http.Request, resp *internal.Response, pu *util.ProxyUrl) *internal.Response {
	urlString := pu.GetUrlStr()
	encodedBody := util.CSSURLReplace(resp.Body, urlString, pu, r)
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
		return 1 * time.Hour
	}

	ct := internal.ContentType(raw.Header.Get("Content-Type"))
	if ct.IsStaticFile() {
		return 12 * time.Hour
	}

	if ct.IsCss() {
		return time.Hour
	}

	path := raw.Request.URL.Path
	for _, e := range staticExts {
		if strings.HasSuffix(path, e) {
			return 2 * time.Hour
		}
	}
	return 0
}

var staticExts = []string{".png", ".jpg", ".jpeg", ".gif", ".css", ".js"}

func hasFile(name string) bool {
	info, err := os.Stat("./" + name)
	if err != nil {
		return false
	}
	return !info.IsDir() && info.Size() > 0
}
