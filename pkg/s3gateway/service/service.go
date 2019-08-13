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
	"os"

	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/s3gateway"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/s3gateway/models"
	"yunion.io/x/onecloud/pkg/s3gateway/options"
)

func StartService() {
	opts := &options.Options
	commonOpts := &opts.CommonOptions
	baseOpts := &opts.BaseOptions
	dbOpts := &opts.DBOptions
	common_options.ParseOptions(opts, os.Args, "s3gateway.conf", api.SERVICE_TYPE)

	app_common.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})

	cloudcommon.InitDB(dbOpts)

	app := app_common.InitApp(&opts.BaseOptions, true)
	initHandlers(app)

	cloudcommon.InitDB(&opts.DBOptions)

	if !db.CheckSync(opts.AutoSyncTable) {
		log.Fatalf("database schema not in sync!")
	}

	models.InitDB()

	if opts.ExitAfterDBInit {
		log.Infof("Exiting after db initialization ...")
		os.Exit(0)
	}

	/*if !opts.IsSlaveNode {
		cron := cronman.GetCronJobManager(true)
		cron.AddJobAtIntervals("CleanPendingDeleteImages", time.Duration(options.Options.PendingDeleteCheckSeconds)*time.Second, models.ImageManager.CleanPendingDeleteImages)
		cron.AddJobAtIntervals("CalculateQuotaUsages", time.Duration(opts.CalculateQuotaUsageIntervalSeconds)*time.Second, models.QuotaManager.CalculateQuotaUsages)

		cron.Start()
	}*/

	cloudcommon.AppDBInit(app)
	app_common.ServeForeverWithCleanup(app, baseOpts, func() {
		cloudcommon.CloseDB()

	})
}
