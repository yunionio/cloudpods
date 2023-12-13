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

type SLVMDisk struct {
	SBaseDisk
}

func (d *SLVMDisk) GetSnapshotDir() string {
	return ""
}

func (d *SLVMDisk) GetType() string {
	return api.STORAGE_LVM
}

// /dev/<vg>/<lvm>
func (d *SLVMDisk) GetLvPath() string {
	return path.Join("/dev", d.Storage.GetPath(), d.Id)
}

func (d *SLVMDisk) GetDevMapperDiskId() string {
	return "dm_" + d.Id
}

func (d *SLVMDisk) GetDevMapperPath() string {
	return path.Join("/dev/mapper", "dm_"+d.Id)
}

func (d *SLVMDisk) GetExtendDiskId() string {
	return "ex_" + d.Id
}

func (d *SLVMDisk) GetSysDiskExtendPath() string {
	return path.Join("/dev", d.Storage.GetPath(), d.GetExtendDiskId())
}

func (d *SLVMDisk) GetPath() string {
	var diskPath = d.GetLvPath()
	if fileutils2.Exists(d.GetDevMapperPath()) {
		diskPath = d.GetDevMapperPath()
	}
	return diskPath
}

func (d *SLVMDisk) GetDiskSetupScripts(idx int) string {
	return fmt.Sprintf("DISK_%d='%s'\n", idx, d.GetPath())
}

func (d *SLVMDisk) GetDiskDesc() jsonutils.JSONObject {
	qemuImg, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		log.Errorln(err)
		return nil
	}

	var desc = jsonutils.NewDict()
	desc.Set("disk_id", jsonutils.NewString(d.Id))
	desc.Set("disk_size", jsonutils.NewInt(qemuImg.SizeBytes/1024/1024))
	desc.Set("format", jsonutils.NewString(string(qemuimg.RAW)))
	desc.Set("disk_path", jsonutils.NewString(d.Storage.GetPath()))
	return desc
}

func (d *SLVMDisk) CleanUpDisk() error {
	// device mapper /dev/mapper/dm_<disk_id>
	if fileutils2.Exists(d.GetDevMapperPath()) {
		if err := lvmutils.DmRemove(d.GetDevMapperPath()); err != nil {
			return err
		}
	}
	// sys disk extend lv /dev/<vg>/ex_<disk_id>
	if fileutils2.Exists(d.GetSysDiskExtendPath()) {
		if err := lvmutils.LvRemove(d.GetSysDiskExtendPath()); err != nil {
			return err
		}
	}
	// disk path /dev/<vg>/<disk_id>
	if fileutils2.Exists(d.GetLvPath()) {
		if err := lvmutils.LvRemove(d.GetLvPath()); err != nil {
			return err
		}
	}
	return nil
}

func (d *SLVMDisk) CreateRaw(
	ctx context.Context, sizeMb int, diskFormat string, fsFormat string,
	encryptInfo *apis.SEncryptInfo, diskId string, back string,
) (jsonutils.JSONObject, error) {
	if fileutils2.Exists(d.GetPath()) {
		if err := d.CleanUpDisk(); err != nil {
			return nil, errors.Wrap(err, "failed remove exists lvm")
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

func (d *SLVMDisk) Delete(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	if err := d.CleanUpDisk(); err != nil {
		return nil, errors.Wrap(err, "Delete")
	}
	d.Storage.RemoveDisk(d)
	return nil, nil
}

func (d *SLVMDisk) PostCreateFromImageFuse() {
}

func (d *SLVMDisk) IsFile() bool {
	return false
}

func (d *SLVMDisk) Probe() error {
	if !fileutils2.Exists(d.GetPath()) {
		return errors.Wrapf(cloudprovider.ErrNotFound, "%s", d.GetPath())
	}
	return nil
}

func (d *SLVMDisk) Resize(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	diskInfo, ok := params.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}
	sizeMb, _ := diskInfo.Int("size")
	if err := d.resize(sizeMb * 1024 * 1024); err != nil {
		return nil, err
	}

	resizeFsInfo := &deployapi.DiskInfo{
		Path: d.GetPath(),
	}
	if err := d.ResizeFs(resizeFsInfo); err != nil {
		log.Errorf("Resize fs %s fail %s", d.GetPath(), err)
	}
	return d.GetDiskDesc(), nil
}

func (d *SLVMDisk) resize(newSize int64) error {
	qemuImg, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		return errors.Wrapf(err, "Open image %s", d.GetPath())
	}
	if qemuImg.SizeBytes >= newSize {
		return nil
	}

	resizePath := d.GetLvPath()
	origin, err := lvmutils.GetLvOrigin(resizePath)
	if err != nil {
		return errors.Wrap(err, "get lv origin")
	}

	if origin != "" {
		// lv created from snapshot
		if !fileutils2.Exists(d.GetDevMapperPath()) {
			// create an lvm extend disk directly.
			err = lvmutils.LvCreate(d.Storage.GetPath(), d.GetExtendDiskId(), newSize-qemuImg.SizeBytes)
			if err != nil {
				return errors.Wrap(err, "lv create")
			}
			// create a device mapper disk
			err = lvmutils.DmCreate(d.GetLvPath(), d.GetSysDiskExtendPath(), d.GetDevMapperDiskId())
			if err != nil {
				if errT := lvmutils.LvRemove(d.GetSysDiskExtendPath()); errT != nil {
					log.Errorf("failed remove extend disk path %s", errT)
				}
				return errors.Wrap(err, "dm create")
			}
		} else {
			// resize extend disk
			resizePath = d.GetSysDiskExtendPath()
			extendImg, err := qemuimg.NewQemuImage(resizePath)
			if err != nil {
				return errors.Wrapf(err, "Open image %s", resizePath)
			}

			newSize = newSize - qemuImg.SizeBytes + extendImg.SizeBytes
			err = lvmutils.LvResize(d.Storage.GetPath(), resizePath, newSize)
			if err != nil {
				return errors.Wrap(err, "lv resize")
			}
		}
	} else {
		err = lvmutils.LvResize(d.Storage.GetPath(), resizePath, newSize)
		if err != nil {
			return errors.Wrap(err, "lv resize")
		}
	}
	return nil
}

func (d *SLVMDisk) CreateFromTemplate(
	ctx context.Context, imageId, format string, size int64, encryptInfo *apis.SEncryptInfo,
) (jsonutils.JSONObject, error) {
	if fileutils2.Exists(d.GetPath()) {
		if err := d.CleanUpDisk(); err != nil {
			return nil, errors.Wrap(err, "failed remove exists lvm")
		}
	}

	var imageCacheManager = storageManager.GetStoragecacheById(d.Storage.GetStoragecacheId())
	ret, err := d.createFromTemplate(ctx, imageId, format, size, imageCacheManager, encryptInfo)
	if err != nil {
		return nil, err
	}
	retSize, _ := ret.Int("disk_size")
	log.Infof("REQSIZE: %d, RETSIZE: %d", size, retSize)
	if size > retSize {
		params := jsonutils.NewDict()
		params.Set("size", jsonutils.NewInt(size))
		if encryptInfo != nil {
			params.Set("encrypt_info", jsonutils.Marshal(encryptInfo))
		}
		return d.Resize(ctx, params)
	}
	return ret, nil
}

func (d *SLVMDisk) PrepareSaveToGlance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
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
		Format:   qemuimg.RAW,
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

func (d *SLVMDisk) createFromTemplate(
	ctx context.Context, imageId, format string, sizeMb int64, imageCacheManager IImageCacheManger, encryptInfo *apis.SEncryptInfo,
) (jsonutils.JSONObject, error) {
	input := api.CacheImageInput{ImageId: imageId, Zone: d.GetZoneId()}
	imageCache, err := imageCacheManager.AcquireImage(ctx, input, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "AcquireImage")
	}

	defer imageCacheManager.ReleaseImage(ctx, imageId)
	cacheImagePath := imageCache.GetPath()
	cacheImage, err := qemuimg.NewQemuImage(cacheImagePath)
	if err != nil {
		return nil, errors.Wrapf(err, "NewQemuImage(%s)", cacheImagePath)
	}

	if sizeMb*1024*1024 > cacheImage.SizeBytes {
		if err = lvmutils.LvCreateFromSnapshot(d.GetLvPath(), cacheImagePath, cacheImage.SizeBytes); err != nil {
			return nil, errors.Wrap(err, "lv create from snapshot")
		}
	} else {
		if err = lvmutils.LvCreate(d.Storage.GetPath(), d.Id, cacheImage.SizeBytes); err != nil {
			return nil, errors.Wrap(err, "lv create")
		}
	}

	return d.GetDiskDesc(), nil
}

func NewLVMDisk(storage IStorage, id string) *SLVMDisk {
	return &SLVMDisk{
		SBaseDisk: *NewBaseDisk(storage, id),
	}
}
