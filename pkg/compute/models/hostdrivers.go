package models

import (
	"context"
	"log"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
)

type IHostDriver interface {
	GetHostType() string
	CheckAndSetCacheImage(ctx context.Context, host *SHost, storagecache *SStoragecache, scimg *SStoragecachedimage, task taskman.ITask) error
}

var hostDrivers map[string]IHostDriver

func init() {
	hostDrivers = make(map[string]IHostDriver)
}

func RegisterHostDriver(driver IHostDriver) {
	hostDrivers[driver.GetHostType()] = driver
}

func GetHostDriver(hostType string) IHostDriver {
	driver, ok := hostDrivers[hostType]
	if ok {
		return driver
	} else {
		log.Fatalf("Unsupported hostType %s", hostType)
		return nil
	}
}
