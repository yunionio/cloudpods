package log

import (
	"context"
	"io"
	"log/slog"
	"time"
)

type JsonHandler struct {
	// This is used to output JSON as it provides a more modern way and probably more efficient way
	// to modify log records. You can alter this in place after initing JsonHandler and before
	// logging to it.
	SlogHandler slog.Handler
}

func NewJsonHandler(w io.Writer) *JsonHandler {
	return &JsonHandler{
		SlogHandler: slog.NewJSONHandler(w, &slog.HandlerOptions{
			AddSource:   false,
			Level:       slog.LevelDebug - 4,
			ReplaceAttr: nil,
		}),
	}
}

var _ Handler = (*JsonHandler)(nil)

func (me *JsonHandler) Handle(r Record) {
	slogLevel, ok := toSlogLevel(r.Level)
	if !ok {
		panic(r.Level)
	}
	var pc [1]uintptr
	r.Callers(1, pc[:])
	slogRecord := slog.NewRecord(time.Now(), slogLevel, r.Msg.String(), pc[0])
	anyNames := make([]any, 0, len(r.Names))
	for _, name := range r.Names {
		anyNames = append(anyNames, name)
	}
	slogRecord.AddAttrs(slog.Any("names", r.Names))
	err := me.SlogHandler.Handle(context.Background(), slogRecord)
	if err != nil {
		panic(err)
	}
}
