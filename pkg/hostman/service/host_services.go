package service

import (
	"os"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/service"
	"yunion.io/x/onecloud/pkg/hostman/guestman"
	"yunion.io/x/onecloud/pkg/hostman/options"
)

type SHostService struct {
	service.SServiceBase
}

func (host *SHostService) StartService() {
	host.TrapSignals(host.quitSignalHandler)

	cloudcommon.ParseOptions(&options.HostOptions, &options.HostOptions.Options, os.Args, "host.conf")
	// Hostinfo.Init()
	// Firewall.Init()

	var c = make(chan struct{})
	cloudcommon.InitAuth(&options.HostOptions.Options, func() {
		log.Infof("Auth complete!!")
		// TODO
		// hostinfo.instance().start_register
		// 应该在register 成功后注册handler
		// 上报guest信息，即guestman
		// hostinfo.startregisters
		close(c)
	})
	guestman.Init()
	app := cloudcommon.InitApp(&o.Options.Options)
	host.InitHandlers(app)
	<-c // wait host info registered
	cloudcommon.ServeForever(app, &options.HostOptions)
}

func (host *SHostService) quitSignalHandler() {
	// TODO
	/*
		cloud/yunion/server/clouds/common/handler/__init__.py -> stop()
		1. delay process
		2. work manager
	*/
}

func (host *SHostService) initHandlers(app *appsrv.Application) {
	guestman.AddGuestTaskHandler("", app)
}
