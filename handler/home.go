package handler

import (
	"fmt"
	"github.com/hidu/kx-proxy/util"
	"net/http"
	"net/url"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL.String())
	// 404 for all other url path
	if r.URL.Path[1:] != "" {
		http.NotFound(w, r)
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
		encodedURL, err := util.EncryptURL(validURL.String())
		if err != nil {
			w.Write([]byte("build url failed:" + err.Error()))
			return
		}
		http.Redirect(w, r, "/p/"+encodedURL, 302)
		return
	}
	Assest.FileHandlerFunc("/assets/html/index.html")(w, r)
}
