package handler

import (
	"fmt"
	"github.com/hidu/kx-proxy/util"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL.String())
	// 404 for all other url path
	if r.URL.Path[1:] != "" {
		handler404(w, r)
		return
	}
	r.ParseForm()
	enteredURL := r.FormValue("url")
	if enteredURL != "" {
		validURL, _ := url.Parse(enteredURL)
		// prepend http if not specified
		if validURL.Scheme == "" {
			validURL.Scheme = "http"
		}
		expire, _ := strconv.ParseInt(r.FormValue("expire"), 10, 64)

		pu := util.NewProxyUrl(validURL.String(), expire)
		encodedURL, err := pu.Encode()
		if err != nil {
			w.Write([]byte("build url failed:" + err.Error()))
			return
		}
		http.Redirect(w, r, "/p/"+encodedURL, 302)
		return
	}
	Assest.FileHandlerFunc("/assets/html/index.html")(w, r)
}

//404
//当存在refer的时候发生404跳转，可能是其他js或者meta跳转等情况
func handler404(w http.ResponseWriter, r *http.Request) {
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
