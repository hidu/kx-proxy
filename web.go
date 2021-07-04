package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/hidu/kx-proxy/handler"
)

var addr = flag.String("addr", ":8085", "listen addr,eg :8085")
var cd = flag.String("cache_dir", "./cache/", "cache dir")

func main() {
	flag.Parse()

	handler.InitCache(*cd)
	log.Println("kx-proxy listening on :", *addr)

	proxy := handler.New()
	err := http.ListenAndServe(*addr, proxy)

	log.Println("kx-proxy exit:", err)
}
