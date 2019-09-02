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

package handlers

import (
	"context"
	"net/http"

	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

const (
	S3_OBJECT_REQUEST = appctx.AppContextKey("S3_OBJECT_REQUEST")
)

func s3authenticate(f appsrv.FilterHandler) appsrv.FilterHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		log.Debugf("%s %s %s %s", r.Method, r.Host, r.URL, r.Header)

		o, err := getObjectRequest(r)
		if err != nil {
			SendError(w, BadRequest(ctx, err.Error()))
			return
		}
		ctx = context.WithValue(ctx, S3_OBJECT_REQUEST, o)
		userCred, err := auth.VerifyRequest(*r, o.VirtualHost)
		if err != nil {
			SendError(w, Unauthenticated(ctx, err.Error()))
			return
		}
		ctx = context.WithValue(ctx, auth.AUTH_TOKEN, userCred)

		f(ctx, w, r)
	}
}

func fetchObjectRequest(ctx context.Context) SObjectRequest {
	val := ctx.Value(S3_OBJECT_REQUEST)
	if gotypes.IsNil(val) {
		return SObjectRequest{}
	}
	return val.(SObjectRequest)
}
