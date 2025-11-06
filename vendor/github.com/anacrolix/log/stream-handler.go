package log

import (
	"fmt"
	"io"
)

type StreamHandler struct {
	W   io.Writer
	Fmt ByteFormatter
}

func (me StreamHandler) Handle(r Record) {
	r.Msg = r.Skip(1)
	me.W.Write(me.Fmt(r))
}

type ByteFormatter func(Record) []byte

// Formats like:
// [2023-12-02 14:49:32 +1100 NIL github.com/anacrolix/dht-indexer main.go:417]
//
//	error maintaining search db: signal received: interrupt
func twoLineFormatter(msg Record) []byte {
	b := []byte{'['}
	beforeLen := len(b)
	b = GetDefaultTimeAppendFormatter()(b)
	if len(b) != beforeLen {
		b = append(b, ' ')
	}
	b = append(b, msg.Level.LogString()...)
	for _, name := range msg.Names {
		b = append(b, ' ')
		b = append(b, name...)
	}
	b = append(b, "]\n  "...)
	b = append(b, msg.Text()...)
	msg.Values(func(value interface{}) (more bool) {
		b = append(b, ' ')
		if item, ok := value.(item); ok {
			b = fmt.Appendf(b, "%s=%s", item.key, item.value)
		} else {
			b = fmt.Append(b, value)
		}
		return true
	})
	if b[len(b)-1] != '\n' {
		b = append(b, '\n')
	}
	return b
}

// Formats like: "[2023-12-02 14:34:02 +1100 INF] prefix: text [name name import-path short-file:line]"
func LineFormatter(msg Record) []byte {
	b := []byte{'['}
	beforeLen := len(b)
	b = GetDefaultTimeAppendFormatter()(b)
	if len(b) != beforeLen {
		b = append(b, ' ')
	}
	b = append(b, msg.Level.LogString()...)
	b = append(b, "] "...)
	b = append(b, msg.Text()...)
	b = append(b, " ["...)
	b = append(b, msg.Names[0]...)
	for _, name := range msg.Names[1:] {
		b = append(b, ' ')
		b = append(b, name...)
	}
	b = append(b, ']')
	if b[len(b)-1] != '\n' {
		b = append(b, '\n')
	}
	return b
}
