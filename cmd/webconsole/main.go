package main

import (
	"yunion.io/x/onecloud/pkg/util/atexit"
	"yunion.io/x/onecloud/pkg/webconsole/service"
)

func main() {
	defer atexit.Handle()

	service.StartService()
}
