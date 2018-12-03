package auth

import (
	"context"
	"net/http"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"

	"yunion.io/x/onecloud/pkg/mcclient"
)

const (
	AUTH_TOKEN = appctx.AppContextKey("X_AUTH_TOKEN")
)

func Authenticate(f appsrv.FilterHandler) appsrv.FilterHandler {
	return AuthenticateWithDelayDecision(f, false)
}

func AuthenticateWithDelayDecision(f appsrv.FilterHandler, delayDecision bool) appsrv.FilterHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		tokenStr := r.Header.Get(mcclient.AUTH_TOKEN)
		if len(tokenStr) == 0 {
			log.Errorf("no auth_token found!")
			if !delayDecision {
				httperrors.UnauthorizedError(w, "Unauthorized")
				return
			}
		}
		token, err := Verify(tokenStr)
		if err != nil {
			log.Errorf("Verify token failed: %s", err)
			if !delayDecision {
				httperrors.UnauthorizedError(w, "InvalidToken")
				return
			}
		}
		if token != nil {
			ctx = context.WithValue(ctx, AUTH_TOKEN, token)
		}

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
	tokenValue := ctx.Value(AUTH_TOKEN)
	if tokenValue != nil {
		token := tokenValue.(mcclient.TokenCredential)
		if filter != nil {
			token = filter(token)
		}
		return token
	}
	return nil
}
