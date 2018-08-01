package auth

import (
	"context"
	"net/http"

	"github.com/yunionio/log"
	"github.com/yunionio/onecloud/pkg/appctx"
	"github.com/yunionio/onecloud/pkg/appsrv"
	"github.com/yunionio/onecloud/pkg/httperrors"

	"github.com/yunionio/onecloud/pkg/mcclient"
)

const (
	AUTH_TOKEN = appctx.AppContextKey("X_AUTH_TOKEN")
)

func Authenticate(f appsrv.FilterHandler) appsrv.FilterHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		tokenStr := r.Header.Get("X-Auth-Token")
		if len(tokenStr) == 0 {
			httperrors.UnauthorizedError(w, "Unauthorized")
			return
		}
		token, err := Verify(tokenStr)
		if err != nil {
			log.Errorf("Verify token failed: %s", err)
			httperrors.UnauthorizedError(w, "InvalidToken")
			return
		}
		ctx = context.WithValue(ctx, AUTH_TOKEN, token)
		f(ctx, w, r)
	}
}

func FetchUserCredential(ctx context.Context) mcclient.TokenCredential {
	tokenValue := ctx.Value(AUTH_TOKEN)
	if tokenValue != nil {
		return tokenValue.(mcclient.TokenCredential)
	}
	return nil
}
