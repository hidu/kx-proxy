package util

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ProxyUrl struct {
	UrlStr    string     `json:"u"`
	Expire    int64      `json:"e"`
	ExpireAt  int64      `json:"a"`
	Sign      int64      `json:"s"`
	Extension Extensions `json:"x"`
	Ref       string     `json:"r"`
}

type Extensions []string

func (es Extensions) Has(key string) bool {
	if len(es) == 0 {
		return false
	}
	for _, k := range es {
		if k == key {
			return true
		}
	}
	return false
}

func (es Extensions) Preloading() bool {
	return es.Has("pre")
}

func (es Extensions) PreloadingSameDir() bool {
	return es.Has("pre_sd")
}
func (es Extensions) PreloadingNext() bool {
	return es.Has("pre_next")
}

func (es Extensions) Cache() bool {
	return es.Has("cache")
}
func (es Extensions) NoCache() bool {
	return es.Has("no_cache")
}

var noJSReg = regexp.MustCompile(`(?is)<script.+?<\s*/\s*script>`)
var onReg = regexp.MustCompile(`\son.+["'].+['"]`) // onXXX=""

var noLinkReg = regexp.MustCompile(`(?is)<link\s.+?>`)
var noStyleReg = regexp.MustCompile(`(?is)<style.+?<\s*/\s*style>`)

var noImgReg = regexp.MustCompile(`(?is)<img\s.+?>`)
var noImgReg1 = regexp.MustCompile(`(?is)<input\s+[^>]*type=["\']?image["\']?\s.+?>`)

func (es Extensions) Rewrite(body []byte) []byte {
	if es.Has("no_js") {
		body = noJSReg.ReplaceAll(body, []byte("<!-- script ignore -->"))
		body = onReg.ReplaceAll(body, []byte(""))
	}
	if es.Has("no_css") {
		body = noLinkReg.ReplaceAll(body, []byte("<!-- link ignore -->"))
		body = noStyleReg.ReplaceAll(body, []byte("<!-- style ignore -->"))
	}

	if es.Has("no_images") {
		body = noImgReg.ReplaceAll(body, []byte("<!-- img ignore -->"))
		body = noImgReg1.ReplaceAllFunc(body, func(bs []byte) []byte {
			bs = bytes.ReplaceAll(bs, []byte("image"), []byte("button"))
			bs = bytes.ReplaceAll(bs, []byte("src="), []byte("img_ignore_src="))
			return bs
		})
	}
	if es.Has("clean") {
		body = clean(body)
	}
	return body
}

const URL_STOP_CHAR = '.'

func NewProxyUrl(urlStr string, old *ProxyUrl, r *http.Request) *ProxyUrl {
	r.ParseForm()
	pu := &ProxyUrl{
		UrlStr:    urlStr,
		Expire:    old.GetExpire(),
		ExpireAt:  0,
		Extension: old.Extension,
	}
	pu.setSign(r)
	pu.checkExpireAt()
	return pu
}

func (p *ProxyUrl) checkExpireAt() {
	if p.Expire > 0 {
		p.ExpireAt = time.Now().Unix() + p.Expire
	}
}

func (p *ProxyUrl) setSign(r *http.Request) {
	// only this check sign
	if p.Expire == 1800 {
		p.Sign = GetRequestSign(r)
	}
}

func (p *ProxyUrl) CheckSign(r *http.Request) bool {
	// only this check sign
	if p.Expire == 1800 {
		s := GetRequestSign(r)
		return p.Sign == s
	}
	return true
}

func (p *ProxyUrl) IsExpire() bool {
	if p.Expire < 1 {
		return false
	}

	return p.ExpireAt < time.Now().Unix()
}

func (p *ProxyUrl) GetUrlStr() string {
	return p.UrlStr
}

func (p *ProxyUrl) GetExpire() int64 {
	return p.Expire
}

func (p *ProxyUrl) Encode() (string, error) {
	p.checkExpireAt()

	bf, err := json.Marshal(p)
	if err != nil {
		return "", err
	}

	encodedURL, err := EncryptURL(string(bf))
	if err != nil {
		return "", fmt.Errorf("build url failed:%s", err.Error())
	}
	return fmt.Sprintf("%s%c", encodedURL, URL_STOP_CHAR), nil
}

func (p *ProxyUrl) SwitchUrl(urlNew string) {
	p.UrlStr = urlNew
	p.checkExpireAt()
}

func (p *ProxyUrl) SwitchPath(urlPath string) {
	u, err := url.Parse(p.UrlStr)
	if err == nil {
		u.Path = urlPath
		u.RawQuery = ""
		p.UrlStr = u.String()
	}
	p.checkExpireAt()
}

func (p *ProxyUrl) URLValues() url.Values {
	vs := url.Values{}
	vs.Add("url", p.UrlStr)
	vs.Add("expire", strconv.FormatInt(p.Expire, 10))
	vs.Add("expire", strconv.FormatInt(p.Expire, 10))
	vs.Add("ext", strings.Join(p.Extension, ","))
	return vs
}

func DecodeProxyUrl(encodedURL string) (p *ProxyUrl, err error) {
	if len(encodedURL) < 10 {
		return nil, fmt.Errorf("path is too short")
	}

	arr := strings.SplitN(encodedURL, string(URL_STOP_CHAR), 2)
	urlStr := arr[0]

	urlData, err := DecryptURL(urlStr)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(urlData), &p)
	if err != nil {
		return nil, err
	}

	if len(arr[1]) > 0 {
		p.UrlStr += arr[1]
	}

	return p, nil
}

func GetRequestSign(r *http.Request) int64 {
	info := strings.SplitN(r.RemoteAddr, ":", 2)
	str := fmt.Sprintf("%s:%s", info[0], r.UserAgent())
	fmt.Println("sign_str:", str)
	md5Val := strMd5(str)
	b_buf := bytes.NewBuffer([]byte(md5Val))
	var x int64
	binary.Read(b_buf, binary.BigEndian, &x)
	return x % 2017
}
