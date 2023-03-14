package main

import (
	"context"

	"yunion.io/x/onecloud/pkg/hostman/host_health"
	"yunion.io/x/onecloud/pkg/util/atexit"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func main() {
	defer atexit.Handle()

	go procutils.WaitZombieLoop(context.TODO())

	host_health.StartService()
}
