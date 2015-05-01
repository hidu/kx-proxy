package main

import (
	"encoding/base64"
	"flag"
	"github.com/elazarl/goproxy"
	"log"
	"net/http"
	"net/url"
	"strings"
)
var proxy=flag.String("proxy","https://vast-scrubland-8450.herokuapp.com","gopee proxy url base")
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
	if(strings.HasPrefix(urlOld,*proxy)){
		return r,nil
	}
	var urlReq = base64.StdEncoding.EncodeToString([]byte(urlOld))
	urlNew := *proxy+"/p/" + urlReq
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
