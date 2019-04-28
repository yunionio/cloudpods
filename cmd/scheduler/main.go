package main

import (
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/scheduler/service"
	"yunion.io/x/onecloud/pkg/util/atexit"
)

func main() {
	exitCode := 0
	defer func() {
		atexit.Exit(exitCode)
	}()

	if err := service.StartService(); err != nil {
		log.Errorln(err)
		exitCode = -1
	}
}
