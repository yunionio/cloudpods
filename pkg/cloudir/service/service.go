package service

import (
	"os"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudir/options"
)

func StartService() {
	opts := &options.Options
	commonOpts := &opts.CommonOptions
	common_options.ParseOptions(opts, os.Args, "cloudir.conf", "cloudir")

	app_common.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})

	err := etcd.InitDefaultEtcdClient(&opts.SEtcdOptions)
	if err != nil {
		log.Fatalf("init etcd fail: %s", err)
		return
	}

	app := app_common.InitApp(commonOpts, false)
	cloudcommon.AppDBInit(app)
	initHandlers(app)

	app_common.ServeForeverWithCleanup(app, commonOpts, func() {
		etcd.CloseDefaultEtcdClient()
	})
}
