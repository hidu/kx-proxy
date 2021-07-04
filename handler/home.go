package handler

import (
	_ "embed"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/hidu/kx-proxy/util"
)

var homeTpl *template.Template

//go:embed asset/html/index.html
var homeCode string

func init() {
	myfn := template.FuncMap{
		"selected": func(v1 string, v2 string) string {
			if v1 == v2 {
				return "selected"
			}
			arr := strings.Split(v2, ",")
			for _, vn := range arr {
				if v1 == vn {
					return "selected"
				}
			}
			return "not"
		},
		"selected_def": func(v1 string, v2 string) string {
			if v2 == "" || v1 == v2 {
				return "selected"
			}
			arr := strings.Split(v2, ",")
			for _, vn := range arr {
				if v1 == vn {
					return "selected"
				}
			}
			return "not"
		},
	}
	tpl, err := template.New("home").Funcs(myfn).Parse(homeCode)
	if err != nil {
		panic(err.Error())
	}
	homeTpl = tpl
}

func (k *KxProxy) handlerHome(w http.ResponseWriter, r *http.Request) {
	// 404 for all other url path
	if r.URL.Path[1:] != "" {
		k.handler404(w, r)
		return
	}

	if r.Method == http.MethodPost {
		r.ParseForm()
		enteredURL := r.FormValue("url")
		if enteredURL == "" {
			http.Redirect(w, r, "/", 302)
			return
		}
		validURL, err := url.Parse(enteredURL)
		if err != nil {
			w.Write([]byte("Parse url failed:" + err.Error()))
			return
		}
		// prepend http if not specified
		if validURL.Scheme != "http" && validURL.Scheme != "https" {
			w.Write([]byte("invalid Scheme: " + validURL.Scheme))
			return
		}
		expire, _ := strconv.ParseInt(r.FormValue("expire"), 10, 64)

		opu := &util.ProxyUrl{
			Extension: r.Form["ext"],
			Expire:    expire,
		}

		pu := util.NewProxyUrl(validURL.String(), opu, r)
		encodedURL, err := pu.Encode()
		if err != nil {
			w.Write([]byte("build url failed:" + err.Error()))
			return
		}
		http.Redirect(w, r, "/p/"+encodedURL, 302)
		return
	}

	qs := r.URL.Query()
	datas := map[string]interface{}{
		"url":    qs.Get("url"),
		"expire": qs.Get("expire"),
		"ext":    qs.Get("ext"),
	}
	_ = homeTpl.Execute(w, datas)
}
