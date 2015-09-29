/**
*
*kexue shang wang
*
*use some gopee code: github.com/madhurjain/gopee
*
 */
package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"crypto/rand"

	"crypto/md5"
)

var indexPage []byte
var secreKeys = make(map[string]int)

const (
	aesTable = "kxproxyb8PsyCQ4b"
)

var (
	aesBlock cipher.Block
)

func init() {
	indexPage, _ = ioutil.ReadFile("index.html")
	keys := loadTxtConf("keys.txt")
	for _, key := range keys {
		secreKeys[key] = 1
	}

	var err error
	aesBlock, err = aes.NewCipher([]byte(aesTable))

	if err != nil {
		panic(err)
	}

}

var reBase = regexp.MustCompile("base +href=\"(.*?)\"")
var reHTML = regexp.MustCompile("src=[\"\\'](.*?)[\"\\']|href=[\"\\'](.*?)[\"\\']|action=[\"\\'](.*?)[\"\\']")
var reCSS = regexp.MustCompile("url\\([\"\\']?(.*?)[\"\\']?\\)")

var httpClient = &http.Client{}

var startTime = time.Now()

var etag = fmt.Sprintf("%d", startTime.Unix())

var bodyStreamEnc = flag.Bool("enc", false, "encrypts the body stream")

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
		//		encodedURL := base64.StdEncoding.EncodeToString([]byte(validURL.String()))
		encodedURL, err := encryptURL(validURL.String())
		if err != nil {
			w.Write([]byte("build url failed:" + err.Error()))
			return
		}
		http.Redirect(w, r, "/p/"+encodedURL, 302)
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
	w.Write(indexPage)
}

func encryptURL(srcURL string) (string, error) {
	src := []byte(srcURL)
	padLen := aes.BlockSize - (len(src) % aes.BlockSize)

	for i := 0; i < padLen; i++ {
		src = append(src, byte(padLen))

	}

	srcLen := len(src)

	encryptText := make([]byte, srcLen+aes.BlockSize)

	iv := encryptText[srcLen:]

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	mode := cipher.NewCBCEncrypter(aesBlock, iv)

	mode.CryptBlocks(encryptText[:srcLen], src)
	s := base64.URLEncoding.EncodeToString(encryptText)
	return s, nil

}

func decryptURL(srcURL string) (string, error) {
	if srcURL == "" {
		return "", fmt.Errorf("empty url")
	}
	src, err := base64.URLEncoding.DecodeString(srcURL)
	if err != nil {
		log.Println("base64_decode_failed:", err.Error(), "data:", srcURL[1:])
		return "", err
	}
	if len(src) < aes.BlockSize*2 || len(src)%aes.BlockSize != 0 {
		return "", fmt.Errorf("wrong data size")

	}

	srcLen := len(src) - aes.BlockSize

	decryptText := make([]byte, srcLen)

	iv := src[srcLen:]

	mode := cipher.NewCBCDecrypter(aesBlock, iv)

	mode.CryptBlocks(decryptText, src[:srcLen])

	paddingLen := int(decryptText[srcLen-1])

	if paddingLen > 16 {

		return "", fmt.Errorf("wrong pading size")

	}

	return string(decryptText[:srcLen-paddingLen]), nil

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
	newURL, _ := encryptURL(string(src[start:end]))
	return bytes.Replace(src, src[start:end], []byte(newURL), -1)
}

var copyHeaders = []string{"Referer", "Accept-Language", "Cookie"}

func copyHeader(dst, src http.Header) {
	for k, vs := range src {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}

var kxKey = "KxKey"

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	r.Header.Del("Connection")
	encodedURL := r.URL.Path[len("/p/"):]

	kxURL := r.Header.Get("kx_url")
	if kxURL != "" {
		encodedURL = kxURL
	}
	r.Header.Del("kx_url")

	url, err := decryptURL(encodedURL)
	if err != nil {
		log.Println("decode_url_failed:", err, encodedURL)
		http.Error(w, "decode_url_failed:"+err.Error(), http.StatusInternalServerError)
		return
	}

	isClient := r.Header.Get("is_client") == "1"

	log.Println("visit_url:", url, "is_client:", isClient)

	skey := r.Header.Get(kxKey)
	if isClient {
		r.Header.Del("is_client")
		if len(secreKeys) > 0 {
			_, hasSkey := secreKeys[skey]
			if skey == "" || !hasSkey {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(r.Host + " required " + kxKey + "\nyourkey:" + skey))
				return
			}
		}
		r.Header.Del(kxKey)
	}

	urlString := string(url[:])
	req, _ := http.NewRequest(r.Method, urlString, r.Body)

	if isClient {
		hidden := r.Header.Get("hidden_ip")
		copyHeader(req.Header, r.Header)
		if hidden != "1" {
			addrInfo := strings.Split(r.RemoteAddr, ":")
			req.Header.Set("HTTP_X_FORWARDED_FOR", addrInfo[0])
		}
		req.Header.Del("hidden_ip")
	} else {
		req.Header.Set("Content-Type", r.Header.Get("Content-Type"))
		req.Header.Set("User-Agent", r.Header.Get("User-Agent"))
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	resp, err := transport.RoundTrip(req)

	if err != nil {
		fmt.Println("Error Fetching " + urlString)
		http.Error(w, "Error Fetching "+urlString+"\n"+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	contentType := ""

	//Write all remote resp header to client
	for headerKey, vs := range resp.Header {
		headerVal := resp.Header.Get(headerKey)
		if headerKey == "Content-Type" {
			contentType = headerVal
		}
		for _, v := range vs {
			w.Header().Add(headerKey, v)
		}
	}

	if isClient {
		if *bodyStreamEnc {
			w.Header().Set("_kx_enc_", "1")
			w.Header().Set("_kx_content_encoding", w.Header().Get("Content-Encoding"))
			w.Header().Del("Content-Encoding")
			w.WriteHeader(resp.StatusCode)

			writer := cipherStreamWrite(skey, encodedURL, w)
			n, err := io.Copy(writer, resp.Body)
			fmt.Println(n, err)
		} else {
			w.WriteHeader(resp.StatusCode)
			io.Copy(w, resp.Body)
		}
		return
	}

	if resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusFound {
		location := resp.Header.Get("Location")
		if location != "" {
			encodedURL, err := encryptURL(location)
			if err != nil {
				w.Write([]byte("build url failed:" + err.Error()))
				return
			}
			http.Redirect(w, r, "/p/"+encodedURL, 302)
			return
		}
	}

	// Rewrite all urls
	if strings.Contains(contentType, "text/html") {
		body, _ := ioutil.ReadAll(resp.Body)
		encodedBody := htmlURLReplace(body, urlString)
		encodedBody = cssURLReplace(encodedBody, urlString)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(encodedBody)))
		w.WriteHeader(resp.StatusCode)
		w.Write(encodedBody)
	} else if strings.Contains(contentType, "text/css") {
		body, _ := ioutil.ReadAll(resp.Body)
		encodedBody := cssURLReplace(body, urlString)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(encodedBody)))
		w.WriteHeader(resp.StatusCode)
		w.Write(encodedBody)
	} else {
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}

}

//对数据流进行加密
func cipherStreamWrite(skey string, encodeURL string, writer io.Writer) *cipher.StreamWriter {
	key := strMd5(fmt.Sprintf("%s#kxsw#%s", skey, encodeURL))
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	var iv [aes.BlockSize]byte
	stream := cipher.NewOFB(block, iv[:])
	return &cipher.StreamWriter{S: stream, W: writer}
}

func strMd5(mystr string) []byte {
	h := md5.New()
	h.Write([]byte(mystr))
	return h.Sum(nil)
}

func htmlURLReplace(body []byte, urlString string) []byte {
	baseHrefMatch := reBase.FindSubmatch(body)
	baseHref := ""
	if len(baseHrefMatch) > 0 {
		baseHref = string(baseHrefMatch[1][:])
	}
	return reHTML.ReplaceAllFunc(body, func(s []byte) []byte {
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
}

func cssURLReplace(body []byte, urlString string) []byte {
	baseHref := ""
	return reCSS.ReplaceAllFunc(body, func(s []byte) []byte {
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
}

/**
* handle url http://aaa.com/get/?url=http://www.baidu.com/
 */
func getHandler(w http.ResponseWriter, r *http.Request) {
	cusURL := ""
	if strings.HasPrefix(r.URL.RawQuery, "url=") {
		cusURL = r.URL.RawQuery[4:]
	}
	req, err := http.NewRequest(r.Method, cusURL, r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	addrInfo := strings.Split(r.RemoteAddr, ":")
	req.Header.Set("HTTP_X_FORWARDED_FOR", addrInfo[0])
	resp, err := httpClient.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(err.Error()))
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

var addr = flag.String("addr", ":8085", "listen addr,eg :8085")

func main() {
	flag.Parse()

	//	for i:=0;i<100;i++{
	//	e,_:=encryptURL("http://127.0.0.1/h/c.html")
	//	d,_:=decryptURL(e)
	//	fmt.Println("url:",e,"decode:",d)
	//	}

	var httpHost  = os.Getenv("HOST")
	var httpPort  = os.Getenv("PORT")

	laddr := httpHost + ":" + httpPort

	if httpPort == "" {
		laddr = *addr
	}
	log.Println("bodyStreamEnc:", *bodyStreamEnc)

	if len(laddr) < 2 {
		fmt.Println("listening addr [", laddr, "] is wrong")
		os.Exit(1)
	}

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/p/", proxyHandler)
	http.HandleFunc("/get/", getHandler)

	http.HandleFunc("/assets/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=2592000")
		http.ServeFile(w, r, r.URL.Path[1:])
	})

	fmt.Printf("kx-proxy listening on :%s\n", laddr)

	err := http.ListenAndServe(laddr, nil)
	fmt.Println("exit with err:", err)
}

func loadTxtConf(confPath string) []string {
	var lines []string
	datas, err := ioutil.ReadFile(confPath)
	if err != nil {
		log.Fatalln(err)
	}
	ls := bytes.Split(datas, []byte("\n"))
	for _, lineBs := range ls {
		index := bytes.IndexByte(lineBs, '#')
		if index > -1 {
			lineBs = lineBs[:index]
		}
		lineBs = bytes.TrimSpace(lineBs)
		if len(lineBs) == 0 {
			continue
		}
		lines = append(lines, string(lineBs))
	}
	return lines
}
