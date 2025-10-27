// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/7/3

package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/xanygo/anygo/xhttp"

	"github.com/hidu/kx-proxy/internal/links"
)

var _ http.Handler = (*KxProxy)(nil)

func NewKxProxy() *KxProxy {
	p := &KxProxy{
		router: xhttp.NewRouter(),
	}
	p.init()
	return p
}

type KxProxy struct {
	router *xhttp.Router
}

func (k *KxProxy) init() {
	k.router.HandleFunc("/", k.handlerHome)
	k.router.GetFunc("/p/*", k.handlerProxy)
	k.router.GetFunc("/get/*", k.handlerGet)
	k.router.HandleFunc("/hello", handlerHello)
	k.router.GetFunc("/ucss/*", k.handlerUserCSS)
	k.router.GetFunc("/asset/*", k.handlerAsset)
	k.router.HandleFunc("/favicon.ico", k.handlerFavicon)
}

func (k *KxProxy) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	k.router.ServeHTTP(writer, request)
}

func handlerHello(w http.ResponseWriter, r *http.Request) {
	t, _ := links.EncryptURL(strconv.FormatInt(time.Now().Unix(), 10))
	_, _ = w.Write([]byte(t))
}
