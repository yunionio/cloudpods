package main

import (
	"yunion.io/x/onecloud/pkg/cloudcommon/service"
	"yunion.io/x/onecloud/pkg/hostman"
)

func main() {
	var srv = &hostman.SHostService{}
	srv.SServiceBase = &service.SServiceBase{
		Service: srv,
	}
	srv.StartService()
}
