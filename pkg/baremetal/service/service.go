package service

import (
	"context"
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
	isExiting bool
}

func New() *BaremetalService {
	return &BaremetalService{}
}

func (s *BaremetalService) StartService() {
	cloudcommon.ParseOptions(&o.Options, os.Args, "baremetal.conf", "baremetal")
	cloudcommon.InitAuth(&o.Options.CommonOptions, s.startAgent)

	app := cloudcommon.InitApp(&o.Options.CommonOptions, false)
	handler.InitHandlers(app)

	s.RegisterSIGUSR1()
	s.RegisterQuitSignals(func() {
		log.Infof("Baremetal agent quit !!!")
		if s.isExiting {
			return
		} else {
			s.isExiting = true
		}

		if app.IsInServe() {
			if err := app.ShotDown(context.Background()); err != nil {
				log.Errorf("App shutdown err: %v", err)
			}
		}
		tasks.OnStop()
		os.Exit(0)
	})

	cloudcommon.ServeForever(app, &o.Options.CommonOptions)
}

func (s *BaremetalService) startAgent() {
	err := baremetal.Start()
	if err != nil {
		log.Fatalf("Start agent error: %v", err)
	}
}
