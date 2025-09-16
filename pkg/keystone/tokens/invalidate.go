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

package tokens

import (
	"context"
	"database/sql"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/models"
)

// swagger:route DELETE /v3/auth/tokens authentication invalidateTokensV3
//
// keystone v3删除token API
//
// keystone v3删除token API
func invalidateTokenV3(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	tokenStr := r.Header.Get(api.AUTH_SUBJECT_TOKEN_HEADER)
	err := invalidateToken(ctx, tokenStr)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendNoContent(w)
}

func invalidateToken(ctx context.Context, tokenStr string) error {
	adminToken := policy.FetchUserCredential(ctx)
	if adminToken == nil || len(tokenStr) == 0 {
		return httperrors.NewForbiddenError("missing auth token")
	}
	token, err := TokenStrDecode(ctx, tokenStr)
	if err != nil {
		return httperrors.NewInvalidCredentialError(errors.Wrapf(err, "invalid token").Error())
	}
	if adminToken.GetUserId() != token.UserId && policy.PolicyManager.Allow(rbacscope.ScopeSystem, adminToken, api.SERVICE_TYPE, "tokens", "delete").Result.IsDeny() {
		return httperrors.NewForbiddenError("%s not allow to delete token", adminToken.GetUserName())
	}
	err = models.TokenCacheManager.Invalidate(ctx, adminToken, tokenStr)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return errors.Wrap(ErrTokenNotFound, tokenStr)
		} else {
			return errors.Wrap(err, "Invalidate")
		}
	}
	return nil
}

// swagger:route GET /v3/auth/tokens/invalid authentication fetchInvalidTokensV3
//
// keystone v3获取被删除的token的列表API
//
// keystone v3获取被删除的token的列表API
func fetchInvalidTokensV3(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	tokens, err := fetchInvalidTokens(ctx)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	resp := jsonutils.NewDict()
	resp.Add(jsonutils.NewStringArray(tokens), "tokens")
	appsrv.SendJSON(w, resp)
}

func fetchInvalidTokens(ctx context.Context) ([]string, error) {
	adminToken := policy.FetchUserCredential(ctx)
	if adminToken == nil {
		return nil, httperrors.NewForbiddenError("missing auth token")
	}
	if policy.PolicyManager.Allow(rbacscope.ScopeSystem, adminToken, api.SERVICE_TYPE, "tokens", "list", "invalid").Result.IsDeny() {
		return nil, httperrors.NewForbiddenError("%s not allow to list invalid tokens", adminToken.GetUserName())
	}
	tokens, err := models.TokenCacheManager.FetchInvalidTokens()
	if err != nil {
		return nil, errors.Wrap(err, "TokenCacheManager.FetchInvalidTokens")
	}
	return tokens, nil
}
