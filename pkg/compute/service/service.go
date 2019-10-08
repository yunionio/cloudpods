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
	"time"

	_ "github.com/go-sql-driver/mysql"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	_ "yunion.io/x/onecloud/pkg/compute/guestdrivers"
	_ "yunion.io/x/onecloud/pkg/compute/hostdrivers"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	_ "yunion.io/x/onecloud/pkg/compute/regiondrivers"
	_ "yunion.io/x/onecloud/pkg/compute/storagedrivers"
	_ "yunion.io/x/onecloud/pkg/compute/tasks"

	_ "yunion.io/x/onecloud/pkg/multicloud/loader"
)

func StartService() {

	opts := &options.Options
	commonOpts := &options.Options.CommonOptions
	baseOpts := &options.Options.BaseOptions
	dbOpts := &options.Options.DBOptions
	common_options.ParseOptions(opts, os.Args, "region.conf", "compute")

	if opts.PortV2 > 0 {
		log.Infof("Port V2 %d is specified, use v2 port", opts.PortV2)
		commonOpts.Port = opts.PortV2
	}

	options.InitNameSyncResources()

	app_common.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})

	app := app_common.InitApp(baseOpts, true)
	InitHandlers(app)

	db.EnsureAppInitSyncDB(app, dbOpts, models.InitDB)
	defer cloudcommon.CloseDB()

	err := setInfluxdbRetentionPolicy()
	if err != nil {
		log.Errorf("setInfluxdbRetentionPolicy fail: %s", err)
	}

	models.InitSyncWorkers(options.Options.CloudSyncWorkerCount)

	if !opts.IsSlaveNode {
		cron := cronman.InitCronJobManager(true, options.Options.CronJobWorkerCount)
		cron.AddJobAtIntervals("CleanPendingDeleteServers", time.Duration(opts.PendingDeleteCheckSeconds)*time.Second, models.GuestManager.CleanPendingDeleteServers)
		cron.AddJobAtIntervals("CleanPendingDeleteDisks", time.Duration(opts.PendingDeleteCheckSeconds)*time.Second, models.DiskManager.CleanPendingDeleteDisks)
		cron.AddJobAtIntervals("CleanPendingDeleteLoadbalancers", time.Duration(opts.LoadbalancerPendingDeleteCheckInterval)*time.Second, models.LoadbalancerAgentManager.CleanPendingDeleteLoadbalancers)
		if opts.PrepaidExpireCheck {
			cron.AddJobAtIntervals("CleanExpiredPrepaidServers", time.Duration(opts.PrepaidExpireCheckSeconds)*time.Second, models.GuestManager.DeleteExpiredPrepaidServers)
		}
		cron.AddJobAtIntervals("StartHostPingDetectionTask", time.Duration(opts.HostOfflineDetectionInterval)*time.Second, models.HostManager.PingDetectionTask)

		cron.AddJobAtIntervalsWithStartRun("CalculateQuotaUsages", time.Duration(opts.CalculateQuotaUsageIntervalSeconds)*time.Second, models.QuotaManager.CalculateQuotaUsages, true)

		cron.AddJobAtIntervalsWithStartRun("AutoSyncCloudaccountTask", time.Duration(opts.CloudAutoSyncIntervalSeconds)*time.Second, models.CloudaccountManager.AutoSyncCloudaccountTask, true)

		cron.AddJobEveryFewHour("AutoDiskSnapshot", 1, 5, 0, models.DiskManager.AutoDiskSnapshot, false)
		cron.AddJobEveryFewHour("AutoSyncExtDiskSnapshot", 1, 5, 0, models.DiskManager.AutoSyncExtDiskSnapshot, false)
		cron.AddJobEveryFewDays("SyncSkus", opts.SyncSkusDay, opts.SyncSkusHour, 0, 0, models.SyncSkus, true)
		cron.AddJobEveryFewDays("StorageSnapshotsRecycle", 1, 2, 0, 0, models.StorageManager.StorageSnapshotsRecycle, false)

		cron.Start()
		defer cron.Stop()
	}

	app_common.ServeForever(app, baseOpts)
}
