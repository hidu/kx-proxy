// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/6/25

package internal

import (
	"strings"
)

type ContentType string

func (ct ContentType) IsHTML() bool {
	return ct.Is("text/html")
}
func (ct ContentType) IsCss() bool {
	return ct.Is("text/css")
}

func (ct ContentType) Is(name string) bool {
	return strings.Contains(string(ct), name)
}

var staticTypes = []string{
	"text/css",
	"image/",
	"/javascript",
}

func (ct ContentType) IsStaticFile() bool {
	for _, st := range staticTypes {
		if ct.Is(st) {
			return true
		}
	}
	return true
}
