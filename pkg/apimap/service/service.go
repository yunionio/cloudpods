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

	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"

	"yunion.io/x/onecloud/pkg/apimap/options"
	"yunion.io/x/onecloud/pkg/apis"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
)

func StartService() {
	options.Init()
	opt := options.GetOptions()
	if err := opt.ValidateThenInit(); err != nil {
		log.Fatalf("validate options: %v", err)
	}

	commonOpts := &opt.CommonOptions
	app_common.InitAuth(commonOpts, func() {
		log.Infof("auth finished ok")
	})
	common_options.StartOptionManager(opt, opt.ConfigSyncPeriodSeconds, apis.SERVICE_TYPE_APIMAP, "", options.OnOptionsChange)

	app := app_common.InitApp(&opt.BaseOptions, false)
	app_common.ExportOptionsHandler(app, opt)

	svc, err := NewModelSetsService(opt)
	if err != nil {
		log.Fatalf("new model sets service: %v", err)
	}
	svc.InitHandlers(app)

	go func() {
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)

		wg := &sync.WaitGroup{}
		ctx = context.WithValue(ctx, "wg", wg)
		ctx = context.WithValue(ctx, appctx.APP_CONTEXT_KEY_APPNAME, "apimap")

		wg.Add(1)
		go svc.Start(ctx, app)

		go func() {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT)
			signal.Notify(sigChan, syscall.SIGTERM)
			sig := <-sigChan
			log.Infof("signal received: %s", sig)
			cancelFunc()
		}()
		wg.Wait()
	}()

	app_common.ServeForeverWithCleanup(app, &opt.BaseOptions, nil)
}
