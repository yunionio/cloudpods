package main

import (
	"os"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/scheduler/service"
)

func main() {
	if err := service.StartService(); err != nil {
		log.Errorln(err)
		os.Exit(-1)
	}
	os.Exit(0)
}
