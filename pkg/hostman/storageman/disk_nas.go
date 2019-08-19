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
	"fmt"
	"path"

	"github.com/pkg/errors"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
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

func (d *SNasDisk) CreateFromImageFuse(ctx context.Context, url string, size int64) error {
	return fmt.Errorf("Not implemented")
}

func (d *SNasDisk) CreateFromSnapshotLocation(ctx context.Context, snapshotLocation string, size int64) error {
	snapshotPath := path.Join(d.Storage.GetPath(), snapshotLocation)
	newImg, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		return errors.Wrap(err, "new image from snapshot")
	}
	if newImg.IsValid() {
		if err := newImg.Delete(); err != nil {
			log.Errorln(err)
			return err
		}
	}
	err = newImg.CreateQcow2(0, false, snapshotPath)
	if err != nil {
		return errors.Wrap(err, "create image from snapshot")
	}
	retSize, _ := d.GetDiskDesc().Int("disk_size")
	log.Infof("REQSIZE: %d, RETSIZE: %d", size, retSize)
	if size > retSize {
		params := jsonutils.NewDict()
		params.Set("size", jsonutils.NewInt(size))
		_, err = d.Resize(ctx, params)
		return err
	}
	return nil
}

func (d *SNasDisk) ResetFromSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	resetParams, ok := params.(*SDiskReset)
	if !ok {
		return nil, hostutils.ParamsError
	}

	outOfChain, err := resetParams.Input.Bool("out_of_chain")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("out_of_chain")
	}

	location, err := resetParams.Input.GetString("location")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("location")
	}
	snapshotPath := path.Join(d.Storage.GetPath(), location)
	log.Infof("Snapshot path is %s", snapshotPath)
	return d.resetFromSnapshot(snapshotPath, outOfChain)
}

func (d *SNasDisk) GetSnapshotLocation() string {
	return d.GetSnapshotDir()[len(d.Storage.GetPath())+1:]
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
