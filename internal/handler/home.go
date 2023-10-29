package handler

import (
	_ "embed" // for asset file
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/hidu/kx-proxy/internal/links"
)

var homeTpl *template.Template

//go:embed asset/html/index.html
var homeCode string

func init() {
	myFns := template.FuncMap{
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
	tpl, err := template.New("home").Funcs(myFns).Parse(homeCode)
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
		k.handlerHomePost(w, r)
		return
	}
	datas := map[string]any{
		"expire": "0",
		"ext":    "",
		"mp":     false,
	}
	raw := r.URL.Query().Get("raw")
	if raw != "" {
		pu, _ := links.DecodeProxyURL(raw)
		if pu != nil {
			datas = pu.ToHomeData()
		}
	}
	if r.URL.Query().Get("mp") != "" {
		datas["mp"] = true
	}
	_ = homeTpl.Execute(w, datas)
}

func (k *KxProxy) handlerHomePost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(400)
		_, _ = w.Write([]byte("ParseForm failed:" + err.Error()))
		return
	}
	enteredURL := r.FormValue("url")
	if enteredURL == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	validURL, err := url.Parse(enteredURL)
	if err != nil {
		w.WriteHeader(400)
		_, _ = w.Write([]byte("Parse url failed:" + err.Error()))
		return
	}

	// prepend http if not specified
	if validURL.Scheme != "http" && validURL.Scheme != "https" {
		_, _ = w.Write([]byte("invalid Scheme: " + validURL.Scheme))
		return
	}

	expire, _ := strconv.ParseInt(r.FormValue("expire"), 10, 64)

	opu := &links.ProxyURL{
		Extension: r.Form["ext"],
		Expire:    expire,
	}

	pu := links.NewProxyURL(validURL.String(), opu, r)
	encodedURL, err := pu.Encode()
	if err != nil {
		_, _ = w.Write([]byte("build url failed:" + err.Error()))
		return
	}
	http.Redirect(w, r, "/p/"+encodedURL, http.StatusFound)
}
