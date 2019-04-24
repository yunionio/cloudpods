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

	"yunion.io/x/onecloud/pkg/baremetal"
	"yunion.io/x/onecloud/pkg/baremetal/handler"
	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/baremetal/tasks"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/service"
)

type BaremetalService struct {
	service.SServiceBase
}

func New() *BaremetalService {
	return &BaremetalService{}
}

func (s *BaremetalService) StartService() {
	common_options.ParseOptions(&o.Options, os.Args, "baremetal.conf", "baremetal")
	app_common.InitAuth(&o.Options.CommonOptions, s.startAgent)

	app := app_common.InitApp(&o.Options.CommonOptions, false)
	handler.InitHandlers(app)

	app_common.ServeForeverWithCleanup(app, &o.Options.CommonOptions, func() {
		tasks.OnStop()
		baremetal.Stop()
	})
}

func (s *BaremetalService) startAgent() {
	err := baremetal.Start()
	if err != nil {
		log.Fatalf("Start agent error: %v", err)
	}
}
