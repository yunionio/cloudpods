package utils

import (
	"fmt"
	"strings"
	"unicode"
)

func KeepalivedConfHexQuote(s string) string {
	q := ""
	for _, r := range s {
		if strings.ContainsRune(`"'\#!`, r) || !unicode.IsPrint(r) {
			// cannot use \xNN here for keepalived bug
			q += fmt.Sprintf(`\%03c`, r)
		} else {
			q += string(r)
		}
	}
	q = fmt.Sprintf(`"%s"`, q)
	return q
}

func KeepalivedConfQuoteScriptArgs(args []string) string {
	for i, arg := range args {
		args[i] = KeepalivedConfHexQuote(arg)
	}
	s := strings.Join(args, " ")
	s = strings.Replace(s, `"`, `\"`, -1)
	s = fmt.Sprintf(`"%s"`, s)
	return s
}
