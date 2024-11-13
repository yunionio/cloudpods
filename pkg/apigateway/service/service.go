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
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apigateway/app"
	"yunion.io/x/onecloud/pkg/apigateway/clientman"
	"yunion.io/x/onecloud/pkg/apigateway/options"
	"yunion.io/x/onecloud/pkg/apigateway/report"
	api "yunion.io/x/onecloud/pkg/apis/apigateway"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
)

func StartService() {
	options.Options = &options.GatewayOptions{}
	opts := options.Options
	baseOpts := &opts.BaseOptions
	commonOpts := &opts.CommonOptions
	common_options.ParseOptions(opts, os.Args, "apigateway.conf", api.SERVICE_TYPE)
	InitDefaultPolicy()
	app_common.InitAuth(commonOpts, func() {
		log.Infof("Auth complete.")
	})

	common_options.StartOptionManager(opts, opts.ConfigSyncPeriodSeconds, api.SERVICE_TYPE, api.SERVICE_VERSION, options.OnOptionsChange)

	if err := clientman.InitClient(); err != nil {
		log.Fatalf("Init client token manager: %v", err)
	}

	serviceApp := app.NewApp(app_common.InitApp(baseOpts, false))
	serviceApp.InitHandlers().Bind()

	// mods, jmods := modulebase.GetRegisterdModules()
	// log.Infof("Modules: %s", jsonutils.Marshal(mods).PrettyString())
	// log.Infof("Modules: %s", jsonutils.Marshal(jmods).PrettyString())

	if !options.Options.DisableReporting {
		cron := cronman.InitCronJobManager(true, 1)
		rand.Seed(time.Now().Unix())
		cron.AddJobEveryFewDays("AutoReport", 1, rand.Intn(23), rand.Intn(59), 0, report.Report, true)
		go cron.Start()
	}

	listenAddr := net.JoinHostPort(options.Options.Address, strconv.Itoa(options.Options.Port))
	if opts.EnableSsl {
		serviceApp.ListenAndServeTLS(listenAddr, opts.SslCertfile, opts.SslKeyfile)
	} else {
		serviceApp.ListenAndServe(listenAddr)
	}
}
