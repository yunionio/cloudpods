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
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/baremetal"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/baremetal"
	"yunion.io/x/onecloud/pkg/baremetal/handler"
	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/baremetal/tasks"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
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

	if !utils.IsInStringArray(o.Options.BootLoader, []string{o.BOOT_LOADER_GRUB, o.BOOT_LOADER_SYSLINUX}) {
		log.Fatalf("invalid boot_loader option: %q", o.Options.BootLoader)
	}

	app_common.InitAuth(&o.Options.CommonOptions, func() {
		log.Infof("auth complete")
	})

	fsdriver.Init("")
	app := app_common.InitApp(&o.Options.BaseOptions, false)

	common_options.StartOptionManager(&o.Options, o.Options.ConfigSyncPeriodSeconds, api.SERVICE_TYPE, api.SERVICE_VERSION, o.OnOptionsChange)

	handler.InitHandlers(app)

	s.startAgent(app)

	cron := cronman.InitCronJobManager(false, o.Options.CronJobWorkerCount)
	cron.AddJobAtIntervals("BaremetalCronJobs", 10*time.Second, baremetal.DoCronJobs)
	cron.Start()
	defer cron.Stop()

	app_common.ServeForeverWithCleanup(app, &o.Options.BaseOptions, func() {
		tasks.OnStop()
		baremetal.Stop()
	})
}

func (s *BaremetalService) startAgent(app *appsrv.Application) {
	// init lockman
	lm := lockman.NewInMemoryLockManager()
	lockman.Init(lm)

	err := baremetal.Start(app)
	if err != nil {
		log.Fatalf("Start agent error: %v", err)
	}
}
