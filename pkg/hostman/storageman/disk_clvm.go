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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman/lvmutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman/storageutils"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

type SCLVMDisk struct {
	SBaseDisk
}

func NewCLVMDisk(storage IStorage, id string) *SCLVMDisk {
	return &SCLVMDisk{
		SBaseDisk: *NewBaseDisk(storage, id),
	}
}

func (d *SCLVMDisk) GetType() string {
	return api.STORAGE_CLVM
}

// /dev/<vg>/<lvm>
func (d *SCLVMDisk) GetLvPath() string {
	return path.Join("/dev", d.Storage.GetPath(), d.Id)
}

func (d *SCLVMDisk) GetPath() string {
	return path.Join("/dev", d.Storage.GetPath(), d.Id)
}

func (d *SCLVMDisk) GetDiskSetupScripts(idx int) string {
	return fmt.Sprintf("DISK_%d='%s'\n", idx, d.GetPath())
}

func (d *SCLVMDisk) GetDiskDesc() jsonutils.JSONObject {
	qemuImg, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		log.Errorln(err)
		return nil
	}

	var desc = jsonutils.NewDict()
	desc.Set("disk_id", jsonutils.NewString(d.Id))
	desc.Set("disk_size", jsonutils.NewInt(qemuImg.SizeBytes/1024/1024))
	desc.Set("format", jsonutils.NewString(string(qemuImg.Format)))
	desc.Set("disk_path", jsonutils.NewString(d.Storage.GetPath()))
	return desc
}

func (d *SCLVMDisk) CreateRaw(
	ctx context.Context, sizeMb int, diskFormat string, fsFormat string,
	encryptInfo *apis.SEncryptInfo, diskId string, back string,
) (jsonutils.JSONObject, error) {
	if fileutils2.Exists(d.GetPath()) {
		if err := lvmutils.LvRemove(d.GetLvPath()); err != nil {
			return nil, errors.Wrap(err, "CreateRaw lvremove")
		}
	}
	if err := lvmutils.LvCreate(d.Storage.GetPath(), d.Id, int64(sizeMb)*1024*1024); err != nil {
		return nil, errors.Wrap(err, "CreateRaw")
	}

	diskInfo := &deployapi.DiskInfo{
		Path: d.GetPath(),
	}
	if utils.IsInStringArray(fsFormat, []string{"swap", "ext2", "ext3", "ext4", "xfs"}) {
		d.FormatFs(fsFormat, diskId, diskInfo)
	}
	return d.GetDiskDesc(), nil
}

func (d *SCLVMDisk) CreateFromTemplate(
	ctx context.Context, imageId, format string, sizeMb int64, encryptInfo *apis.SEncryptInfo,
) (jsonutils.JSONObject, error) {
	if fileutils2.Exists(d.GetPath()) {
		if err := lvmutils.LvRemove(d.GetLvPath()); err != nil {
			return nil, errors.Wrap(err, "CreateRaw lvremove")
		}
	}

	var imageCacheManager = storageManager.GetStoragecacheById(d.Storage.GetStoragecacheId())
	ret, err := d.createFromTemplate(ctx, imageId, format, sizeMb, imageCacheManager, encryptInfo)
	if err != nil {
		return nil, err
	}
	retSize, _ := ret.Int("disk_size")
	log.Infof("REQSIZE: %d, RETSIZE: %d", sizeMb, retSize)
	if sizeMb > retSize {
		params := jsonutils.NewDict()
		params.Set("size", jsonutils.NewInt(sizeMb))
		if encryptInfo != nil {
			params.Set("encrypt_info", jsonutils.Marshal(encryptInfo))
		}
		return d.Resize(ctx, params)
	}
	return ret, nil
}

func (d *SCLVMDisk) createFromTemplate(
	ctx context.Context, imageId, format string, sizeMb int64, imageCacheManager IImageCacheManger, encryptInfo *apis.SEncryptInfo,
) (jsonutils.JSONObject, error) {
	input := api.CacheImageInput{ImageId: imageId, Zone: d.GetZoneId()}
	imageCache, err := imageCacheManager.AcquireImage(ctx, input, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "AcquireImage")
	}

	defer imageCacheManager.ReleaseImage(ctx, imageId)
	cacheImagePath := imageCache.GetPath()

	lvSizeMb := d.getQcow2LvSize(sizeMb)
	if err := lvmutils.LvCreate(d.Storage.GetPath(), d.Id, lvSizeMb*1024*1024); err != nil {
		return nil, errors.Wrap(err, "CreateRaw")
	}
	newImg, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		return nil, errors.Wrapf(err, "NewQemuImage(%s)", d.GetPath())
	}
	err = newImg.CreateQcow2(int(sizeMb), false, cacheImagePath, "", "", "")
	if err != nil {
		return nil, errors.Wrapf(err, "CreateQcow2(%s)", cacheImagePath)
	}

	return d.GetDiskDesc(), nil
}

func (d *SCLVMDisk) Probe() error {
	if !fileutils2.Exists(d.GetPath()) {
		return errors.Wrapf(cloudprovider.ErrNotFound, "%s", d.GetPath())
	}
	return nil
}

func (d *SCLVMDisk) GetSnapshotDir() string {
	return ""
}

func (d *SCLVMDisk) OnRebuildRoot(ctx context.Context, params api.DiskAllocateInput) error {
	_, err := d.Delete(ctx, api.DiskDeleteInput{})
	return err
}

func (d *SCLVMDisk) Delete(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	if err := lvmutils.LvRemove(d.GetLvPath()); err != nil {
		return nil, errors.Wrap(err, "Delete lvremove")
	}
	d.Storage.RemoveDisk(d)
	return nil, nil
}

func (d *SCLVMDisk) getQcow2LvSize(sizeMb int64) int64 {
	// Qcow2 cluster size 2M, 100G reserve 1M for qcow2 metadata
	metaSize := sizeMb/1024/100 + 2
	return sizeMb + metaSize
}

func (d *SCLVMDisk) Resize(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	diskInfo, ok := params.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}
	sizeMb, _ := diskInfo.Int("size")

	qemuImg, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		return nil, errors.Wrap(err, "lvm qemuimg.NewQemuImage")
	}

	lvsize := sizeMb
	if qemuImg.Format == qemuimg.QCOW2 {
		lvsize = d.getQcow2LvSize(sizeMb)
	}

	err = lvmutils.LvResize(d.Storage.GetPath(), d.GetPath(), lvsize*1024*1024)
	if err != nil {
		return nil, errors.Wrap(err, "lv resize")
	}
	err = qemuImg.Resize(int(sizeMb))
	if err != nil {
		return nil, errors.Wrap(err, "qemuImg resize")
	}

	resizeFsInfo := &deployapi.DiskInfo{
		Path: d.GetPath(),
	}
	if err := d.ResizeFs(resizeFsInfo); err != nil {
		log.Errorf("Resize fs %s fail %s", d.GetPath(), err)
	}
	return d.GetDiskDesc(), nil
}

func (d *SCLVMDisk) PrepareSaveToGlance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	if err := d.Probe(); err != nil {
		return nil, err
	}
	destDir := d.Storage.GetImgsaveBackupPath()
	if err := procutils.NewCommand("mkdir", "-p", destDir).Run(); err != nil {
		log.Errorln(err)
		return nil, err
	}
	freeSizeMb, err := storageutils.GetFreeSizeMb(destDir)
	if err != nil {
		return nil, errors.Wrap(err, "lvm storageutils.GetFreeSizeMb")
	}
	qemuImg, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		return nil, errors.Wrap(err, "lvm qemuimg.NewQemuImage")
	}
	if int(qemuImg.SizeBytes/1024/1024) >= freeSizeMb*4/5 {
		return nil, errors.Errorf("image cache dir free size is not enough")
	}

	backupPath := path.Join(destDir, fmt.Sprintf("%s.%s", d.Id, appctx.AppContextTaskId(ctx)))
	srcInfo := qemuimg.SImageInfo{
		Path:     d.GetPath(),
		Format:   qemuImg.Format,
		IoLevel:  qemuimg.IONiceNone,
		Password: "",
	}
	destInfo := qemuimg.SImageInfo{
		Path:     backupPath,
		Format:   qemuimg.QCOW2,
		IoLevel:  qemuimg.IONiceNone,
		Password: "",
	}
	if err = qemuimg.Convert(srcInfo, destInfo, true, nil); err != nil {
		log.Errorln(err)
		procutils.NewCommand("rm", "-f", backupPath).Run()
		return nil, err
	}
	res := jsonutils.NewDict()
	res.Set("backup", jsonutils.NewString(backupPath))
	return res, nil
}

func (d *SCLVMDisk) IsFile() bool {
	return false
}
