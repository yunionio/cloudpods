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

package appsrv

import (
	"context"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/appctx"
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
	Body      jsonutils.JSONObject

	Request  *http.Request
	Response http.ResponseWriter

	OverrideResponseBodyWrapper bool

	// Cancel context.CancelFunc
}

func AppContextGetParams(ctx context.Context) *SAppParams {
	val := ctx.Value(APP_CONTEXT_KEY_APP_PARAMS)
	if val != nil {
		return val.(*SAppParams)
	} else {
		return nil
	}
}
