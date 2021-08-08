package handler

import (
	"embed"
	"net/http"
	"strings"
)

//go:embed asset
var asset embed.FS

// handle /ucss/abcd.css
func (k *KxProxy) handlerUcss(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, ".css") || strings.Count(r.URL.Path, "/") > 2 {
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}
	http.StripPrefix("/ucss/", http.FileServer(http.Dir("./ucss/"))).ServeHTTP(w, r)
}

func (k *KxProxy) handlerAsset(w http.ResponseWriter, r *http.Request) {
	http.FileServer(http.FS(asset)).ServeHTTP(w, r)
}

func (k *KxProxy) handlerFavicon(w http.ResponseWriter, r *http.Request) {
	f, err := http.FS(asset).Open("favicon.png")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	info, err := f.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, "favicon.png", info.ModTime(), f)
}
