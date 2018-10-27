package service

import (
	"os"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/baremetal"
	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/cloudcommon"
)

func StartService() {
	cloudcommon.ParseOptions(&o.Options, &o.Options.Options, os.Args, "baremetal.conf")
	cloudcommon.InitAuth(&o.Options.Options, startAgent)

	app := cloudcommon.InitApp(&o.Options.Options)
	baremetal.InitHandlers(app)
	cloudcommon.ServeForever(app, &o.Options.Options)
}

func startAgent() {
	go func() {
		err := baremetal.Start()
		if err != nil {
			log.Fatalf("Start agent error: %v", err)
		}
	}()
}
