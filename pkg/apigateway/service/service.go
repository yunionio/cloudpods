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
	"net"
	"os"
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"

	"yunion.io/x/onecloud/pkg/apigateway/app"
	"yunion.io/x/onecloud/pkg/apigateway/clientman"
	"yunion.io/x/onecloud/pkg/apigateway/options"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func StartService() {
	opts := &options.Options
	baseOpts := &opts.BaseOptions
	commonOpts := &opts.CommonOptions
	common_options.ParseOptions(opts, os.Args, "apigateway.conf", "apigateway")
	app_common.InitAuth(commonOpts, func() {
		log.Infof("Auth complete.")
	})

	if opts.DisableModuleApiVersion {
		mcclient.DisableApiVersionByModule()
	}

	if err := clientman.InitClient(opts.SqlitePath); err != nil {
		log.Fatalf("Init client token manager: %v", err)
	}

	serviceApp := app.NewApp(app_common.InitApp(baseOpts, false))
	serviceApp.InitHandlers().Bind()

	mods, jmods := modulebase.GetRegisterdModules()
	log.Infof("Modules: %s", jsonutils.Marshal(mods).PrettyString())
	log.Infof("Modules: %s", jsonutils.Marshal(jmods).PrettyString())

	listenAddr := net.JoinHostPort(options.Options.Address, strconv.Itoa(options.Options.Port))
	if opts.EnableSsl {
		serviceApp.ListenAndServeTLS(listenAddr, opts.SslCertfile, opts.SslKeyfile)
	} else {
		serviceApp.ListenAndServe(listenAddr)
	}
}
