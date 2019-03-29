// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
