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
	"os"
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
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman/remotefile"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	identity_modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/fuseutils"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

var _ALTER_SUFFIX_ = ".alter"

type SLocalDisk struct {
	SBaseDisk
	isAlter bool
}

func NewLocalDisk(storage IStorage, id string) *SLocalDisk {
	var ret = new(SLocalDisk)
	ret.SBaseDisk = *NewBaseDisk(storage, id)
	return ret
}

func (d *SBaseDisk) GetType() string {
	return api.STORAGE_LOCAL
}

func (d *SLocalDisk) getPath() string {
	return path.Join(d.Storage.GetPath(), d.Id)
}

func (d *SLocalDisk) getAlterPath() string {
	return path.Join(d.Storage.GetPath(), d.Id, _ALTER_SUFFIX_)
}

func (d *SLocalDisk) GetPath() string {
	if d.isAlter {
		return d.getAlterPath()
	} else {
		return d.getPath()
	}
}

func (d *SLocalDisk) GetFormat() (string, error) {
	disk, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		return "", err
	}
	return string(disk.Format), nil
}

func (d *SLocalDisk) GetSnapshotDir() string {
	return path.Join(d.Storage.GetSnapshotDir(), d.Id+options.HostOptions.SnapshotDirSuffix)
}

func (d *SLocalDisk) GetBackupDir() string {
	return d.Storage.GetBackupDir()
}

func (d *SLocalDisk) GetSnapshotLocation() string {
	return d.GetSnapshotDir()
}

func (d *SLocalDisk) Probe() error {
	if fileutils2.Exists(d.getPath()) {
		d.isAlter = false
		return nil
	}
	if fileutils2.Exists(d.getAlterPath()) {
		d.isAlter = true
		return nil
	}
	return errors.Wrapf(cloudprovider.ErrNotFound, "%s", d.getPath())
}

func (d *SLocalDisk) UmountFuseImage() {
	mntPath := path.Join(d.Storage.GetFuseMountPath(), d.Id)
	procutils.NewCommand("umount", mntPath).Run()
	procutils.NewCommand("rm", "-rf", mntPath).Run()
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

func (d *SLocalDisk) Delete(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	p := params.(api.DiskDeleteInput)
	dpath := d.GetPath()
	log.Infof("Delete guest disk %s", dpath)
	if err := d.Storage.DeleteDiskfile(dpath, p.SkipRecycle != nil && *p.SkipRecycle); err != nil {
		return nil, err
	}
	d.UmountFuseImage()
	if p.EsxiFlatFilePath != "" {
		connections := &deployapi.EsxiDisksConnectionInfo{Disks: []*deployapi.EsxiDiskInfo{{DiskPath: p.EsxiFlatFilePath}}}
		_, err := deployclient.GetDeployClient().DisconnectEsxiDisks(ctx, connections)
		if err != nil {
			log.Errorf("Disconnect %s esxi disks failed %s", p.EsxiFlatFilePath, err)
			return nil, err
		}
	}

	d.Storage.RemoveDisk(d)
	return nil, nil
}

func (d *SLocalDisk) OnRebuildRoot(ctx context.Context, params api.DiskAllocateInput) error {
	_, err := d.Delete(ctx, api.DiskDeleteInput{})
	return err
}

func (d *SLocalDisk) Resize(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	diskInfo, ok := params.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}

	sizeMb, _ := diskInfo.Int("size")
	disk, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		log.Errorf("qemuimg.NewQemuImage %s fail: %s", d.GetPath(), err)
		return nil, err
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
			disk.SetPassword(encryptInfo.Key)
			resizeFsInfo.EncryptPassword = encryptInfo.Key
			resizeFsInfo.EncryptAlg = string(encryptInfo.Alg)
		}
	}
	if disk.SizeBytes/1024/1024 < sizeMb {
		if err := disk.Resize(int(sizeMb)); err != nil {
			return nil, err
		}
	}
	if options.HostOptions.EnableFallocateDisk {
		err := d.fallocate()
		if err != nil {
			log.Errorf("fallocate fail %s", err)
		}
	}

	if err := d.ResizeFs(resizeFsInfo); err != nil {
		log.Errorf("Resize fs %s fail %s", d.GetPath(), err)
		// return nil, errors.Wrapf(err, "resize fs %s", d.GetPath())
	}

	return d.GetDiskDesc(), nil
}

func (d *SLocalDisk) CreateFromImageFuse(ctx context.Context, url string, size int64, encryptInfo *apis.SEncryptInfo) error {
	log.Infof("Create from image fuse %s", url)

	localPath := d.Storage.GetFuseTmpPath()
	mntPath := path.Join(d.Storage.GetFuseMountPath(), d.Id)
	contentPath := path.Join(mntPath, "content")
	newImg, err := qemuimg.NewQemuImage(d.getPath())

	if err != nil {
		log.Errorf("qemuimg.NewQemuImage %s fail: %s", d.getPath(), err)
		return err
	}

	if newImg.IsValid() && newImg.IsChained() && newImg.BackFilePath != contentPath {
		if err := newImg.Delete(); err != nil {
			log.Errorln(err)
			return err
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

func (d *SLocalDisk) CreateFromTemplate(ctx context.Context, imageId, format string, size int64, encryptInfo *apis.SEncryptInfo) (jsonutils.JSONObject, error) {
	imageCacheManager := storageManager.LocalStorageImagecacheManager
	return d.createFromTemplateAndResize(ctx, imageId, format, imageCacheManager, encryptInfo, size)
}

func (d *SLocalDisk) createFromTemplateAndResize(
	ctx context.Context, imageId, format string, imageCacheManager IImageCacheManger, encryptInfo *apis.SEncryptInfo, size int64,
) (jsonutils.JSONObject, error) {
	ret, err := d.createFromTemplate(ctx, imageId, format, imageCacheManager, encryptInfo)
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

func (d *SLocalDisk) createFromTemplate(
	ctx context.Context, imageId, format string, imageCacheManager IImageCacheManger, encryptInfo *apis.SEncryptInfo,
) (jsonutils.JSONObject, error) {
	input := api.CacheImageInput{
		ImageId: imageId,
		Zone:    d.GetZoneId(),
	}
	imageCache, err := imageCacheManager.AcquireImage(ctx, input, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "AcquireImage")
	}

	defer imageCacheManager.ReleaseImage(ctx, imageId)
	cacheImagePath := imageCache.GetPath()

	if fileutils2.Exists(d.GetPath()) {
		err := os.Remove(d.GetPath())
		if err != nil {
			return nil, errors.Wrapf(err, "os.Remove(%s)", d.GetPath())
		}
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

func (d *SLocalDisk) CreateFromUrl(ctx context.Context, url string, size int64, callback func(progress, progressMbps float64, totalSizeMb int64)) error {
	remoteFile := remotefile.NewRemoteFile(ctx, url, d.getPath(), false, "", -1, nil, "", "")
	err := remoteFile.Fetch(callback)
	if err != nil {
		return errors.Wrapf(err, "fetch image from %s", url)
	}
	if options.HostOptions.EnableFallocateDisk {
		err := d.fallocate()
		if err != nil {
			log.Errorf("fallocate fail %s", err)
		}
	}
	return nil
}

func (d *SLocalDisk) CreateRaw(ctx context.Context, sizeMB int, diskFormat, fsFormat string,
	encryptInfo *apis.SEncryptInfo, uuid string, back string) (jsonutils.JSONObject, error) {
	if fileutils2.Exists(d.GetPath()) {
		os.Remove(d.GetPath())
	}

	img, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		log.Errorln(err)
		return nil, err
	}

	switch diskFormat {
	case "qcow2":
		if encryptInfo != nil {
			err = img.CreateQcow2(sizeMB, false, back, encryptInfo.Key, qemuimg.EncryptFormatLuks, encryptInfo.Alg)
		} else {
			err = img.CreateQcow2(sizeMB, false, back, "", "", "")
		}
	case "vmdk":
		err = img.CreateVmdk(sizeMB, false)
	default:
		err = img.CreateRaw(sizeMB)
	}

	if err != nil {
		return nil, fmt.Errorf("create_raw: Fail to create disk: %s", err)
	}

	if options.HostOptions.EnableFallocateDisk {
		err := d.fallocate()
		if err != nil {
			log.Errorf("fallocate fail %s", err)
		}
	}

	diskInfo := &deployapi.DiskInfo{
		Path: d.GetPath(),
	}
	if encryptInfo != nil {
		diskInfo.EncryptPassword = encryptInfo.Key
		diskInfo.EncryptAlg = string(encryptInfo.Alg)
	}
	if utils.IsInStringArray(fsFormat, []string{"swap", "ext2", "ext3", "ext4", "xfs"}) {
		d.FormatFs(fsFormat, uuid, diskInfo)
	}

	return d.GetDiskDesc(), nil
}

func (d *SLocalDisk) GetDiskDesc() jsonutils.JSONObject {
	qemuImg, err := qemuimg.NewQemuImage(d.getPath())
	if err != nil {
		log.Errorln(err)
		return nil
	}

	var desc = jsonutils.NewDict()
	desc.Set("disk_id", jsonutils.NewString(d.Id))
	desc.Set("disk_size", jsonutils.NewInt(qemuImg.SizeBytes/1024/1024))
	desc.Set("format", jsonutils.NewString(qemuImg.Format.String()))
	desc.Set("disk_path", jsonutils.NewString(d.getPath()))
	return desc
}

func (d *SLocalDisk) GetDiskSetupScripts(diskIndex int) string {
	cmd := ""
	cmd += fmt.Sprintf("DISK_%d=%s\n", diskIndex, d.getPath())
	cmd += fmt.Sprintf("if [ ! -f $DISK_%d ]; then\n", diskIndex)
	cmd += fmt.Sprintf("    DISK_%d=$DISK_%d%s\n", diskIndex, diskIndex, _ALTER_SUFFIX_)
	cmd += "fi\n"
	return cmd
}

func (d *SLocalDisk) PostCreateFromImageFuse() {
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

func (d *SLocalDisk) DiskBackup(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	diskBackup := params.(*SDiskBakcup)

	encKey := ""
	if len(diskBackup.EncryptKeyId) > 0 {
		session := auth.GetSession(ctx, diskBackup.UserCred, consts.GetRegion())
		secKey, err := identity_modules.Credentials.GetEncryptKey(session, diskBackup.EncryptKeyId)
		if err != nil {
			return nil, errors.Wrap(err, "GetEncryptKey")
		}
		encKey = secKey.Key
	}

	snapshotDir := d.GetSnapshotDir()
	snapshotPath := path.Join(snapshotDir, diskBackup.SnapshotId)
	backupDir := d.GetBackupDir()
	if !fileutils2.Exists(backupDir) {
		output, err := procutils.NewCommand("mkdir", "-p", backupDir).Output()
		if err != nil {
			log.Errorf("mkdir %s failed: %s", backupDir, output)
			return nil, errors.Wrapf(err, "mkdir %s failed: %s", backupDir, output)
		}
	}
	backupPath := path.Join(backupDir, diskBackup.BackupId)
	img, err := qemuimg.NewQemuImage(snapshotPath)
	if err != nil {
		log.Errorln(err)
		procutils.NewCommand("mv", "-f", backupPath, d.getPath()).Run()
		return nil, err
	}
	if len(encKey) > 0 {
		img.SetPassword(encKey)
	}
	newImage, err := img.Clone(backupPath, qemuimgfmt.QCOW2, true)
	if err != nil {
		return nil, errors.Wrap(err, "unable to backup snapshot")
	}
	_, err = d.Storage.StorageBackup(ctx, &SStorageBackup{
		BackupId:                diskBackup.BackupId,
		BackupStorageId:         diskBackup.BackupStorageId,
		BackupStorageAccessInfo: diskBackup.BackupStorageAccessInfo,
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to SStorageBackup")
	}
	data := jsonutils.NewDict()
	data.Set("size_mb", jsonutils.NewInt(int64(newImage.GetActualSizeMB())))
	return data, nil
}

func (d *SLocalDisk) CreateSnapshot(snapshotId string, encryptKey string, encFormat qemuimg.TEncryptFormat, encAlg seclib2.TSymEncAlg) error {
	snapshotDir := d.GetSnapshotDir()
	log.Infof("snapshotDir of LocalDisk %s: %s", d.Id, snapshotDir)
	if !fileutils2.Exists(snapshotDir) {
		output, err := procutils.NewCommand("mkdir", "-p", snapshotDir).Output()
		if err != nil {
			log.Errorf("mkdir %s failed: %s", snapshotDir, output)
			return errors.Wrapf(err, "mkdir %s failed: %s", snapshotDir, output)
		}
	}
	snapshotPath := path.Join(snapshotDir, snapshotId)
	output, err := procutils.NewCommand("mv", "-f", d.getPath(), snapshotPath).Output()
	if err != nil {
		log.Errorf("mv %s to %s failed %s", d.getPath(), snapshotPath, output)
		return errors.Wrapf(err, "mv %s to %s failed %s", d.getPath(), snapshotPath, output)
	}
	img, err := qemuimg.NewQemuImage(d.getPath())
	if err != nil {
		log.Errorln(err)
		procutils.NewCommand("mv", "-f", snapshotPath, d.getPath()).Run()
		return err
	}
	if err := img.CreateQcow2(0, false, snapshotPath, encryptKey, encFormat, encAlg); err != nil {
		log.Errorf("Snapshot create image error %s", err)
		procutils.NewCommand("mv", "-f", snapshotPath, d.getPath()).Run()
		return err
	}
	return nil
}

func (d *SLocalDisk) ConvertSnapshot(convertSnapshotId string) error {
	snapshotDir := d.GetSnapshotDir()
	snapshotPath := path.Join(snapshotDir, convertSnapshotId)
	img, err := qemuimg.NewQemuImage(snapshotPath)
	if err != nil {
		log.Errorln(err)
		return err
	}
	convertedDisk := snapshotPath + ".tmp"
	if err = img.Convert2Qcow2To(convertedDisk, false, "", "", ""); err != nil {
		log.Errorln(err)
		if fileutils2.Exists(convertedDisk) {
			os.Remove(convertedDisk)
		}
		return err
	}
	if output, err := procutils.NewCommand("mv", "-f", convertedDisk, snapshotPath).Output(); err != nil {
		log.Errorf("mv %s to %s failed: %s, %s", convertedDisk, snapshotPath, err, output)
		return err
	}
	return nil
}

func (d *SLocalDisk) DeleteSnapshot(snapshotId, convertSnapshot string, blockStream bool) error {
	snapshotDir := d.GetSnapshotDir()
	return DeleteLocalSnapshot(snapshotDir, snapshotId, d.getPath(), convertSnapshot, blockStream)
}

func (d *SLocalDisk) PrepareSaveToGlance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	if err := d.Probe(); err != nil {
		return nil, err
	}
	destDir := d.Storage.GetImgsaveBackupPath()
	if err := procutils.NewCommand("mkdir", "-p", destDir).Run(); err != nil {
		log.Errorln(err)
		return nil, err
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

func (d *SLocalDisk) ResetFromSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	resetParams, ok := params.(*SDiskReset)
	if !ok {
		return nil, hostutils.ParamsError
	}

	outOfChain, err := resetParams.Input.Bool("out_of_chain")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("out_of_chain")
	}

	snapshotDir := d.GetSnapshotDir()
	snapshotPath := path.Join(snapshotDir, resetParams.SnapshotId)

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

	return d.resetFromSnapshot(snapshotPath, outOfChain, encryptInfo)
}

func (d *SLocalDisk) resetFromSnapshot(snapshotPath string, outOfChain bool, encryptInfo *apis.SEncryptInfo) (jsonutils.JSONObject, error) {
	img, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		return nil, err
	}
	diskSizeMB := int(img.SizeBytes / 1024 / 1024)

	diskTmpPath := d.GetPath() + "_reset.tmp"
	if output, err := procutils.NewCommand("mv", "-f", d.GetPath(), diskTmpPath).Output(); err != nil {
		err = errors.Wrapf(err, "mv disk to tmp failed: %s", output)
		return nil, err
	}
	if !outOfChain {
		img, err := qemuimg.NewQemuImage(d.GetPath())
		if err != nil {
			err = errors.Wrap(err, "new qemu img")
			procutils.NewCommand("mv", "-f", diskTmpPath, d.GetPath()).Run()
			return nil, err
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
		if err := img.CreateQcow2(diskSizeMB, false, snapshotPath, encKey, encFmt, encAlg); err != nil {
			err = errors.Wrap(err, "qemu-img create disk by snapshot")
			procutils.NewCommand("mv", "-f", diskTmpPath, d.GetPath()).Run()
			return nil, err
		}
	} else {
		if output, err := procutils.NewCommand("cp", "-f", snapshotPath, d.GetPath()).Output(); err != nil {
			err = errors.Wrapf(err, "cp snapshot to disk %s", output)
			procutils.NewCommand("mv", "-f", diskTmpPath, d.GetPath()).Run()
			return nil, err
		}
	}

	output, err := procutils.NewCommand("rm", "-f", diskTmpPath).Output()
	if err != nil {
		err = errors.Wrapf(err, "rm disk tmp path %s", output)
		return nil, err
	}
	return nil, nil
}

func (d *SLocalDisk) CleanupSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	cleanupParams, ok := params.(*SDiskCleanupSnapshots)
	if !ok {
		return nil, hostutils.ParamsError
	}
	snapshotDir := d.GetSnapshotDir()
	for _, snapshotId := range cleanupParams.ConvertSnapshots {
		snapId, _ := snapshotId.GetString()
		snapshotPath := path.Join(snapshotDir, snapId)
		output := snapshotPath + "_convert.tmp"
		img, err := qemuimg.NewQemuImage(snapshotPath)
		if err != nil {
			log.Errorln(err)
			return nil, err
		}
		if err = img.Convert2Qcow2To(output, true, "", "", ""); err != nil {
			log.Errorln(err)
			return nil, err
		}
		if err := procutils.NewCommand("mv", "-f", output, snapshotPath).Run(); err != nil {
			procutils.NewCommand("rm", "-f", output).Run()
			log.Errorln(err)
			return nil, err
		}
	}

	for _, snapshotId := range cleanupParams.DeleteSnapshots {
		snapId, _ := snapshotId.GetString()
		if err := procutils.NewCommand("rm", "-f", path.Join(snapshotDir, snapId)).Run(); err != nil {
			log.Errorln(err)
			return nil, err
		}
	}
	return nil, nil
}

func (d *SLocalDisk) DeleteAllSnapshot(skipRecycle bool) error {
	snapshotDir := d.GetSnapshotDir()
	if !fileutils2.Exists(snapshotDir) {
		return nil
	}
	if options.HostOptions.RecycleDiskfile {
		return d.Storage.DeleteDiskfile(snapshotDir, skipRecycle)
	} else {
		log.Infof("Delete disk(%s) snapshot dir %s", d.Id, snapshotDir)
		return procutils.NewCommand("rm", "-rf", snapshotDir).Run()
	}
}

func (d *SLocalDisk) PrepareMigrate(liveMigrate bool) ([]string, string, bool, error) {
	disk, err := qemuimg.NewQemuImage(d.getPath())
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
			} else if !strings.HasPrefix(backingChain[i], options.HostOptions.ImageCachePath) {
				return nil, "", false, errors.Errorf("backing file path %s unsupported", backingChain[i])
			}
		}
		hasTemplate := strings.HasPrefix(backingChain[len(backingChain)-1], options.HostOptions.ImageCachePath)
		return snapshots, backingChain[0], hasTemplate, nil
	}
	return nil, "", false, nil
}

func (d *SLocalDisk) DoDeleteSnapshot(snapshotId string) error {
	snapshotPath := path.Join(d.GetSnapshotDir(), snapshotId)
	return d.Storage.DeleteDiskfile(snapshotPath, false)
}

func (d *SLocalDisk) IsFile() bool {
	return true
}

func (d *SLocalDisk) RebuildSlaveDisk(diskUri string) error {
	diskPath := d.getPath()
	if output, err := procutils.NewCommand("rm", "-f", diskPath).Output(); err != nil {
		return errors.Errorf("failed delete slave top disk file %s %s", output, err)
	}
	diskUrl := fmt.Sprintf("%s/%s", diskUri, d.Id)
	if err := d.CreateFromImageFuse(context.Background(), diskUrl, 0, nil); err != nil {
		return errors.Wrap(err, "failed create slave disk")
	}
	return nil
}

func (d *SLocalDisk) fallocate() error {
	img, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		return errors.Wrap(err, "NewQemuImage")
	}
	err = img.Fallocate()
	if err != nil {
		return errors.Wrap(err, "Fallocate")
	}
	return nil
}
