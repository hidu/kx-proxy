package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/xanygo/anygo/xattr"
	"github.com/xanygo/anygo/xcfg"
	"github.com/xanygo/ext"

	"github.com/hidu/kx-proxy/internal"
	"github.com/hidu/kx-proxy/internal/handler"
)

func init() {
	ext.Init()
}

var c = flag.String("conf", "conf/app.yml", "app main config file")

func main() {
	flag.Parse()
	flag.Parse()
	xattr.MustInitAppMain(*c, xcfg.Parse)

	handler.InitCache(filepath.Join(xattr.TempDir(), "cache"))
	internal.SetupDNS()

	proxy := handler.NewKxProxy()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	listen := xattr.AppMain().GetListen("main")
	l, err := net.Listen("tcp", listen)
	if err != nil {
		log.Fatalln(err)
	}

	ser := &http.Server{
		Handler: proxy,
	}
	go func() {
		sig := <-ch
		log.Println("received signal", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = ser.Shutdown(ctx)
	}()

	err = ser.Serve(l)

	log.Println("kx-proxy exit:", err)
}
