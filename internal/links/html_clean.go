package links

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
	// html.DefaultMinifier.KeepQuotes = true
	// html.DefaultMinifier.KeepEndTags = true
	// html.DefaultMinifier.KeepWhitespace = true
	m.AddFunc("text/html", html.Minify)
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
