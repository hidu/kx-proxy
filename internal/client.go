// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/8/7

package internal

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/fsgo/fsgo/fsnet/fsconn"
	"github.com/fsgo/fsgo/fsnet/fsdialer"
	"github.com/fsgo/fsgo/fsnet/fsresolver"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/hidu/kx-proxy/internal/metrics"
)

var client1 = newClient(false)
var client2 = newClient(true)

func GetClient(skipVerify bool) *http.Client {
	if skipVerify {
		return client2
	}
	return client1
}

var allowVisitVlan bool

func init() {
	flag.BoolVar(&allowVisitVlan, "private", false, "allow visit local network, e.g. 127.*,10.*,192.168.*")
}

func inPrivateAddr(addr net.Addr) bool {
	ip := net.ParseIP(addr.String())
	return ip.IsLoopback() || ip.IsPrivate()
}

func newClient(skipVerify bool) *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				conn, err := fsdialer.DialContext(ctx, network, addr)
				// 禁止连接到本地 IP
				if conn != nil && !allowVisitVlan && inPrivateAddr(conn.LocalAddr()) {
					_ = conn.Close()
					return nil, fmt.Errorf("forbidden, cannot connect to %s", conn.LocalAddr().String())
				}
				if err != nil || Dumper == nil {
					return conn, err
				}
				conn = fsconn.WithService("http_client", conn)
				return fsconn.Wrap(conn, Dumper.ClientConnInterceptor()), nil
			},
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          10,
			IdleConnTimeout:       10 * time.Second,
			TLSHandshakeTimeout:   20 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: skipVerify,
			},
		},
	}
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

	type dialCtxKey struct{}

	dialIt := &fsdialer.Interceptor{
		BeforeDialContext: func(ctx context.Context, network string, address string) (context.Context, string, string) {
			ctx = context.WithValue(ctx, dialCtxKey{}, time.Now())
			return ctx, network, address
		},
		AfterDialContext: func(ctx context.Context, network string, address string, conn net.Conn, err error) (net.Conn, error) {
			start := ctx.Value(dialCtxKey{}).(time.Time)
			cost := time.Since(start)
			if err == nil {
				dialVec.WithLabelValues("success").Observe(cost.Seconds())
			} else {
				dialVec.WithLabelValues("fail").Observe(cost.Seconds())
			}
			if err != nil {
				return nil, err
			}
			return fsconn.Wrap(conn, stIt), nil
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

	type resolverCtxKey struct{}

	it := &fsresolver.Interceptor{
		BeforeLookupIP: func(ctx context.Context, network, host string) (c context.Context, n, h string) {
			ctx = context.WithValue(ctx, resolverCtxKey{}, time.Now())
			return ctx, network, host
		},
		AfterLookupIP: func(ctx context.Context, network, host string, ips []net.IP, err error) ([]net.IP, error) {
			start := ctx.Value(resolverCtxKey{}).(time.Time)
			cost := time.Since(start)
			if err == nil {
				resolverVec.WithLabelValues("success").Observe(cost.Seconds())
			} else {
				resolverVec.WithLabelValues("fail").Observe(cost.Seconds())
			}
			return ips, err
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
