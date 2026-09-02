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

package server

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"

	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

const yunionAuthCookie = "yunionauth"

// AuthenticateSftp authenticates SFTP HTTP endpoints.
// Browser calls go through apigateway or hit webconsole directly with the
// yunionauth cookie (no X-Auth-Token).
func AuthenticateSftp(f appsrv.FilterHandler) appsrv.FilterHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		token, err := fetchSftpUserCred(ctx, r)
		if err != nil || token == nil {
			log.Errorf("sftp auth failed: %v", err)
			httperrors.UnauthorizedError(ctx, w, "Unauthorized")
			return
		}
		ctx = context.WithValue(ctx, appctx.APP_CONTEXT_KEY_AUTH_TOKEN, token)
		f(ctx, w, r)
	}
}

func fetchSftpUserCred(ctx context.Context, r *http.Request) (mcclient.TokenCredential, error) {
	if tokenStr := r.Header.Get(identityapi.AUTH_TOKEN_HEADER); tokenStr != "" {
		token, err := auth.DefaultTokenVerifier(ctx, tokenStr)
		if err != nil {
			return nil, errors.Wrap(err, "verify X-Auth-Token")
		}
		return token, nil
	}

	return tokenFromYunionAuthCookie(r)
}

func tokenFromYunionAuthCookie(r *http.Request) (mcclient.TokenCredential, error) {
	cookie, err := r.Cookie(yunionAuthCookie)
	if err != nil || cookie == nil || cookie.Value == "" {
		return nil, errors.Wrap(httperrors.ErrInvalidCredential, "no auth token or yunionauth cookie")
	}
	raw, err := decodeYunionAuthCookie(cookie.Value)
	if err != nil {
		return nil, errors.Wrap(err, "decode yunionauth cookie")
	}
	info, err := jsonutils.ParseString(raw)
	if err != nil {
		return nil, errors.Wrap(err, "parse yunionauth cookie")
	}
	if expStr, _ := info.GetString("exp"); expStr != "" {
		exp, err := time.Parse(time.RFC3339, expStr)
		if err == nil && time.Now().After(exp) {
			return nil, errors.Wrap(httperrors.ErrInvalidCredential, "yunionauth cookie expired")
		}
	}
	userId, _ := info.GetString("user_id")
	if userId == "" {
		return nil, errors.Wrap(httperrors.ErrInvalidCredential, "yunionauth cookie missing user_id")
	}
	user, _ := info.GetString("user")
	return &mcclient.SSimpleToken{
		User:   user,
		UserId: userId,
	}, nil
}

func decodeYunionAuthCookie(val string) (string, error) {
	s := strings.ReplaceAll(val, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	for len(s)%4 != 0 {
		s += "="
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
