package handler

import (
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/hidu/kx-proxy/internal"
)

// handle url http://aaa.com/get/?url=http://www.baidu.com/
func (k *KxProxy) handlerGet(w http.ResponseWriter, r *http.Request) {
	var cusURL string
	if strings.HasPrefix(r.URL.RawQuery, "url=") {
		cusURL = r.URL.RawQuery[4:]
	}
	req, err := http.NewRequest(r.Method, cusURL, r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	req.Header.Set("HTTP_X_FORWARDED_FOR", host)
	resp, err := internal.GetClient(false).Do(req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
