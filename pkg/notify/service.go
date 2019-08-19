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

package notify

import (
	"context"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/notify/cache"
	"yunion.io/x/onecloud/pkg/notify/models"
	"yunion.io/x/onecloud/pkg/notify/options"
	"yunion.io/x/onecloud/pkg/notify/rpc"
)

func StartService() {
	// parse options
	opts := &options.Options
	commonOpts := &options.Options.CommonOptions
	dbOpts := &options.Options.DBOptions
	baseOpts := &options.Options.BaseOptions
	common_options.ParseOptions(opts, os.Args, "notify.conf", "notify")

	// init auth
	app.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!")
	})

	// init handler
	applicaion := app.InitApp(baseOpts, true)
	InitHandlers(applicaion)

	// init database
	db.EnsureAppInitSyncDB(applicaion, dbOpts, models.InitDB)
	defer cloudcommon.CloseDB()

	// init cache
	cache.RegistUserCredCacheUpdater()

	// init notify service
	models.NotifyService = rpc.NewSRpcService(opts.SocketFileDir, models.ConfigManager)
	models.NotifyService.InitAll()
	defer models.NotifyService.StopAll()

	cron := cronman.GetCronJobManager(true)
	// update service
	cron.AddJobAtIntervals("UpdateServices", time.Duration(opts.UpdateInterval)*time.Second, models.NotifyService.UpdateServices)

	// resend notifications
	resend := func(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
		models.ReSend(opts.ReSendScope)
	}
	cron.AddJobAtIntervals("ReSendNotifications", time.Duration(opts.ReSendScope)*time.Minute, resend)

	app.ServeForever(applicaion, baseOpts)
}
