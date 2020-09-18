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

package httputils

import (
	"testing"
)

func TestVariadic(t *testing.T) {
	conv := func(v interface{}) interface{} { return v }
	cases := []struct {
		name   string
		msg    string
		params []interface{}
		out    string
	}{
		{
			name: "no params",
			msg:  "hello",
			out:  "hello",
		},
		{
			name: "no params (with fmt escape)",
			msg:  "hello %s %d %v",
			out:  "hello %s %d %v",
		},
		{
			name:   "with params (no fmt escape)",
			msg:    "hello",
			params: []interface{}{conv("world")},
			out:    "hello%!(EXTRA string=world)",
		},
		{
			name:   "with params (with fmt escape)",
			msg:    "hello %s",
			params: []interface{}{conv("world")},
			out:    "hello world",
		},
	}
	for _, c := range cases {
		t.Run(c.name+"_New", func(t *testing.T) {
			err := NewJsonClientError(400, "InputParameterError", c.msg, c.params...)
			if err.Details != c.out {
				t.Errorf("want %s, got %s", c.out, err.Details)
			}
		})
	}
}

func TestMsgFmtTmplConversion(t *testing.T) {
	cases := []struct {
		name    string
		msgFmt  string
		msgTmpl string
		msgFmt2 string
		params  []interface{}
		out     string
	}{
		{
			name:    "empty",
			msgFmt:  "",
			msgTmpl: "",
			msgFmt2: "",
		},
		{
			name:    "non-empty",
			msgFmt:  "%% baremetals %s delete.time %d%",
			msgTmpl: "% baremetals {0} delete.time {1}%",
			msgFmt2: "% baremetals %s delete.time %s%",
		},
		{
			name:    "non-empty with zh-utf8",
			msgFmt:  "%% baremetals %s 中文%d ¥%%",
			msgTmpl: "% baremetals {0} 中文{1} ¥%",
			msgFmt2: "% baremetals %s 中文%s ¥%",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			msgTmpl := msgFmtToTmpl(c.msgFmt)
			if msgTmpl != c.msgTmpl {
				t.Errorf("msgFmtToTmpl: want %s, got %s", c.msgTmpl, msgTmpl)
			}
			msgFmt2 := msgTmplToFmt(msgTmpl)
			if msgFmt2 != c.msgFmt2 {
				t.Errorf("msgTmplToFmt: want %s, got %s", c.msgFmt2, msgFmt2)
			}
		})
	}
}
