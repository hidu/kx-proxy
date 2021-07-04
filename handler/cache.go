// Copyright(C) 2021 github.com/hidu  All Rights Reserved.
// Author: hidu
// Date: 2021/6/25

package handler

import (
	"github.com/hidu/kx-proxy/internal"
)

var fileCache *internal.FileCache
var preLoader *internal.PreLoad

func InitCache(dir string) {
	fileCache = internal.NewFileCache(dir)
	preLoader = &internal.PreLoad{
		FileCache: fileCache,
	}
}
