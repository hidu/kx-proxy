package util

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type ProxyUrl struct {
	UrlStr   string `json:"u"`
	Expire   int64  `json:"e"`
	ExpireAt int64  `json:"a"`
}

const URL_STOP_CHAR = '.'

func NewProxyUrl(urlStr string, expire int64) *ProxyUrl {
	pu := &ProxyUrl{
		UrlStr:   urlStr,
		Expire:   expire,
		ExpireAt: 0,
	}
	pu.checkExpireAt()
	return pu
}

func (p *ProxyUrl) checkExpireAt() {
	if p.Expire > 0 {
		p.ExpireAt = time.Now().Unix() + p.Expire
	}
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
