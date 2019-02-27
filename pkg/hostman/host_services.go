package hostman

import (
	"os"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/service"
	"yunion.io/x/onecloud/pkg/hostman/diskhandlers"
	"yunion.io/x/onecloud/pkg/hostman/downloader"
	"yunion.io/x/onecloud/pkg/hostman/guesthandlers"
	"yunion.io/x/onecloud/pkg/hostman/guestman"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo"
	"yunion.io/x/onecloud/pkg/hostman/hostmetrics"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/metadata"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
)

type SHostService struct {
	service.SServiceBase
}

func (host *SHostService) StartService() {
	cloudcommon.ParseOptions(&options.HostOptions, os.Args, "host.conf", "host")
	options.HostOptions.EnableRbac = false // disable rbac

	app := cloudcommon.InitApp(&options.HostOptions.CommonOptions, false)
	hostInstance := hostinfo.Instance()
	if err := hostInstance.Init(); err != nil {
		log.Fatalf(err.Error())
	}

	if err := storageman.Init(hostInstance); err != nil {
		log.Fatalf(err.Error())
	}

	guestman.Init(hostInstance, options.HostOptions.ServersPath)
	cloudcommon.InitAuth(&options.HostOptions.CommonOptions, func() {
		log.Infof("Auth complete!!")
		// ??? Why wait 5 seconds

		hostInstance.StartRegister(2, func() {
			guestman.GetGuestManager().Bootstrap()
			// hostmetrics after guestmanager bootstrap
			hostmetrics.Init()
			hostmetrics.Start()
		})
	})
	host.initHandlers(app)
	<-hostinfo.Instance().IsRegistered // wait host and guest init

	// Init Metadata handler
	go metadata.StartService(
		cloudcommon.InitApp(&options.HostOptions.CommonOptions, false),
		options.HostOptions.Address, options.HostOptions.Port+1000)

	cronManager := cronman.GetCronJobManager(false)
	cronManager.AddJob2(
		"CleanRecycleDiskFiles", 1, 3, 0, 0, storageman.CleanRecycleDiskfiles, false)

	cloudcommon.ServeForeverWithCleanup(app, &options.HostOptions.CommonOptions, func() {
		hostinfo.Stop()
		storageman.Stop()
		hostmetrics.Stop()
		guestman.Stop()
		hostutils.GetWorkManager().Stop()
	})
}

func (host *SHostService) initHandlers(app *appsrv.Application) {
	guesthandlers.AddGuestTaskHandler("", app)
	storageman.AddStorageHandler("", app)
	diskhandlers.AddDiskHandler("", app)
	downloader.AddDownloadHandler("", app)
	addKubeAgentHandler("", app)
}
