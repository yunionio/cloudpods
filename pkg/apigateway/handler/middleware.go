package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apigateway/clientman"
	"yunion.io/x/onecloud/pkg/apigateway/constants"
	"yunion.io/x/onecloud/pkg/appctx"
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
		return nil, errors.New("invalid base64url encoding")
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

func setAuthHeader(w http.ResponseWriter, tid string) {
	w.Header().Set(constants.AUTH_HEADER, fmt.Sprintf("%s%s", constants.AUTH_PREFIX, tid))
}

func SetAuthToken(ctx context.Context, w http.ResponseWriter, r *http.Request) (context.Context, error) {
	var token mcclient.TokenCredential
	auth1 := getAuthToken(r)
	auth := "" // getAuthToken(r)
	if len(auth) == 0 {
		authCookieStr := getAuthCookie(r)
		if len(authCookieStr) > 0 {
			authCookie, err := jsonutils.ParseString(authCookieStr)
			if err != nil {
				return ctx, fmt.Errorf("Auth cookie decode error: %s", err)
			}
			auth, err = authCookie.GetString("session")
			if err != nil {
				return ctx, err
			}
		}
	}
	if len(auth) > 0 && auth != auth1 { // hack!!! browser cache problem???
		log.Errorf("XXXX Auth cookie and header mismatch!!! %s:%s", auth, auth1)
		auth = auth1
	}
	if len(auth) > 0 {
		token = clientman.TokenMan.Get(auth)
	}
	if token == nil {
		return ctx, fmt.Errorf("No token in header")
	} else if !token.IsValid() {
		return ctx, fmt.Errorf("Token in header invalid")
	}
	setAuthHeader(w, auth)
	ctx = context.WithValue(ctx, appctx.AppContextKey(constants.AUTH_TOKEN), token)
	return ctx, nil
}

func FetchAuthToken(f func(context.Context, http.ResponseWriter, *http.Request)) func(context.Context, http.ResponseWriter, *http.Request) {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		ctx, err := SetAuthToken(ctx, w, r)
		if err != nil {
			httperrors.InvalidCredentialError(w, "invalid token: %v", err)
			return
		}
		// 启用双因子认证
		t := AppContextToken(ctx)
		if isUserEnableTotp(ctx, r, t) {
			tid := getAuthToken(r)
			totp := clientman.TokenMan.GetTotp(tid)
			if !totp.IsVerified() {
				httperrors.UnauthorizedError(w, "TOTP authentication failed")
				return
			}
		}

		f(ctx, w, r)
	}
}
