package util

import (
	"bytes"
	"regexp"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
)

var m = minify.New()

func init() {
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("text/html", html.Minify)
}

var brReg = regexp.MustCompile(`(<br>){2,}`)

func clean(code []byte) []byte {
	ret, err := m.Bytes("text/html", code)
	if err != nil {
		return code
	}
	ret = brReg.ReplaceAll(ret, []byte("<br>"))
	ret = bytes.ReplaceAll(ret, []byte(`<script>tj();</script>`), []byte(""))
	return ret
}
