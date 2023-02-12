package links

import (
	"bytes"
	"io"
	"regexp"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
)

var m = minify.New()

func init() {
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("text/html", htmlMinify)
}

func htmlMinify(m *minify.M, w io.Writer, r io.Reader, params map[string]string) error {
	ins := &html.Minifier{
		KeepQuotes:     true,
		KeepEndTags:    true,
		KeepWhitespace: true,
	}
	return ins.Minify(m, w, r, params)
}

var brReg = regexp.MustCompile(`(?is)(<\s*br\s*/?\s*>\s*?){2,}`)

func clean(code []byte) []byte {
	code = brReg.ReplaceAll(code, []byte("<br>"))
	code = bytes.ReplaceAll(code, []byte(`<script>tj();</script>`), []byte(""))
	ret, err := m.Bytes("text/html", code)
	if err != nil {
		return code
	}
	return ret
}
