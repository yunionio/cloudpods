package appctx

import (
	"context"
)

var (
	Background context.Context
)

func init() {
	Background = context.Background()
}
