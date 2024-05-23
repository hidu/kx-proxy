// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/8/8

package links

import (
	"strings"
)

var staticExts = []string{
	".png", ".jpg", ".jpeg", ".gif", ".bmp",
	".css", ".js", ".svg",
	".ttf", ".woff", ".woff2",
	".ico",
}

func IsStaticPath(path string) bool {
	prefix, _, _ := strings.Cut(path, "?")
	for _, e := range staticExts {
		if strings.HasSuffix(prefix, e) {
			return true
		}
	}
	return false
}
