// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/10/30

package internal

import (
	"context"
	"log"
	"net"

	"github.com/xanygo/anygo/xcfg"
	"github.com/xanygo/anygo/xnet"

	"github.com/hidu/kx-proxy/internal/dnsgroup"
)

func SetupDNS() {
	confPath := "dns"
	if !xcfg.Exists(confPath) {
		log.Println(confPath + " not exists, skip SetupDNS")
		return
	}
	var cfg *dnsgroup.Config
	if err := xcfg.Parse(confPath, &cfg); err != nil {
		panic(err)
	}
	g := cfg.ToGroup()

	xnet.WithInterceptor(&xnet.ResolverInterceptor{
		LookupIP: func(ctx context.Context, network string, host string, invoker xnet.LookupIPFunc) ([]net.IP, error) {
			return g.LookupIP(ctx, network, host)
		},
	})
	log.Println("use dns group with config:", confPath)
}
