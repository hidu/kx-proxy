// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/10/30

package internal

import (
	"context"
	"io/ioutil"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fsgo/fsgo/fsnet"
	"github.com/fsgo/fsgo/fsnet/fsdns"
)

var NameServer string

var DNSBlockFile string

func SetUpDNS() {
	db := newDNSBlock(DNSBlockFile)

	var client = &fsdns.DNSClient{
		HostsFile: fsdns.DefaultHostsFile,
		LookupIPHook: func(ctx context.Context, network, host string, ns net.Addr, result []net.IP) (ret []net.IP, errRet error) {
			defer func() {
				log.Printf("LookupIP network=%q host=%q nameserver=%s rawResult=%v lastResult=%v",
					network, host, ns.String(), result, ret)
			}()
			ret = make([]net.IP, 0, len(result))
			for _, ip := range result {
				if !db.IsBlock(ip) {
					ret = append(ret, ip)
				}
			}
			return ret, nil
		},
	}

	list1 := fsdns.ParserServers(strings.Split(NameServer, ","))
	if len(list1) == 0 {
		log.Println("ignore user DNSClient with empty nameserver list")
		return
	}

	client.SetServers(list1)
	log.Println("use DNSClient with nameserver:", list1)
	fsnet.MustRegisterResolverHook(client.ResolverHook())
}

func newDNSBlock(cf string) *dnsBlock {
	b := &dnsBlock{
		fileName: cf,
	}
	go b.autoLoad(5 * time.Second)
	return b
}

type dnsBlock struct {
	fileName  string
	BlockIPs  map[string]bool
	BlockRegs []*regexp.Regexp
	mux       sync.RWMutex
}

func (db *dnsBlock) autoLoad(dur time.Duration) {
	tk := time.NewTicker(dur)
	defer tk.Stop()
	var last time.Time
	for {
		select {
		case <-tk.C:
			last = db.load(last)
		}
	}
}

func (db *dnsBlock) load(last time.Time) time.Time {
	info, err := os.Stat(db.fileName)
	if err != nil {
		return time.Time{}
	}
	if last.Equal(info.ModTime()) {
		return info.ModTime()
	}
	bf, err := ioutil.ReadFile(db.fileName)
	if err != nil {
		return time.Time{}
	}
	lines := strings.Split(string(bf), "\n")
	result := map[string]bool{}
	var regs []*regexp.Regexp
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		if ip := net.ParseIP(line); ip != nil {
			result[ip.String()] = true
		} else if strings.Contains(line, "*") {
			rstr := strings.ReplaceAll(line, ".", `\.`)
			rstr = "^" + strings.ReplaceAll(rstr, "*", `\d+`) + "$"
			if reg, err := regexp.Compile(rstr); err == nil {
				regs = append(regs, reg)
			}
		}
	}
	log.Printf("parser %q success, BlockIPs=%v BlockRegs=%v\n", db.fileName, result, regs)
	db.mux.Lock()
	db.BlockIPs = result
	db.BlockRegs = regs
	db.mux.Unlock()
	return info.ModTime()
}

func (db *dnsBlock) IsBlock(ip net.IP) bool {
	str := ip.String()
	db.mux.RLock()
	defer db.mux.RUnlock()
	if db.BlockIPs[str] {
		return true
	}
	for _, reg := range db.BlockRegs {
		if reg.MatchString(str) {
			return true
		}
	}
	return false
}
