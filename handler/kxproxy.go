// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/7/3

package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/hidu/kx-proxy/util"
)

var _ http.Handler = (*KxProxy)(nil)

func New() *KxProxy {
	p := &KxProxy{
		router: http.NewServeMux(),
	}
	p.init()
	return p
}

type KxProxy struct {
	router *http.ServeMux
}

func (k *KxProxy) init() {
	k.router.HandleFunc("/", k.handlerHome)
	k.router.HandleFunc("/p/", k.handlerProxy)
	k.router.HandleFunc("/get/", k.handlerGet)
	k.router.HandleFunc("/hello", handlerHello)
	k.router.HandleFunc("/ucss/", k.handlerUcss)
	k.router.HandleFunc("/favicon.ico", k.handlerFavicon)
	k.router.HandleFunc("/asset/", k.handlerAsset)
}

func (k *KxProxy) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	k.router.ServeHTTP(writer, request)
}

func handlerHello(w http.ResponseWriter, r *http.Request) {
	t, _ := util.EncryptURL(fmt.Sprintf("%d", time.Now().Unix()))
	w.Write([]byte(t))
}
