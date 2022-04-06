// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/6/25

package internal

import (
	"io"
	"net/http"
	"strconv"
	"strings"
)

type ContentType string

func (ct ContentType) IsHTML() bool {
	return ct.Is("text/html")
}

func (ct ContentType) IsCSS() bool {
	return ct.Is("text/css")
}

func (ct ContentType) Is(name string) bool {
	return strings.Contains(string(ct), name)
}

func (ct ContentType) IsText() bool {
	return strings.Contains(string(ct), "text/") ||
		ct.Is("/javascript")
}

func (ct ContentType) CanCache() bool {
	return ct.IsText() || ct.IsStaticFile()
}

var staticTypes = []string{
	"text/css",
	"image/",
	"/javascript",
}

func (ct ContentType) IsStaticFile() bool {
	for _, st := range staticTypes {
		if ct.Is(st) {
			return true
		}
	}
	return true
}

type LogData map[string]interface{}

func (l LogData) Merge(b LogData) {
	for k, v := range b {
		l[k] = v
	}
}

type Response struct {
	StatusCode  int
	ContentType ContentType
	Header      http.Header
	HeaderMap   map[string]string
	Body        []byte

	Raw *http.Response
}

func (resp *Response) HeaderGet(name string) string {
	if len(resp.Header) > 0 {
		if got := resp.Header.Get(name); got != "" {
			return got
		}
	}
	if len(resp.HeaderMap) > 0 {
		if got := resp.HeaderMap[name]; got != "" {
			return got
		}
	}
	if resp.Raw != nil {
		return resp.Raw.Header.Get(name)
	}
	return ""
}

func (resp *Response) WriteTo(w http.ResponseWriter) (int64, error) {
	if resp.Raw != nil {
		defer resp.Raw.Body.Close()
	}

	if len(resp.Header) > 0 {
		for k, vs := range resp.Header {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
	}

	if len(resp.HeaderMap) > 0 {
		for k, v := range resp.HeaderMap {
			w.Header().Set(k, v)
		}
	}

	if resp.Body != nil {
		w.Header().Set("Content-Length", strconv.Itoa(len(resp.Body)))
		w.WriteHeader(resp.StatusCode)
		n, err := w.Write(resp.Body)
		return int64(n), err
	}

	w.WriteHeader(resp.StatusCode)
	return io.Copy(w, resp.Raw.Body)
}
