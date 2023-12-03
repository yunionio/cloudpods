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

	_ "yunion.io/x/cloudmux/pkg/multicloud/loader"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudmon/misc"
	"yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/cloudmon/resources"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

func StartService() {
	opts := &options.Options
	baseOpts := &options.Options.BaseOptions
	common_options.ParseOptions(opts, os.Args, "cloudmon.conf", "cloudmon")

	commonOpts := &opts.CommonOptions
	app_common.InitAuth(commonOpts, func() {
		log.Infof("Auth complete")
	})
	common_options.StartOptionManager(opts, opts.ConfigSyncPeriodSeconds, apis.SERVICE_TYPE_CLOUDMON, "", options.OnOptionsChange)

	res := resources.NewResources()
	if !opts.IsSlaveNode {
		cron := cronman.InitCronJobManager(true, options.Options.CronJobWorkerCount)
		cron.AddJobAtIntervalsWithStartRun("InitResources", time.Duration(opts.ResourcesSyncInterval)*time.Minute, res.Init, true)
		cron.AddJobAtIntervals("IncrementResources", time.Duration(opts.ResourcesSyncInterval)*time.Minute, res.IncrementSync)
		cron.AddJobAtIntervals("DecrementResources", time.Duration(opts.ResourcesSyncInterval)*time.Minute, res.DecrementSync)
		cron.AddJobAtIntervals("UpdateResources", time.Duration(opts.ResourcesSyncInterval)*time.Minute, res.UpdateSync)
		cron.AddJobAtIntervalsWithStarTime("CollectResources", time.Duration(opts.CollectMetricInterval)*time.Minute, res.CollectMetrics)

		cron.AddJobAtIntervalsWithStartRun("PingProb", time.Duration(opts.PingProbIntervalHours)*time.Hour, misc.PingProbe, true)

		cron.AddJobEveryFewDays("UsageMetricCollect", 1, 23, 10, 10, misc.UsegReport, false)
		cron.AddJobEveryFewDays("AlertHistoryMetricCollect", 1, 23, 59, 59, misc.AlertHistoryReport, false)

		cron.AddJobAtIntervals("CollectServiceMetrics", time.Duration(opts.CollectServiceMetricIntervalMinute)*time.Minute, misc.CollectServiceMetrics)
		go cron.Start()
	}

	ctx := context.Background()
	if opts.HistoryMetricPullDays > 0 {
		go func() {
			for !res.IsInit() {
				log.Infof("wait resources init...")
				time.Sleep(time.Second * 10)
			}
			now := time.Now()
			start := now.AddDate(0, 0, -1*opts.HistoryMetricPullDays)
			s := auth.GetAdminSession(ctx, opts.BaseOptions.Region)
			for start.Before(now) {
				log.Infof("start collect history metric from %s", start.Format(time.RFC3339))
				res.CollectMetrics(ctx, s.GetToken(), start, false)
				start = start.Add(time.Duration(opts.CollectMetricInterval) * time.Minute)
			}
			log.Infof("collect history metric end")
		}()
	}

	app := app_common.InitApp(baseOpts, true).
		OnException(func(method, path string, body jsonutils.JSONObject, err error) {
			session := auth.GetAdminSession(ctx, commonOpts.Region)
			notifyclient.EventNotifyServiceAbnormal(ctx, session.GetToken(), consts.GetServiceType(), method, path, body, err)
		})
	app_common.ServeForever(app, baseOpts)
}
