// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/7/3

package handler

import (
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/hidu/kx-proxy/internal/links"
	"github.com/hidu/kx-proxy/internal/metrics"
)

var _ http.Handler = (*KxProxy)(nil)

func NewKxProxy() *KxProxy {
	p := &KxProxy{
		router: mux.NewRouter(),
	}
	p.init()
	return p
}

type KxProxy struct {
	router *mux.Router
}

func (k *KxProxy) init() {
	k.router.Use(metricsMiddleware)

	k.router.HandleFunc("/", k.handlerHome)
	k.router.PathPrefix("/p/").HandlerFunc(k.handlerProxy)
	k.router.PathPrefix("/get/").HandlerFunc(k.handlerGet)
	k.router.HandleFunc("/hello", handlerHello)
	k.router.PathPrefix("/ucss/").HandlerFunc(k.handlerUserCSS)
	k.router.HandleFunc("/favicon.ico", k.handlerFavicon)
	k.router.PathPrefix("/asset/").HandlerFunc(k.handlerAsset)

	metricsHandler := promhttp.HandlerFor(metrics.DefaultReg, promhttp.HandlerOpts{})
	k.router.Handle("/metrics", metricsHandler)
}

func (k *KxProxy) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	k.router.ServeHTTP(writer, request)
}

func handlerHello(w http.ResponseWriter, r *http.Request) {
	t, _ := links.EncryptURL(fmt.Sprintf("%d", time.Now().Unix()))
	_, _ = w.Write([]byte(t))
}

var pvVec *prometheus.HistogramVec

func init() {
	pvVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "server_requests_seconds",
	}, []string{"method", "api"})

	metrics.DefaultReg.MustRegister(pvVec)

	c1 := collectors.NewGoCollector()
	metrics.DefaultReg.MustRegister(c1)

	c2 := collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})
	metrics.DefaultReg.MustRegister(c2)
}

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		cost := time.Since(start)

		pathInfo := strings.SplitN(path.Clean(r.URL.Path), "/", 3)
		api := pathInfo[1]
		if api == "" {
			api = "home"
		}
		pvVec.WithLabelValues(r.Method, api).Observe(cost.Seconds())
	})
}
