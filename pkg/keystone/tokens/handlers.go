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
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/cache"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/netutils2"
)

func AddHandler(app *appsrv.Application) {
	app.AddHandler2("POST", "/v2.0/tokens", authenticateTokensV2, nil, "auth_tokens_v2", nil)
	app.AddHandler2("POST", "/v3/auth/tokens", authenticateTokensV3, nil, "auth_tokens_v3", nil)
	app.AddHandler2("GET", "/v2.0/tokens/<token>", authenticateToken(verifyTokensV2), nil, "verify_tokens_v2", nil)
	app.AddHandler2("GET", "/v3/auth/tokens", authenticateToken(verifyTokensV3), nil, "verify_tokens_v3", nil)
	app.AddHandler2("GET", "/v3/auth/policies", authenticateToken(fetchTokenPolicies), nil, "fetch_token_policies", nil)
	app.AddHandler2("POST", "/v3/auth/policies", authenticateToken(postTokenPolicies), nil, "post_token_policies", nil)

	app.AddHandler2("DELETE", "/v3/auth/tokens", authenticateToken(invalidateTokenV3), nil, "delete_tokens_v3", nil)
	app.AddHandler2("GET", "/v3/auth/tokens/invalid", authenticateToken(fetchInvalidTokensV3), nil, "fetch_revoked_tokens_v3", nil)
}

func FetchAuthContext(authCtx mcclient.SAuthContext, r *http.Request) mcclient.SAuthContext {
	if len(authCtx.Source) == 0 {
		authCtx.Source = mcclient.AuthSourceAPI
	}
	if len(authCtx.Ip) == 0 || authCtx.Ip == "0.0.0.0" {
		authCtx.Ip = netutils2.GetHttpRequestIp(r)
	}
	return authCtx
}

func authenticateTokensV2(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, _, body := appsrv.FetchEnv(ctx, w, r)
	input := mcclient.SAuthenticationInputV2{}
	err := body.Unmarshal(&input)
	if err != nil {
		httperrors.InvalidInputError(ctx, w, "unrecognized input %s", err)
		return
	}
	input.Auth.Context = FetchAuthContext(input.Auth.Context, r)
	token, err := AuthenticateV2(ctx, input)
	if err != nil {
		log.Errorf("AuthenticateV2 error %s", err)
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	if token == nil {
		httperrors.UnauthorizedError(ctx, w, "unauthorized %s", err)
		return
	}
	appsrv.SendJSON(w, jsonutils.Marshal(token))

	models.UserManager.TraceLoginV2(ctx, &token.Access)
}

func authenticateTokensV3(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, _, body := appsrv.FetchEnv(ctx, w, r)
	if body == nil {
		httperrors.InvalidInputError(ctx, w, "fail to decode request body")
		return
	}
	input := mcclient.SAuthenticationInputV3{}
	err := body.Unmarshal(&input)
	if err != nil {
		httperrors.InvalidInputError(ctx, w, "unrecognized input %s", err)
		return
	}
	input.Auth.Context = FetchAuthContext(input.Auth.Context, r)
	token, err := AuthenticateV3(ctx, input)
	if err != nil {
		log.Errorf("AuthenticateV3 error %s", err)
		switch errors.Cause(err) {
		case sqlchemy.ErrDuplicateEntry:
			httperrors.ConflictError(ctx, w, "duplicate username")
		case httperrors.ErrTooManyAttempts,
			httperrors.ErrUserNotFound,
			httperrors.ErrUserDisabled,
			httperrors.ErrUserLocked,
			httperrors.ErrInvalidIdpStatus,
			httperrors.ErrWrongPassword:
			httperrors.GeneralServerError(ctx, w, err)
		default:
			httperrors.UnauthorizedError(ctx, w, "unauthorized %s", err)
		}
		return
	}
	if token == nil {
		httperrors.UnauthorizedError(ctx, w, "user not found or not enabled")
		return
	}
	w.Header().Set(api.AUTH_SUBJECT_TOKEN_HEADER, token.Id)

	appsrv.SendJSON(w, jsonutils.Marshal(token))

	models.UserManager.TraceLoginV3(ctx, token)
}

// swagger:parameters verifyTokensV2
type VerifyTokenV2Param struct {
	// keystone V2验证token
	// in:path
	// required:true
	Token string
}

// swagger:route GET /v2.0/tokens/{token} authentication verifyTokensV2
//
// keystone v2验证token API
//
// keystone v2验证token API
//
//	Responses:
//	  200: tokens_AuthenticateV2Output
func verifyTokensV2(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, _ := appsrv.FetchEnv(ctx, w, r)
	tokenStr := params["<token>"]

	cachedToken := cache.Get(tokenStr)
	if cachedToken != nil {
		if v2token, ok := cachedToken.(*mcclient.TokenCredentialV2); ok && v2token.IsValid() {
			ret := jsonutils.NewDict()
			ret.Add(jsonutils.Marshal(v2token), "access")
			appsrv.SendJSON(w, ret)
			return
		} else {
			cache.Remove(tokenStr)
		}
	}

	token, err := verifyCommon(ctx, w, tokenStr)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	user, err := models.UserManager.FetchUserExtended(token.UserId, "", "", "")
	if err != nil {
		httperrors.InvalidCredentialError(ctx, w, "invalid user")
		return
	}
	project, err := models.ProjectManager.FetchProject(token.ProjectId, "", "", "")
	if err != nil {
		httperrors.InvalidCredentialError(ctx, w, "invalid project")
		return
	}
	projExt, err := project.FetchExtend()
	if err != nil {
		httperrors.InvalidCredentialError(ctx, w, "invalid project")
		return
	}
	v2token, err := token.getTokenV2(ctx, user, projExt)
	if err != nil {
		httperrors.InternalServerError(ctx, w, "internal server error %s", err)
		return
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.Marshal(v2token), "access")
	appsrv.SendJSON(w, ret)
}

// swagger:parameters verifyTokensV3
type VerifyTokenV3Param struct {
	// keystone V3验证token
	// in:header
	// required:true
	Token string `json:"X-Subject-Token"`
}

// swagger:route GET /v3/auth/tokens authentication verifyTokensV3
//
// keystone v3验证token API
//
// keystone v3验证token API
//
//	Responses:
//	  200: tokens_AuthenticateV3Output
func verifyTokensV3(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	tokenStr := r.Header.Get(api.AUTH_SUBJECT_TOKEN_HEADER)

	cachedToken := cache.Get(tokenStr)
	if cachedToken != nil {
		if v3token, ok := cachedToken.(*mcclient.TokenCredentialV3); ok && v3token.IsValid() {
			w.Header().Set(api.AUTH_SUBJECT_TOKEN_HEADER, v3token.Id)
			v3token.Id = ""
			appsrv.SendJSON(w, jsonutils.Marshal(v3token))
			return
		} else {
			cache.Remove(tokenStr)
		}
	}

	token, err := verifyCommon(ctx, w, tokenStr)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	user, err := models.UserManager.FetchUserExtended(token.UserId, "", "", "")
	if err != nil {
		httperrors.InvalidCredentialError(ctx, w, "invalid user")
		return
	}
	var projExt *models.SProjectExtended
	var domain *models.SDomain
	if len(token.ProjectId) > 0 {
		project, err := models.ProjectManager.FetchProject(token.ProjectId, "", "", "")
		if err != nil {
			httperrors.InvalidCredentialError(ctx, w, "invalid project")
			return
		}
		projExt, err = project.FetchExtend()
		if err != nil {
			httperrors.InvalidCredentialError(ctx, w, "invalid project")
			return
		}
	} else if len(token.DomainId) > 0 {
		domain, err = models.DomainManager.FetchDomainById(token.DomainId)
		if err != nil {
			httperrors.InvalidCredentialError(ctx, w, "invalid domain")
			return
		}
	}

	v3token, err := token.getTokenV3(ctx, user, projExt, domain, api.SAccessKeySecretInfo{})
	if err != nil {
		httperrors.InternalServerError(ctx, w, "internal server error %s", err)
		return
	}
	w.Header().Set(api.AUTH_SUBJECT_TOKEN_HEADER, v3token.Id)
	v3token.Id = ""
	appsrv.SendJSON(w, jsonutils.Marshal(v3token))
}

func verifyCommon(ctx context.Context, w http.ResponseWriter, tokenStr string) (*SAuthToken, error) {
	adminToken := policy.FetchUserCredential(ctx)
	if adminToken == nil || len(tokenStr) == 0 {
		return nil, httperrors.NewForbiddenError("missing auth token")
	}
	result := policy.PolicyManager.Allow(rbacscope.ScopeSystem, adminToken, api.SERVICE_TYPE, "tokens", "perform", "auth")
	if result.Result.IsDeny() {
		return nil, httperrors.NewForbiddenError("%s not allow to auth", adminToken.GetUserName())
	}
	token, err := TokenStrDecode(ctx, tokenStr)
	if err != nil {
		return nil, httperrors.NewInvalidCredentialError(errors.Wrapf(err, "invalid token").Error())
	}
	return token, nil
}

func authenticateToken(f appsrv.FilterHandler) appsrv.FilterHandler {
	return authenticateTokenWithDelayDecision(f, true)
}

func authenticateTokenWithDelayDecision(f appsrv.FilterHandler, delayDecision bool) appsrv.FilterHandler {
	return auth.AuthenticateWithDelayDecision(f, delayDecision)
}
