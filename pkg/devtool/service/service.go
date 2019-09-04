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

	_ "github.com/go-sql-driver/mysql"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/devtool/models"
	"yunion.io/x/onecloud/pkg/devtool/options"
)

func StartService() {
	opts := &options.Options
	commonOpts := &opts.CommonOptions
	dbOpts := &options.Options.DBOptions
	baseOpts := &opts.BaseOptions
	common_options.ParseOptions(opts, os.Args, "devtool.conf", "devtool")

	app_common.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})

	app := app_common.InitApp(&opts.BaseOptions, false)

	db.EnsureAppInitSyncDB(app, dbOpts, models.InitDB)

	InitHandlers(app)
	app_common.ServeForeverWithCleanup(app, baseOpts, func() {
		cloudcommon.CloseDB()
	})
}
