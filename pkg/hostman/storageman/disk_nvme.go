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

package storageman

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/qemuimgfmt"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
)

var _ IDisk = (*SNVMEDisk)(nil)

type SNVMEDisk struct {
	SBaseDisk
}

func (d *SNVMEDisk) GetSnapshotDir() string {
	return ""
}

func (d *SNVMEDisk) GetDiskDesc() jsonutils.JSONObject {
	var desc = jsonutils.NewDict()

	desc.Set("disk_id", jsonutils.NewString(d.Id))
	desc.Set("disk_size", jsonutils.NewInt(int64(d.Storage.GetCapacityMb())))
	desc.Set("format", jsonutils.NewString(string(qemuimgfmt.RAW)))
	desc.Set("disk_path", jsonutils.NewString(d.Storage.GetPath()))
	return desc
}

func (d *SNVMEDisk) CreateRaw(ctx context.Context, sizeMb int, diskFormat string, fsFormat string, fsFeatures *api.DiskFsFeatures, encryptInfo *apis.SEncryptInfo, diskId string, back string) (jsonutils.JSONObject, error) {
	// do nothing
	return nil, nil
}

func (s *SNVMEDisk) Delete(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (d *SNVMEDisk) IsFile() bool {
	return false
}

func (d *SNVMEDisk) Probe() error {
	return nil
}

func (d *SNVMEDisk) DiskBackup(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, errors.ErrNotImplemented
}

func NewNVMEDisk(storage IStorage, id string) *SNVMEDisk {
	return &SNVMEDisk{
		SBaseDisk: *NewBaseDisk(storage, id),
	}
}
