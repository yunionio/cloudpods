package storageagent

import (
	"os"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/service"
	"yunion.io/x/onecloud/pkg/hostman/diskhandlers"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/storageagent/options"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

type SStorageAgent struct {
	*service.SServiceBase
}

func (s *SStorageAgent) InitService() {
	common_options.ParseOptions(&options.StorageAgentOptions, os.Args, "storage.conf", "storage")
	isRoot := sysutils.IsRootPermission()
	if !isRoot {
		log.Fatalf("Storage agent must running with root permissions")
	}

	options.StorageAgentOptions.EnableRbac = false // disable rbac
	s.SServiceBase.O = &options.StorageAgentOptions.BaseOptions
}

func (s *SStorageAgent) RunService() {
	app := app_common.InitApp(&options.StorageAgentOptions.BaseOptions, false)
	app_common.InitAuth(&options.StorageAgentOptions.CommonOptions, func() {
		log.Infof("Auth complete!!")
		// TODO
		storageman.InitAgent()
	})
	s.initHandlers(app)
	app_common.ServeForeverWithCleanup(app, &options.StorageAgentOptions.BaseOptions, func() {
		log.Infof("Exiting ...")
	})
}

func (s *SStorageAgent) initHandlers(app *appsrv.Application) {
	diskhandlers.AddDiskHandler("", app)
}

func (s *SStorageAgent) OnExitService() {
	// pass
}
