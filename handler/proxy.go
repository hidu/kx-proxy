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
	"sync"
	"time"

	"github.com/hidu/kx-proxy/util"
)

var kxKey = "KxKey"
var cache = newRespCache()

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	logData := make(map[string]interface{})
	startTime := time.Now()
	defer func() {
		used := time.Now().Sub(startTime)
		log.Println(
			"remote:", r.RemoteAddr,
			"path:", r.URL.Path,
			"used:", used,
			logData)
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

	req, _ := http.NewRequest(r.Method, urlString, r.Body)
	req.Header.Set("Content-Type", r.Header.Get("Content-Type"))
	req.Header.Set("User-Agent", r.Header.Get("User-Agent"))

	var resp *http.Response

	resp = cache.Get(req)
	if resp != nil {
		logData["from_cache"] = 1
	}

	if resp == nil {
		resp, err = http.DefaultTransport.RoundTrip(req)

		if err != nil {
			logData["emsg"] = "fetch_failed:" + err.Error()
			http.Error(w, "Error Fetching "+urlString+"\n"+err.Error(), http.StatusBadGateway)
			return
		}
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")

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

	// Rewrite all urls
	if strings.Contains(contentType, "text/html") {
		body, _ := ioutil.ReadAll(resp.Body)

		body = pu.Extension.Rewrite(body)
		if pu.Extension.PreloadingNext() {
			go cache.Cache(body, urlString, req, cacheTypeNext)
		} else if pu.Extension.PreloadingSameDir() {
			go cache.Cache(body, urlString, req, cacheTypeSameDir)

		} else if pu.Extension.Preloading() {
			go cache.Cache(body, urlString, req, cacheTypeAll)
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
		w.WriteHeader(resp.StatusCode)
		w.Write(hBuf.Bytes())
	} else if strings.Contains(contentType, "text/css") {
		body, _ := ioutil.ReadAll(resp.Body)

		encodedBody := util.CSSURLReplace(body, urlString, pu, r)

		w.Header().Set("Content-Length", strconv.Itoa(len(encodedBody)))
		w.WriteHeader(resp.StatusCode)
		w.Write(encodedBody)
	} else {
		w.WriteHeader(resp.StatusCode)
		copySize, copyErr := io.Copy(w, resp.Body)
		logData["io_copy_size"] = copySize
		logData["io_copy_err"] = copyErr
	}
}

func newRespCache() *respCache {
	cd := &respCache{
		datas: make(map[string]*cacheData),
	}
	go cd.gc()
	return cd
}

type cacheData struct {
	Resp *http.Response
	Tm   time.Time
}

func (cd *cacheData) IsExpire() bool {
	return cd.Tm.Unix() < time.Now().Unix()-5*60
}

type respCache struct {
	datas map[string]*cacheData
	lock  sync.RWMutex
}

func (rc *respCache) Get(r *http.Request) *http.Response {
	key := rc.cacheKey(r)
	rc.lock.RLock()
	data := rc.datas[key]
	defer rc.lock.RUnlock()
	if data == nil {
		return nil
	}
	delete(rc.datas, key)
	return data.Resp
}

func (rc *respCache) gc() {
	tk := time.NewTicker(10 * time.Minute)
	for range tk.C {
		rc.checkExpire()
	}
}

func (rc *respCache) checkExpire() {
	rc.lock.RLock()
	var eKeys []string
	for key, data := range rc.datas {
		if data.IsExpire() {
			eKeys = append(eKeys, key)
		}
	}
	log.Printf("checkExpire total cache %d, expire=%d\n", len(rc.datas), len(eKeys))
	rc.lock.RUnlock()

	for _, key := range eKeys {
		rc.lock.Lock()
		delete(rc.datas, key)
		rc.lock.Unlock()
	}
}

func (rc *respCache) cacheKey(r *http.Request) string {
	return r.URL.String()
}

const cacheMaxSize = 1024 * 1024
const (
	cacheTypeAll     = "all"
	cacheTypeSameDir = "same"
	cacheTypeNext    = "next"
)

func (rc *respCache) filterURLS(cacheType string, urlNow string, urls []string) []string {
	if cacheType == cacheTypeAll {
		return urls
	}

	pos := strings.LastIndex(urlNow, "/")
	prefix := urlNow[0:pos]
	var result []string
	for _, u := range urls {
		if !strings.HasSuffix(u, "/") && strings.HasPrefix(u, prefix) {
			result = append(result, u)
		}
	}

	if cacheType == cacheTypeNext {
		for _, u := range result {
			if strings.Compare(u, urlNow) > 0 {
				return []string{u}
			}
		}
	}

	return result
}

func (rc *respCache) Cache(body []byte, urlNow string, reqRaw *http.Request, cacheType string) {
	defer func() {
		if re := recover(); re != nil {
			log.Printf("CacheAll panic:%v \n", re)
		}
	}()

	baseHref := util.BaseHref(body)
	urls := util.AllLinks(body, baseHref, urlNow)

	urls = rc.filterURLS(cacheType, urlNow, urls)

	if len(urls) == 0 {
		return
	}

	for _, u := range urls {
		rc.fetch(u, reqRaw)
	}
}

func (rc *respCache) fetch(preloadURL string, reqRaw *http.Request) {
	logData := map[string]interface{}{
		"loaded": false,
	}

	defer func() {
		log.Println("Preloading ", preloadURL, logData)
	}()

	req, err := http.NewRequest("GET", preloadURL, nil)
	if err != nil {
		log.Println("build request failed, error=", err)
		return
	}

	req.Header.Set("User-Agent", reqRaw.Header.Get("User-Agent"))

	key := rc.cacheKey(req)
	rc.lock.RLock()
	vHas := rc.datas[key]
	rc.lock.RUnlock()

	// already has cache
	if vHas != nil {
		log.Println("cache already has key=", key)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("get resp failed, err=", err)
		return
	}

	contentType := resp.Header.Get("Content-Type")

	logData["status"] = resp.StatusCode
	logData["content_len"] = resp.ContentLength
	logData["content_type"] = contentType

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return
	}

	if !strings.HasPrefix(contentType, "text/") {
		return
	}

	buf := bytes.NewBuffer(nil)
	_, errCopy := io.Copy(buf, io.LimitReader(resp.Body, cacheMaxSize))
	if errCopy != nil {
		return
	}
	logData["body_len"] = buf.Len()

	resp.Body = ioutil.NopCloser(buf)
	rc.lock.Lock()
	rc.datas[key] = &cacheData{
		Resp: resp,
		Tm:   time.Now(),
	}
	rc.lock.Unlock()

	logData["loaded"] = true
}
