// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/6/25

package internal

import (
	"context"
	"time"

	"github.com/fsgo/fscache"
	"github.com/fsgo/fscache/filecache"
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
	opt := filecache.Option{
		Dir:        dir,
		GCInterval: time.Minute,
	}
	fc, err := filecache.New(opt)
	if err != nil {
		panic(err.Error())
	}
	return &FileCache{
		cache: fc,
	}
}

type FileCache struct {
	cache fscache.ICache
}

func (fc *FileCache) Get(key string) *CacheData {
	ret := fc.cache.Get(context.Background(), key)
	if err := ret.Err(); err != nil {
		return nil
	}
	var cd *CacheData
	has, err := ret.Value(&cd)
	if !has || err != nil {
		return nil
	}
	return cd
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
	ret := fc.cache.Has(context.Background(), key)
	return ret.Has()
}
