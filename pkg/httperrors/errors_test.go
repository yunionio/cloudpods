package httperrors

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
		t.Run(c.name, func(t *testing.T) {
			msg, _ := errorMessage(c.msg, c.params...)
			if msg != c.out {
				t.Errorf("want %s, got %s", c.out, msg)
			}
		})
		t.Run(c.name+"_New", func(t *testing.T) {
			err := NewInputParameterError(c.msg, c.params...)
			if err.Details != c.out {
				t.Errorf("want %s, got %s", c.out, err.Details)
			}
		})
	}
}

func TestMsgToTemplate(t *testing.T) {
	cases := []struct {
		name   string
		msg    string
		params []interface{}
		out    string
	}{
		{
			name: "non-empty msg to template",
			msg:  "%% baremetals %s delete.time %d%",
			out:  "% baremetals {0} delete.time {1}%",
		},
		{
			name: "empty msg to template",
			msg:  "",
			out:  "",
		},
		{
			name: "non-empty with zh-utf8 characters msg to template",
			msg:  "%% baremetals %s 中文%d ¥%%",
			out:  "% baremetals {0} 中文{1} ¥%",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp := msgToTemplate(c.msg)
			if resp != c.out {
				t.Errorf("want %s, got %s", c.out, resp)
			}
		})
	}
}
