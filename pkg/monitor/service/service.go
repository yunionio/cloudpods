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

	"golang.org/x/sync/errgroup"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	_ "yunion.io/x/sqlchemy/backends"

	"yunion.io/x/onecloud/pkg/cloudcommon"
	common_app "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	_ "yunion.io/x/onecloud/pkg/monitor/alerting"
	_ "yunion.io/x/onecloud/pkg/monitor/alerting/conditions"
	_ "yunion.io/x/onecloud/pkg/monitor/alerting/notifiers"
	_ "yunion.io/x/onecloud/pkg/monitor/alertresourcedrivers"
	"yunion.io/x/onecloud/pkg/monitor/controller/balancer"
	"yunion.io/x/onecloud/pkg/monitor/models"
	_ "yunion.io/x/onecloud/pkg/monitor/notifydrivers"
	"yunion.io/x/onecloud/pkg/monitor/options"
	"yunion.io/x/onecloud/pkg/monitor/registry"
	"yunion.io/x/onecloud/pkg/monitor/subscriptionmodel"
	_ "yunion.io/x/onecloud/pkg/monitor/tsdb/driver/influxdb"
	_ "yunion.io/x/onecloud/pkg/monitor/tsdb/driver/victoriametrics"
	"yunion.io/x/onecloud/pkg/monitor/worker"
)

func StartService() {
	opts := &options.Options
	common_options.ParseOptions(opts, os.Args, "monitor.conf", "monitor")

	commonOpts := &opts.CommonOptions
	common_app.InitAuth(commonOpts, func() {
		log.Infof("Auth complete")
	})

	dbOpts := &opts.DBOptions
	baseOpts := &opts.BaseOptions

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

	go startServices()

	if !opts.IsSlaveNode {
		err := taskman.TaskManager.InitializeData()
		if err != nil {
			log.Fatalf("TaskManager.InitializeData fail %s", err)
		}

		cron := cronman.InitCronJobManager(true, opts.CronJobWorkerCount)
		cron.AddJobAtIntervalsWithStartRun("InitAlertResourceAdminRoleUsers", time.Duration(opts.InitAlertResourceAdminRoleUsersIntervalSeconds)*time.Second, models.GetAlertResourceManager().GetAdminRoleUsers, true)
		cron.AddJobEveryFewDays("DeleteRecordsOfThirtyDaysAgoRecords", 1, 0, 0, 0,
			models.AlertRecordManager.DeleteRecordsOfThirtyDaysAgo, false)
		//cron.AddJobAtIntervalsWithStartRun("MonitorResourceSync", time.Duration(opts.MonitorResourceSyncIntervalSeconds)*time.Minute*60, models.MonitorResourceManager.SyncResources, true)
		cron.AddJobEveryFewHour("AutoPurgeSplitable", 4, 30, 0, db.AutoPurgeSplitable, false)

		cron.AddJobAtIntervals("TaskCleanupJob", time.Duration(options.Options.TaskArchiveIntervalHours)*time.Hour, taskman.TaskManager.TaskCleanupJob)

		cron.Start()
		defer cron.Stop()

		worker, err := worker.NewWorker(opts)
		if err != nil {
			log.Fatalf("new worker failed: %v", err)
		}
		go worker.Start(app.GetContext(), app, "")
	}

	InitInfluxDBSubscriptionHandlers(app, baseOpts)

	// start migration recover routine
	go func() {
		if err := balancer.RecoverInProcessAlerts(app.GetContext(), auth.GetAdminSession(app.GetContext(), options.Options.Region)); err != nil {
			log.Errorf("RecoverInProcessAlerts error: %v", err)
		}
	}()
}

func startServices() {
	services := registry.GetServices()
	// Initialize services
	for _, svc := range services {
		if registry.IsDisabled(svc.Instance) {
			continue
		}

		log.Infof("Initializing " + svc.Name)
		if err := svc.Instance.Init(); err != nil {
			log.Fatalf("Service %s init failed: %v", svc.Name, err)
		}
	}

	subscriptionmodel.SubscriptionManager.AddSubscription()
	models.CommonAlertManager.SetSubscriptionManager(subscriptionmodel.SubscriptionManager)

	childRoutines, ctx := errgroup.WithContext(context.Background())
	// Start background services
	for _, svc := range services {
		service, ok := svc.Instance.(registry.BackgroundService)
		if !ok {
			continue
		}

		if registry.IsDisabled(svc.Instance) {
			continue
		}

		// Variable is needed for accessing loop variable in callback
		descriptor := svc
		childRoutines.Go(func() error {
			if err := service.Run(ctx); err != nil {
				log.Errorf("Stopped %s: %v", descriptor.Name, err)
				return err
			}
			return nil
		})
	}
	defer func() {
		log.Debugf("Waiting on services...")
		if waitErr := childRoutines.Wait(); waitErr != nil {
			log.Errorf("A service failed: %v", waitErr)
		}
	}()
}
