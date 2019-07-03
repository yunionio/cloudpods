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
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SNasDisk struct {
	SLocalDisk
}

func NewNasDisk(storage IStorage, id string) *SNasDisk {
	return &SNasDisk{*NewLocalDisk(storage, id)}
}

func (d *SNasDisk) CreateFromTemplate(ctx context.Context, imageId, format string, size int64) (jsonutils.JSONObject, error) {
	imageCacheManager := storageManager.GetStoragecacheById(d.Storage.GetStoragecacheId())
	ret, err := d.SLocalDisk.createFromTemplate(ctx, imageId, format, imageCacheManager)
	if err != nil {
		return nil, err
	}
	retSize, _ := ret.Int("disk_size")
	log.Infof("REQSIZE: %d, RETSIZE: %d", size, retSize)
	if size > retSize {
		params := jsonutils.NewDict()
		params.Set("size", jsonutils.NewInt(size))
		return d.Resize(ctx, params)
	}
	return ret, nil
}

type SNFSDisk struct {
	SNasDisk
}

func NewNFSDisk(storage IStorage, id string) *SNFSDisk {
	return &SNFSDisk{
		SNasDisk: *NewNasDisk(storage, id),
	}
}

func (d *SNFSDisk) GetType() string {
	return api.STORAGE_NFS
}

type SGPFSDisk struct {
	SNasDisk
}

func NewGPFSDisk(storage IStorage, id string) *SGPFSDisk {
	return &SGPFSDisk{
		SNasDisk: *NewNasDisk(storage, id),
	}
}

func (d *SGPFSDisk) GetType() string {
	return api.STORAGE_GPFS
}
