package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
)

type IHostDriver interface {
	GetHostType() string
	CheckAndSetCacheImage(ctx context.Context, host *SHost, storagecache *SStoragecache, scimg *SStoragecachedimage, task taskman.ITask) error
	RequestAllocateDiskOnStorage(host *SHost, storage *SStorage, disk *SDisk, task taskman.ITask, content *jsonutils.JSONDict) error
	RequestDeallocateDiskOnHost(host *SHost, storage *SStorage, disk *SDisk, task taskman.ITask) error
	RequestResizeDiskOnHostOnline(host *SHost, storage *SStorage, disk *SDisk, size int64, task taskman.ITask) error
	RequestResizeDiskOnHost(host *SHost, storage *SStorage, disk *SDisk, size int64, task taskman.ITask) error
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
