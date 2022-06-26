// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/8/7

package internal

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/fsgo/fsgo/fsnet/fsconn"
	"github.com/fsgo/fsgo/fsnet/fsdialer"
	"github.com/fsgo/fsgo/fsnet/fsresolver"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/hidu/kx-proxy/internal/metrics"
)

var Client = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := fsdialer.DialContext(ctx, network, addr)
			if err != nil || Dumper == nil {
				return conn, err
			}
			conn = fsconn.WithService("http_client", conn)
			return fsconn.WithInterceptor(conn, Dumper.ClientConnInterceptor()), nil
		},
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

	stIt := &fsconn.Interceptor{
		AfterRead: func(info fsconn.Info, b []byte, readSize int, err error) {
			readBytesCounter.Add(float64(readSize))
		},
		AfterWrite: func(info fsconn.Info, b []byte, wroteSize int, err error) {
			writeBytesCounter.Add(float64(wroteSize))
		},
	}

	dialIt := &fsdialer.Interceptor{
		DialContext: func(ctx context.Context, network string, address string, invoker fsdialer.DialContextFunc) (conn net.Conn, err error) {
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
			return fsconn.WithInterceptor(conn, stIt), nil
		},
	}
	fsdialer.MustRegisterInterceptor(dialIt)
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

	it := &fsresolver.Interceptor{
		LookupIP: func(ctx context.Context, network, host string, invoker fsresolver.LookupIPFunc) (ret []net.IP, err error) {
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
	fsresolver.MustRegisterInterceptor(it)
}

func init() {
	// cp:=&fsnet.ConnCopy{
	// 	WriterTo: os.Stdout,
	// 	ReadTo: os.Stdout,
	// }
	// fsnet.RegisterConnInterceptor(cp.Interceptor())
}
