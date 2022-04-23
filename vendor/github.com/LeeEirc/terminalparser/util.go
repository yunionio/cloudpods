package terminalparser

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

func DebugString(p string) string {
	var s strings.Builder
	for _, v := range []rune(p) {
		if unicode.IsPrint(v) {
			s.WriteRune(v)
		} else {
			s.WriteString(fmt.Sprintf("%q", v))
		}
	}
	return s.String()
}

func IsAlphabetic(r rune) bool {
	index := bytes.IndexRune([]byte(string(Alphabetic)), r)
	if index < 0 {
		return false
	}
	return true
}

func ReadRunePacket(p []byte) (code rune, rest []byte) {
	r, l := utf8.DecodeRune(p)
	if r == utf8.RuneError {
		return utf8.RuneError, p
	}
	return r, p[l:]
}