// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/6/25

package internal

import (
	"context"
	"time"

	"github.com/xanygo/anygo/store/xcache"
	"github.com/xanygo/anygo/xcodec"
)

type CacheData struct {
	Header map[string]string
	Body   []byte
}

func (cd *CacheData) ContentType() ContentType {
	if len(cd.Header) == 0 {
		return ""
	}
	return ContentType(cd.Header["Content-Type"])
}

var defaultTTL = 30 * 24 * time.Hour

func NewFileCache(dir string) *FileCache {
	return &FileCache{
		cache: &xcache.File[string, *CacheData]{
			Dir:   dir,
			Codec: xcodec.JSON,
			GC:    time.Minute,
		},
	}
}

type FileCache struct {
	cache *xcache.File[string, *CacheData]
}

func (fc *FileCache) Get(key string) *CacheData {
	val, _ := fc.cache.Get(context.Background(), key)
	return val
}

func (fc *FileCache) Set(key string, data *CacheData) {
	fc.SetWithTTL(key, data, defaultTTL)
}

func (fc *FileCache) SetWithTTL(key string, data *CacheData, ttl time.Duration) {
	fc.cache.Set(context.Background(), key, data, ttl)
}

func (fc *FileCache) Del(key string) {
	fc.cache.Delete(context.Background(), key)
}

func (fc *FileCache) Has(key string) bool {
	return fc.Get(key) != nil
}
