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
	"os/signal"
	"sync"
	"syscall"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"

	"yunion.io/x/onecloud/pkg/apis"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	identity_modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/vpcagent/options"
	_ "yunion.io/x/onecloud/pkg/vpcagent/ovn"
	"yunion.io/x/onecloud/pkg/vpcagent/worker"
)

func updateDnsOptions(ctx context.Context, opts *options.Options) {
	regionConfig, err := identity_modules.ServicesV3.GetConfig(auth.GetAdminSession(ctx, opts.Region), apis.SERVICE_TYPE_REGION)
	if err != nil {
		log.Fatalf("fail to retrieve compute config")
	}
	if opts.DNSDomain == "" {
		opts.DNSDomain, _ = regionConfig.GetString("default", "dns_domain")
	}
	if opts.DNSServer == "" {
		opts.DNSServer, _ = regionConfig.GetString("default", "dns_server")
	}
}

func StartService() {
	opts := &options.Options{}
	commonOpts := &opts.CommonOptions
	{
		common_options.ParseOptions(opts, os.Args, "vpcagent.conf", "vpcagent")
		app_common.InitAuth(commonOpts, func() {
			log.Infof("auth finished ok")
		})
		common_options.StartOptionManager(opts, opts.ConfigSyncPeriodSeconds, apis.SERVICE_TYPE_VPCAGENT, "", options.OnOptionsChange)

	}
	if err := opts.ValidateThenInit(); err != nil {
		log.Fatalf("opts validate: %s", err)
	}

	app := app_common.InitApp(&opts.BaseOptions, true).
		OnException(func(method, path string, body jsonutils.JSONObject, err error) {
			ctx := context.Background()
			session := auth.GetAdminSession(ctx, commonOpts.Region)
			notifyclient.EventNotifyServiceAbnormal(ctx, session.GetToken(), consts.GetServiceType(), method, path, body, err)
		})

	w := worker.NewWorker(opts)
	if w == nil {
		log.Fatalf("new worker failed")
	}

	go func() {
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)

		wg := &sync.WaitGroup{}
		ctx = context.WithValue(ctx, "wg", wg)
		ctx = context.WithValue(ctx, appctx.APP_CONTEXT_KEY_APPNAME, "vpcagent")

		if opts.DNSDomain == "" || opts.DNSServer == "" {
			updateDnsOptions(ctx, opts)
		}

		wg.Add(1)
		go w.Start(ctx, app, "vpcagent")

		go func() {
			sigChan := make(chan os.Signal)
			signal.Notify(sigChan, syscall.SIGINT)
			signal.Notify(sigChan, syscall.SIGTERM)
			sig := <-sigChan
			log.Infof("signal received: %s", sig)
			cancelFunc()
		}()
		wg.Wait()
	}()

	app_common.ServeForeverWithCleanup(app, &opts.BaseOptions, nil)
}
