package service

import (
	"os"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
	"yunion.io/x/onecloud/pkg/cloudir/options"
)

func StartService() {
	opts := &options.Options
	commonOpts := &opts.CommonOptions
	cloudcommon.ParseOptions(opts, os.Args, "cloudir.conf", "cloudir")

	cloudcommon.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})

	err := etcd.InitDefaultEtcdClient(&opts.SEtcdOptions)
	if err != nil {
		log.Fatalf("init etcd fail: %s", err)
		return
	}

	app := cloudcommon.InitApp(commonOpts, false)

	initHandlers(app)

	cloudcommon.ServeForever(app, commonOpts, func() {
		etcd.CloseDefaultEtcdClient()
	})
}
