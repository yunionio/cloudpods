package service

import (
	"os"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/baremetal"
	"yunion.io/x/onecloud/pkg/baremetal/handler"
	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
)

func StartService() {
	consts.SetServiceType("baremetal")

	cloudcommon.ParseOptions(&o.Options, &o.Options.CommonOptions, os.Args, "baremetal.conf")
	cloudcommon.InitAuth(&o.Options.CommonOptions, startAgent)

	app := cloudcommon.InitApp(&o.Options.CommonOptions, false)
	handler.InitHandlers(app)
	cloudcommon.ServeForever(app, &o.Options.CommonOptions)
}

func startAgent() {
	err := baremetal.Start()
	if err != nil {
		log.Fatalf("Start agent error: %v", err)
	}
}
