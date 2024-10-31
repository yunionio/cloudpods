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
	"io/ioutil"
	"path"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/qemuimgfmt"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman/lvmutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman/storageutils"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	identity_modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/fuseutils"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

var _ IDisk = (*SLVMDisk)(nil)

type SLVMDisk struct {
	SBaseDisk
}

func (d *SLVMDisk) GetSnapshotDir() string {
	return d.GetSnapshotPrefix()
}

func (d *SLVMDisk) GetSnapshotPrefix() string {
	return path.Join("/dev", d.Storage.GetPath(), "snap_")
}

func (d *SLVMDisk) GetImageCachePrefix() string {
	return path.Join("/dev", d.Storage.GetPath(), "imagecache_")
}

func (d *SLVMDisk) GetType() string {
	return api.STORAGE_LVM
}

// /dev/<vg>/<lvm>
func (d *SLVMDisk) GetLvPath() string {
	return path.Join("/dev", d.Storage.GetPath(), d.Id)
}

func (d *SLVMDisk) GetPath() string {
	return path.Join("/dev", d.Storage.GetPath(), d.Id)
}

// The LVM logical volume name is limited to 64 characters.
func (d *SLVMDisk) GetSnapshotName(snapshotId string) string {
	return "snap_" + snapshotId
}

func (d *SLVMDisk) GetSnapshotPath(snapshotId string) string {
	return path.Join("/dev", d.Storage.GetPath(), d.GetSnapshotName(snapshotId))
}

func (d *SLVMDisk) GetDiskSetupScripts(idx int) string {
	return fmt.Sprintf("DISK_%d='%s'\n", idx, d.GetPath())
}

func (d *SLVMDisk) GetDiskDesc() jsonutils.JSONObject {
	qemuImg, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		log.Errorf("qemuimg.NewQemuImage %s: %s", d.GetPath(), err)
		return nil
	}

	var desc = jsonutils.NewDict()
	desc.Set("disk_id", jsonutils.NewString(d.Id))
	desc.Set("disk_size", jsonutils.NewInt(qemuImg.SizeBytes/1024/1024))
	desc.Set("format", jsonutils.NewString(string(qemuImg.Format)))
	desc.Set("disk_path", jsonutils.NewString(d.GetPath()))
	return desc
}

func (d *SLVMDisk) CreateRaw(ctx context.Context, sizeMB int, diskFormat string, fsFormat string, fsFeatures *api.DiskFsFeatures, encryptInfo *apis.SEncryptInfo, diskId string, back string) (jsonutils.JSONObject, error) {
	if fileutils2.Exists(d.GetPath()) {
		if err := lvmutils.LvRemove(d.GetLvPath()); err != nil {
			return nil, errors.Wrap(err, "CreateRaw lvremove")
		}
	}

	img, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		log.Errorln(err)
		return nil, err
	}

	qcow2Size := lvmutils.GetQcow2LvSize(int64(sizeMB))
	err = lvmutils.LvCreate(d.Storage.GetPath(), d.Id, qcow2Size*1024*1024)
	if err != nil {
		return nil, errors.Wrap(err, "CreateRaw")
	}

	if encryptInfo != nil {
		err = img.CreateQcow2(sizeMB, false, back, encryptInfo.Key, qemuimg.EncryptFormatLuks, encryptInfo.Alg)
	} else {
		err = img.CreateQcow2(sizeMB, false, back, "", "", "")
	}

	if err != nil {
		return nil, fmt.Errorf("create_raw: Fail to create disk: %s", err)
	}

	diskInfo := &deployapi.DiskInfo{
		Path: d.GetPath(),
	}
	if encryptInfo != nil {
		diskInfo.EncryptPassword = encryptInfo.Key
		diskInfo.EncryptAlg = string(encryptInfo.Alg)
	}
	if utils.IsInStringArray(fsFormat, []string{"swap", "ext2", "ext3", "ext4", "xfs"}) {
		d.FormatFs(fsFormat, nil, diskId, diskInfo)
	}
	return d.GetDiskDesc(), nil
}

func (d *SLVMDisk) Delete(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	if err := lvmutils.LvRemove(d.GetLvPath()); err != nil {
		return nil, errors.Wrap(err, "Delete lvremove")
	}
	d.Storage.RemoveDisk(d)
	return nil, nil
}

func (d *SLVMDisk) PostCreateFromImageFuse() {
	mntPath := path.Join(d.Storage.GetFuseMountPath(), d.Id)
	if output, err := procutils.NewCommand("umount", mntPath).Output(); err != nil {
		log.Errorf("umount %s failed: %s, %s", mntPath, err, output)
	}
	if output, err := procutils.NewCommand("rm", "-rf", mntPath).Output(); err != nil {
		log.Errorf("rm %s failed: %s, %s", mntPath, err, output)
	}
	tmpPath := d.Storage.GetFuseTmpPath()
	tmpFiles, err := ioutil.ReadDir(tmpPath)
	if err != nil {
		for _, f := range tmpFiles {
			if strings.HasPrefix(f.Name(), d.Id) {
				procutils.NewCommand("rm", "-f", path.Join(tmpPath, f.Name()))
			}
		}
	}
}

func (d *SLVMDisk) CreateFromImageFuse(ctx context.Context, url string, size int64, encryptInfo *apis.SEncryptInfo) error {
	log.Infof("Create from image fuse %s", url)

	localPath := d.Storage.GetFuseTmpPath()
	mntPath := path.Join(d.Storage.GetFuseMountPath(), d.Id)
	contentPath := path.Join(mntPath, "content")
	newImg, err := qemuimg.NewQemuImage(d.GetPath())

	if err != nil {
		log.Errorf("qemuimg.NewQemuImage %s fail: %s", d.GetPath(), err)
		return err
	}

	if newImg.IsValid() && newImg.IsChained() && newImg.BackFilePath != contentPath {
		if err := lvmutils.LvRemove(d.GetPath()); err != nil {
			return errors.Wrap(err, "remove disk")
		}
	}
	if !newImg.IsValid() || newImg.IsChained() {
		if err := fuseutils.MountFusefs(
			options.HostOptions.FetcherfsPath, url, localPath,
			auth.GetTokenString(), mntPath, options.HostOptions.FetcherfsBlockSize, encryptInfo,
		); err != nil {
			log.Errorln(err)
			return err
		}
	}
	if !newImg.IsValid() {
		lvSize := lvmutils.GetQcow2LvSize(size)
		if err := lvmutils.LvCreate(d.Storage.GetPath(), d.Id, lvSize*1024*1024); err != nil {
			return errors.Wrap(err, "lvcreate")
		}

		if encryptInfo != nil {
			err = newImg.CreateQcow2(0, false, contentPath, encryptInfo.Key, qemuimg.EncryptFormatLuks, encryptInfo.Alg)
		} else {
			err = newImg.CreateQcow2(0, false, contentPath, "", "", "")
		}
		if err != nil {
			return errors.Wrapf(err, "create from fuse")
		}
	}

	return nil
}

func (d *SLVMDisk) IsFile() bool {
	return true
}

func (d *SLVMDisk) Probe() error {
	if !fileutils2.Exists(d.GetPath()) {
		return errors.Wrapf(cloudprovider.ErrNotFound, "%s", d.GetPath())
	}
	return nil
}

func (d *SLVMDisk) OnRebuildRoot(ctx context.Context, params api.DiskAllocateInput) error {
	_, err := d.Delete(ctx, api.DiskDeleteInput{})
	return err
}

func (d *SLVMDisk) PreResize(ctx context.Context, sizeMb int64) error {
	qemuImg, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		return errors.Wrap(err, "lvm qemuimg.NewQemuImage")
	}

	lvsize := sizeMb
	if qemuImg.Format == qemuimgfmt.QCOW2 {
		lvsize = lvmutils.GetQcow2LvSize(sizeMb)
	}

	err = lvmutils.LvResize(d.Storage.GetPath(), d.GetPath(), lvsize*1024*1024)
	if err != nil {
		return errors.Wrap(err, "lv resize")
	}
	return nil
}

func (d *SLVMDisk) Resize(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
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
	if qemuImg.Format == qemuimgfmt.QCOW2 {
		lvsize = lvmutils.GetQcow2LvSize(sizeMb)
	}

	err = lvmutils.LvResize(d.Storage.GetPath(), d.GetPath(), lvsize*1024*1024)
	if err != nil {
		return nil, errors.Wrap(err, "lv resize")
	}

	resizeFsInfo := &deployapi.DiskInfo{
		Path: d.GetPath(),
	}
	if diskInfo.Contains("encrypt_info") {
		var encryptInfo apis.SEncryptInfo
		err := diskInfo.Unmarshal(&encryptInfo, "encrypt_info")
		if err != nil {
			log.Errorf("fail to fetch encryptInfo %s", err)
			return nil, errors.Wrap(err, "Unmarshal encrpt_info")
		} else {
			qemuImg.SetPassword(encryptInfo.Key)
			resizeFsInfo.EncryptPassword = encryptInfo.Key
			resizeFsInfo.EncryptAlg = string(encryptInfo.Alg)
		}
	}

	err = qemuImg.Resize(int(sizeMb))
	if err != nil {
		return nil, errors.Wrap(err, "qemuImg resize")
	}

	if err := d.ResizeFs(resizeFsInfo); err != nil {
		log.Errorf("Resize fs %s fail %s", d.GetPath(), err)
	}
	return d.GetDiskDesc(), nil
}

func (d *SLVMDisk) CreateFromTemplate(
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

	lvSizeMb := lvmutils.GetQcow2LvSize(imageCache.GetDesc().SizeMb)
	if err := lvmutils.LvCreate(d.Storage.GetPath(), d.Id, lvSizeMb*1024*1024); err != nil {
		return nil, errors.Wrap(err, "CreateRaw")
	}
	newImg, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		return nil, errors.Wrapf(err, "NewQemuImage(%s)", d.GetPath())
	}
	if encryptInfo != nil {
		err = newImg.CreateQcow2(0, false, cacheImagePath, encryptInfo.Key, qemuimg.EncryptFormatLuks, encryptInfo.Alg)
	} else {
		err = newImg.CreateQcow2(0, false, cacheImagePath, "", "", "")
	}
	if err != nil {
		return nil, errors.Wrapf(err, "CreateQcow2(%s)", cacheImagePath)
	}

	return d.GetDiskDesc(), nil
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
	if err := procutils.NewCommand("cp", "--sparse=always", "-f", d.GetPath(), backupPath).Run(); err != nil {
		log.Errorln(err)
		procutils.NewCommand("rm", "-f", backupPath).Run()
		return nil, err
	}

	res := jsonutils.NewDict()
	res.Set("backup", jsonutils.NewString(backupPath))
	return res, nil
}

func (d *SLVMDisk) GetBackupName(backupId string) string {
	return "backup_" + backupId
}

func (d *SLVMDisk) GetBackupPath(backupId string) string {
	return path.Join("/dev", d.Storage.GetPath(), d.GetBackupName(backupId))
}

func (d *SLVMDisk) DiskBackup(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	diskBackup := params.(*SDiskBackup)

	var encKey = ""
	var encAlg seclib2.TSymEncAlg
	if len(diskBackup.EncryptKeyId) > 0 {
		session := auth.GetSession(ctx, diskBackup.UserCred, consts.GetRegion())
		secKey, err := identity_modules.Credentials.GetEncryptKey(session, diskBackup.EncryptKeyId)
		if err != nil {
			return nil, errors.Wrap(err, "GetEncryptKey")
		}
		encAlg = secKey.Alg
		encKey = secKey.Key
	}

	snapshotPath := d.GetSnapshotPath(diskBackup.SnapshotId)
	snapshotImg, err := qemuimg.NewQemuImage(snapshotPath)
	if err != nil {
		return nil, errors.Wrap(err, "lvm disk backup snapshotPath NewQemuImage")
	}

	// create backup lv
	lvSizeMb := lvmutils.GetQcow2LvSize(snapshotImg.SizeBytes / 1024 / 1024)
	err = lvmutils.LvCreate(d.Storage.GetPath(), d.GetBackupName(diskBackup.BackupId), lvSizeMb*1024*1024)
	if err != nil {
		return nil, errors.Wrap(err, "lvcreate backup")
	}

	backupPath := d.GetBackupPath(diskBackup.BackupId)
	srcInfo := qemuimg.SImageInfo{
		Path:          snapshotPath,
		Format:        snapshotImg.Format,
		IoLevel:       qemuimg.IONiceNone,
		Password:      encKey,
		EncryptAlg:    encAlg,
		EncryptFormat: qemuimg.EncryptFormatLuks,
	}
	destInfo := qemuimg.SImageInfo{
		Path:          backupPath,
		Format:        qemuimgfmt.QCOW2,
		IoLevel:       qemuimg.IONiceNone,
		Password:      encKey,
		EncryptAlg:    encAlg,
		EncryptFormat: qemuimg.EncryptFormatLuks,
	}
	if err = qemuimg.Convert(srcInfo, destInfo, true, nil); err != nil {
		if errRm := lvmutils.LvRemove(backupPath); errRm != nil {
			log.Errorf("failed delete backup lv %s", errRm)
		}
		return nil, errors.Wrap(err, "failed convert snapshot to backup")
	}

	_, err = d.Storage.StorageBackup(ctx, &SStorageBackup{
		BackupId:                diskBackup.BackupId,
		BackupLocalPath:         backupPath,
		BackupStorageId:         diskBackup.BackupStorageId,
		BackupStorageAccessInfo: diskBackup.BackupStorageAccessInfo,
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to SStorageBackup")
	}
	data := jsonutils.NewDict()
	data.Set("size_mb", jsonutils.NewInt(snapshotImg.SizeBytes/1024/1024))
	return data, nil
}

func (d *SLVMDisk) CreateSnapshot(snapshotId string, encryptKey string, encFormat qemuimg.TEncryptFormat, encAlg seclib2.TSymEncAlg) error {
	snapName := d.GetSnapshotName(snapshotId)
	log.Infof("Start create snapshot %s of lvm Disk %s", snapName, d.Id)
	lvSize, err := lvmutils.GetLvSize(d.GetPath())
	if err != nil {
		return err
	}

	err = lvmutils.LvRename(d.Storage.GetPath(), d.Id, snapName)
	if err != nil {
		return err
	}
	if err := lvmutils.LvCreate(d.Storage.GetPath(), d.Id, lvSize); err != nil {
		if e := lvmutils.LvRename(d.Storage.GetPath(), snapName, d.Id); e != nil {
			log.Errorf("failed rename lv %s to %s: %s", snapName, d.GetPath(), e)
		}
		return errors.Wrap(err, "snapshot LvCreate")
	}
	img, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		if e := lvmutils.LvRemove(d.GetPath()); e != nil {
			log.Errorf("failed remove lv %s: %s", d.GetPath(), e)
		}
		if e := lvmutils.LvRename(d.Storage.GetPath(), snapName, d.Id); e != nil {
			log.Errorf("failed rename lv %s to %s: %s", snapName, d.GetPath(), e)
		}
		return errors.Wrapf(err, "failed qemuimg.NewQemuImage(%s))", d.GetPath())
	}

	snapPath := d.GetSnapshotPath(snapshotId)
	err = img.CreateQcow2(0, false, snapPath, encryptKey, encFormat, encAlg)
	if err != nil {
		if e := lvmutils.LvRemove(d.GetPath()); e != nil {
			log.Errorf("failed remove lv %s: %s", d.GetPath(), e)
		}
		if e := lvmutils.LvRename(d.Storage.GetPath(), snapName, d.Id); e != nil {
			log.Errorf("failed rename lv %s to %s: %s", snapName, d.GetPath(), e)
		}
		return errors.Wrapf(err, "CreateQcow2(%s)", snapPath)
	}
	return nil
}

func (d *SLVMDisk) ResetFromSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	resetParams, ok := params.(*SDiskReset)
	if !ok {
		return nil, hostutils.ParamsError
	}

	img, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		return nil, err
	}
	diskSizeMB := int(img.SizeBytes / 1024 / 1024)

	lvSize, err := lvmutils.GetLvSize(d.GetPath())
	if err != nil {
		return nil, err
	}
	// rename disk to temp logical volume
	tmpVolume := d.Id + "-reset.tmp"
	err = lvmutils.LvRename(d.Storage.GetPath(), d.Id, tmpVolume)
	if err != nil {
		return nil, err
	}
	if err := lvmutils.LvCreate(d.Storage.GetPath(), d.Id, lvSize); err != nil {
		return nil, errors.Wrap(err, "reset snapshot LvCreate")
	}

	imgNew, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		lvmutils.LvRemove(d.GetPath())
		lvmutils.LvRename(d.Storage.GetPath(), tmpVolume, d.Id)
		return nil, errors.Wrapf(err, "failed qemuimg.NewQemuImage(%s))", d.GetPath())
	}

	var encryptInfo *apis.SEncryptInfo
	if resetParams.Input.Contains("encrypt_info") {
		encInfo := apis.SEncryptInfo{}
		err := resetParams.Input.Unmarshal(&encInfo, "encrypt_info")
		if err != nil {
			log.Errorf("unmarshal encrypt_info fail %s", err)
		} else {
			encryptInfo = &encInfo
		}
	}
	var (
		encKey string
		encAlg seclib2.TSymEncAlg
		encFmt qemuimg.TEncryptFormat
	)
	if encryptInfo != nil {
		encKey = encryptInfo.Key
		encFmt = qemuimg.EncryptFormatLuks
		encAlg = encryptInfo.Alg
	}

	snapPath := d.GetSnapshotPath(resetParams.SnapshotId)
	err = imgNew.CreateQcow2(diskSizeMB, false, snapPath, encKey, encFmt, encAlg)
	if err != nil {
		lvmutils.LvRemove(d.GetPath())
		lvmutils.LvRename(d.Storage.GetPath(), tmpVolume, d.Id)
		return nil, errors.Wrapf(err, "CreateQcow2(%s)", snapPath)
	}
	tmpVolumePath := path.Join("/dev", d.Storage.GetPath(), tmpVolume)
	err = lvmutils.LvRemove(tmpVolumePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed remove tmp volume")
	}
	return nil, nil
}

func (d *SLVMDisk) DeleteSnapshot(snapshotId, convertSnapshot string, blockStream bool, encryptInfo apis.SEncryptInfo) error {
	if blockStream {
		if err := ConvertLVMDisk(d.Storage.GetPath(), d.Id, encryptInfo); err != nil {
			return err
		}
	} else if len(convertSnapshot) > 0 {
		if err := d.ConvertSnapshot(convertSnapshot, encryptInfo); err != nil {
			return err
		}
	}
	return lvmutils.LvRemove(d.GetSnapshotPath(snapshotId))
}

func (d *SLVMDisk) DeleteAllSnapshot(skipRecycle bool) error {
	lvNames, err := lvmutils.GetLvNames(d.Storage.GetPath())
	if err != nil {
		log.Errorf("failed get lvm %s lvs %s", d.Storage.GetPath(), err)
		return nil
	}

	snapPrefix := "snap_" + d.Id
	for _, f := range lvNames {
		if strings.HasPrefix(f, snapPrefix) {
			if err := lvmutils.LvRemove(path.Join("/dev", d.Storage.GetPath(), f)); err != nil {
				return errors.Wrap(err, "delele lvm snapshots")
			}
		}
	}
	return nil
}

func (d *SLVMDisk) ConvertSnapshot(convertSnapshot string, encryptInfo apis.SEncryptInfo) error {
	convertSnapshotName := d.GetSnapshotName(convertSnapshot)
	return ConvertLVMDisk(d.Storage.GetPath(), convertSnapshotName, encryptInfo)
}

func (d *SLVMDisk) DoDeleteSnapshot(snapshotId string) error {
	snapshotPath := d.GetSnapshotPath(snapshotId)
	return lvmutils.LvRemove(snapshotPath)
}

func (d *SLVMDisk) RollbackDiskOnSnapshotFail(snapshotId string) error {
	diskPath := d.GetPath()
	if fileutils2.Exists(diskPath) {
		if err := lvmutils.LvRemove(diskPath); err != nil {
			return errors.Wrap(err, "rollback disk on snapshot fail delete disk")
		}
	}
	snapshotName := d.GetSnapshotName(snapshotId)
	if err := lvmutils.LvRename(d.Storage.GetPath(), snapshotName, d.Id); err != nil {
		return errors.Wrapf(err, "RollbackDiskOnSnapshotFail rename %s to %s failed: %s", snapshotName, d.Id, err)
	}
	return nil
}

func (d *SLVMDisk) PrepareMigrate(liveMigrate bool) ([]string, string, bool, error) {
	disk, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		log.Errorln(err)
		return nil, "", false, err
	}
	ret, err := disk.WholeChainFormatIs("qcow2")
	if err != nil {
		log.Errorln(err)
		return nil, "", false, err
	}
	if liveMigrate && !ret {
		return nil, "", false, fmt.Errorf("Disk format doesn't support live migrate")
	}
	if disk.IsChained() {
		backingChain, err := disk.GetBackingChain()
		if err != nil {
			return nil, "", false, err
		}
		snapshots := []string{}
		for i := range backingChain {
			if strings.HasPrefix(backingChain[i], d.GetSnapshotDir()) {
				snapshots = append(snapshots, path.Base(backingChain[i]))
			} else if !strings.HasPrefix(backingChain[i], d.GetImageCachePrefix()) {
				return nil, "", false, errors.Errorf("backing file path %s unsupported", backingChain[i])
			}
		}
		hasTemplate := strings.HasPrefix(backingChain[len(backingChain)-1], d.GetImageCachePrefix())
		return snapshots, backingChain[0], hasTemplate, nil
	}
	return nil, "", false, nil
}

func (d *SLVMDisk) RebuildSlaveDisk(diskUri string) error {
	if err := lvmutils.LvRemove(d.GetPath()); err != nil {
		return errors.Wrap(err, "lvremove")
	}
	diskUrl := fmt.Sprintf("%s/%s", diskUri, d.Id)
	if err := d.CreateFromImageFuse(context.Background(), diskUrl, 0, nil); err != nil {
		return errors.Wrap(err, "failed create slave disk")
	}
	return nil
}

func NewLVMDisk(storage IStorage, id string) *SLVMDisk {
	return &SLVMDisk{
		SBaseDisk: *NewBaseDisk(storage, id),
	}
}
