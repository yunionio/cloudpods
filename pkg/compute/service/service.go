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

	_ "github.com/go-sql-driver/mysql"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/elect"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	_ "yunion.io/x/onecloud/pkg/compute/guestdrivers"
	_ "yunion.io/x/onecloud/pkg/compute/hostdrivers"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	_ "yunion.io/x/onecloud/pkg/compute/policy"
	_ "yunion.io/x/onecloud/pkg/compute/regiondrivers"
	_ "yunion.io/x/onecloud/pkg/compute/storagedrivers"
	_ "yunion.io/x/onecloud/pkg/compute/tasks"
	"yunion.io/x/onecloud/pkg/controller/autoscaling"
	_ "yunion.io/x/onecloud/pkg/multicloud/loader"
)

func StartService() {
	opts := &options.Options
	commonOpts := &options.Options.CommonOptions
	baseOpts := &options.Options.BaseOptions
	dbOpts := &options.Options.DBOptions
	common_options.ParseOptions(opts, os.Args, "region.conf", api.SERVICE_TYPE)

	if opts.PortV2 > 0 {
		log.Infof("Port V2 %d is specified, use v2 port", opts.PortV2)
		commonOpts.Port = opts.PortV2
	}

	app_common.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})

	app := app_common.InitApp(baseOpts, true)
	InitHandlers(app)

	db.EnsureAppInitSyncDB(app, dbOpts, models.InitDB)
	defer cloudcommon.CloseDB()

	common_options.StartOptionManager(opts, opts.ConfigSyncPeriodSeconds, api.SERVICE_TYPE, api.SERVICE_VERSION, options.OnOptionsChange)

	options.InitNameSyncResources()

	err := setInfluxdbRetentionPolicy()
	if err != nil {
		log.Errorf("setInfluxdbRetentionPolicy fail: %s", err)
	}

	models.InitSyncWorkers(options.Options.CloudSyncWorkerCount)

	var (
		electObj        *elect.Elect
		ctx, cancelFunc = context.WithCancel(context.Background())
	)
	defer cancelFunc()

	if opts.LockmanMethod == common_options.LockMethodEtcd {
		etcdCfg, err := elect.NewEtcdConfigFromDBOptions(dbOpts)
		if err != nil {
			log.Fatalf("etcd config for elect: %v", err)
		}
		electObj, err = elect.NewElect(etcdCfg, "@master-role")
		if err != nil {
			log.Fatalf("new elect instance: %v", err)
		}
		go electObj.Start(ctx)
	}

	if opts.EnableHostHealthCheck {
		if err := initDefaultEtcdClient(dbOpts); err != nil {
			log.Fatalf("init etcd client failed %s", err)
		}
		models.InitHostHealthChecker(etcd.Default(), opts.HostHealthTimeout).
			StartHostsHealthCheck(context.Background())
	}

	if !opts.IsSlaveNode {
		cron := cronman.InitCronJobManager(true, options.Options.CronJobWorkerCount)
		cron.AddJobAtIntervals("CleanPendingDeleteServers", time.Duration(opts.PendingDeleteCheckSeconds)*time.Second, models.GuestManager.CleanPendingDeleteServers)
		cron.AddJobAtIntervals("CleanPendingDeleteDisks", time.Duration(opts.PendingDeleteCheckSeconds)*time.Second, models.DiskManager.CleanPendingDeleteDisks)
		cron.AddJobAtIntervals("CleanPendingDeleteLoadbalancers", time.Duration(opts.LoadbalancerPendingDeleteCheckInterval)*time.Second, models.LoadbalancerAgentManager.CleanPendingDeleteLoadbalancers)
		if opts.PrepaidExpireCheck {
			cron.AddJobAtIntervals("CleanExpiredPrepaidServers", time.Duration(opts.PrepaidExpireCheckSeconds)*time.Second, models.GuestManager.DeleteExpiredPrepaidServers)
		}
		if opts.PrepaidAutoRenew {
			cron.AddJobAtIntervals("AutoRenewPrepaidServers", time.Duration(opts.PrepaidAutoRenewHours)*time.Hour, models.GuestManager.AutoRenewPrepaidServer)
		}
		cron.AddJobAtIntervals("CleanExpiredPostpaidServers", time.Duration(opts.PrepaidExpireCheckSeconds)*time.Second, models.GuestManager.DeleteExpiredPostpaidServers)
		cron.AddJobAtIntervals("StartHostPingDetectionTask", time.Duration(opts.HostOfflineDetectionInterval)*time.Second, models.HostManager.PingDetectionTask)

		cron.AddJobAtIntervalsWithStartRun("CalculateQuotaUsages", time.Duration(opts.CalculateQuotaUsageIntervalSeconds)*time.Second, models.QuotaManager.CalculateQuotaUsages, true)
		cron.AddJobAtIntervalsWithStartRun("CalculateRegionQuotaUsages", time.Duration(opts.CalculateQuotaUsageIntervalSeconds)*time.Second, models.RegionQuotaManager.CalculateQuotaUsages, true)
		cron.AddJobAtIntervalsWithStartRun("CalculateZoneQuotaUsages", time.Duration(opts.CalculateQuotaUsageIntervalSeconds)*time.Second, models.ZoneQuotaManager.CalculateQuotaUsages, true)
		cron.AddJobAtIntervalsWithStartRun("CalculateProjectQuotaUsages", time.Duration(opts.CalculateQuotaUsageIntervalSeconds)*time.Second, models.ProjectQuotaManager.CalculateQuotaUsages, true)
		cron.AddJobAtIntervalsWithStartRun("CalculateDomainQuotaUsages", time.Duration(opts.CalculateQuotaUsageIntervalSeconds)*time.Second, models.DomainQuotaManager.CalculateQuotaUsages, true)
		cron.AddJobAtIntervalsWithStartRun("CalculateInfrasQuotaUsages", time.Duration(opts.CalculateQuotaUsageIntervalSeconds)*time.Second, models.InfrasQuotaManager.CalculateQuotaUsages, true)

		cron.AddJobAtIntervalsWithStartRun("AutoSyncCloudaccountTask", time.Duration(opts.CloudAutoSyncIntervalSeconds)*time.Second, models.CloudaccountManager.AutoSyncCloudaccountTask, true)

		cron.AddJobEveryFewHour("AutoDiskSnapshot", 1, 5, 0, models.DiskManager.AutoDiskSnapshot, false)
		cron.AddJobEveryFewHour("SnapshotsCleanup", 1, 35, 0, models.SnapshotManager.CleanupSnapshots, false)
		cron.AddJobEveryFewHour("AutoSyncExtDiskSnapshot", 1, 10, 0, models.DiskManager.AutoSyncExtDiskSnapshot, false)
		cron.AddJobEveryFewDays("SyncSkus", opts.SyncSkusDay, opts.SyncSkusHour, 0, 0, models.SyncServerSkus, true)
		cron.AddJobEveryFewDays("SyncDBInstanceSkus", opts.SyncSkusDay, opts.SyncSkusHour, 0, 0, models.SyncDBInstanceSkus, true)
		cron.AddJobEveryFewDays("SyncElasticCacheSkus", opts.SyncSkusDay, opts.SyncSkusHour, 0, 0, models.SyncElasticCacheSkus, true)
		cron.AddJobEveryFewDays("StorageSnapshotsRecycle", 1, 2, 0, 0, models.StorageManager.StorageSnapshotsRecycle, false)
		go cron.Start2(ctx, electObj)

		// init auto scaling controller
		autoscaling.ASController.Init(options.Options.SASControllerOptions, cron)
	}

	app_common.ServeForever(app, baseOpts)
}

func initDefaultEtcdClient(opts *common_options.DBOptions) error {
	if etcd.Default() != nil {
		return nil
	}
	tlsConfig, err := opts.GetEtcdTLSConfig()
	if err != nil {
		return err
	}
	err = etcd.InitDefaultEtcdClient(&etcd.SEtcdOptions{
		EtcdEndpoint:  opts.EtcdEndpoints,
		EtcdUsername:  opts.EtcdUsername,
		EtcdPassword:  opts.EtcdPassword,
		EtcdEnabldSsl: opts.EtcdUseTLS,
		TLSConfig:     tlsConfig,
	}, nil)
	if err != nil {
		return errors.Wrap(err, "init default etcd client")
	}
	return nil
}
