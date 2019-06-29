package main

import (
	"yunion.io/x/onecloud/pkg/s3gateway/service"
	"yunion.io/x/onecloud/pkg/util/atexit"
)

func main() {
	defer atexit.Handle()

	service.StartService()
}
