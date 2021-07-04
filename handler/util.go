package handler

import (
	"net/http"
)

var copyHeaders = []string{"Referer", "Accept-Language", "Cookie"}

func copyHeader(dst, src http.Header) {
	for k, vs := range src {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}
