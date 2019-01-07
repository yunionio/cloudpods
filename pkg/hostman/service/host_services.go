package service

import (
	"context"
	"os"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/service"
	"yunion.io/x/onecloud/pkg/hostman/guestman"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
)

type SHostService struct {
	service.SServiceBase

	guestmanager *guestman.SGuestManager
	hostinstance *hostinfo.SHostInfo
}

func (host *SHostService) StartService() {
	cloudcommon.ParseOptions(&options.HostOptions, &options.HostOptions.CommonOptions, os.Args, "host.conf")

	// isolatedman.Init()

	hostInstance := hostinfo.Instance()
	if err := hostInstance.Init(); err != nil {
		log.Fatalf(err)
	}

	if err := storageman.Init(hostInstance.HostId, hostInstance.Zone); err != nil {
		log.Fatalf(err)
	}
	// wait host registerd

	// Firewall.Init()

	var c = make(chan struct{})
	cloudcommon.InitAuth(&options.HostOptions.Options, func() {
		log.Infof("Auth complete!!")

		hostinfo.Instance().StartRegister()
		<-hostinfo.Instance().IsRegistered

		guestman.Init(options.HostOptions.ServersPath)
		guestman.GetGuestManager().Bootstrap()

		close(c)
	})

	app := cloudcommon.InitApp(&options.HostOptions.Options)
	host.TrapSignals(func() { host.quitSignalHandler(app) })
	host.InitHandlers(app)

	<-c // wait host and guest init

	cloudcommon.ServeForever(app, &options.HostOptions)
}

func (host *SHostService) quitSignalHandler(app *appsrv.Application) {
	err := app.ShowDown(context.Background())
	if err != nil {
		log.Errorln(err.Error())
	}
	hostutils.GetWorkManager().Stop()
}

func (host *SHostService) initHandlers(app *appsrv.Application) {
	guestman.AddGuestTaskHandler("", app)
	storageman.AddStorageHandler("", app)
	storageman.AddDiskHandler("", app)
}
