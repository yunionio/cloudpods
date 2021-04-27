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

// +build linux,cgo

package storageman

import (
	"context"
	"fmt"

	"github.com/ceph/go-ceph/rbd"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
)

type SRBDDisk struct {
	SBaseDisk
}

func NewRBDDisk(storage IStorage, id string) *SRBDDisk {
	var ret = new(SRBDDisk)
	ret.SBaseDisk = *NewBaseDisk(storage, id)
	return ret
}

func (d *SRBDDisk) GetType() string {
	return api.STORAGE_RBD
}

func (d *SRBDDisk) Probe() error {
	storage := d.Storage.(*SRbdStorage)
	storageConf := d.Storage.GetStorageConf()
	pool, _ := storageConf.GetString("pool")
	_, err := storage.withImage(pool, d.Id, func(image *rbd.Image) (interface{}, error) {
		return image.GetSize()
	})
	return err
}

func (d *SRBDDisk) getPath() string {
	storageConf := d.Storage.GetStorageConf()
	pool, _ := storageConf.GetString("pool")
	return fmt.Sprintf("rbd:%s/%s", pool, d.Id)
}

func (d *SRBDDisk) GetPath() string {
	storage := d.Storage.(*SRbdStorage)
	return fmt.Sprintf("%s%s", d.getPath(), storage.getStorageConfString())
}

func (d *SRBDDisk) GetSnapshotDir() string {
	return ""
}

func (d *SRBDDisk) GetDiskDesc() jsonutils.JSONObject {
	storage := d.Storage.(*SRbdStorage)
	storageConf := d.Storage.GetStorageConf()
	pool, _ := storageConf.GetString("pool")
	desc := map[string]interface{}{
		"disk_id":     d.Id,
		"disk_format": "raw",
		"disk_path":   d.GetPath(),
		"disk_size":   storage.getImageSizeMb(pool, d.Id),
	}
	return jsonutils.Marshal(desc)
}

func (d *SRBDDisk) GetDiskSetupScripts(idx int) string {
	return fmt.Sprintf("DISK_%d=%s\n", idx, d.GetPath())
}

func (d *SRBDDisk) DeleteAllSnapshot() error {
	return fmt.Errorf("Not Impl")
}

func (d *SRBDDisk) Delete(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	storage := d.Storage.(*SRbdStorage)
	storageConf := d.Storage.GetStorageConf()
	pool, _ := storageConf.GetString("pool")
	return nil, storage.deleteImage(pool, d.Id)
}

func (d *SRBDDisk) OnRebuildRoot(ctx context.Context, params jsonutils.JSONObject) error {
	backingDiskId, err := params.GetString("backing_disk_id")
	if err != nil {
		_, err := d.Delete(ctx, params)
		return err
	} else {
		storage := d.Storage.(*SRbdStorage)
		storageConf := d.Storage.GetStorageConf()
		pool, _ := storageConf.GetString("pool")
		return storage.renameImage(pool, d.Id, backingDiskId)
	}
}

func (d *SRBDDisk) Resize(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	diskInfo, ok := params.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}
	storage := d.Storage.(*SRbdStorage)
	storageConf := d.Storage.GetStorageConf()
	pool, _ := storageConf.GetString("pool")
	sizeMb, _ := diskInfo.Int("size")
	if err := storage.resizeImage(pool, d.Id, uint64(sizeMb)); err != nil {
		return nil, err
	}

	if err := d.ResizeFs(d.GetPath()); err != nil {
		return nil, errors.Wrapf(err, "resize fs %s", d.GetPath())
	}

	return d.GetDiskDesc(), nil
}

func (d *SRBDDisk) PrepareSaveToGlance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	if err := d.Probe(); err != nil {
		return nil, err
	}
	imageName := fmt.Sprintf("image_cache_%s_%s", d.Id, appctx.AppContextTaskId(ctx))
	imageCache := storageManager.GetStoragecacheById(d.Storage.GetStoragecacheId())
	if imageCache == nil {
		return nil, fmt.Errorf("failed to find image cache for prepare save to glance")
	}
	storage := d.Storage.(*SRbdStorage)
	pool, _ := storage.GetStorageConf().GetString("pool")
	if err := storage.cloneImage(ctx, pool, d.Id, imageCache.GetPath(), imageName); err != nil {
		log.Errorf("clone image %s from pool %s to %s/%s error: %v", d.Id, pool, imageCache.GetPath(), imageName, err)
		return nil, err
	}
	return jsonutils.Marshal(map[string]string{"backup": imageName}), nil
}

func (d *SRBDDisk) CleanupSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	storage := d.Storage.(*SRbdStorage)
	pool, _ := storage.StorageConf.GetString("pool")
	return nil, storage.deleteSnapshot(pool, d.Id, "")
}

func (d *SRBDDisk) PrepareMigrate(liveMigrate bool) (string, error) {
	return "", fmt.Errorf("Not support")
}

func (d *SRBDDisk) CreateFromTemplate(ctx context.Context, imageId string, format string, size int64) (jsonutils.JSONObject, error) {
	ret, err := d.createFromTemplate(ctx, imageId, format)
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

func (d *SRBDDisk) createFromTemplate(ctx context.Context, imageId, format string) (jsonutils.JSONObject, error) {
	var imageCacheManager = storageManager.GetStoragecacheById(d.Storage.GetStoragecacheId())
	if imageCacheManager == nil {
		return nil, fmt.Errorf("failed to find image cache manger for storage %s", d.Storage.GetStorageName())
	}
	imageCache := imageCacheManager.AcquireImage(ctx, imageId, d.GetZoneName(), "", "", "")
	if imageCache == nil {
		return nil, fmt.Errorf("failed to qcquire image for storage %s", d.Storage.GetStorageName())
	}
	defer imageCacheManager.ReleaseImage(ctx, imageId)
	storage := d.Storage.(*SRbdStorage)
	destPool, _ := storage.StorageConf.GetString("pool")
	storage.deleteImage(destPool, d.Id) //重装系统时，需要删除以前的系统盘
	if err := storage.cloneImage(ctx, imageCacheManager.GetPath(), imageCache.GetName(), destPool, d.Id); err != nil {
		return nil, err
	}
	return d.GetDiskDesc(), nil
}

func (d *SRBDDisk) CreateFromImageFuse(ctx context.Context, url string, size int64) error {
	return fmt.Errorf("Not support")
}

func (d *SRBDDisk) CreateRaw(ctx context.Context, sizeMb int, diskFromat string, fsFormat string, encryption bool, diskId string, back string) (jsonutils.JSONObject, error) {
	storage := d.Storage.(*SRbdStorage)
	pool, _ := storage.StorageConf.GetString("pool")
	if err := storage.createImage(pool, diskId, uint64(sizeMb)); err != nil {
		return nil, err
	}

	if utils.IsInStringArray(fsFormat, []string{"swap", "ext2", "ext3", "ext4", "xfs"}) {
		d.FormatFs(fsFormat, diskId, d.GetPath())
	}

	return d.GetDiskDesc(), nil
}

func (d *SRBDDisk) PostCreateFromImageFuse() {
	log.Errorf("Not support PostCreateFromImageFuse")
}

func (d *SRBDDisk) CreateSnapshot(snapshotId string) error {
	storage := d.Storage.(*SRbdStorage)
	pool, _ := storage.StorageConf.GetString("pool")
	return storage.createSnapshot(pool, d.Id, snapshotId)
}

func (d *SRBDDisk) DeleteSnapshot(snapshotId, convertSnapshot string, pendingDelete bool) error {
	storage := d.Storage.(*SRbdStorage)
	pool, _ := storage.StorageConf.GetString("pool")
	return storage.deleteSnapshot(pool, d.Id, snapshotId)
}

func (d *SRBDDisk) DiskSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	snapshotId, ok := params.(string)
	if !ok {
		return nil, hostutils.ParamsError
	}
	return nil, d.CreateSnapshot(snapshotId)
}

func (d *SRBDDisk) DiskDeleteSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	snapshotId, ok := params.(string)
	if !ok {
		return nil, hostutils.ParamsError
	}
	err := d.DeleteSnapshot(snapshotId, "", false)
	if err != nil {
		return nil, err
	} else {
		res := jsonutils.NewDict()
		res.Set("deleted", jsonutils.JSONTrue)
		return res, nil
	}
}

func (d *SRBDDisk) ResetFromSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	resetParams, ok := params.(*SDiskReset)
	if !ok {
		return nil, hostutils.ParamsError
	}
	storage := d.Storage.(*SRbdStorage)
	pool, _ := storage.StorageConf.GetString("pool")
	return nil, storage.resetDisk(pool, d.GetId(), resetParams.SnapshotId)
}

func (d *SRBDDisk) CreateFromRbdSnapshot(ctx context.Context, snapshot, srcDiskId, srcPool string) error {
	storage := d.Storage.(*SRbdStorage)
	pool, _ := storage.StorageConf.GetString("pool")
	return storage.cloneFromSnapshot(srcDiskId, srcPool, snapshot, d.GetId(), pool)
}
