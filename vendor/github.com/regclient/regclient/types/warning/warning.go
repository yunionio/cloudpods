// Package warning is used to handle HTTP warning headers
package warning

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
)

type contextKey string

var key contextKey = "key"

type Warning struct {
	List []string
	Hook *func(context.Context, *logrus.Logger, string)
	mu   sync.Mutex
}

func (w *Warning) Handle(ctx context.Context, log *logrus.Logger, msg string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	// dedup
	for _, entry := range w.List {
		if entry == msg {
			return
		}
	}
	w.List = append(w.List, msg)
	// handle new warning if hook defined
	if w.Hook != nil {
		(*w.Hook)(ctx, log, msg)
	}
}

func NewContext(ctx context.Context, w *Warning) context.Context {
	return context.WithValue(ctx, key, w)
}

func FromContext(ctx context.Context) *Warning {
	wAny := ctx.Value(key)
	if wAny == nil {
		return nil
	}
	w, ok := wAny.(*Warning)
	if !ok {
		return nil
	}
	return w
}

func NewHook(log *logrus.Logger) *func(context.Context, *logrus.Logger, string) {
	hook := func(_ context.Context, _ *logrus.Logger, msg string) {
		logMsg(log, msg)
	}
	return &hook
}

func DefaultHook() *func(context.Context, *logrus.Logger, string) {
	hook := func(_ context.Context, log *logrus.Logger, msg string) {
		logMsg(log, msg)
	}
	return &hook
}

func Handle(ctx context.Context, log *logrus.Logger, msg string) {
	// check for context
	if w := FromContext(ctx); w != nil {
		w.Handle(ctx, log, msg)
		return
	}

	// fallback to log
	logMsg(log, msg)
}

func logMsg(log *logrus.Logger, msg string) {
	log.WithFields(logrus.Fields{
		msg: msg,
	}).Warn("Registry warning message")
}
