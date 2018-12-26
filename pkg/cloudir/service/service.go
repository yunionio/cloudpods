package service

import (
	"os"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
	"yunion.io/x/onecloud/pkg/cloudir/options"
)

func StartService() {
	consts.SetServiceType("cloudir")

	cloudcommon.ParseOptions(&options.Options, &options.Options.Options, os.Args, "cloudir.conf")

	cloudcommon.InitAuth(&options.Options.Options, func() {
		log.Infof("Auth complete!!")
	})

	err := etcd.InitDefaultEtcdClient(&options.Options.SEtcdOptions)
	if err != nil {
		log.Fatalf("init etcd fail: %s", err)
		return
	}
	defer etcd.CloseDefaultEtcdClient()

	app := cloudcommon.InitApp(&options.Options.Options)

	initHandlers(app)

	cloudcommon.ServeForever(app, &options.Options.Options)
}
