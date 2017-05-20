package handler

import (
	"fmt"
	"github.com/hidu/kx-proxy/util"
	"net/http"
	"time"
)

func init() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/p/", proxyHandler)
	http.HandleFunc("/get/", getHandler)
	http.HandleFunc("/hello", helloHandler)
	http.HandleFunc("/favicon.ico", Assest.FileHandlerFunc("/assest/favicon.png"))
	http.Handle("/assest/", Assest.HTTPHandler("/"))
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := util.EncryptURL(fmt.Sprintf("%d", time.Now().Unix()))
	w.Write([]byte(t))
}
