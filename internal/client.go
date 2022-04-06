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
	dialVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "net",
			Name:      "dial_seconds",
			Help:      "dial latency distributions.",
		},
		[]string{"status"},
	)
	metrics.DefaultReg.MustRegister(dialVec)

	readBytesCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "net",
		Name:      "conn_read_bytes",
	})
	metrics.DefaultReg.MustRegister(readBytesCounter)

	writeBytesCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "net",
		Name:      "conn_write_bytes",
	})
	metrics.DefaultReg.MustRegister(writeBytesCounter)

	stIt := &fsnet.ConnInterceptor{
		Read: func(b []byte, invoker func([]byte) (int, error)) (n int, err error) {
			defer func() {
				readBytesCounter.Add(float64(n))
			}()
			return invoker(b)
		},
		Write: func(b []byte, invoker func([]byte) (int, error)) (n int, err error) {
			defer func() {
				writeBytesCounter.Add(float64(n))
			}()
			return invoker(b)
		},
	}

	dialIt := &fsnet.DialerInterceptor{
		DialContext: func(ctx context.Context, network string, address string, invoker fsnet.DialContextFunc) (conn net.Conn, err error) {
			tm := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
				if err == nil {
					dialVec.WithLabelValues("success").Observe(v)
				} else {
					dialVec.WithLabelValues("fail").Observe(v)
				}
			}))

			conn, err = invoker(ctx, network, address)

			tm.ObserveDuration()
			if err != nil {
				return nil, err
			}
			return fsnet.WrapConn(conn, stIt), nil
		},
	}
	fsnet.MustRegisterDialerInterceptor(dialIt)
}

func init() {
	resolverVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "net",
			Name:      "lookupip_seconds",
			Help:      "Resolver LookupIP latency distributions.",
		},
		[]string{"status"},
	)
	metrics.DefaultReg.MustRegister(resolverVec)

	it := &fsnet.ResolverInterceptor{
		LookupIP: func(ctx context.Context, network, host string, invoker fsnet.LookupIPFunc) (ret []net.IP, err error) {
			tm := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
				if err == nil {
					resolverVec.WithLabelValues("success").Observe(v)
				} else {
					resolverVec.WithLabelValues("fail").Observe(v)
				}
			}))

			defer tm.ObserveDuration()

			return invoker(ctx, network, host)
		},
	}
	fsnet.MustRegisterResolverInterceptor(it)
}

func init() {
	// cp:=&fsnet.ConnDuplicate{
	// 	WriterTo: os.Stdout,
	// 	ReadTo: os.Stdout,
	// }
	// fsnet.RegisterConnInterceptor(cp.ConnInterceptor())
}
