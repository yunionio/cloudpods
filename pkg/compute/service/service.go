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
	_ "yunion.io/x/onecloud/pkg/util/aliyun/provider"
	_ "yunion.io/x/onecloud/pkg/util/aws/provider"
	_ "yunion.io/x/onecloud/pkg/util/azure/provider"
	_ "yunion.io/x/onecloud/pkg/util/esxi/provider"
	_ "yunion.io/x/onecloud/pkg/util/huawei/provider"
	_ "yunion.io/x/onecloud/pkg/util/openstack/provider"
	_ "yunion.io/x/onecloud/pkg/util/qcloud/provider"
	_ "yunion.io/x/onecloud/pkg/util/ucloud/provider"
)

func StartService() {

	opts := &options.Options
	commonOpts := &options.Options.CommonOptions
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

	cloudcommon.InitDB(dbOpts)
	defer cloudcommon.CloseDB()

	app := app_common.InitApp(commonOpts, true)
	cloudcommon.AppDBInit(app)
	InitHandlers(app)

	if !db.CheckSync(opts.AutoSyncTable) {
		log.Fatalf("database schema not in sync!")
	}

	err := models.InitDB()
	if err != nil {
		log.Errorf("InitDB fail: %s", err)
	}

	err = setInfluxdbRetentionPolicy()
	if err != nil {
		log.Errorf("setInfluxdbRetentionPolicy fail: %s", err)
	}

	models.InitSyncWorkers(options.Options.CloudSyncWorkerCount)

	cron := cronman.GetCronJobManager(true)
	cron.AddJob1("CleanPendingDeleteServers", time.Duration(opts.PendingDeleteCheckSeconds)*time.Second, models.GuestManager.CleanPendingDeleteServers)
	cron.AddJob1("CleanPendingDeleteDisks", time.Duration(opts.PendingDeleteCheckSeconds)*time.Second, models.DiskManager.CleanPendingDeleteDisks)
	cron.AddJob1("CleanPendingDeleteLoadbalancers", time.Duration(opts.LoadbalancerPendingDeleteCheckInterval)*time.Second, models.LoadbalancerAgentManager.CleanPendingDeleteLoadbalancers)
	if opts.PrepaidExpireCheck {
		cron.AddJob1("CleanExpiredPrepaidServers", time.Duration(opts.PrepaidExpireCheckSeconds)*time.Second, models.GuestManager.DeleteExpiredPrepaidServers)
	}
	cron.AddJob1("StartHostPingDetectionTask", time.Duration(opts.HostOfflineDetectionInterval)*time.Second, models.HostManager.PingDetectionTask)

	cron.AddJob1WithStartRun("AutoSyncCloudaccountTask", time.Duration(opts.CloudAutoSyncIntervalSeconds)*time.Second, models.CloudaccountManager.AutoSyncCloudaccountTask, true)

	cron.AddJob2("AutoDiskSnapshot", opts.AutoSnapshotDay, opts.AutoSnapshotHour, 0, 0, models.DiskManager.AutoDiskSnapshot, false)
	cron.AddJob2("SyncSkus", opts.SyncSkusDay, opts.SyncSkusHour, 0, 0, models.SyncSkus, true)

	cron.Start()
	defer cron.Stop()

	app_common.ServeForever(app, commonOpts)
}
