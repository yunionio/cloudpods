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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

var _ IDisk = (*SRBDDisk)(nil)

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
	exist, err := storage.IsImageExist(d.Id)
	if err != nil {
		return errors.Wrapf(err, "IsImageExist")
	}
	if !exist {
		return cloudprovider.ErrNotFound
	}
	return nil
}

func (d *SRBDDisk) getPath() string {
	return d.Storage.(*SRbdStorage).getDiskPath(d.Id)
}

func (d *SRBDDisk) GetPath() string {
	storage := d.Storage.(*SRbdStorage)
	return fmt.Sprintf("%s%s", d.getPath(), storage.getStorageConfString())
}

func (d *SRBDDisk) GetFormat() (string, error) {
	return "raw", nil
}

func (d *SRBDDisk) GetSnapshotDir() string {
	return ""
}

func (d *SRBDDisk) GetDiskDesc() jsonutils.JSONObject {
	storage := d.Storage.(*SRbdStorage)

	sizeMb, _ := storage.getImageSizeMb(d.Id)
	desc := map[string]interface{}{
		"disk_id":     d.Id,
		"disk_format": "raw",
		"disk_path":   d.getPath(),
		"disk_size":   sizeMb,
	}
	return jsonutils.Marshal(desc)
}

func (d *SRBDDisk) GetDiskSetupScripts(idx int) string {
	return fmt.Sprintf("DISK_%d='%s'\n", idx, d.GetPath())
}

func (d *SRBDDisk) DeleteAllSnapshot(skipRecycle bool) error {
	return fmt.Errorf("Not Impl")
}

func (d *SRBDDisk) Delete(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	p := params.(api.DiskDeleteInput)
	storage := d.Storage.(*SRbdStorage)

	return nil, storage.deleteImage(d.Id, p.SkipRecycle != nil && *p.SkipRecycle)
}

func (d *SRBDDisk) OnRebuildRoot(ctx context.Context, params api.DiskAllocateInput) error {
	if len(params.BackingDiskId) == 0 {
		_, err := d.Delete(ctx, api.DiskDeleteInput{})
		return err
	}
	storage := d.Storage.(*SRbdStorage)
	storageConf := d.Storage.GetStorageConf()
	pool, _ := storageConf.GetString("pool")
	return storage.renameImage(pool, d.Id, params.BackingDiskId)
}

func (d *SRBDDisk) Resize(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	diskInfo, ok := params.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}
	storage := d.Storage.(*SRbdStorage)
	sizeMb, _ := diskInfo.Int("size")
	if err := storage.resizeImage(d.Id, uint64(sizeMb)); err != nil {
		return nil, err
	}

	resizeFsInfo := &deployapi.DiskInfo{
		Path: d.GetPath(),
	}
	if err := d.ResizeFs(resizeFsInfo); err != nil {
		log.Errorf("Resize fs %s fail %s", d.GetPath(), err)
		// return nil, errors.Wrapf(err, "resize fs %s", d.GetPath())
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
	return nil, storage.deleteSnapshot(d.Id, "")
}

func (d *SRBDDisk) PrepareMigrate(liveMigrate bool) ([]string, string, bool, error) {
	return nil, "", false, fmt.Errorf("Not support")
}

func (d *SRBDDisk) CreateFromTemplate(ctx context.Context, imageId string, format string, size int64, encryptInfo *apis.SEncryptInfo) (jsonutils.JSONObject, error) {
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
	input := api.CacheImageInput{
		ImageId: imageId,
		Zone:    d.GetZoneId(),
	}
	imageCache, err := imageCacheManager.AcquireImage(ctx, input, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "AcquireImage")
	}

	defer imageCacheManager.ReleaseImage(ctx, imageId)

	storage := d.Storage.(*SRbdStorage)

	storage.deleteImage(d.Id, false) //重装系统时，需要删除以前的系统盘
	err = storage.cloneImage(ctx, imageCacheManager.GetPath(), imageCache.GetName(), storage.Pool, d.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "cloneImage(%s)", imageCache.GetName())
	}
	return d.GetDiskDesc(), nil
}

func (d *SRBDDisk) CreateFromImageFuse(ctx context.Context, url string, size int64, encryptInfo *apis.SEncryptInfo) error {
	return fmt.Errorf("Not support")
}

func (d *SRBDDisk) CreateRaw(ctx context.Context, sizeMb int, diskFormat string, fsFormat string, fsFeatures *api.DiskFsFeatures, encryptInfo *apis.SEncryptInfo, diskId string, back string) (jsonutils.JSONObject, error) {
	if encryptInfo != nil {
		return nil, errors.Wrap(httperrors.ErrNotSupported, "rbd not support encryptInfo")
	}
	storage := d.Storage.(*SRbdStorage)

	if err := storage.createImage(diskId, uint64(sizeMb)); err != nil {
		return nil, err
	}

	diskInfo := &deployapi.DiskInfo{
		Path: d.GetPath(),
	}
	if utils.IsInStringArray(fsFormat, []string{"swap", "ext2", "ext3", "ext4", "xfs"}) {
		d.FormatFs(fsFormat, nil, diskId, diskInfo)
	}

	return d.GetDiskDesc(), nil
}

func (d *SRBDDisk) PostCreateFromImageFuse() {
	log.Errorf("Not support PostCreateFromImageFuse")
}

func (d *SRBDDisk) DiskBackup(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	diskBackup := params.(*SDiskBackup)
	storage := d.Storage.(*SRbdStorage)

	sizeMb, err := storage.createBackup(ctx, d.Id, diskBackup)
	if err != nil {
		return nil, err
	}
	data := jsonutils.NewDict()
	data.Set("size_mb", jsonutils.NewInt(int64(sizeMb)))
	return data, nil
}

func (d *SRBDDisk) CreateSnapshot(snapshotId string, encryptKey string, encFormat qemuimg.TEncryptFormat, encAlg seclib2.TSymEncAlg) error {
	storage := d.Storage.(*SRbdStorage)
	return storage.createSnapshot(d.Id, snapshotId)
}

func (d *SRBDDisk) ConvertSnapshot(convertSnapshotId string, encryptInfo apis.SEncryptInfo) error {
	return nil
}

func (d *SRBDDisk) DeleteSnapshot(snapshotId, convertSnapshot string, blockStream bool, encryptInfo apis.SEncryptInfo) error {
	storage := d.Storage.(*SRbdStorage)
	return storage.deleteSnapshot(d.Id, snapshotId)
}

func (d *SRBDDisk) DiskSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	snapshotId, ok := params.(string)
	if !ok {
		return nil, hostutils.ParamsError
	}
	return nil, d.CreateSnapshot(snapshotId, "", "", "")
}

func (d *SRBDDisk) DiskDeleteSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	snapshotId, ok := params.(string)
	if !ok {
		return nil, hostutils.ParamsError
	}
	err := d.DeleteSnapshot(snapshotId, "", false, apis.SEncryptInfo{})
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
	diskId := resetParams.BackingDiskId
	if len(diskId) == 0 {
		diskId = d.GetId()
	}
	storage := d.Storage.(*SRbdStorage)

	return nil, storage.resetDisk(diskId, resetParams.SnapshotId)
}

func (d *SRBDDisk) CreateFromRbdSnapshot(ctx context.Context, snapshot, srcDiskId, srcPool string) error {
	storage := d.Storage.(*SRbdStorage)
	pool, _ := storage.StorageConf.GetString("pool")
	return storage.cloneFromSnapshot(srcDiskId, srcPool, snapshot, d.GetId(), pool)
}

func (d *SRBDDisk) IsFile() bool {
	return false
}
