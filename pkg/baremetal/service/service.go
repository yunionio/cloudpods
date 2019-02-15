package service

import (
	"os"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/baremetal"
	"yunion.io/x/onecloud/pkg/baremetal/handler"
	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/baremetal/tasks"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/service"
)

type BaremetalService struct {
	service.SServiceBase
}

func New() *BaremetalService {
	return &BaremetalService{}
}

func (s *BaremetalService) StartService() {
	cloudcommon.ParseOptions(&o.Options, os.Args, "baremetal.conf", "baremetal")
	cloudcommon.InitAuth(&o.Options.CommonOptions, s.startAgent)

	app := cloudcommon.InitApp(&o.Options.CommonOptions, false)
	handler.InitHandlers(app)

	cloudcommon.ServeForeverWithCleanup(app, &o.Options.CommonOptions, func() {
		tasks.OnStop()
	})
}

func (s *BaremetalService) startAgent() {
	err := baremetal.Start()
	if err != nil {
		log.Fatalf("Start agent error: %v", err)
	}
}
