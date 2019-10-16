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
	"path/filepath"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/baremetal"
	"yunion.io/x/onecloud/pkg/baremetal/handler"
	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/baremetal/tasks"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/service"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"

	_ "yunion.io/x/onecloud/pkg/util/redfish/loader"
)

type BaremetalService struct {
	service.SServiceBase
}

func New() *BaremetalService {
	return &BaremetalService{}
}

func (s *BaremetalService) StartService() {
	common_options.ParseOptions(&o.Options, os.Args, "baremetal.conf", "baremetal")

	if len(o.Options.CachePath) == 0 {
		o.Options.CachePath = filepath.Join(filepath.Dir(o.Options.BaremetalsPath), "bm_image_cache")
		log.Infof("No cachepath, use default %s", o.Options.CachePath)
	}
	if len(o.Options.BootIsoPath) == 0 {
		o.Options.BootIsoPath = filepath.Join(filepath.Dir(o.Options.BaremetalsPath), "bm_boot_iso")
		log.Infof("No BootIsoPath, use default %s", o.Options.BootIsoPath)
		err := os.MkdirAll(o.Options.BootIsoPath, os.FileMode(0760))
		if err != nil {
			log.Fatalf("fail to create BootIsoPath %s", o.Options.BootIsoPath)
		}
	}

	app_common.InitAuth(&o.Options.CommonOptions, func() {
		log.Infof("auth complete")
	})

	fsdriver.Init(nil)
	app := app_common.InitApp(&o.Options.BaseOptions, false)
	handler.InitHandlers(app)

	s.startAgent(app)

	app_common.ServeForeverWithCleanup(app, &o.Options.BaseOptions, func() {
		tasks.OnStop()
		baremetal.Stop()
	})
}

func (s *BaremetalService) startAgent(app *appsrv.Application) {
	err := baremetal.Start(app)
	if err != nil {
		log.Fatalf("Start agent error: %v", err)
	}
}
