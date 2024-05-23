package links

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ProxyURL struct {
	URLStr    string     `json:"u"`
	Expire    int64      `json:"e"`
	ExpireAt  int64      `json:"a"`
	Sign      int64      `json:"s"`
	Extension Extensions `json:"x"`
	Ref       string     `json:"r"`
	UserCss   string     `json:"uc"` // 页面自定义的 css

	ctxParams map[any]any
}

const _URLStopChar = '.'

func NewProxyURL(urlStr string, old *ProxyURL, r *http.Request) *ProxyURL {
	pu := &ProxyURL{
		URLStr:    urlStr,
		Expire:    old.GetExpire(),
		ExpireAt:  0,
		Extension: old.Extension,
		UserCss:   r.FormValue("ucss"),
	}
	pu.setSign(r)
	pu.checkExpireAt()
	return pu
}

const (
	ctxParamsKeyNoCache = "no_cache"
)

func (p *ProxyURL) SetNoCache() {
	p.SetCtxParams(ctxParamsKeyNoCache, 1)
}

func (p *ProxyURL) SetCtxParams(key any, val any) {
	if p.ctxParams == nil {
		p.ctxParams = make(map[any]any, 1)
	}
	p.ctxParams[key] = val
}

func (p *ProxyURL) GetCtxParams(key any) any {
	if p.ctxParams == nil {
		return nil
	}
	return p.ctxParams[key]
}

func (p *ProxyURL) checkExpireAt() {
	if p.Expire > 0 {
		p.ExpireAt = time.Now().Unix() + p.Expire
	}
}

func (p *ProxyURL) setSign(r *http.Request) {
	// only this check sign
	if p.Expire == 1800 {
		p.Sign = GetRequestSign(r)
	}
}

func (p *ProxyURL) CheckSign(r *http.Request) bool {
	// only this check sign
	if p.Expire == 1800 {
		s := GetRequestSign(r)
		return p.Sign == s
	}
	return true
}

func (p *ProxyURL) IsExpire() bool {
	if p.Expire < 1 {
		return false
	}

	return p.ExpireAt < time.Now().Unix()
}

func (p *ProxyURL) GetURLStr() string {
	return p.URLStr
}

func (p *ProxyURL) GetExpire() int64 {
	return p.Expire
}

func (p *ProxyURL) Encode() (string, error) {
	p.checkExpireAt()

	bf, err := json.Marshal(p)
	if err != nil {
		return "", err
	}

	encodedURL, err := EncryptURL(string(bf))
	if err != nil {
		return "", fmt.Errorf("build url failed: %w", err)
	}
	return fmt.Sprintf("%s%c", encodedURL, _URLStopChar), nil
}

func (p *ProxyURL) SwitchURL(urlNew string) {
	p.URLStr = urlNew
	p.checkExpireAt()
}

func (p *ProxyURL) SwitchPath(urlPath string) {
	u, err := url.Parse(p.URLStr)
	if err == nil {
		u.Path = urlPath
		u.RawQuery = ""
		p.URLStr = u.String()
	}
	p.checkExpireAt()
}

func (p *ProxyURL) ToHomeData() map[string]any {
	datas := map[string]any{
		"url":    p.URLStr,
		"expire": strconv.FormatInt(p.Expire, 10),
		"ext":    strings.Join(p.Extension, ","),
		"mp":     p.UserCss != "",
		"ucss":   p.UserCss,
	}
	return datas
}

func (p *ProxyURL) CacheAble() bool {
	if p.NoCache() {
		return false
	}
	if p.Extension.Cache() {
		return true
	}
	if p.IsStaticURL() && p.Extension.CacheStatic() {
		return true
	}
	return false
}

func (p *ProxyURL) NoCache() bool {
	if val := p.GetCtxParams(ctxParamsKeyNoCache); val != nil {
		return true
	}
	return p.Extension.NoCache()
}

func emptyFunc() {}

func (p *ProxyURL) Context(ctx context.Context) (context.Context, context.CancelFunc) {
	if tl := p.Extension.Timeout(); tl > 0 {
		return context.WithTimeout(ctx, tl)
	}
	return ctx, emptyFunc
}

func (p *ProxyURL) HeadHTML() []byte {
	var bf bytes.Buffer
	if p.Extension.Has("raw_url") {
		bf.WriteString(`<input style='border:none' readonly type='text' value="`)
		bf.WriteString(p.GetURLStr())
		bf.WriteString(`">`)

		bf.WriteString(`&nbsp;<a class='raw_url' href="/?`)

		u, _ := p.Encode()
		uv := url.Values{}
		uv.Add("raw", u)
		// raw := p.URLValues().Encode()
		// bf.WriteString(raw)
		bf.WriteString(uv.Encode())
		bf.WriteString(`">`)
		bf.WriteString("Home")
		bf.WriteString("</a>")
	}

	if p.Extension.Cache() {
		bf.WriteString(`&nbsp;<a href="?cache=no">NoCache</a>`)
	}

	if bf.Len() > 0 {
		bf.WriteString("<br/>\n")
	}

	return bf.Bytes()
}

func (p *ProxyURL) IsStaticURL() bool {
	return IsStaticPath(p.URLStr)
}

func (p *ProxyURL) Hostname() string {
	ru, erru := url.Parse(p.URLStr)
	if erru == nil {
		return ru.Hostname()
	}
	panic(erru)
}

func DecodeProxyURL(encodedURL string) (p *ProxyURL, err error) {
	if len(encodedURL) < 10 {
		return nil, errors.New("path is too short")
	}

	arr := strings.SplitN(encodedURL, string(_URLStopChar), 2)
	if len(arr) != 2 {
		return nil, errors.New("invalid encodedURL")
	}

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
		p.URLStr += arr[1]
	}

	return p, nil
}

func GetRequestSign(r *http.Request) int64 {
	info := strings.SplitN(r.RemoteAddr, ":", 2)
	str := fmt.Sprintf("%s:%s", info[0], r.UserAgent())
	md5Val := strMd5(str)
	buf := bytes.NewBuffer(md5Val)
	var x int64
	_ = binary.Read(buf, binary.BigEndian, &x)
	return x % 2017
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

func (es Extensions) CacheStatic() bool {
	return es.Has("cache_static")
}

func (es Extensions) NoCache() bool {
	return es.Has("no_cache")
}

func (es Extensions) SkipVerify() bool {
	return es.Has("SkipVerify")
}

func (es Extensions) InverseColor() bool {
	return es.Has("ic")
}

func (es Extensions) Timeout() time.Duration {
	if es.Has("tl_60") {
		return 60 * time.Second
	}
	if es.Has("tl_30") {
		return 30 * time.Second
	}
	if es.Has("tl_10") {
		return 10 * time.Second
	}
	if es.Has("tl_5") {
		return 5 * time.Second
	}
	return 0
}

func (es Extensions) TryTotal() int {
	if es.Has("retry_2") {
		return 3
	}
	if es.Has("retry_1") {
		return 2
	}
	return 1
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
