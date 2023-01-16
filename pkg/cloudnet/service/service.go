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

	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudnet/models"
	"yunion.io/x/onecloud/pkg/cloudnet/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

func StartService() {
	opts := &options.Options
	common_options.ParseOptions(opts, os.Args, "cloudnet.conf", "cloudnet")

	commonOpts := &opts.CommonOptions
	app_common.InitAuth(commonOpts, func() {
		log.Infof("Auth complete")
	})

	dbOpts := &opts.DBOptions
	baseOpts := &opts.BaseOptions

	app := app_common.InitApp(baseOpts, true).
		OnException(func(method, path string, body jsonutils.JSONObject, err error) {
			ctx := context.Background()
			session := auth.GetAdminSession(ctx, commonOpts.Region)
			notifyclient.EventNotifyServiceAbnormal(ctx, session.GetToken(), consts.GetServiceType(), method, path, body, err)
		})

	cloudcommon.InitDB(dbOpts)

	InitHandlers(app)

	db.EnsureAppSyncDB(app, dbOpts, models.InitDB)
	defer cloudcommon.CloseDB()

	app_common.ServeForever(app, baseOpts)
}
