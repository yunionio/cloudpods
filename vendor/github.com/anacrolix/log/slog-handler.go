package log

import (
	"context"
	"log/slog"
)

type slogHandler struct {
	l Logger
}

func (s slogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// See IsEnabledFor for reasons why we probably should just return true here.
	return s.l.IsEnabledFor(fromSlogLevel(level))
}

func (s slogHandler) Handle(ctx context.Context, record slog.Record) error {
	s.l.LazyLog(fromSlogLevel(record.Level), func() Msg { return Msg{slogMsg{record}} })
	return nil
}

func (s slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	//TODO implement me
	panic("implement me")
}

func (s slogHandler) WithGroup(name string) slog.Handler {
	//TODO implement me
	panic("implement me")
}

type slogMsg struct {
	record slog.Record
}

func (s slogMsg) Text() string {
	return s.record.Message
}

func (s slogMsg) Callers(skip int, pc []uintptr) int {
	if len(pc) >= 1 {
		pc[0] = s.record.PC
		return 1
	}
	return 0
}

func (s slogMsg) Values(callback valueIterCallback) {
	s.record.Attrs(func(attr slog.Attr) bool {
		return callback(item{attr.Key, attr.Value})
	})
}
