// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/7/4

package handler

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/hidu/kx-proxy/util"
)

// 404
// 当存在refer的时候发生404跳转，可能是其他js或者meta跳转等情况
func (k *KxProxy) handler404(w http.ResponseWriter, r *http.Request) {
	refer := r.Referer()

	if refer == "" || !strings.Contains(refer, "/p/") {
		http.NotFound(w, r)
		return
	}

	u, err := url.Parse(refer)
	if err != nil || strings.Index(u.Path, "/p/") != 0 {
		http.NotFound(w, r)
		return
	}

	pu, err := util.DecodeProxyUrl(u.Path[3:])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	us := r.URL.Path
	if r.URL.RawQuery != "" {
		us += "?" + r.URL.RawQuery
	}

	pu.SwitchPath(us)
	encodedURL, err := pu.Encode()

	if err != nil {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, "/p/"+encodedURL, 302)
}
