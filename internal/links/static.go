// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/8/8

package links

import (
	"strings"
)

var staticExts = []string{".png", ".jpg", ".jpeg", ".gif", ".css", ".js", ".woff2"}

func IsStaticPath(path string) bool {
	for _, e := range staticExts {
		if strings.HasSuffix(path, e) {
			return true
		}
	}
	return false
}
