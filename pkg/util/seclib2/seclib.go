package seclib2

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
)

const (
	DIGITS  = "23456789"
	LETTERS = "abcdefghjkmnpqrstuvwxyz"
	PUNC    = "()~@#$%^&*-+={}[]:;<>,.?/"
)

var CHARS = fmt.Sprintf("%s%s%s%s", DIGITS, LETTERS, strings.ToUpper(LETTERS), PUNC)

func RandomPassword2(width int) string {
	if width < 6 {
		width = 6
	}
	for {
		var buf bytes.Buffer
		digitsCnt := 0
		letterCnt := 0
		upperCnt := 0
		puncCnt := 0
		for i := 0; i < width; i += 1 {
			index := rand.Intn(len(CHARS))
			ch := CHARS[index]
			if strings.IndexByte(DIGITS, ch) >= 0 {
				digitsCnt += 1
			} else if strings.IndexByte(LETTERS, ch) >= 0 {
				letterCnt += 1
			} else if strings.IndexByte(LETTERS, ch+32) >= 0 {
				upperCnt += 1
			} else if strings.IndexByte(PUNC, ch) >= 0 {
				puncCnt += 1
			}
			buf.WriteByte(ch)
		}
		if digitsCnt > 1 && letterCnt > 1 && upperCnt > 1 && puncCnt >= 1 && puncCnt <= 2 {
			return buf.String()
		}
	}
	return ""
}
