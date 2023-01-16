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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	_ "yunion.io/x/sqlchemy/backends"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudid/models"
	"yunion.io/x/onecloud/pkg/cloudid/options"
	_ "yunion.io/x/onecloud/pkg/cloudid/policy"
	"yunion.io/x/onecloud/pkg/cloudid/saml"
	_ "yunion.io/x/onecloud/pkg/cloudid/tasks"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	_ "yunion.io/x/onecloud/pkg/multicloud/loader"
)

func StartService() {
	opts := &options.Options
	dbOpts := &opts.DBOptions
	baseOpts := &opts.BaseOptions
	commonOpts := &opts.CommonOptions
	common_options.ParseOptions(opts, os.Args, "cloudid.conf", api.SERVICE_TYPE)

	app_common.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})
	common_options.StartOptionManager(opts, opts.ConfigSyncPeriodSeconds, api.SERVICE_TYPE, api.SERVICE_VERSION, options.OnOptionsChange)

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

	err := saml.InitSAML(app, api.SAML_IDP_PREFIX)
	if err != nil {
		log.Errorf("SAML initialization fail %s", err)
		return
	}

	if !opts.IsSlaveNode {
		cron := cronman.InitCronJobManager(true, options.Options.CronJobWorkerCount)
		cron.AddJobAtIntervalsWithStartRun("SyncCloudaccounts", time.Duration(opts.CloudaccountSyncIntervalMinutes)*time.Minute, models.CloudaccountManager.SyncCloudaccounts, true)
		cron.AddJobAtIntervalsWithStartRun("SyncSAMLProviders", time.Duration(opts.SAMLProviderSyncIntervalHours)*time.Hour, models.CloudaccountManager.SyncSAMLProviders, true)
		cron.AddJobAtIntervalsWithStartRun("SyncSystemCloudpolicies", time.Duration(opts.SystemPoliciesSyncIntervalHours)*time.Hour, models.CloudaccountManager.SyncCloudidSystemPolicies, true)
		cron.AddJobAtIntervalsWithStartRun("SyncCloudIdResources", time.Duration(opts.CloudIdResourceSyncIntervalHours)*time.Hour, models.CloudaccountManager.SyncCloudidResources, true)
		cron.AddJobAtIntervalsWithStartRun("SyncCloudroles", time.Duration(opts.CloudroleSyncIntervalHours)*time.Hour, models.CloudaccountManager.SyncCloudroles, true)

		cron.AddJobEveryFewHour("AutoPurgeSplitable", 4, 30, 0, db.AutoPurgeSplitable, false)

		cron.Start()
		defer cron.Stop()
	}

	app_common.ServeForever(app, baseOpts)
}
