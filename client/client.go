package main

import (
	"encoding/base64"
	"flag"
	"github.com/elazarl/goproxy"
	"log"
	"net/http"
	"net/url"
)

func main() {
	verbose := flag.Bool("v", false, "should every proxy request be logged to stdout")
	addr := flag.String("addr", ":8080", "proxy listen address")
	flag.Parse()
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = *verbose
	proxy.OnRequest().DoFunc(requestHanderFunc)
	log.Fatal(http.ListenAndServe(*addr, proxy))
}

func requestHanderFunc(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	urlOld := r.URL.String()
	var urlReq = base64.StdEncoding.EncodeToString([]byte(urlOld))
	urlNew := "https://vast-scrubland-8450.herokuapp.com/p/" + urlReq
	log.Println(urlOld, "--->", urlNew)
	var err error
	r.URL, err = url.Parse(urlNew)
	r.Host = r.URL.Host
	r.Header.Add("is_client", "1")
	if err != nil {
		log.Println("parse new url failed", err)
	}
	return r, nil
}
