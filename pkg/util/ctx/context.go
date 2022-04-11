package ctx

import (
	"context"
	"time"
)

const (
	CTX_TIME_KEY = "time"
)

func CtxWithTime() context.Context {
	return context.WithValue(context.Background(), CTX_TIME_KEY, time.Now())
}
