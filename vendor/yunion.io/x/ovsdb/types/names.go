package types

import (
	"unicode"
	"unicode/utf8"
)

func Kebab2Camel(s string) string {
	var (
		rs = make([]rune, 0, len(s))
		up bool
	)
	for _, r := range s {
		if r != '_' {
			if up {
				r = unicode.ToUpper(r)
				up = false
			}
			rs = append(rs, r)
		} else {
			up = true
		}
	}
	return string(rs)
}

func ExportName(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}
