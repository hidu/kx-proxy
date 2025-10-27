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

	"github.com/xanygo/anygo/xnet"
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
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return true
	}
	ip := net.ParseIP(host)
	return ip.IsLoopback() || ip.IsPrivate()
}

func newClient(skipVerify bool) *http.Client {
	return &http.Client{
		Timeout: 3 * time.Minute,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				conn, err := xnet.DialContext(ctx, network, addr)
				// 禁止连接到本地 IP
				if conn != nil && !allowVisitVlan && inPrivateAddr(conn.RemoteAddr()) {
					_ = conn.Close()
					return nil, fmt.Errorf("forbidden, cannot connect to %s", conn.RemoteAddr().String())
				}
				return conn, err
			},
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          2,
			IdleConnTimeout:       10 * time.Second,
			TLSHandshakeTimeout:   20 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: skipVerify,
			},
		},
	}
}
