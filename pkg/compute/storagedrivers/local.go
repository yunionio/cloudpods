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

package storagedrivers

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SLocalStorageDriver struct {
	SBaseStorageDriver
}

func init() {
	driver := SLocalStorageDriver{}
	models.RegisterStorageDriver(&driver)
	lvmDriver := SLVMStorageDriver{}
	models.RegisterStorageDriver(&lvmDriver)
}

func (self *SLocalStorageDriver) GetStorageType() string {
	return api.STORAGE_LOCAL
}

func (self *SLocalStorageDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.StorageCreateInput) error {
	return nil
}

func (self *SLocalStorageDriver) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, storage *models.SStorage, data jsonutils.JSONObject) {
}

type SLVMStorageDriver struct {
	SBaseStorageDriver
}

func (self *SLVMStorageDriver) GetStorageType() string {
	return api.STORAGE_LVM
}

func (self *SLVMStorageDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.StorageCreateInput) error {
	return nil
}

func (self *SLVMStorageDriver) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, storage *models.SStorage, data jsonutils.JSONObject) {
	sc := &models.SStoragecache{}
	sc.SetModelManager(models.StoragecacheManager, sc)
	sc.Name = fmt.Sprintf("lvm-imagecache-%s", storage.Id)
	if err := models.StoragecacheManager.TableSpec().Insert(ctx, sc); err != nil {
		log.Errorf("insert storagecache for storage %s error: %v", storage.Name, err)
		return
	}
	_, err := db.Update(storage, func() error {
		storage.StoragecacheId = sc.Id
		return nil
	})
	if err != nil {
		log.Errorf("update storagecache info for storage %s error: %v", storage.Name, err)
	}
}

func (self *SLVMStorageDriver) ValidateCreateSnapshotData(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, input *api.SnapshotCreateInput) error {
	return errors.Errorf("lvm storage unsupported create snapshot")
}

func (self *SLVMStorageDriver) ValidateSnapshotDelete(ctx context.Context, snapshot *models.SSnapshot) error {
	return nil
}
