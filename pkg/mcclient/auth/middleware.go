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

package auth

import (
	"context"
	"net/http"

	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

var (
	GUEST_USER  = "guest"
	GUEST_TOKEN = "guest_token"
	GuestToken  = mcclient.SSimpleToken{
		User:  GUEST_USER,
		Token: GUEST_TOKEN,
	}

	DefaultTokenVerifier = Verify
)

const (
	AUTH_TOKEN = appctx.APP_CONTEXT_KEY_AUTH_TOKEN
)

type TokenVerifyFunc func(string) (mcclient.TokenCredential, error)

func Authenticate(f appsrv.FilterHandler) appsrv.FilterHandler {
	return AuthenticateWithDelayDecision(f, false)
}

func AuthenticateWithDelayDecision(f appsrv.FilterHandler, delayDecision bool) appsrv.FilterHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		tokenStr := r.Header.Get(api.AUTH_TOKEN_HEADER)
		var token mcclient.TokenCredential
		if len(tokenStr) == 0 {
			log.Errorf("no auth_token found! delayDecision=%v", delayDecision)
			if !delayDecision {
				httperrors.UnauthorizedError(ctx, w, "Unauthorized")
				return
			}
			token = &GuestToken
		} else {
			var err error
			token, err = DefaultTokenVerifier(ctx, tokenStr)
			if err != nil {
				log.Errorf("Verify token failed: %s", err)
				if !delayDecision {
					httperrors.UnauthorizedError(ctx, w, "InvalidToken")
					return
				}
			}
		}
		ctx = context.WithValue(ctx, appctx.APP_CONTEXT_KEY_AUTH_TOKEN, token)

		if taskId := r.Header.Get(mcclient.TASK_ID); taskId != "" {
			ctx = context.WithValue(ctx, appctx.APP_CONTEXT_KEY_TASK_ID, taskId)
		}
		if taskNotifyUrl := r.Header.Get(mcclient.TASK_NOTIFY_URL); taskNotifyUrl != "" {
			ctx = context.WithValue(ctx, appctx.APP_CONTEXT_KEY_TASK_NOTIFY_URL, taskNotifyUrl)
		}

		f(ctx, w, r)
	}
}

func FetchUserCredential(ctx context.Context, filter func(mcclient.TokenCredential) mcclient.TokenCredential) mcclient.TokenCredential {
	tokenValue := ctx.Value(appctx.APP_CONTEXT_KEY_AUTH_TOKEN)
	if tokenValue != nil {
		token := tokenValue.(mcclient.TokenCredential)
		if filter != nil {
			token = filter(token)
		}
		return token
	}
	return nil
}

func IsGuestToken(userCred rbacutils.IRbacIdentity) bool {
	return userCred.GetTokenString() == GUEST_TOKEN
}
