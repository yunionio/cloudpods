package aws

import (
	"context"
	"github.com/ks3sdklib/aws-sdk-go/internal/apierr"
)

type Context = context.Context

// SetContext adds a Context to the current request that can be used to cancel
func (r *Request) SetContext(ctx Context) {
	if ctx == nil {
		r.Error = apierr.New("InvalidParameter", "context cannot be nil", nil)
	}
	r.context = ctx
	r.HTTPRequest = r.HTTPRequest.WithContext(ctx)
}

// BackgroundContext returns a context that will never be canceled
func BackgroundContext() Context {
	return context.Background()
}
