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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SSLVMStorageDriver struct {
	SBaseStorageDriver
}

func init() {
	driver := SSLVMStorageDriver{}
	models.RegisterStorageDriver(&driver)
}

func (s *SSLVMStorageDriver) GetStorageType() string {
	return api.STORAGE_SLVM
}

func (s *SSLVMStorageDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.StorageCreateInput) error {
	input.SLVMVgName = strings.TrimSpace(input.SLVMVgName)
	if len(input.SLVMVgName) == 0 {
		return httperrors.NewMissingParameterError("slvm_vg_name")
	}
	if len(input.MasterHost) == 0 {
		return httperrors.NewMissingParameterError("master_host")
	}
	host, err := models.HostManager.FetchByIdOrName(userCred, input.MasterHost)
	if err != nil {
		return httperrors.NewInputParameterError("get host %s failed", input.MasterHost)
	}
	input.MasterHost = host.GetId()

	input.StorageConf = jsonutils.NewDict()
	input.StorageConf.Set("slvm_vg_name", jsonutils.NewString(input.SLVMVgName))
	return nil
}

func (self *SSLVMStorageDriver) ValidateSnapshotDelete(ctx context.Context, snapshot *models.SSnapshot) error {
	return nil
}

func (s *SSLVMStorageDriver) ValidateCreateSnapshotData(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, input *api.SnapshotCreateInput) error {
	return errors.Errorf("lvm storage unsupported create snapshot")
}

func (s *SSLVMStorageDriver) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, storage *models.SStorage, data jsonutils.JSONObject) {
	sc := &models.SStoragecache{}
	sc.ExternalId = storage.Id
	sc.MasterHost = storage.MasterHost
	sc.Name = "slvm-" + storage.Name + time.Now().Format("2006-01-02 15:04:05")
	if err := models.StoragecacheManager.TableSpec().Insert(ctx, sc); err != nil {
		log.Errorf("insert storagecache for storage %s error: %v", storage.Name, err)
		return
	}
	_, err := db.Update(storage, func() error {
		storage.StoragecacheId = sc.Id
		storage.Status = api.STORAGE_ONLINE
		return nil
	})
	if err != nil {
		log.Errorf("update storagecache info for storage %s error: %v", storage.Name, err)
	}
}
