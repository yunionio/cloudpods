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

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd/models"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloutpost/options"
)

const (
	SERVICE_TYPE = "cloutpost"
)

func StartService() {
	opts := &options.Options
	baseOpts := &opts.BaseOptions
	commonOpts := &opts.CommonOptions
	common_options.ParseOptions(opts, os.Args, "cloutpost.conf", SERVICE_TYPE)

	app_common.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})

	err := etcd.InitDefaultEtcdClient(&opts.SEtcdOptions, nil)
	if err != nil {
		log.Fatalf("init etcd fail: %s", err)
	}
	defer etcd.CloseDefaultEtcdClient()

	app := app_common.InitApp(baseOpts, false)
	cloudcommon.AppDBInit(app)
	initHandlers(app)

	err = models.ServiceRegistryManager.Register(
		app.GetContext(),
		options.Options.Address,
		options.Options.Port,
		options.Options.Provider,
		options.Options.Environment,
		options.Options.Cloudregion,
		options.Options.Zone,
		SERVICE_TYPE,
	)

	if err != nil {
		log.Fatalf("fail to register service %s", err)
	}

	app_common.ServeForever(app, baseOpts)
}
