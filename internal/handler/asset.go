package handler

import (
	"embed"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/fsgo/fsenv"
)

//go:embed asset
var asset embed.FS

// handle /ucss/abcd.css
func (k *KxProxy) handlerUserCSS(w http.ResponseWriter, r *http.Request) {
	isCSS := strings.HasSuffix(r.URL.Path, ".css")
	isJS := strings.HasSuffix(r.URL.Path, ".js")

	if !(isCSS || isJS) || strings.Count(r.URL.Path, "/") > 2 {
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}
	ucssDir := filepath.Join(fsenv.ConfDir(), "ucss")
	http.StripPrefix("/ucss/", http.FileServer(http.Dir(ucssDir))).ServeHTTP(w, r)
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
