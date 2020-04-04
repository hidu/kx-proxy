package util

import (
	"net/http"
	// 	"fmt"
)

var encHeaders = map[string]int{"Cookie": 1, "User-Agent": 1, "Referer": 1, "Authorization": 1, "Set-Cookie": 1, "Refresh": 1, "Content-Encoding": 1}

// HeaderEnc 对http header中的部分字段进行加密
func HeaderEnc(header http.Header) {
	for k, vs := range header {
		_, enc := encHeaders[k]
		for i, v := range vs {
			if enc && v != "" {
				vs[i], _ = EncryptURL(v)
			}
		}
		header[k] = vs
	}
}

// HeaderDec 对 header进行解密
func HeaderDec(header http.Header) {
	for k, vs := range header {
		_, enc := encHeaders[k]
		for i, v := range vs {
			if enc && v != "" {
				vs[i], _ = DecryptURL(v)
			}
		}
		header[k] = vs
	}
}
