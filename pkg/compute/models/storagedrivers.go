package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IStorageDriver interface {
	GetStorageType() string

	ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)

	PostCreate(ctx context.Context, userCred mcclient.TokenCredential, storage *SStorage, data jsonutils.JSONObject)
}

var storageDrivers map[string]IStorageDriver

func init() {
	storageDrivers = make(map[string]IStorageDriver)
}

func RegisterStorageDriver(driver IStorageDriver) {
	storageDrivers[driver.GetStorageType()] = driver
}

func GetStorageDriver(storageType string) IStorageDriver {
	driver, ok := storageDrivers[storageType]
	if ok {
		return driver
	}
	log.Fatalf("Unsupported storageType %s", storageType)
	return nil
}
