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

package handler

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apigateway/clientman"
	"yunion.io/x/onecloud/pkg/apigateway/constants"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func Base64UrlEncode(data []byte) string {
	str := base64.StdEncoding.EncodeToString(data)
	str = strings.Replace(str, "+", "-", -1)
	str = strings.Replace(str, "/", "_", -1)
	str = strings.Replace(str, "=", "", -1)
	return str
}

func Base64UrlDecode(str string) ([]byte, error) {
	if strings.ContainsAny(str, "+/") {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "invalid base64url encoding")
	}
	str = strings.Replace(str, "-", "+", -1)
	str = strings.Replace(str, "_", "/", -1)
	for len(str)%4 != 0 {
		str += "="
	}
	return base64.StdEncoding.DecodeString(str)
}

func getAuthToken(r *http.Request) string {
	auth := r.Header.Get(constants.AUTH_HEADER)
	if len(auth) > 0 && auth[:len(constants.AUTH_PREFIX)] == constants.AUTH_PREFIX {
		return auth[len(constants.AUTH_PREFIX):]
	} else {
		return ""
	}
}

func getAuthCookie(r *http.Request) string {
	return getCookie(r, constants.YUNION_AUTH_COOKIE)
}

/*func setAuthHeader(w http.ResponseWriter, tid string) {
	w.Header().Set(constants.AUTH_HEADER, fmt.Sprintf("%s%s", constants.AUTH_PREFIX, tid))
}*/

func fetchAuthInfo(ctx context.Context, r *http.Request) (mcclient.TokenCredential, *clientman.SAuthToken, error) {
	var token mcclient.TokenCredential
	var authToken *clientman.SAuthToken

	// no more use Auth header
	// auth1 := getAuthToken(r)
	auth := getAuthToken(r)
	if len(auth) == 0 {
		authCookieStr := getAuthCookie(r)
		if len(authCookieStr) > 0 {
			authCookie, err := jsonutils.ParseString(authCookieStr)
			if err != nil {
				return nil, nil, errors.Wrap(httperrors.ErrInputParameter, "Auth cookie decode")
			}
			auth, err = authCookie.GetString("session")
			if err != nil {
				return nil, nil, errors.Wrap(httperrors.ErrInputParameter, "authCookie missing session field")
			}
		}
	}
	// if len(auth) > 0 && auth != auth1 { // hack!!! browser cache problem???
	//	log.Errorf("XXXX Auth cookie and header mismatch!!! %s:%s", auth, auth1)
	//	auth = auth1
	// }
	if len(auth) > 0 {
		var err error
		authToken, err = clientman.Decode(auth)
		if err != nil {
			return nil, nil, errors.Wrap(httperrors.ErrInputParameter, "clientman.Decode auth token fail")
		}
		token, err = authToken.GetToken(ctx)
		if err != nil {
			return nil, nil, errors.Wrap(httperrors.ErrInputParameter, "authToken.GetToken fail")
		}
	}
	if token == nil {
		return nil, nil, errors.Wrap(httperrors.ErrInvalidCredential, "No token in header")
	} else if !token.IsValid() {
		return nil, nil, errors.Wrap(httperrors.ErrInvalidCredential, "Token in header invalid")
	}
	return token, authToken, nil
}

func fetchAndSetAuthContext(ctx context.Context, w http.ResponseWriter, r *http.Request) (context.Context, error) {
	token, authToken, err := fetchAuthInfo(ctx, r)
	if err != nil {
		return ctx, errors.Wrap(err, "fetchAuthInfo")
	}
	// 启用双因子认证
	if !authToken.IsTotpVerified() {
		return ctx, errors.Wrap(httperrors.ErrInvalidCredential, "TOTP authentication failed")
	}
	// no more send auth header, save auth info in cookie
	// setAuthHeader(w, authHeader)
	ctx = context.WithValue(ctx, appctx.APP_CONTEXT_KEY_AUTH_TOKEN, token)
	return ctx, nil
}

func FetchAuthToken(f func(context.Context, http.ResponseWriter, *http.Request)) func(context.Context, http.ResponseWriter, *http.Request) {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		ctx, err := fetchAndSetAuthContext(ctx, w, r)
		if err != nil {
			httperrors.InvalidCredentialError(ctx, w, "No token in header: %v", err)
			return
		}
		f(ctx, w, r)
	}
}
