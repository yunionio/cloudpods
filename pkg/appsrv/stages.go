package appsrv

import (
	"context"
	"net/http"
)

type MiddlewareFunc func(f func(context.Context, http.ResponseWriter, *http.Request)) func(context.Context, http.ResponseWriter, *http.Request)

func (app *Application) RegisterMiddleware(f MiddlewareFunc) {
	app.middlewares = append(app.middlewares, f)
}
