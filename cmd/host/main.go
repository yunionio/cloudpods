package main

import (
	"yunion.io/x/onecloud/pkg/cloudcommon/service"
	"yunion.io/x/onecloud/pkg/hostman"
	"yunion.io/x/onecloud/pkg/util/atexit"
)

func main() {
	defer atexit.Handle()

	var srv = &hostman.SHostService{}
	srv.SServiceBase = &service.SServiceBase{
		Service: srv,
	}
	srv.StartService()
}
