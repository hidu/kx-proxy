package handler

import (
	"fmt"
	"github.com/hidu/kx-proxy/util"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
	"regexp"
	"strconv"
)

var kxKey = "KxKey"

var reEncodedURL=regexp.MustCompile(`^(\d+)\|(.+)$`)

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	logData:=make(map[string]interface{})
	startTime:=time.Now()
	defer (func(){
		used:=time.Now().Sub(startTime)
		log.Println(
			"remote:",r.RemoteAddr,
			"path:",r.URL.Path,
			"used:",used,
			logData)
	})()

	r.Header.Del("Connection")
	encodedURL := r.URL.Path[len("/p/"):]
	
	kxURL := r.Header.Get("kx_url")
	if kxURL != "" {
		encodedURL = kxURL
	}
	r.Header.Del("kx_url")

	url, err := util.DecryptURL(encodedURL)
	if err != nil {
		logData["emsg"]="decode_url_failed:"+err.Error()
		http.Error(w, "decode_url_failed:"+err.Error(), http.StatusInternalServerError)
		return
	}
	
	matchStrs:=reEncodedURL.FindStringSubmatch(url)
	//检查url是否过期
	if(len(matchStrs)==3){
		url=matchStrs[2]
		expireAt,_:=strconv.ParseInt(matchStrs[1],10,64)
		expiredN:=startTime.Unix()-(expireAt+2)
		logData["expiredN"]=expiredN
		if(expireAt>0 && expiredN>0){
			http.Error(w, fmt.Sprintf("expired:%d",expiredN), http.StatusBadRequest)
			return
		}
	}
	
	isClient := r.Header.Get("is_client") == "1"
	
	logData["visit_url"]=url
	logData["is_client"]=isClient

	skey := r.Header.Get(kxKey)
	if isClient {
		logData["skey"]=skey
		
		r.Header.Del("is_client")
		if len(secreKeys) > 0 {
			_, hasSkey := secreKeys[skey]
			if skey == "" || !hasSkey {
				logData["emsg"]="required kxkey,get:"+skey
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
		
		logData["hidden_ip"]=hidden
		
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
		logData["emsg"]="fetch_failed:"+err.Error()
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

	logData["resp_status"]=resp.StatusCode
	
	if isClient {
		var copySize int64
		var copyErr error
		if BodyStreamEnc {
			w.Header().Set("_kx_enc_", "1")
			w.Header().Set("_kx_content_encoding", w.Header().Get("Content-Encoding"))
			w.Header().Del("Content-Encoding")
			w.WriteHeader(resp.StatusCode)

			writer := util.CipherStreamWrite(skey, encodedURL, w)
			copySize, copyErr = io.Copy(writer, resp.Body)
			
		} else {
			w.WriteHeader(resp.StatusCode)
			copySize, copyErr=io.Copy(w, resp.Body)
		}
		logData["io_copy_size"]=copySize
		logData["io_copy_err"]=copyErr
		return
	}

	if resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusFound {
		location := resp.Header.Get("Location")
		if location != "" {
			encodedURL, err := util.EncryptURL(location)
			if err != nil {
				logData["emsg"]="Location_build_url failed"+err.Error()
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
		encodedBody := util.HTMLURLReplace(body, urlString)
		encodedBody = util.CSSURLReplace(encodedBody, urlString)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(encodedBody)))
		w.WriteHeader(resp.StatusCode)
		w.Write(encodedBody)
	} else if strings.Contains(contentType, "text/css") {
		body, _ := ioutil.ReadAll(resp.Body)
		encodedBody := util.CSSURLReplace(body, urlString)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(encodedBody)))
		w.WriteHeader(resp.StatusCode)
		w.Write(encodedBody)
	} else {
		w.WriteHeader(resp.StatusCode)
		copySize, copyErr:=io.Copy(w, resp.Body)
		logData["io_copy_size"]=copySize
		logData["io_copy_err"]=copyErr
	}

}
