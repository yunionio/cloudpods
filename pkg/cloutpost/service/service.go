package service

import (
	"os"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd/models"
	"yunion.io/x/onecloud/pkg/cloutpost/options"
)

const (
	SERVICE_TYPE = "cloutpost"
)

func StartService() {
	opts := &options.Options
	commonOpts := &opts.CommonOptions
	cloudcommon.ParseOptions(opts, os.Args, "cloutpost.conf", SERVICE_TYPE)

	cloudcommon.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})

	err := etcd.InitDefaultEtcdClient(&opts.SEtcdOptions)
	if err != nil {
		log.Fatalf("init etcd fail: %s", err)
	}
	defer etcd.CloseDefaultEtcdClient()

	app := cloudcommon.InitApp(commonOpts, false)

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

	cloudcommon.ServeForever(app, commonOpts)
}
