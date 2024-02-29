/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package util

import (
	"bytes"
	"runtime"
	"strings"
	"time"
	"unicode"
)

type stringutil struct{}

var StringUtil = &stringutil{}

/*----------------------------------------------------*/
func (StringUtil *stringutil) LineSeparator() string {
	var lineSeparator string
	if strings.Contains(runtime.GOOS, "windos") {
		lineSeparator = "\r\n"
	} else if strings.Contains(runtime.GOOS, "mac") {
		lineSeparator = "\r"
	} else {
		lineSeparator = "\n"
	}

	return lineSeparator
}

func (StringUtil *stringutil) Equals(str1 string, str2 string) bool {
	return str1 == str2
}

func (StringUtil *stringutil) EqualsIgnoreCase(str1 string, str2 string) bool {
	return strings.ToUpper(str1) == strings.ToUpper(str2)
}

func (StringUtil *stringutil) StartsWith(s string, subStr string) bool {
	return strings.Index(s, subStr) == 0
}

func (StringUtil *stringutil) StartWithIgnoreCase(s string, subStr string) bool {
	return strings.HasPrefix(strings.ToLower(s), strings.ToLower(subStr))
}

func (StringUtil *stringutil) EndsWith(s string, subStr string) bool {
	return strings.LastIndex(s, subStr) == len(s)-1
}

func (StringUtil *stringutil) IsDigit(str string) bool {
	if str == "" {
		return false
	}
	sz := len(str)
	for i := 0; i < sz; i++ {
		if unicode.IsDigit(rune(str[i])) {
			continue
		} else {
			return false
		}
	}
	return true
}

func (StringUtil *stringutil) FormatDir(dir string) string {
	dir = strings.TrimSpace(dir)
	if dir != "" {
		if !StringUtil.EndsWith(dir, PathSeparator) {
			dir += PathSeparator
		}
	}
	return dir
}

func (StringUtil *stringutil) HexStringToBytes(s string) []byte {
	str := s

	bs := make([]byte, 0)
	flag := false

	str = strings.TrimSpace(str)
	if strings.Index(str, "0x") == 0 || strings.Index(str, "0X") == 0 {
		str = str[2:]
	}

	if len(str) == 0 {
		return bs
	}

	var bsChr []byte
	l := len(str)

	if l%2 == 0 {
		bsChr = []byte(str)
	} else {
		l += 1
		bsChr = make([]byte, l)
		bsChr[0] = '0'
		for i := 0; i < l-1; i++ {
			bsChr[i+1] = str[i]
		}
	}

	bs = make([]byte, l/2)

	pos := 0
	for i := 0; i < len(bsChr); i += 2 {
		bt := convertHex(bsChr[i])
		bt2 := convertHex(bsChr[i+1])
		if int(bt) == 0xff || int(bt2) == 0xff {
			flag = true
			break
		}

		bs[pos] = byte(bt*16 + bt2)
		pos++
	}

	if flag {
		bs = ([]byte)(str)
	}

	return bs
}

func convertHex(chr byte) byte {
	if chr >= '0' && chr <= '9' {
		return chr - '0'
	} else if chr >= 'a' && chr <= 'f' {
		return chr - 'a' + 10
	} else if chr >= 'A' && chr <= 'F' {
		return chr - 'A' + 10
	} else {
		return 0xff
	}
}

func (StringUtil *stringutil) BytesToHexString(bs []byte, pre bool) string {
	if bs == nil {
		return ""
	}
	if len(bs) == 0 {
		return ""
	}

	hexDigits := "0123456789ABCDEF"
	ret := new(strings.Builder)
	for _, b := range bs {
		ret.WriteByte(hexDigits[0x0F&(b>>4)])
		ret.WriteByte(hexDigits[0x0F&b])
	}
	if pre {
		return "0x" + ret.String()
	}
	return ret.String()
}

func (StringUtil *stringutil) ProcessSingleQuoteOfName(name string) string {
	return StringUtil.processQuoteOfName(name, "'")
}

func (StringUtil *stringutil) ProcessDoubleQuoteOfName(name string) string {
	return StringUtil.processQuoteOfName(name, "\"")
}

func (StringUtil *stringutil) processQuoteOfName(name string, quote string) string {
	if quote == "" || name == "" {
		return name
	}

	temp := name
	result := bytes.NewBufferString("")
	index := -1
	quetoLength := len(quote)
	index = strings.Index(temp, quote)
	for index != -1 {
		result.WriteString(temp[:index+quetoLength])
		result.WriteString(quote)
		temp = temp[index+quetoLength:]
		index = strings.Index(temp, quote)
	}
	result.WriteString(temp)
	return result.String()
}

func (StringUtil *stringutil) FormatTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func (StringUtil *stringutil) SubstringBetween(str string, open string, close string) string {
	if str == "" {
		return ""
	}

	iopen := -1
	if open != "" {
		iopen = strings.Index(str, open)
	}

	iclose := -1
	if close != "" {
		iclose = strings.LastIndex(str, close)
	}

	if iopen == -1 && iclose == -1 {
		return ""
	} else if iopen == -1 {
		return str[0:iclose]
	} else if iclose == -1 {
		return str[iopen:]
	} else {
		return str[iopen:iclose]
	}
}
