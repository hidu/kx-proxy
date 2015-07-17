package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
	"net"
)

// Cache templates
var templates = template.Must(template.ParseFiles("home.html"))

// Pre-compile RegEx
var reBase = regexp.MustCompile("base +href=\"(.*?)\"")
var reHTML = regexp.MustCompile("src=[\"\\'](.*?)[\"\\']|href=[\"\\'](.*?)[\"\\']|action=[\"\\'](.*?)[\"\\']")
var reCSS = regexp.MustCompile("url\\([\"\\']?(.*?)[\"\\']?\\)")

var httpClient *http.Client = &http.Client{}

var startTime = time.Now()

var etag = fmt.Sprintf("%d", startTime.Unix())

func homeHandler(w http.ResponseWriter, r *http.Request) {
	// 404 for all other url path
	if r.URL.Path[1:] != "" {
		http.NotFound(w, r)
		return
	}
	r.ParseForm()
	enteredUrl := r.FormValue("url")
	if enteredUrl != "" {
		validUrl, _ := url.Parse(enteredUrl)
		// prepend http if not specified
		if validUrl.Scheme == "" {
			validUrl.Scheme = "http"
		}
		encodedUrl := base64.StdEncoding.EncodeToString([]byte(validUrl.String()))
		http.Redirect(w, r, "/p/"+encodedUrl, 302)
		return
	}
	w.Header().Set("Cache-Control", "max-age=2592000")

	etagClient := r.Header.Get("If-None-Match")
	if etagClient != "" {
		if etag == etagClient {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	w.Header().Set("etag", etag)
	templates.ExecuteTemplate(w, "home.html", nil)
}

func encodeURL(src []byte, baseHref string, urlString string, start int, end int) []byte {
	relURL := string(src[start:end])
	// keep anchor and javascript links intact
	if strings.Index(relURL, "#") == 0 || strings.Index(relURL, "javascript") == 0 {
		return src
	}
	// Check if url is relative and make it absolute
	if strings.Index(relURL, "http") != 0 {
		var basePath *url.URL
		if baseHref == "" {
			basePath, _ = url.Parse(urlString)
		} else {
			basePath, _ = url.Parse(baseHref)
		}
		relPath, err := url.Parse(relURL)
		if err != nil {
			return src
		}
		absURL := basePath.ResolveReference(relPath).String()
		src = bytes.Replace(src, []byte(relURL), []byte(absURL), -1)
		end = start + len(absURL)
	}
	var encodedPath []byte = make([]byte, base64.StdEncoding.EncodedLen(end-start))
	base64.StdEncoding.Encode(encodedPath, src[start:end])
	return bytes.Replace(src, src[start:end], encodedPath, -1)
}
var copyHeaders=[]string{"Referer","Accept-Language","Cookie"}

func copyHeader(dst, src http.Header) {
	for k, vs := range src {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	r.Header.Del("Connection")
	encodedUrl := r.URL.Path[len("/p/"):]
	url, err := base64.StdEncoding.DecodeString(encodedUrl)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	is_client := r.Header.Get("is_client") == "1"
	if is_client {
		r.Header.Del("is_client")
	}
	
	urlString := string(url[:])
	req, _ := http.NewRequest(r.Method, urlString, r.Body)
	
	if(is_client){
		copyHeader(req.Header,r.Header)
	}else{
		req.Header.Set("Content-Type", r.Header.Get("Content-Type"))
		// Set request user agent to that of user's
		req.Header.Set("User-Agent", r.Header.Get("User-Agent"))
	}
	
//	resp, err := httpClient.Do(req)
	
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	resp, err:=transport.RoundTrip(req)
	

	if err != nil {
		fmt.Println("Error Fetching " + urlString)
		http.NotFound(w, r)
		return
	}
	defer resp.Body.Close()

	contentType := ""

	//Write all remote resp header to client
	for headerKey,vs := range resp.Header {
		headerVal := resp.Header.Get(headerKey)
		if headerKey == "Content-Type" {
			contentType = headerVal
		}
		for _,v:=range vs{
			w.Header().Add(headerKey,v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	if is_client {
//		if is_replace, body := googleApis(resp); is_replace {
//			w.Write(body)
//		} else {
			io.Copy(w, resp.Body)
//		}
		return
	}

	// Rewrite all urls
	baseHref := ""
	if strings.Contains(contentType, "text/html") {
		body, _ := ioutil.ReadAll(resp.Body)
		baseHrefMatch := reBase.FindSubmatch(body)
		if len(baseHrefMatch) > 0 {
			baseHref = string(baseHrefMatch[1][:])
		}
		encodedBody := reHTML.ReplaceAllFunc(body, func(s []byte) []byte {
			parts := reHTML.FindSubmatchIndex(s)
			if parts != nil {
				// replace src attribute
				srcIndex := parts[2:4]
				if srcIndex[0] != -1 {
					return encodeURL(s, baseHref, urlString, srcIndex[0], srcIndex[1])
				}

				// replace href attribute
				hrefIndex := parts[4:6]
				if hrefIndex[0] != -1 {
					return encodeURL(s, baseHref, urlString, hrefIndex[0], hrefIndex[1])
				}

				// replace form action attribute
				actionIndex := parts[6:8]
				if actionIndex[0] != -1 {
					return encodeURL(s, baseHref, urlString, actionIndex[0], actionIndex[1])
				}
			}
			return s
		})
		w.Write(encodedBody)
	} else if strings.Contains(contentType, "text/css") {
		body, _ := ioutil.ReadAll(resp.Body)
		encodedBody := reCSS.ReplaceAllFunc(body, func(s []byte) []byte {
			parts := reCSS.FindSubmatchIndex(s)
			if parts != nil {
				// replace url attribute in css
				pathIndex := parts[2:4]
				if pathIndex[0] != -1 {
					return encodeURL(s, baseHref, urlString, pathIndex[0], pathIndex[1])
				}
			}
			return s
		})
		w.Write(encodedBody)
	} else {
		io.Copy(w, resp.Body)
	}
}

func googleApis(resp *http.Response) (bool, []byte) {
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		return false, []byte("")
	}
	body, _ := ioutil.ReadAll(resp.Body)
	body = bytes.Replace(body, []byte("https://ajax.googleapis.com/"), []byte("http://ajax.useso.com/"), -1)
	body = bytes.Replace(body, []byte("http://ajax.googleapis.com/"), []byte("http://ajax.useso.com/"), -1)
	body = bytes.Replace(body, []byte("https://fonts.googleapis.com/"), []byte("http://fonts.useso.com/"), -1)
	body = bytes.Replace(body, []byte("http://fonts.googleapis.com/"), []byte("http://fonts.useso.com/"), -1)

	return true, body
}

func main() {
	var httpHost string = os.Getenv("HOST")
	var httpPort string = os.Getenv("PORT")
	if httpPort == "" {
		httpPort = "8085"
	}
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/p/", proxyHandler)

	http.HandleFunc("/assets/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=2592000")
		http.ServeFile(w, r, r.URL.Path[1:])
	})

	fmt.Printf("web proxy listening on :%s\n", httpPort)

	http.ListenAndServe(httpHost+":"+httpPort, nil)

}
