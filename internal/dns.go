// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/10/30

package internal

import (
	"log"

	"github.com/fsgo/fsconf"
	"github.com/fsgo/fsgo/fsnet/fsresolver"

	"github.com/hidu/kx-proxy/internal/dnsgroup"
)

func SetupDNS() {
	confPath := "dns.toml"
	if !fsconf.Exists(confPath) {
		log.Println(confPath + " not exists, skip SetupDNS")
		return
	}
	var cfg *dnsgroup.Config
	if err := fsconf.Parse(confPath, &cfg); err != nil {
		panic(err)
	}
	g := cfg.ToGroup()

	fsresolver.MustRegisterInterceptor(fsresolver.ToInterceptor(g))
	log.Println("use dns group with config:", confPath)
}

func init() {
	cache := &fsresolver.CacheInterceptor{}
	fsresolver.MustRegisterInterceptor(cache.Interceptor())
}
