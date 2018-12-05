package main

import "yunion.io/x/onecloud/pkg/hostman/service"

func main() {
	var srv = service.SHostService{}
	srv.StartService()
}
