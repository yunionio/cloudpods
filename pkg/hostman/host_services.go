package hostman

import (
	"context"
	"os"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/service"
	"yunion.io/x/onecloud/pkg/hostman/downloader"
	"yunion.io/x/onecloud/pkg/hostman/guestman"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
)

type SHostService struct {
	service.SServiceBase

	isExiting bool
}

func (host *SHostService) StartService() {
	cloudcommon.ParseOptions(&options.HostOptions, os.Args, "host.conf", "host")
	options.HostOptions.EnableRbac = false // disable rbac

	app := cloudcommon.InitApp(&options.HostOptions.CommonOptions, false)
	hostInstance := hostinfo.Instance()
	if err := hostInstance.Init(); err != nil {
		log.Fatalf(err.Error())
	}

	host.RegisterSIGUSR1()
	host.RegisterQuitSignals(func() { // register quit handler
		if host.isExiting {
			return
		} else {
			host.isExiting = true
		}

		if app.IsInServe() {
			err := app.ShowDown(context.Background())
			if err != nil {
				log.Errorln(err.Error())
			}
		}

		hostinfo.Stop()
		storageman.Stop()
		guestman.Stop()
		hostutils.GetWorkManager().Stop()

		os.Exit(0)
	})

	if err := storageman.Init(hostInstance); err != nil {
		log.Fatalf(err.Error())
	}

	guestman.Init(hostInstance, options.HostOptions.ServersPath)
	cloudcommon.InitAuth(&options.HostOptions.CommonOptions, func() {
		log.Infof("Auth complete!!")
		// ??? Why wait 5 seconds
		hostInstance.StartRegister(2, guestman.GetGuestManager().Bootstrap)
	})
	host.initHandlers(app)

	<-hostinfo.Instance().IsRegistered // wait host and guest init
	cloudcommon.ServeForever(app, &options.HostOptions.CommonOptions)
}

func (host *SHostService) initHandlers(app *appsrv.Application) {
	guestman.AddGuestTaskHandler("", app)
	storageman.AddStorageHandler("", app)
	storageman.AddDiskHandler("", app)
	downloader.AddDownloadHandler("", app)
	addKubeAgentHandler("", app)
}
