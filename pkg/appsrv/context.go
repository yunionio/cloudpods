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

package appsrv

import (
	"context"
	"database/sql"

	"yunion.io/x/pkg/appctx"

	"yunion.io/x/onecloud/pkg/util/hashcache"
)

func AppContextDB(ctx context.Context) *sql.DB {
	val := ctx.Value(appctx.APP_CONTEXT_KEY_DB)
	if val == nil {
		return nil
	}
	return val.(*sql.DB)
}

func AppContextCache(ctx context.Context) *hashcache.Cache {
	val := ctx.Value(appctx.APP_CONTEXT_KEY_CACHE)
	if val == nil {
		return nil
	}
	return val.(*hashcache.Cache)
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
