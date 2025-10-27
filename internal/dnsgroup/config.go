// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/12/5

package dnsgroup

import (
	"context"
	"math/rand/v2"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/xanygo/anygo/xnet"
)

type Config struct {
	// Timeout 一次 DNS 查询的总超时时间，单位 ms，可选，默认为 500
	Timeout int

	// Disable 是否禁用 NameServer 配置，默认为 false
	Disable bool

	NameServer []*ConfigNameServer

	resolvers []*resolver
}

func (c *Config) parser() {
	c.resolvers = make([]*resolver, 0)
	if !c.Disable {
		for _, item := range c.NameServer {
			c.resolvers = append(c.resolvers, item.toResolver())
		}
	}

	if len(c.resolvers) == 0 {
		def := &resolver{
			Name:     "fsnet_default",
			Resolver: xnet.LookupIPFunc(xnet.LookupIP),
		}
		c.resolvers = append(c.resolvers, def)
	}
}

func (c *Config) ToGroup() *ResolverGroup {
	c.parser()
	return &ResolverGroup{
		Config: c,
	}
}

func (c *Config) findResolvers(domain string) []*resolver {
	var matches []*resolver
	var backup []*resolver
	for _, item := range c.resolvers {
		if item.Match(domain) {
			matches = append(matches, item)
		} else {
			backup = append(backup, item)
		}
	}

	if len(matches) == 0 {
		return c.resolvers
	}

	matches = append(matches, backup...)
	return matches
}

type DomainRule string

func (dr DomainRule) ToRegexp() *regexp.Regexp {
	str := regexp.QuoteMeta(string(dr))
	str = strings.ReplaceAll(str, regexp.QuoteMeta("."), ".")
	str = strings.ReplaceAll(str, regexp.QuoteMeta("*"), "*")
	return regexp.MustCompile("^" + str + "$")
}

type ConfigNameServer struct {
	Name  string
	Hosts []string

	// Timeout 使用此组配置查询的超时时间，可选，默认为 500
	Timeout int

	Domains []string
}

func (cn *ConfigNameServer) getTimeout() time.Duration {
	if cn.Timeout > 0 {
		return time.Duration(cn.Timeout) * time.Millisecond
	}
	return 200 * time.Millisecond
}

func (cn *ConfigNameServer) parser() *net.Resolver {
	if len(cn.Hosts) == 0 {
		panic("empty hosts")
	}
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{}
			ctx, cancel := context.WithTimeout(ctx, cn.getTimeout())
			defer cancel()
			idx := rand.IntN(len(cn.Hosts))
			return d.DialContext(ctx, "udp", cn.Hosts[idx])
		},
	}
}

func (cn *ConfigNameServer) toResolver() *resolver {
	res := &resolver{
		Name:     cn.Name,
		Timeout:  cn.Timeout,
		Resolver: cn.parser(),
	}

	for _, domain := range cn.Domains {
		res.DomainRule = append(res.DomainRule, DomainRule(domain).ToRegexp())
	}
	return res
}
