package handler

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hidu/kx-proxy/internal"
	"github.com/hidu/kx-proxy/util"
)

var kxKey = "KxKey"

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	logData := make(map[string]interface{})
	reqStart := time.Now()
	defer func() {
		used := time.Now().Sub(reqStart)
		log.Println("remote:", r.RemoteAddr, "path:", r.URL.Path, "used:", used, logData)
	}()

	r.Header.Del("Connection")
	encodedURL := r.URL.Path[len("/p/"):]

	kxURL := r.Header.Get("kx_url")
	if kxURL != "" {
		encodedURL = kxURL
	}
	r.Header.Del("kx_url")

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

	var contentType internal.ContentType
	var statusCode int
	var body []byte

	cd := fileCache.Get(urlString)
	if cd != nil {
		logData["from_cache"] = 1
		statusCode = 200
		contentType = cd.ContentType()
		body = cd.Body
		for k, v := range cd.Header {
			w.Header().Add(k, v)
		}
	} else {
		logData["from_cache"] = 0
		req, _ := http.NewRequest(r.Method, urlString, r.Body)
		req.Header.Set("Content-Type", r.Header.Get("Content-Type"))
		req.Header.Set("User-Agent", r.Header.Get("User-Agent"))
		resp, err1 := internal.Client.Do(req)
		if err1 != nil {
			logData["emsg"] = "fetch_failed:" + err1.Error()
			http.Error(w, "Error Fetching "+urlString+"\n"+err1.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		contentType = internal.ContentType(resp.Header.Get("Content-Type"))
		// Write all remote resp header to client
		for headerKey, vs := range resp.Header {
			for _, v := range vs {
				w.Header().Add(headerKey, v)
			}
		}

		logData["resp_status"] = resp.StatusCode

		if resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusFound {
			location := resp.Header.Get("Location")
			if location != "" {
				pu.SwitchUrl(location)
				encodedURL, err := pu.Encode()
				if err != nil {
					logData["emsg"] = "Location_build_url failed" + err.Error()
					w.Write([]byte("build url failed:" + err.Error()))
					return
				}
				http.Redirect(w, r, "/p/"+encodedURL, 302)
				return
			}
		}
		statusCode = resp.StatusCode

		// 缓存静态资源
		if life := staticCacheTime(resp); life > 0 {
			body, err = ioutil.ReadAll(resp.Body)
			if err == nil && len(body) > 0 {
				ncd := &internal.CacheData{
					Header: map[string]string{
						"Content-Type": string(contentType),
					},
					Body: body,
				}
				fileCache.SetWithTTL(urlString, ncd, life)
				logData["static"] = "save_cache"
			}
		} else {
			if contentType.IsHTML() || contentType.IsCss() {
				body, _ = ioutil.ReadAll(resp.Body)
			} else {
				w.WriteHeader(resp.StatusCode)
				copySize, copyErr := io.Copy(w, resp.Body)
				logData["io_copy_size"] = copySize
				logData["io_copy_err"] = copyErr
				return
			}
		}
	}

	// Rewrite all urls
	if contentType.IsHTML() {
		body = pu.Extension.Rewrite(body)
		if pu.Extension.PreloadingNext() {
			go preLoader.PreLoad(body, urlString, internal.PreLoadTypeNext)
		} else if pu.Extension.PreloadingSameDir() {
			go preLoader.PreLoad(body, urlString, internal.PreLoadTypeSameDir)

		} else if pu.Extension.Preloading() {
			go preLoader.PreLoad(body, urlString, internal.PreLoadTypeAll)
		}

		encodedBody := util.HTMLURLReplace(body, urlString, pu, r)
		encodedBody = util.CSSURLReplace(encodedBody, urlString, pu, r)

		var hBuf bytes.Buffer
		if pu.Extension.Has("raw_url") {
			raw := url.QueryEscape(urlString)
			hBuf.WriteString(`<a href="/?raw=`)
			hBuf.WriteString(raw)
			hBuf.WriteString(`" target='_blank'>`)
			hBuf.WriteString(urlString)
			hBuf.WriteString("</a>")
			hBuf.WriteString("<br/>")
		}
		hBuf.Write(encodedBody)

		if pu.Extension.Has("ucss") {
			hBuf.WriteString(`<link rel="stylesheet" href="/ucss/all.css" ?>`)
			ru, erru := url.Parse(urlString)
			if erru == nil {
				hBuf.WriteString(`<link rel="stylesheet" href="/ucss/`)
				hBuf.WriteString(ru.Hostname())
				hBuf.WriteString(`.css" />`)
			}
		}

		w.Header().Set("Content-Length", strconv.Itoa(hBuf.Len()))
		w.WriteHeader(statusCode)
		w.Write(hBuf.Bytes())
		return
	}

	if contentType.IsCss() {
		encodedBody := util.CSSURLReplace(body, urlString, pu, r)
		w.Header().Set("Content-Length", strconv.Itoa(len(encodedBody)))
		w.WriteHeader(statusCode)
		w.Write(encodedBody)
		return
	}

	w.WriteHeader(statusCode)
	w.Write(body)
}

func staticCacheTime(resp *http.Response) time.Duration {
	if resp.Header.Get("ETag") != "" &&
		resp.ContentLength > 0 &&
		resp.ContentLength < internal.CacheMaxSize {
		return 12 * time.Hour
	}

	if resp.Header.Get("X-Cache-Status") == "HIT" {
		return 12 * time.Hour
	}

	ct := internal.ContentType(resp.Header.Get("Content-Type"))
	if ct.IsStaticFile() {
		return 12 * time.Hour
	}

	if ct.IsCss() {
		return time.Hour
	}

	path := resp.Request.URL.Path
	for _, e := range staticExts {
		if strings.HasSuffix(path, e) {
			return 6 * time.Hour
		}
	}

	return 0
}

var staticExts = []string{".png", ".jpg", ".jpeg", ".gif", ".css", ".js"}
