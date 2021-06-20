package handler

import (
	"net/http"
	"strings"
)

// handle /ucss/abcd.css
func ucssHandler(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, ".css") || strings.Count(r.URL.Path, "/") > 2 {
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}
	http.StripPrefix("/ucss/", http.FileServer(http.Dir("./ucss/"))).ServeHTTP(w, r)
}
