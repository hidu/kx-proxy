// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/6/25

package internal

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hidu/kx-proxy/util"
)

const (
	PreLoadTypeAll     = "all"
	PreLoadTypeSameDir = "same"
	PreLoadTypeNext    = "next"
)

type PreLoad struct {
	FileCache *FileCache
}

func (rc *PreLoad) filterURLS(cacheType string, urlNow string, urls []string) []string {
	if cacheType == PreLoadTypeAll {
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

	if cacheType == PreLoadTypeNext {
		for _, u := range result {
			if strings.Compare(u, urlNow) > 0 {
				return []string{u}
			}
		}
	}

	return result
}

var ua = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36"

var Client = &http.Client{
	Timeout: 30 * time.Second,
}

const CacheMaxSize = 1024 * 1024

func (rc *PreLoad) Fetch(pu *util.ProxyUrl, preloadURL string) {
	logData := map[string]interface{}{
		"loaded": false,
	}

	defer func() {
		log.Println("Preloading ", preloadURL, logData)
	}()

	if !pu.Extension.NoCache() {
		// already has cache
		if rc.FileCache.Has(preloadURL) {
			logData["msg"] = "cache already exists"
			return
		}
	}

	req, err := http.NewRequest("GET", preloadURL, nil)
	if err != nil {
		logData["err"] = err.Error()
		return
	}

	req.Header.Set("User-Agent", ua)
	resp, err := Client.Do(req)
	if err != nil {
		logData["msg"] = fmt.Sprintf("get resp failed, err=%v", err)
		return
	}

	contentType := resp.Header.Get("Content-Type")

	logData["status"] = resp.StatusCode
	logData["content_len"] = resp.ContentLength
	logData["content_type"] = contentType

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		logData["skip_by"] = fmt.Sprintf("StatusCode=%d", resp.StatusCode)
		return
	}

	if !strings.HasPrefix(contentType, "text/") {
		logData["skip_by"] = "content-type:" + contentType
		return
	}

	buf := bytes.NewBuffer(nil)
	_, errCopy := io.Copy(buf, io.LimitReader(resp.Body, CacheMaxSize))
	if errCopy != nil {
		logData["err"] = err.Error()
		return
	}
	logData["body_len"] = buf.Len()
	cd := &CacheData{
		Header: map[string]string{
			"Content-Type": contentType,
		},
		Body: buf.Bytes(),
	}

	rc.FileCache.Set(preloadURL, cd)

	logData["loaded"] = true
}

func (rc *PreLoad) PreLoad(pu *util.ProxyUrl, body []byte, cacheType string) {
	defer func() {
		if re := recover(); re != nil {
			log.Printf("CacheAll panic:%v \n", re)
		}
	}()

	urlNow := pu.GetUrlStr()

	baseHref := util.BaseHref(body)
	urls := util.AllLinks(body, baseHref, urlNow)

	urls = rc.filterURLS(cacheType, urlNow, urls)

	if len(urls) == 0 {
		return
	}

	for _, u := range urls {
		rc.Fetch(pu, u)
	}
}
