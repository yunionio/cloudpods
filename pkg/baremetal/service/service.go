package service

import (
	"os"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/baremetal"
	"yunion.io/x/onecloud/pkg/baremetal/handler"
	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/service"
)

type BaremetalService struct {
	service.SServiceBase
}

func (s *BaremetalService) StartService() {
	cloudcommon.ParseOptions(&o.Options, os.Args, "baremetal.conf", "baremetal")
	cloudcommon.InitAuth(&o.Options.CommonOptions, s.startAgent)

	s.RegisterSIGUSR1()

	s.RegisterQuitSignals(func() {
		log.Infof("Baremetal agent quit !!!")
	})
	app := cloudcommon.InitApp(&o.Options.CommonOptions, false)
	handler.InitHandlers(app)
	cloudcommon.ServeForever(app, &o.Options.CommonOptions)
}

func (s *BaremetalService) startAgent() {
	err := baremetal.Start()
	if err != nil {
		log.Fatalf("Start agent error: %v", err)
	}
}
