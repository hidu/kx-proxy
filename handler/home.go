package handler

import (
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/hidu/kx-proxy/util"
)

var homeTpl *template.Template

func init() {
	homeCode := Asset.GetContent("/asset/html/index.html")
	tpl, err := template.New("home").Parse(string(homeCode))
	if err != nil {
		panic(err.Error())
	}
	homeTpl = tpl
}
func homeHandler(w http.ResponseWriter, r *http.Request) {
	// 404 for all other url path
	if r.URL.Path[1:] != "" {
		handler404(w, r)
		return
	}
	r.ParseForm()
	enteredURL := r.FormValue("url")
	if enteredURL != "" {
		validURL, err := url.Parse(enteredURL)
		if err != nil {
			w.Write([]byte("Parse url failed:" + err.Error()))
			return
		}
		// prepend http if not specified
		if validURL.Scheme == "" {
			validURL.Scheme = "http"
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

	rawURL := r.FormValue("raw")
	datas := map[string]interface{}{}
	datas["Raw"] = rawURL
	_ = homeTpl.Execute(w, datas)

	// Asset.FileHandlerFunc("/asset/html/index.html")(w, r)
}

// 404
// 当存在refer的时候发生404跳转，可能是其他js或者meta跳转等情况
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
