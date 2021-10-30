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

	"github.com/fsgo/fsgo/fsos"
	"github.com/fsgo/fsgo/fsos/fsfs"

	"github.com/hidu/kx-proxy/internal"
	"github.com/hidu/kx-proxy/internal/handler"
)

var addr = flag.String("addr", "127.0.0.1:8085", "listen addr,eg :8085")
var cd = flag.String("cache_dir", "./cache/", "cache dir")
var alog = flag.String("log", "./log/kx.log", "log file")

func main() {
	flag.StringVar(&internal.NameServer, "nameserver", "114.114.114.114,1.1.1.1", "nameserver list")
	flag.StringVar(&internal.DNSBlockFile, "dns_block", "./conf/dns_block.txt", "dns block file")

	flag.Parse()

	setupLogFile(*alog)

	handler.InitCache(*cd)
	log.Println("kx-proxy listening on :", *addr)

	internal.SetUpDNS()

	proxy := handler.NewKxProxy()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ser := &http.Server{Addr: *addr, Handler: proxy}
	go func() {
		sig := <-ch
		log.Println("received signal", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ser.Shutdown(ctx)
	}()

	err := ser.ListenAndServe()

	log.Println("kx-proxy exit:", err)
}

func setupLogFile(fp string) {
	af := &fsfs.Rotator{
		Path:     fp,
		ExtRule:  "1hour",
		MaxFiles: 72,
	}
	af.AfterChange(func(f *os.File) {
		fsos.HookStderr(f)
	})
	if err := af.Init(); err != nil {
		panic(err)
	}
}
