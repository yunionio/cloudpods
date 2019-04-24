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
	"os"
	"path"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman/remotefile"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/fuseutils"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
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

func (d *SLocalDisk) GetSnapshotDir() string {
	return path.Join(d.Storage.GetSnapshotDir(), d.Id+options.HostOptions.SnapshotDirSuffix)
}

func (d *SLocalDisk) Probe() error {
	if fileutils2.Exists(d.getPath()) {
		d.isAlter = false
		return nil
	} else if fileutils2.Exists(d.getAlterPath()) {
		d.isAlter = true
		return nil
	}
	return fmt.Errorf("Disk not found")
}

func (d *SLocalDisk) UmountFuseImage() {
	mntPath := path.Join(d.Storage.GetFuseMountPath(), d.Id)
	procutils.NewCommand("umount", mntPath).Run()
	procutils.NewCommand("rm", "-rf", mntPath).Run()
}

func (d *SLocalDisk) Delete(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	dpath := d.GetPath()
	log.Infof("Delete guest disk %s", dpath)
	if err := d.Storage.DeleteDiskfile(dpath); err != nil {
		return nil, err
	}
	d.UmountFuseImage()

	/* ????????????????
	   files = os.listdir(self.storage.path)
	   for f in files:
	       if f.startswith(self.id):
	           if not re.match(r'[a-z0-9\-]*\.\d{14}', f):
	               path = os.path.join(self.storage.path, f)
	               print 'delete backing-file:', path
	               self.storage.delete_diskfile(path)
	*/

	d.Storage.RemoveDisk(d)
	return nil, nil
}

func (d *SLocalDisk) Resize(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	diskInfo, ok := params.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}

	sizeMb, _ := diskInfo.Int("size")
	disk, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	if err := disk.Resize(int(sizeMb)); err != nil {
		return nil, err
	}
	if options.HostOptions.EnableFallocateDisk {
		// TODO
		// d.Fallocate()
	}

	if err = d.ResizeFs(); err != nil {
		return nil, err
	}

	return d.GetDiskDesc(), nil
}

func (d *SLocalDisk) ResizeFs() error {
	disk := NewKVMGuestDisk(d.GetPath())
	if disk.Connect() {
		defer disk.Disconnect()
		if err := disk.ResizePartition(); err != nil {
			return err
		}
	}
	return nil
}

func (d *SLocalDisk) CreateFromImageFuse(ctx context.Context, url string) error {
	log.Infof("Create from image fuse %s", url)

	var (
		localPath   = d.Storage.GetFuseTmpPath()
		mntPath     = path.Join(d.Storage.GetFuseMountPath())
		contentPath = path.Join(mntPath, "content")
		newImg, err = qemuimg.NewQemuImage(d.getPath())
	)

	if err != nil {
		log.Errorln(err)
		return err
	}

	if newImg.IsValid() && newImg.IsChained() && newImg.BackFilePath != contentPath {
		if err := newImg.Delete(); err != nil {
			log.Errorln(err)
			return err
		}
	}
	if !newImg.IsValid() || newImg.IsChained() {
		if err := fuseutils.MountFusefs(options.HostOptions.FetcherfsPath, url, localPath,
			auth.GetTokenString(), mntPath, fuseutils.DEFAULT_BLOCKSIZE); err != nil {
			log.Errorln(err)
			return err
		}
	}
	if !newImg.IsValid() {
		if err := newImg.CreateQcow2(0, false, contentPath); err != nil {
			log.Errorln(err)
			return err
		}
	}

	return nil
}

func (d *SLocalDisk) CreateFromTemplate(ctx context.Context, imageId, format string, size int64) (jsonutils.JSONObject, error) {
	var imageCacheManager = storageManager.LocalStorageImagecacheManager
	ret, err := d.createFromTemplate(ctx, imageId, format, imageCacheManager)
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

func (d *SLocalDisk) createFromTemplate(
	ctx context.Context, imageId, format string, imageCacheManager IImageCacheManger,
) (jsonutils.JSONObject, error) {
	imageCache := imageCacheManager.AcquireImage(ctx, imageId, d.GetZone(), "", "")
	if imageCache != nil {
		defer imageCacheManager.ReleaseImage(imageId)
		cacheImagePath := imageCache.GetPath()

		if fileutils2.Exists(d.GetPath()) {
			err := os.Remove(d.GetPath())
			if err != nil {
				log.Errorln(err)
				return nil, fmt.Errorf("Fail to Create disk %s", d.Id)
			}
		}

		newImg, err := qemuimg.NewQemuImage(d.GetPath())
		if err != nil {
			log.Errorln(err)
			return nil, err
		}
		if err := newImg.CreateQcow2(0, false, cacheImagePath); err != nil {
			log.Errorln(err)
			return nil, fmt.Errorf("Fail to create disk %s", d.Id)
		}
		return d.GetDiskDesc(), nil

	} else {
		return nil, fmt.Errorf("Fail to fetch image %s", imageId)
	}
}

func (d *SLocalDisk) CreateFromUrl(ctx context.Context, url string) error {
	remoteFile := remotefile.NewRemoteFile(ctx, url, d.getPath(), false, "", -1, nil, "", "")
	if remoteFile.Fetch() {
		if options.HostOptions.EnableFallocateDisk {
			//TODO
			// d.fallocate()
		}
		return nil
	} else {
		return fmt.Errorf("Fail to fetch image from %s", url)
	}
}

func (d *SLocalDisk) CreateRaw(ctx context.Context, sizeMB int, diskFormat, fsFormat string,
	encryption bool, uuid string, back string) (jsonutils.JSONObject, error) {
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
		err = img.CreateQcow2(sizeMB, false, back)
	case "vmdk":
		err = img.CreateVmdk(sizeMB, false)
	default:
		err = img.CreateRaw(sizeMB)
	}

	if err != nil {
		log.Errorln(err)
		fmt.Errorf("create_raw: Fail to create disk")
	}

	if options.HostOptions.EnableFallocateDisk {
		// TODO
		// d.Fallocate
	}

	if utils.IsInStringArray(fsFormat, []string{"swap", "ext2", "ext3", "ext4", "xfs"}) {
		d.FormatFs(fsFormat, uuid)
	}

	return d.GetDiskDesc(), nil
}

func (d *SLocalDisk) FormatFs(fsFormat, uuid string) {
	log.Infof("Make disk %s fs %s", uuid, fsFormat)
	gd := NewKVMGuestDisk(d.GetPath())
	if gd.Connect() {
		defer gd.Disconnect()
		if err := gd.MakePartition(fsFormat); err == nil {
			err = gd.FormatPartition(fsFormat, uuid)
			if err != nil {
				log.Errorln(err)
			}
		} else {
			log.Errorln(err)
		}
	}
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
	if _, err := procutils.NewCommand("umount", mntPath).Run(); err != nil {
		log.Errorln(err)
	}
	if _, err := procutils.NewCommand("rm", "-rf", mntPath).Run(); err != nil {
		log.Errorln(err)
	}
}

func (d *SLocalDisk) CreateSnapshot(snapshotId string) error {
	snapshotDir := d.GetSnapshotDir()
	if !fileutils2.Exists(snapshotDir) {
		_, err := procutils.NewCommand("mkdir", "-p", snapshotDir).Run()
		if err != nil {
			log.Errorln(err)
			return err
		}
	}
	snapshotPath := path.Join(snapshotDir, snapshotId)
	_, err := procutils.NewCommand("mv", "-f", d.getPath(), snapshotPath).Run()
	if err != nil {
		log.Errorln(err)
		return err
	}
	img, err := qemuimg.NewQemuImage(d.getPath())
	if err != nil {
		log.Errorln(err)
		procutils.NewCommand("mv", "-f", snapshotPath, d.getPath()).Run()
		return err
	}
	if err := img.CreateQcow2(0, false, snapshotPath); err != nil {
		log.Errorf("Snapshot create image error %s", err)
		procutils.NewCommand("mv", "-f", snapshotPath, d.getPath()).Run()
		return err
	}
	return nil
}

func (d *SLocalDisk) DeleteSnapshot(snapshotId, convertSnapshot string, pendingDelete bool) error {
	snapshotDir := d.GetSnapshotDir()
	if len(convertSnapshot) > 0 {
		if !fileutils2.Exists(snapshotDir) {
			_, err := procutils.NewCommand("mkdir", "-p", snapshotDir).Run()
			if err != nil {
				log.Errorln(err)
				return err
			}
		}
		convertSnapshotPath := path.Join(snapshotDir, convertSnapshot)
		output := convertSnapshotPath + ".tmp"
		if fileutils2.Exists(output) {
			procutils.NewCommand("rm", "-f", output).Run()
		}
		img, err := qemuimg.NewQemuImage(convertSnapshotPath)
		if err != nil {
			log.Errorln(err)
			return err
		}
		if err = img.Convert2Qcow2To(output, true); err != nil {
			log.Errorln(err)
			procutils.NewCommand("rm", "-f", output).Run()
			return err
		}
		if _, err = procutils.NewCommand("rm", "-f", convertSnapshotPath).Run(); err != nil {
			log.Errorln(err)
			return err
		}
		if _, err = procutils.NewCommand("mv", "-f", output, convertSnapshotPath).Run(); err != nil {
			log.Errorln(err)
			return err
		}
		if !pendingDelete {
			_, err = procutils.NewCommand("rm", "-f", path.Join(snapshotDir, snapshotId)).Run()
			if err != nil {
				log.Errorln(err)
				return err
			}
		}
		return nil
	} else {
		_, err := procutils.NewCommand("rm", "-f", path.Join(snapshotDir, snapshotId)).Run()
		if err != nil {
			log.Errorln(err)
			return err
		}
		return nil
	}
}

func (d *SLocalDisk) PrepareSaveToGlance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	// diskInfo, ok := params.(*jsonutils.JSONDict)
	// if !ok {
	// 	return nil, hostutils.ParamsError
	// }
	if err := d.Probe(); err != nil {
		return nil, err
	}
	destDir := d.Storage.GetImgsaveBackupPath()
	if _, err := procutils.NewCommand("mkdir", "-p", destDir).Run(); err != nil {
		log.Errorln(err)
		return nil, err
	}
	backupPath := path.Join(destDir, fmt.Sprintf("%s.%s", d.Id, appctx.AppContextTaskId(ctx)))
	if _, err := procutils.NewCommand("cp", "--sparse=always", "-f", d.GetPath(), backupPath).Run(); err != nil {
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

	snapshotDir := d.GetSnapshotDir()
	snapshotPath := path.Join(snapshotDir, resetParams.SnapshotId)
	diskTmpPath := d.GetPath() + "_reset.tmp"
	if _, err := procutils.NewCommand("mv", "-f", d.GetPath(), diskTmpPath).Run(); err != nil {
		log.Errorln(err)
		return nil, err
	}
	if !resetParams.OutOfChain {
		img, err := qemuimg.NewQemuImage(d.GetPath())
		if err != nil {
			log.Errorln(err)
			procutils.NewCommand("mv", "-f", diskTmpPath, d.GetPath()).Run()
			return nil, err
		}
		if err := img.CreateQcow2(0, false, snapshotPath); err != nil {
			log.Errorln(err)
			procutils.NewCommand("mv", "-f", diskTmpPath, d.GetPath()).Run()
			return nil, err
		}
	} else {
		if _, err := procutils.NewCommand("cp", "-f", snapshotPath, d.GetPath()).Run(); err != nil {
			log.Errorln(err)
			procutils.NewCommand("mv", "-f", diskTmpPath, d.GetPath()).Run()
			return nil, err
		}
	}
	_, err := procutils.NewCommand("rm", "-f", diskTmpPath).Run()
	return nil, err
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
		if err = img.Convert2Qcow2To(output, true); err != nil {
			log.Errorln(err)
			return nil, err
		}
		if procutils.NewCommand("mv", "-f", output, snapshotPath).Run(); err != nil {
			procutils.NewCommand("rm", "-f", output).Run()
			log.Errorln(err)
			return nil, err
		}
	}

	for _, snapshotId := range cleanupParams.DeleteSnapshots {
		snapId, _ := snapshotId.GetString()
		if _, err := procutils.NewCommand("rm", "-f", path.Join(snapshotDir, snapId)).Run(); err != nil {
			log.Errorln(err)
			return nil, err
		}
	}
	return nil, nil
}

func (d *SLocalDisk) DeleteAllSnapshot() error {
	snapshotDir := d.GetSnapshotDir()
	log.Infof("Delete disk(%s) snapshot dir %s", d.Id, snapshotDir)
	_, err := procutils.NewCommand("rm", "-rf", snapshotDir).Run()
	return err
}

func (d *SLocalDisk) PrepareMigrate(liveMigrate bool) (string, error) {
	disk, err := qemuimg.NewQemuImage(d.getPath())
	if err != nil {
		log.Errorln(err)
		return "", err
	}
	ret, err := disk.WholeChainFormatIs("qcow2")
	if err != nil {
		log.Errorln(err)
		return "", err
	}
	if liveMigrate && !ret {
		return "", fmt.Errorf("Disk format doesn't support live migrate")
	}
	if disk.IsChained() {
		return disk.BackFilePath, nil
	}
	return "", nil
}
