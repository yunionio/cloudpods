package main

import (
	"yunion.io/x/onecloud/pkg/hostimage"
	"yunion.io/x/onecloud/pkg/util/atexit"
)

func main() {
	defer atexit.Handle()

	hostimage.StartService()
}
