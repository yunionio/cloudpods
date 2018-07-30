package main

import (
	"os"

	"github.com/yunionio/log"
	"github.com/yunionio/onecloud/cmd/scheduler/app"
)

func main() {
	if err := app.Execute(); err != nil {
		log.Errorln(err)
		os.Exit(-1)
	}
	os.Exit(0)
}
