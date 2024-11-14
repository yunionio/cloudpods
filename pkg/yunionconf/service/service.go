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

package service

import (
	"context"
	"os"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	_ "yunion.io/x/sqlchemy/backends"

	api "yunion.io/x/onecloud/pkg/apis/yunionconf"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/yunionconf/models"
	"yunion.io/x/onecloud/pkg/yunionconf/options"
	"yunion.io/x/onecloud/pkg/yunionconf/policy"
)

func StartService() {

	opts := &options.Options
	baseOpts := &options.Options.BaseOptions
	commonOpts := &options.Options.CommonOptions
	dbOpts := &options.Options.DBOptions
	common_options.ParseOptions(opts, os.Args, "yunionconf.conf", api.SERVICE_TYPE)
	policy.Init()
	app_common.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})

	cloudcommon.InitDB(dbOpts)

	app := app_common.InitApp(baseOpts, true).
		OnException(func(method, path string, body jsonutils.JSONObject, err error) {
			ctx := context.Background()
			session := auth.GetAdminSession(ctx, baseOpts.Region)
			notifyclient.EventNotifyServiceAbnormal(ctx, session.GetToken(), consts.GetServiceType(), method, path, body, err)
		})
	InitHandlers(app)
	db.AppDBInit(app)

	if db.CheckSync(opts.AutoSyncTable, opts.EnableDBChecksumTables, opts.DBChecksumSkipInit) {
		err := models.InitDB()
		if err == nil {
			app_common.ServeForeverWithCleanup(app, baseOpts, func() {
				cloudcommon.CloseDB()
			})
		} else {
			log.Errorf("InitDB fail: %s", err)
		}
	}
}
