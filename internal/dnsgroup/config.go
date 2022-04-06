// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/12/5

package dnsgroup

import (
	"regexp"
	"strings"

	"github.com/fsgo/fsgo/fsnet"
	"github.com/fsgo/fsgo/fsnet/fsdns"
)

type Config struct {
	// Timeout 一次 DNS 查询的总超时时间，单位 ms，可选，默认为 500
	Timeout int

	// Disable 是否禁用 NameServer 配置，默认为 false
	Disable bool

	NameServer []*ConfigNameServer

	resolvers []*resolver
}

type ConfigNameServer struct {
	Name  string
	Hosts []string
	// Timeout 使用此组配置查询的超时时间，可选，默认为 500
	Timeout int

	Domains []string
}

func (c *Config) parser() {
	c.resolvers = make([]*resolver, 0)
	if !c.Disable {
		for _, item := range c.NameServer {
			c.resolvers = append(c.resolvers, c.toResolver(item))
		}
	}

	if len(c.resolvers) == 0 {
		def := &resolver{
			Name:     "fsnet_default",
			Resolver: fsnet.DefaultResolver,
		}
		c.resolvers = append(c.resolvers, def)
	}
}

func (c *Config) toResolver(item *ConfigNameServer) *resolver {
	client := &fsdns.Client{
		HostsFile: fsdns.DefaultHostsFile,
		Servers:   fsdns.ParserServers(item.Hosts),
	}
	rc := &fsnet.ResolverCached{
		Invoker: client,
	}
	rc.Expiration = rc.ExpirationFromEnv()

	res := &resolver{
		Name:     item.Name,
		Timeout:  item.Timeout,
		Resolver: rc,
	}

	for _, domain := range item.Domains {
		res.DomainRule = append(res.DomainRule, DomainRule(domain).ToRegexp())
	}

	return res
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

	// 其他的 dns 服务配置放在后面作为备份
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
