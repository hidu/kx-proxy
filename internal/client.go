// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/8/7

package internal

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/fsgo/fsgo/fsnet"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/hidu/kx-proxy/internal/metrics"
)

var Client = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           fsnet.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       10 * time.Second,
		TLSHandshakeTimeout:   20 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

func init() {
	dialSum := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: "net",
			Name:      "dial_seconds",
			Help:      "dial latency distributions.",
			Objectives: map[float64]float64{
				0.5:  0.05,
				0.9:  0.01,
				0.99: 0.001,
			},
		},
		[]string{"status"},
	)
	metrics.DefaultReg.MustRegister(dialSum)

	hk := &fsnet.DialerHook{
		DialContext: func(ctx context.Context, network string, address string, fn fsnet.DialContextFunc) (conn net.Conn, err error) {
			tm := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
				if err == nil {
					dialSum.WithLabelValues("success").Observe(v)
				} else {
					dialSum.WithLabelValues("fail").Observe(v)
				}
			}))
			defer tm.ObserveDuration()
			return fn(ctx, network, address)
		},
	}
	fsnet.MustRegisterDialerHook(hk)
}

func init() {
	resolverSum := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: "net",
			Name:      "lookupip_seconds",
			Help:      "Resolver LookupIP latency distributions.",
			Objectives: map[float64]float64{
				0.5:  0.05,
				0.9:  0.01,
				0.99: 0.001,
			},
		},
		[]string{"status"},
	)
	metrics.DefaultReg.MustRegister(resolverSum)

	hook := &fsnet.ResolverHook{
		LookupIP: func(ctx context.Context, network, host string, fn fsnet.LookupIPFunc) (ret []net.IP, err error) {
			tm := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
				if err == nil {
					resolverSum.WithLabelValues("success").Observe(v)
				} else {
					resolverSum.WithLabelValues("fail").Observe(v)
				}
			}))

			defer tm.ObserveDuration()

			return fn(ctx, network, host)
		},
	}
	fsnet.MustRegisterResolverHook(hook)
}
