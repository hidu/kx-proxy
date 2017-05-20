package util

import (
	"bytes"
	"net/url"
	"regexp"
	"strings"
)

var reBase = regexp.MustCompile("base +href=\"(.*?)\"")
var reHTML = regexp.MustCompile("src=[\"\\'](.*?)[\"\\']|href=[\"\\'](.*?)[\"\\']|action=[\"\\'](.*?)[\"\\']")
var reCSS = regexp.MustCompile("url\\([\"\\']?(.*?)[\"\\']?\\)")

func encodeURL(src []byte, baseHref string, urlString string, start int, end int, expire int64) []byte {
	relURL := string(src[start:end])
	// keep anchor and javascript links intact
	if strings.Index(relURL, "#") == 0 || strings.Index(relURL, "javascript") == 0 {
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
	pu := NewProxyUrl(urlStrNew, expire)

	newURL, _ := pu.Encode()

	return bytes.Replace(src, src[start:end], []byte(newURL), -1)
}

// HTMLURLReplace 对html内容中的url替换 成代理的url地址
func HTMLURLReplace(body []byte, urlString string, expire int64) []byte {
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
				return encodeURL(s, baseHref, urlString, srcIndex[0], srcIndex[1], expire)
			}

			// replace href attribute
			hrefIndex := parts[4:6]
			if hrefIndex[0] != -1 {
				return encodeURL(s, baseHref, urlString, hrefIndex[0], hrefIndex[1], expire)
			}

			// replace form action attribute
			actionIndex := parts[6:8]
			if actionIndex[0] != -1 {
				return encodeURL(s, baseHref, urlString, actionIndex[0], actionIndex[1], expire)
			}
		}
		return s
	})
}

// CSSURLReplace 对css内容中的url替换为代理的地址
func CSSURLReplace(body []byte, urlString string, expire int64) []byte {
	baseHref := ""
	return reCSS.ReplaceAllFunc(body, func(s []byte) []byte {
		parts := reCSS.FindSubmatchIndex(s)
		if parts != nil {
			// replace url attribute in css
			pathIndex := parts[2:4]
			if pathIndex[0] != -1 {
				return encodeURL(s, baseHref, urlString, pathIndex[0], pathIndex[1], expire)
			}
		}
		return s
	})
}
