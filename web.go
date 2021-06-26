package main

//go:generate goasset

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/hidu/kx-proxy/handler"
)

var addr = flag.String("addr", ":8085", "listen addr,eg :8085")
var cd = flag.String("cache_dir", "./cache/", "cache dir")

func init() {
	flag.BoolVar(&handler.BodyStreamEnc, "enc", false, "encrypts the body stream")
}

func main() {
	flag.Parse()

	var httpHost = os.Getenv("HOST")
	var httpPort = os.Getenv("PORT")

	laddr := httpHost + ":" + httpPort

	if httpPort == "" {
		laddr = *addr
	}

	if len(laddr) < 2 {
		fmt.Println("listening addr [", laddr, "] is wrong")
		os.Exit(1)
	}
	handler.InitCache(*cd)

	fmt.Printf("kx-proxy listening on :%s\n", laddr)
	err := http.ListenAndServe(laddr, nil)
	fmt.Println("exit with err:", err)
}
