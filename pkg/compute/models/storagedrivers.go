// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
