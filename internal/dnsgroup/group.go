// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/12/5

package dnsgroup

import (
	"context"
	"fmt"
	"log"
	"net"
	"regexp"
	"time"

	"github.com/fsgo/fsgo/fsnet/fsresolver"
)

var _ fsresolver.Resolver = (*ResolverGroup)(nil)

type ResolverGroup struct {
	Config *Config
}

func (r *ResolverGroup) getTimeout() time.Duration {
	if r.Config.Timeout > 0 {
		return time.Duration(r.Config.Timeout) * time.Millisecond
	}
	return 500 * time.Millisecond
}

func (r *ResolverGroup) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	ctx, cancel := context.WithTimeout(ctx, r.getTimeout())
	defer cancel()
	return r.find(host).LookupIP(ctx, network, host)
}

func (r *ResolverGroup) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	ctx, cancel := context.WithTimeout(ctx, r.getTimeout())
	defer cancel()
	return r.find(host).LookupIPAddr(ctx, host)
}

func (r *ResolverGroup) find(host string) resolvers {
	return r.Config.findResolvers(host)
}

var _ fsresolver.Resolver = (*resolver)(nil)

type resolver struct {
	Name       string
	Timeout    int
	Resolver   fsresolver.Resolver
	DomainRule []*regexp.Regexp
}

func (r *resolver) Match(domain string) bool {
	for _, item := range r.DomainRule {
		if item.MatchString(domain) {
			return true
		}
	}
	return false
}

func (r *resolver) getTimeout() time.Duration {
	if r.Timeout > 0 {
		return time.Duration(r.Timeout) * time.Millisecond
	}
	return 200 * time.Millisecond
}

func (r *resolver) LookupIP(ctx context.Context, network, host string) (ret []net.IP, err error) {
	start := time.Now()
	defer func() {
		cost := time.Since(start)
		log.Printf("ns.%s LookupIP(%q,%q)=(%v,%v) cost=%s\n", r.Name, network, host, ret, err, cost)
	}()

	ctx, cancel := context.WithTimeout(ctx, r.getTimeout())
	defer cancel()
	ret, err = r.Resolver.LookupIP(ctx, network, host)
	if err != nil {
		err = fmt.Errorf("%w NS=%s", err, r.Name)
	}
	return ret, err
}

func (r *resolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	ctx, cancel := context.WithTimeout(ctx, r.getTimeout())
	defer cancel()
	ret, err := r.Resolver.LookupIPAddr(ctx, host)
	if err != nil {
		err = fmt.Errorf("%w NS=%s", err, r.Name)
	}
	return ret, err
}

var _ fsresolver.Resolver = (*resolvers)(nil)

type resolvers []*resolver

func (rs resolvers) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	for _, r := range rs {
		val, err := r.LookupIP(ctx, network, host)
		if err == nil {
			return val, nil
		}
		if ctx.Err() != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf(" all %d resolver LookupIP failed", len(rs))
}

func (rs resolvers) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	for _, r := range rs {
		val, err := r.LookupIPAddr(ctx, host)
		if err == nil {
			return val, nil
		}
		if ctx.Err() != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf(" all %d resolver LookupIPAddr failed", len(rs))
}
