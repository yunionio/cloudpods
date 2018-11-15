package service

import (
	"os"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd/models"
	"yunion.io/x/onecloud/pkg/cloutpost/options"
)

const (
	SERVICE_TYPE = "cloutpost"
)

func StartService() {
	consts.SetServiceType(SERVICE_TYPE)

	cloudcommon.ParseOptions(&options.Options, &options.Options.Options, os.Args, "cloutpost.conf")

	cloudcommon.InitAuth(&options.Options.Options, func() {
		log.Infof("Auth complete!!")
	})

	err := etcd.InitDefaultEtcdClient(&options.Options.SEtcdOptions)
	if err != nil {
		log.Fatalf("init etcd fail: %s", err)
	}
	defer etcd.CloseDefaultEtcdClient()

	app := cloudcommon.InitApp(&options.Options.Options)

	initHandlers(app)

	err = models.ServiceRegistryManager.Register(
		app.GetContext(),
		options.Options.Address,
		options.Options.Port,
		options.Options.Provider,
		options.Options.Environment,
		options.Options.Cloudregion,
		options.Options.Zone,
		SERVICE_TYPE,
	)

	if err != nil {
		log.Fatalf("fail to register service %s", err)
	}

	cloudcommon.ServeForever(app, &options.Options.Options)
}
