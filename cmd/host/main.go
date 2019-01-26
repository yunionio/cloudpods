package main

import "yunion.io/x/onecloud/pkg/hostman"

func main() {
	var srv = hostman.SHostService{}
	srv.StartService()
}
