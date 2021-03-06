package handler

import (
	"net/http"
	"time"

	"github.com/hidu/kx-proxy/util"
)

// BodyStreamEnc 作为服务端代理时时候对输出的内容进行加密处理
var BodyStreamEnc = false

var copyHeaders = []string{"Referer", "Accept-Language", "Cookie"}

func copyHeader(dst, src http.Header) {
	for k, vs := range src {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}

var startTime = time.Now()

var secretKeys = make(map[string]int)

func init() {
	keys := util.LoadTxtConf("keys.txt")
	for _, key := range keys {
		secretKeys[key] = 1
	}
}
