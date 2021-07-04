package handler

import (
	"io"
	"net/http"
	"strings"
)

//
// handle url http://aaa.com/get/?url=http://www.baidu.com/
func (k *KxProxy) handlerGet(w http.ResponseWriter, r *http.Request) {
	cusURL := ""
	if strings.HasPrefix(r.URL.RawQuery, "url=") {
		cusURL = r.URL.RawQuery[4:]
	}
	req, err := http.NewRequest(r.Method, cusURL, r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	addrInfo := strings.Split(r.RemoteAddr, ":")
	req.Header.Set("HTTP_X_FORWARDED_FOR", addrInfo[0])
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(err.Error()))
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
