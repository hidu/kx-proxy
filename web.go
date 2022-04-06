package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fsgo/fsgo/fsfs"
	"github.com/fsgo/fsgo/fsos"

	"github.com/hidu/kx-proxy/internal"
	"github.com/hidu/kx-proxy/internal/handler"
)

var addr = flag.String("addr", "127.0.0.1:8085", "listen addr,eg :8085")
var cd = flag.String("cache_dir", "./cache/", "cache dir")
var alog = flag.String("log", "./log/kx.log", "log file")

// DNS 配置文件，若文件不存在将跳过
var dnsConf = flag.String("dns", "./conf/dns.toml", "dns group config file")

func main() {
	flag.Parse()

	setupLogFile(*alog)

	handler.InitCache(*cd)
	log.Println("kx-proxy listening on :", *addr)

	internal.SetupDNS(*dnsConf)

	proxy := handler.NewKxProxy()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ser := &http.Server{Addr: *addr, Handler: proxy}
	go func() {
		sig := <-ch
		log.Println("received signal", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = ser.Shutdown(ctx)
	}()

	err := ser.ListenAndServe()

	log.Println("kx-proxy exit:", err)
}

func setupLogFile(fp string) {
	af := &fsfs.Rotator{
		Path:     fp,
		ExtRule:  "1hour",
		MaxFiles: 24,
	}
	af.AfterChange(func(f *os.File) {
		_ = fsos.HookStderr(f)
	})
	if err := af.Init(); err != nil {
		panic(err)
	}
}
