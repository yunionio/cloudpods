package appsrv

import (
	"context"
	"net/http"

	"yunion.io/x/onecloud/pkg/appctx"
)

const (
	APP_CONTEXT_KEY_APP_PARAMS = appctx.AppContextKey("app_params")
)

type SAppParams struct {
	Name      string
	SkipLog   bool
	SkipTrace bool
	Params    map[string]string
	Path      []string

	Request  *http.Request
	Response http.ResponseWriter

	OverrideResponseBodyWrapper bool

	Cancel context.CancelFunc
}

func AppContextGetParams(ctx context.Context) *SAppParams {
	val := ctx.Value(APP_CONTEXT_KEY_APP_PARAMS)
	if val != nil {
		return val.(*SAppParams)
	} else {
		return nil
	}
}
