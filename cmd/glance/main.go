package main

import (
	"yunion.io/x/onecloud/pkg/image/service"
	"yunion.io/x/onecloud/pkg/util/atexit"
)

func main() {
	defer atexit.Handle()

	service.StartService()
}
