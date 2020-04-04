package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/hidu/kx-proxy/util"
)

func init() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/p/", proxyHandler)
	http.HandleFunc("/get/", getHandler)
	http.HandleFunc("/hello", helloHandler)
	http.HandleFunc("/favicon.ico", Asset.FileHandlerFunc("/assest/favicon.png"))
	http.Handle("/asset/", Asset.HTTPHandler("/"))
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := util.EncryptURL(fmt.Sprintf("%d", time.Now().Unix()))
	w.Write([]byte(t))
}
