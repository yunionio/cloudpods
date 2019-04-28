package main

import (
	"yunion.io/x/onecloud/pkg/baremetal/service"
	"yunion.io/x/onecloud/pkg/util/atexit"
)

func main() {
	defer atexit.Handle()

	service.New().StartService()
}
