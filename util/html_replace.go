package util

import (
	"bytes"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var reBase = regexp.MustCompile("base +href=\"(.*?)\"")
var reHTML = regexp.MustCompile("src=[\"\\'](.*?)[\"\\']|href=[\"\\'](.*?)[\"\\']|action=[\"\\'](.*?)[\"\\']")
var reCSS = regexp.MustCompile("url\\([\"\\']?(.*?)[\"\\']?\\)")

func encodeURL(src []byte, baseHref string, urlString string, start int, end int, pu *ProxyUrl, r *http.Request) []byte {
	relURL := string(src[start:end])
	// keep anchor and javascript links intact
	if strings.Index(relURL, "#") == 0 || strings.Index(relURL, "javascript") == 0 {
		return src
	}

	relURL1 := strings.ToLower(relURL)
	if strings.Index(relURL1, "javascript") == 0 {
		if pu.Extension.Has("no_js") {
			return []byte("")
		}
		return src
	}

	// keep url(data:image/png;base64
	if strings.Index(relURL, "data:") == 0 {
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
	urlStrNew := string(src[start:end])

	puNew := NewProxyUrl(urlStrNew, pu, r)

	newURL, _ := puNew.Encode()

	return bytes.Replace(src, src[start:end], []byte(newURL), -1)
}

func BaseHref(body []byte) string {
	baseHrefMatch := reBase.FindSubmatch(body)
	var baseHref string
	if len(baseHrefMatch) > 0 {
		baseHref = string(baseHrefMatch[1][:])
	}
	return baseHref
}

// HTMLURLReplace 对html内容中的url替换 成代理的url地址
func HTMLURLReplace(body []byte, urlString string, pu *ProxyUrl, r *http.Request) []byte {
	baseHref := BaseHref(body)
	return reHTML.ReplaceAllFunc(body, func(s []byte) []byte {
		parts := reHTML.FindSubmatchIndex(s)
		if parts != nil {
			// replace src attribute
			srcIndex := parts[2:4]
			if srcIndex[0] != -1 {
				return encodeURL(s, baseHref, urlString, srcIndex[0], srcIndex[1], pu, r)
			}

			// replace href attribute
			hrefIndex := parts[4:6]
			if hrefIndex[0] != -1 {
				return encodeURL(s, baseHref, urlString, hrefIndex[0], hrefIndex[1], pu, r)
			}

			// replace form action attribute
			actionIndex := parts[6:8]
			if actionIndex[0] != -1 {
				return encodeURL(s, baseHref, urlString, actionIndex[0], actionIndex[1], pu, r)
			}
		}
		return s
	})
}

// CSSURLReplace 对css内容中的url替换为代理的地址
func CSSURLReplace(body []byte, urlString string, pu *ProxyUrl, r *http.Request) []byte {
	baseHref := ""
	return reCSS.ReplaceAllFunc(body, func(s []byte) []byte {
		parts := reCSS.FindSubmatchIndex(s)
		if parts != nil {
			// replace url attribute in css
			pathIndex := parts[2:4]
			if pathIndex[0] != -1 {
				return encodeURL(s, baseHref, urlString, pathIndex[0], pathIndex[1], pu, r)
			}
		}
		return s
	})
}

var reAlink = regexp.MustCompile("(?i)<a href=[\"\\'](.*?)[\"\\']")

func AllLinks(body []byte, baseHref string, urlNow string) []string {
	if urlNow == "" {
		return nil
	}
	var basePath *url.URL
	if baseHref == "" {
		basePath, _ = url.Parse(urlNow)
	} else {
		basePath, _ = url.Parse(baseHref)
	}

	var js = []byte("javascript")

	rs := reAlink.FindAllSubmatch(body, -1)
	var result []string
	for _, rr := range rs {
		for _, r := range rr {
			if bytes.HasPrefix(r, []byte("<")) {
				continue
			}
			if bytes.HasPrefix(r, []byte("#")) {
				continue
			}
			if bytes.HasPrefix(r, []byte("data:")) {
				continue
			}

			if bytes.HasPrefix(r, js) {
				continue
			}

			if len(r) >= len(js) && bytes.Equal(bytes.ToLower(r[0:len(js)]), js) {
				continue
			}

			str := string(r)
			relPath, err := url.Parse(str)
			if err != nil {
				continue
			}

			relPath.Fragment = ""
			scheme := relPath.Scheme
			if relPath.Scheme == "" {
				relPath.Scheme = basePath.Scheme
			}

			if relPath.Host == "" {
				relPath.Host = basePath.Host
			}

			if scheme == "" && !strings.HasPrefix(str, "/") {
				up := basePath.String()
				pos := strings.LastIndex(up, "/")
				result = append(result, up[0:pos+1]+str)
			} else {
				result = append(result, relPath.String())
			}
		}
	}

	var last []string
	keys := make(map[string]bool)
	for _, k := range result {
		if _, has := keys[k]; has {
			continue
		}
		last = append(last, k)
		keys[k] = true
	}

	return last
}
