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
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"yunion.io/x/cloudmux/pkg/multicloud/esxi"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	_ "yunion.io/x/sqlchemy/backends"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	common_app "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/elect"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	_ "yunion.io/x/onecloud/pkg/compute/container_drivers/device"
	_ "yunion.io/x/onecloud/pkg/compute/container_drivers/volume_mount"
	_ "yunion.io/x/onecloud/pkg/compute/guestdrivers"
	_ "yunion.io/x/onecloud/pkg/compute/hostdrivers"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	_ "yunion.io/x/onecloud/pkg/compute/policy"
	_ "yunion.io/x/onecloud/pkg/compute/regiondrivers"
	_ "yunion.io/x/onecloud/pkg/compute/storagedrivers"
	"yunion.io/x/onecloud/pkg/compute/tasks"
	"yunion.io/x/onecloud/pkg/controller/autoscaling"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

func StartService() {
	StartServiceWithJobs(nil)
}

func StartServiceWithJobs(jobs func(cron *cronman.SCronJobManager)) {
	opts := &options.Options
	commonOpts := &options.Options.CommonOptions
	baseOpts := &options.Options.BaseOptions
	dbOpts := &options.Options.DBOptions
	common_options.ParseOptions(opts, os.Args, "region.conf", api.SERVICE_TYPE)

	if opts.PortV2 > 0 {
		log.Infof("Port V2 %d is specified, use v2 port", opts.PortV2)
		commonOpts.Port = opts.PortV2
	}

	common_app.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})
	common_options.StartOptionManager(opts, opts.ConfigSyncPeriodSeconds, api.SERVICE_TYPE, api.SERVICE_VERSION, options.OnOptionsChange)

	serviceUrl, err := auth.GetServiceURL(api.SERVER_TYPE_V2, opts.Region, "", identity.EndpointInterfaceInternal)
	if err != nil {
		log.Fatalf("unable to get service url: %v", err)
	}
	log.Infof("serviceUrl: %s", serviceUrl)
	taskman.SetServiceUrl(serviceUrl)
	err = taskman.UpdateWorkerCount(opts.TaskWorkerCount)
	if err != nil {
		log.Fatalf("failed update task manager worker count %s", err)
	}

	err = esxi.InitEsxiConfig(opts.EsxiOptions)
	if err != nil {
		log.Fatalf("unable to init esxi configs: %v", err)
	}

	// always try to init etcd options
	if err := initEtcdLockOpts(opts); err != nil {
		log.Errorf("try to init etcd options error: %v", err)
	}

	app := common_app.InitApp(baseOpts, true).
		OnException(func(method, path string, body jsonutils.JSONObject, err error) {
			ctx := context.Background()
			session := auth.GetAdminSession(ctx, commonOpts.Region)
			notifyclient.EventNotifyServiceAbnormal(ctx, session.GetToken(), consts.GetServiceType(), method, path, body, err)
		})

	cloudcommon.InitDB(dbOpts)

	InitHandlers(app)

	db.EnsureAppSyncDB(app, dbOpts, models.InitDB)
	defer cloudcommon.CloseDB()

	setInfluxdbRetentionPolicy()

	models.InitSyncWorkers(options.Options.CloudSyncWorkerCount)
	tasks.InitCloudproviderSyncWorkers(options.Options.CloudProviderSyncWorkerCount)

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
		if err := models.InitHostHealthChecker(etcd.Default(), opts.HostHealthTimeout).
			StartHostsHealthCheck(context.Background()); err != nil {
			log.Fatalf("failed start host health checker %s", err)
		}
	}

	cronFunc := func() {
		db.StartTenantCacheSync(app.GetContext(), opts.TenantCacheExpireSeconds)

		cron := cronman.InitCronJobManager(true, options.Options.CronJobWorkerCount)
		cron.AddJobAtIntervals("CleanPendingDeleteServers", time.Duration(opts.PendingDeleteCheckSeconds)*time.Second, models.GuestManager.CleanPendingDeleteServers)
		cron.AddJobAtIntervals("CleanPendingDeleteDisks", time.Duration(opts.PendingDeleteCheckSeconds)*time.Second, models.DiskManager.CleanPendingDeleteDisks)
		if opts.PrepaidExpireCheck {
			cron.AddJobAtIntervals("CleanExpiredPrepaidServers", time.Duration(opts.PrepaidExpireCheckSeconds)*time.Second, models.GuestManager.DeleteExpiredPrepaidServers)
		}
		if opts.PrepaidAutoRenew {
			cron.AddJobAtIntervals("AutoRenewPrepaidServers", time.Duration(opts.PrepaidAutoRenewHours)*time.Hour, models.GuestManager.AutoRenewPrepaidServer)
		}
		cron.AddJobAtIntervals("CleanExpiredPostpaidElasticCaches", time.Duration(opts.PrepaidExpireCheckSeconds)*time.Second, models.ElasticcacheManager.DeleteExpiredPostpaids)
		cron.AddJobAtIntervals("CleanExpiredPostpaidDBInstances", time.Duration(opts.PrepaidExpireCheckSeconds)*time.Second, models.DBInstanceManager.DeleteExpiredPostpaids)
		cron.AddJobAtIntervals("CleanExpiredPostpaidServers", time.Duration(opts.PrepaidExpireCheckSeconds)*time.Second, models.GuestManager.DeleteExpiredPostpaidServers)
		cron.AddJobAtIntervals("CleanExpiredPostpaidNatGateways", time.Duration(opts.PrepaidExpireCheckSeconds)*time.Second, models.NatGatewayManager.DeleteExpiredPostpaids)
		cron.AddJobAtIntervals("CleanExpiredPostpaidNas", time.Duration(opts.PrepaidExpireCheckSeconds)*time.Second, models.FileSystemManager.DeleteExpiredPostpaids)

		if !opts.EnableHostHealthCheck {
			cron.AddJobAtIntervals("StartHostPingDetectionTask", time.Duration(opts.HostOfflineDetectionInterval)*time.Second, models.HostManager.PingDetectionTask)
		}

		cron.AddJobAtIntervals("RefreshCloudproviderHostStatus", time.Duration(opts.ManagedHostSyncStatusIntervalSeconds)*time.Second, models.RefreshCloudproviderHostStatus)

		cron.AddJobAtIntervalsWithStartRun("CalculateQuotaUsages", time.Duration(opts.CalculateQuotaUsageIntervalSeconds)*time.Second, models.QuotaManager.CalculateQuotaUsages, true)
		cron.AddJobAtIntervalsWithStartRun("CalculateRegionQuotaUsages", time.Duration(opts.CalculateQuotaUsageIntervalSeconds)*time.Second, models.RegionQuotaManager.CalculateQuotaUsages, true)
		cron.AddJobAtIntervalsWithStartRun("CalculateZoneQuotaUsages", time.Duration(opts.CalculateQuotaUsageIntervalSeconds)*time.Second, models.ZoneQuotaManager.CalculateQuotaUsages, true)
		cron.AddJobAtIntervalsWithStartRun("CalculateProjectQuotaUsages", time.Duration(opts.CalculateQuotaUsageIntervalSeconds)*time.Second, models.ProjectQuotaManager.CalculateQuotaUsages, true)
		cron.AddJobAtIntervalsWithStartRun("CalculateDomainQuotaUsages", time.Duration(opts.CalculateQuotaUsageIntervalSeconds)*time.Second, models.DomainQuotaManager.CalculateQuotaUsages, true)
		cron.AddJobAtIntervalsWithStartRun("CalculateInfrasQuotaUsages", time.Duration(opts.CalculateQuotaUsageIntervalSeconds)*time.Second, models.InfrasQuotaManager.CalculateQuotaUsages, true)
		cron.AddJobAtIntervalsWithStartRun("AutoSyncCloudaccountStatusTask", time.Duration(opts.CloudAutoSyncIntervalSeconds)*time.Second, models.CloudaccountManager.AutoSyncCloudaccountStatusTask, true)
		cron.AddJobAtIntervalsWithStartRun("SyncCapacityUsedForEsxiStorage", time.Duration(opts.SyncStorageCapacityUsedIntervalMinutes)*time.Minute, models.StorageManager.SyncCapacityUsedForEsxiStorage, true)

		cron.AddJobEveryFewHour("AutoPurgeSplitable", 4, 30, 0, db.AutoPurgeSplitable, false)

		cron.AddJobEveryFewHour("AutoDiskSnapshot", 1, 5, 0, models.DiskManager.AutoDiskSnapshot, false)
		cron.AddJobEveryFewHour("SnapshotsCleanup", 1, 35, 0, models.SnapshotManager.CleanupSnapshots, false)

		cron.AddJobEveryFewHour("AutoCleanImageCache", 1, 5, 0, models.CachedimageManager.AutoCleanImageCaches, false)

		cron.AddJobAtIntervalsWithStartRun("SyncSkus", time.Duration(opts.ServerSkuSyncIntervalMinutes)*time.Minute, models.SyncServerSkus, true)
		cron.AddJobAtIntervalsWithStartRun("SyncManagedWafGroups", time.Duration(opts.ServerSkuSyncIntervalMinutes)*time.Minute, models.SyncWafGroups, true)

		cron.AddJobEveryFewDays("SyncDBInstanceSkus", opts.SyncSkusDay, opts.SyncSkusHour, 0, 0, models.SyncDBInstanceSkus, true)
		cron.AddJobEveryFewDays("SyncNatSkus", opts.SyncSkusDay, opts.SyncSkusHour, 0, 0, models.SyncNatSkus, true)
		cron.AddJobEveryFewDays("SyncNasSkus", opts.SyncSkusDay, opts.SyncSkusHour, 0, 0, models.SyncNasSkus, true)
		cron.AddJobEveryFewDays("SyncElasticCacheSkus", opts.SyncSkusDay, opts.SyncSkusHour, 0, 0, models.SyncElasticCacheSkus, true)
		cron.AddJobEveryFewDays("StorageSnapshotsRecycle", 1, 2, 0, 0, models.StorageManager.StorageSnapshotsRecycle, false)

		cron.AddJobEveryFewDays("SnapshotDataCleaning", 1, 0, 0, 0, models.SnapshotManager.DataCleaning, true)

		cron.AddJobAtIntervalsWithStartRun("SyncCloudImages", time.Duration(opts.CloudImagesSyncIntervalHours)*time.Hour, models.SyncPublicCloudImages, true)

		cron.AddJobEveryFewHour("InspectAllTemplate", 1, 0, 0, models.GuestTemplateManager.InspectAllTemplate, true)

		cron.AddJobEveryFewHour("CheckBillingResourceExpireAt", 1, 0, 0, models.CheckBillingResourceExpireAt, true)
		if jobs != nil {
			jobs(cron)
		}
		// init auto scaling controller
		autoscaling.ASController.Init(options.Options.SASControllerOptions, cron)

		go cron.Start2(ctx, electObj)
	}
	if !opts.IsSlaveNode {
		go cronFunc()
	}

	common_app.ServeForever(app, baseOpts)
}

func initDefaultEtcdClient(opts *common_options.DBOptions) error {
	if etcd.Default() != nil {
		return nil
	}
	tlsConfig, err := opts.GetEtcdTLSConfig()
	if err != nil {
		return err
	}
	onKeepaliveFailure := func() {
		cli := etcd.Default()
		if opts.LockmanMethod == common_options.LockMethodEtcd {
			log.Fatalf("etcd keepalive failed and exit when lockman_method is %s", common_options.LockMethodEtcd)
		}
		if err := cli.RestartSession(); err != nil {
			log.Errorf("restart default session error: %v", err)
			return
		}
	}
	err = etcd.InitDefaultEtcdClient(&etcd.SEtcdOptions{
		EtcdEndpoint:  opts.EtcdEndpoints,
		EtcdUsername:  opts.EtcdUsername,
		EtcdPassword:  opts.EtcdPassword,
		EtcdEnabldSsl: opts.EtcdUseTLS,
		TLSConfig:     tlsConfig,
	}, onKeepaliveFailure)
	if err != nil {
		return errors.Wrap(err, "init default etcd client")
	}
	return nil
}

func initEtcdLockOpts(opts *options.ComputeOptions) error {
	etcdEndpoint, err := common_app.FetchEtcdServiceInfo()
	if err != nil {
		if errors.Cause(err) == httperrors.ErrNotFound {
			return nil
		}
		return errors.Wrap(err, "fetch etcd service info")
	}
	if etcdEndpoint != nil {
		opts.EtcdEndpoints = []string{etcdEndpoint.Url}
		if len(etcdEndpoint.CertId) > 0 {
			dir, err := ioutil.TempDir("", "etcd-cluster-tls")
			if err != nil {
				return errors.Wrap(err, "create dir etcd cluster tls")
			}
			opts.EtcdCert, err = writeFile(dir, "etcd.crt", []byte(etcdEndpoint.Certificate))
			if err != nil {
				return errors.Wrap(err, "write file certificate")
			}
			opts.EtcdKey, err = writeFile(dir, "etcd.key", []byte(etcdEndpoint.PrivateKey))
			if err != nil {
				return errors.Wrap(err, "write file private key")
			}
			opts.EtcdCacert, err = writeFile(dir, "etcd-ca.crt", []byte(etcdEndpoint.CaCertificate))
			if err != nil {
				return errors.Wrap(err, "write file  cacert")
			}
			opts.EtcdUseTLS = true
		}
	}
	return nil
}

func writeFile(dir, file string, data []byte) (string, error) {
	p := filepath.Join(dir, file)
	return p, ioutil.WriteFile(p, data, 0600)
}
