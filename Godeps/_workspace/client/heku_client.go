package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"github.com/elazarl/goproxy"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var confPath = flag.String("conf", "conf.json", "json conf")
var verbose = flag.Bool("v", false, "should every proxy request be logged to stdout")
var addr = flag.String("addr", ":8080", "proxy listen address")

var reHTML = regexp.MustCompile("src=[\"\\'](.*?)[\"\\']|href=[\"\\'](.*?)[\"\\']|action=[\"\\'](.*?)[\"\\']")

type ClientConf struct {
	Proxies     []ProxyItem `json:"proxy"`
	Proxy_All   []string
	Total       int
	CDN_ALL     map[string]string `json:"cdn"`
	ParentProxy string            `json:"parent"`
	SSlOn	    string `json:"ssl"`
}

type ProxyItem struct {
	Url    string `json:"url"`
	Weight int    `json:"weight"`
}

func (conf *ClientConf) GetOneHost() string {
	if conf.Total < 1 {
		return ""
	}
	n := rand.Int() % conf.Total
	return conf.Proxy_All[n]
}

func (conf *ClientConf) IsProxyHost(urlClient string) bool {
	urlClient = strings.ToLower(urlClient)
	for _, host := range conf.Proxies {
		if strings.HasPrefix(urlClient, host.Url) {
			return true
		}
	}
	return false
}

var conf *ClientConf

func loadConf() {
	data, err := ioutil.ReadFile(*confPath)
	if err != nil {
		log.Fatalln("load json conf failed,err:", err)
	}
	err = json.Unmarshal(data, &conf)
	if err != nil {
		log.Fatalln("parse json conf failed,err:", err)
	}
	if len(conf.Proxies) < 1 {
		log.Fatalln("no hosts")
	}
	for _, item := range conf.Proxies {
		if item.Weight < 1 {
			log.Println("skip ",item.Url)
			continue
		}
		for i := 0; i < item.Weight; i++ {
			conf.Proxy_All = append(conf.Proxy_All, strings.TrimRight(item.Url, "/"))
		}
	}
	conf.Total = len(conf.Proxy_All)
	rand.Seed(time.Now().Unix())
	log.Println("load conf success")
}

var MitmConnect = &goproxy.ConnectAction{
	Action:    goproxy.ConnectMitm,
	TLSConfig: goproxy.TLSConfigFromCA(&GoproxyCa),
}


var AlwaysMitm goproxy.FuncHttpsHandler = func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
	log.Println("https conn", host, ctx.Req.URL.String())
	return MitmConnect, host
}

var HttpMitmConnect = &goproxy.ConnectAction{
	Action:    goproxy.ConnectHTTPMitm,
}
var AlwaysHttpMitm goproxy.FuncHttpsHandler = func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
	log.Println("https conn", host, ctx.Req.URL.String())
	return HttpMitmConnect, host
}

func responseHanderFunc(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	if resp != nil {
		resp.Header.Set("Connection", "close")
	}
	return resp
}

func copyHeaders(dst, src http.Header) {
	for k, vs := range src {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}

func requestHanderFunc(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	urlOld := r.URL.String()
	log.Println("url->", urlOld)
	r.Header.Set("Connection","Close")
	r.Header.Del("Proxy-Connection")

	if conf.IsProxyHost(urlOld) {
		log.Println(urlOld, "direct")
		return r, nil
	}
	if hostCdn, has := conf.CDN_ALL[r.URL.Host]; has {
		r.URL.Host = hostCdn
		r.Host = hostCdn
		log.Println(urlOld, "<use cdn>", r.URL.String())
		return r, nil
	}

	var urlReq = base64.StdEncoding.EncodeToString([]byte(urlOld))
	urlNew := conf.GetOneHost() + "/p/" + urlReq
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

func main() {
	flag.Parse()
	loadConf()

	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = *verbose
	if(conf.SSlOn=="on"){
		proxy.OnRequest().HandleConnectFunc(AlwaysMitm)
	}else{
		proxy.OnRequest().HandleConnectFunc(AlwaysHttpMitm)
	}
	proxy.OnRequest().DoFunc(requestHanderFunc)
	proxy.OnResponse().DoFunc(responseHanderFunc)
	if conf.ParentProxy != "" {
		proxy.Tr = &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return url.Parse(conf.ParentProxy)
			},
		}
	}
	log.Println("proxy client listen at ", *addr)
	err := http.ListenAndServe(*addr, proxy)
	log.Fatal(err)
}
