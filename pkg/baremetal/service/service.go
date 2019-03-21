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
