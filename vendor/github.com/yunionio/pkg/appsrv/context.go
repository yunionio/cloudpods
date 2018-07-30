package appsrv

import (
	"context"
	"database/sql"

	"github.com/yunionio/pkg/appctx"
)

func AppContextDB(ctx context.Context) *sql.DB {
	val := ctx.Value(appctx.APP_CONTEXT_KEY_DB)
	if val == nil {
		return nil
	}
	return val.(*sql.DB)
}

func AppContextCache(ctx context.Context) *Cache {
	val := ctx.Value(appctx.APP_CONTEXT_KEY_CACHE)
	if val == nil {
		return nil
	}
	return val.(*Cache)
}

func AppContextApp(ctx context.Context) *Application {
	val := ctx.Value(appctx.APP_CONTEXT_KEY_APP)
	if val == nil {
		return nil
	}
	return val.(*Application)
}

func (app *Application) SetContext(key appctx.AppContextKey, val interface{}) {
	app.context = context.WithValue(app.context, key, val)
}
