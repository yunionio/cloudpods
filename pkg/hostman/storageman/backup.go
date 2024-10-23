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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/qemuimgfmt"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	container_storage "yunion.io/x/onecloud/pkg/hostman/container/storage"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman/backupstorage"
	_ "yunion.io/x/onecloud/pkg/hostman/storageman/backupstorage/nfs"
	_ "yunion.io/x/onecloud/pkg/hostman/storageman/backupstorage/object"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	identity_modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

func EnsureBackupDir() (string, error) {
	backupTmpDir := options.HostOptions.LocalBackupTempPath
	if !fileutils2.Exists(backupTmpDir) {
		output, err := procutils.NewCommand("mkdir", "-p", backupTmpDir).Output()
		if err != nil {
			log.Errorf("mkdir %s failed: %s", backupTmpDir, output)
			return "", errors.Wrapf(err, "mkdir %s failed: %s", backupTmpDir, output)
		}
	}
	tmpFileDir, err := ioutil.TempDir(backupTmpDir, "backuptmp*")
	if err != nil {
		return "", errors.Wrap(err, "ioutil.TempDir")
	}
	return tmpFileDir, nil
}

func CleanupDirOrFile(path string) {
	log.Debugf("cleanup backup %s", path)
	if output, err := procutils.NewCommand("rm", "-rf", path).Output(); err != nil {
		log.Errorf("unable to rm %s: %s", path, output)
	}
}

func isTarSnapshot(path string) bool {
	return strings.HasSuffix(path, ".tar")
}

func doBackupDisk(ctx context.Context, snapshotPath string, diskBackup *SDiskBackup) (int, error) {
	backupTmpDir, err := EnsureBackupDir()
	if err != nil {
		return 0, errors.Wrap(err, "EnsureBackupDir")
	}
	defer CleanupDirOrFile(backupTmpDir)

	backupPath := path.Join(backupTmpDir, diskBackup.BackupId)
	var newImageSizeMb int

	if isTarSnapshot(snapshotPath) {
		backupPath = snapshotPath
		fileSize := fileutils2.FileSize(backupPath)
		if fileSize <= 0 {
			return 0, errors.Errorf("get snapshot path %s size failed", snapshotPath)
		}
		newImageSizeMb = int(fileSize / 1024 / 1024)
	} else {
		img, err := qemuimg.NewQemuImage(snapshotPath)
		if err != nil {
			return 0, errors.Wrap(err, "NewQemuImage snapshot")
		}
		encKey := ""
		if len(diskBackup.EncryptKeyId) > 0 {
			session := auth.GetSession(ctx, diskBackup.UserCred, consts.GetRegion())
			secKey, err := identity_modules.Credentials.GetEncryptKey(session, diskBackup.EncryptKeyId)
			if err != nil {
				return 0, errors.Wrap(err, "GetEncryptKey")
			}
			encKey = secKey.Key
		}
		if len(encKey) > 0 {
			img.SetPassword(encKey)
		}
		newImage, err := img.Clone(backupPath, qemuimgfmt.QCOW2, true)
		if err != nil {
			return 0, errors.Wrap(err, "unable to backup snapshot")
		}

		newImageSizeMb = newImage.GetActualSizeMB()
	}

	backupStorage, err := backupstorage.GetBackupStorage(diskBackup.BackupStorageId, diskBackup.BackupStorageAccessInfo)
	if err != nil {
		return 0, errors.Wrap(err, "GetBackupStorage")
	}

	err = backupStorage.SaveBackupFrom(ctx, backupPath, diskBackup.BackupId)
	if err != nil {
		return 0, errors.Wrap(err, "SaveBackupFrom")
	}

	return newImageSizeMb, nil
}

type IDiskCreator interface {
	CreateRawDisk(ctx context.Context, disk IDisk, input *SDiskCreateByDiskinfo) (jsonutils.JSONObject, error)
}

func doRestoreDisk(ctx context.Context, dc IDiskCreator, input *SDiskCreateByDiskinfo, disk IDisk, destImgPath string) error {
	diskInfo := input.DiskInfo
	format := diskInfo.Format
	backupTmpDir, err := EnsureBackupDir()
	if err != nil {
		return errors.Wrap(err, "EnsureBackupDir")
	}
	defer CleanupDirOrFile(backupTmpDir)

	backupStorage, err := backupstorage.GetBackupStorage(diskInfo.Backup.BackupStorageId, diskInfo.Backup.BackupStorageAccessInfo)
	if err != nil {
		return errors.Wrap(err, "GetBackupStorage")
	}
	backupPath := path.Join(backupTmpDir, diskInfo.Backup.BackupId)
	err = backupStorage.RestoreBackupTo(ctx, backupPath, diskInfo.Backup.BackupId)
	if err != nil {
		return errors.Wrap(err, "RestoreBackupTo")
	}

	backupInput := diskInfo.Backup

	if backupInput.BackupAsTar != nil {
		return doRestoreTarDisk(ctx, dc, disk, input, destImgPath, backupPath)
	} else {
		return doRestoreQCOW2Disk(ctx, diskInfo, destImgPath, format, backupPath)
	}
}

func doRestoreQCOW2Disk(ctx context.Context, diskInfo api.DiskAllocateInput, destImgPath string, format string, backupPath string) error {
	img, err := qemuimg.NewQemuImage(backupPath)
	if err != nil {
		return errors.Wrap(err, "NewQemuImage")
	}
	if diskInfo.Encryption {
		img.SetPassword(diskInfo.EncryptInfo.Key)
	}
	if len(format) == 0 {
		format = qemuimgfmt.QCOW2.String()
	}
	_, err = img.Clone(destImgPath, qemuimgfmt.String2ImageFormat(format), false)
	if err != nil {
		return errors.Wrapf(err, "Clone %s", destImgPath)
	}
	return nil
}

func doRestoreTarDisk(ctx context.Context, dc IDiskCreator, disk IDisk, input *SDiskCreateByDiskinfo, destImgPath string, backupPath string) error {
	if err := input.Disk.OnRebuildRoot(ctx, input.DiskInfo); err != nil {
		return errors.Wrapf(err, "call OnRebuildRoot when restore tar disk")
	}
	diskInfo := input.DiskInfo
	backupInput := diskInfo.Backup
	if backupInput.BackupAsTar == nil {
		return errors.Error("backup.backup_as_tar input is empty")
	}
	if backupInput.DiskConfig == nil {
		return errors.Error("backup.disk_config input is empty")
	}
	_, err := dc.CreateRawDisk(ctx, disk, input)
	if err != nil {
		return errors.Wrapf(err, "CreateRawDisk by input: %s", jsonutils.Marshal(input))
	}

	drv, err := disk.GetContainerStorageDriver()
	if err != nil {
		return errors.Wrap(err, "get disk storage driver")
	}
	devPath, isConnected, err := drv.CheckConnect(destImgPath)
	if err != nil {
		return errors.Wrapf(err, "CheckConnect %s", disk.GetPath())
	}
	if !isConnected {
		devPath, err = drv.ConnectDisk(disk.GetPath())
		if err != nil {
			return errors.Wrapf(err, "ConnectDisk %s", disk.GetPath())
		}
	}

	backupMntDir, err := EnsureBackupDir()
	if err != nil {
		return errors.Wrap(err, "EnsureBackupDir")
	}
	defer CleanupDirOrFile(backupMntDir)

	if err := container_storage.Mount(devPath, backupMntDir, diskInfo.FsFormat); err != nil {
		return errors.Wrapf(err, "mount %s to %s", devPath, backupMntDir)
	}

	cmd := fmt.Sprintf("tar -xf %s -C %s", backupPath, backupMntDir)
	log.Infof("start restore %s to %s, disk: %s", backupPath, backupMntDir, disk.GetId())
	if out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", cmd).Output(); err != nil {
		return errors.Wrapf(err, "%s: %s", cmd, out)
	}

	if err := container_storage.Unmount(backupMntDir); err != nil {
		return errors.Wrapf(err, "unmount %s", backupMntDir)
	}
	if err := drv.DisconnectDisk(disk.GetPath(), backupMntDir); err != nil {
		return errors.Wrapf(err, "DisconnectDisk %s %s", disk.GetPath(), backupMntDir)
	}
	return nil
}

const (
	PackageDiskFilename     = "disk"
	PackageMetadataFilename = "metadata"
)

func DoInstancePackBackup(ctx context.Context, backupInfo SStoragePackInstanceBackup) (string, error) {
	backupTmpDir, err := EnsureBackupDir()
	if err != nil {
		return "", errors.Wrap(err, "EnsureBackupDir")
	}
	defer CleanupDirOrFile(backupTmpDir)

	backupStorage, err := backupstorage.GetBackupStorage(backupInfo.BackupStorageId, backupInfo.BackupStorageAccessInfo)
	if err != nil {
		return "", errors.Wrap(err, "GetBackupStorage")
	}

	packagePath := path.Join(backupTmpDir, backupInfo.PackageName)
	{
		// prepare package Path
		output, err := procutils.NewCommand("mkdir", "-p", packagePath).Output()
		if err != nil {
			log.Errorf("mkdir %s failed: %s", packagePath, output)
			return "", errors.Wrapf(err, "mkdir %s failed: %s", packagePath, output)
		}
	}
	{
		// download disk files
		for i, backupId := range backupInfo.BackupIds {
			packageDiskPath := path.Join(packagePath, fmt.Sprintf("%s_%d", PackageDiskFilename, i))
			err := backupStorage.RestoreBackupTo(ctx, packageDiskPath, backupId)
			if err != nil {
				return "", errors.Wrapf(err, "RestoreBackupTo %s %s", backupId, packageDiskPath)
			}
		}
	}
	{
		// save snapshot metadata
		packageMetadataPath := path.Join(packagePath, PackageMetadataFilename)
		err = ioutil.WriteFile(packageMetadataPath, []byte(jsonutils.Marshal(backupInfo.Metadata).PrettyString()), 0644)
		if err != nil {
			return "", errors.Wrapf(err, "unable to write to %s", packageMetadataPath)
		}
	}
	tmpPkgFilename := path.Join(backupTmpDir, backupInfo.PackageName+".tar")
	{
		// tar
		if output, err := procutils.NewRemoteCommandAsFarAsPossible("tar", "-cf", tmpPkgFilename, "-C", backupTmpDir, backupInfo.PackageName).Output(); err != nil {
			log.Errorf("unable to 'tar -cf %s -C %s %s': %s", tmpPkgFilename, backupTmpDir, backupInfo.PackageName, output)
			return "", errors.Wrap(err, "unable to tar")
		}
	}

	var finalPackageName string
	tried := 0
	for {
		var finalPackageFileName string
		if tried == 0 {
			finalPackageFileName = fmt.Sprintf("%s.tar", backupInfo.PackageName)
		} else {
			finalPackageFileName = fmt.Sprintf("%s-%d.tar", backupInfo.PackageName, tried)
		}
		exists, err := backupStorage.IsBackupInstanceExists(finalPackageFileName)
		if err != nil {
			return "", errors.Wrap(err, "IsBackupInstanceExists")
		}
		if exists {
			tried++
		} else {
			err := backupStorage.SaveBackupInstanceFrom(ctx, tmpPkgFilename, finalPackageFileName)
			if err != nil {
				return "", errors.Wrap(err, "SaveBackupInstanceFrom")
			}
			finalPackageName = finalPackageFileName
			break
		}
	}

	return finalPackageName, nil
}

func DoInstanceUnpackBackup(ctx context.Context, backupInfo SStorageUnpackInstanceBackup) ([]string, *api.InstanceBackupPackMetadata, error) {
	backupTmpDir, err := EnsureBackupDir()
	if err != nil {
		return nil, nil, errors.Wrap(err, "EnsureBackupDir")
	}
	defer CleanupDirOrFile(backupTmpDir)

	packageName := backupInfo.PackageName
	metadataOnly := false
	if backupInfo.MetadataOnly != nil && *backupInfo.MetadataOnly {
		metadataOnly = true
	}

	backupStorage, err := backupstorage.GetBackupStorage(backupInfo.BackupStorageId, backupInfo.BackupStorageAccessInfo)
	if err != nil {
		return nil, nil, errors.Wrap(err, "GetBackupStorage")
	}

	packageFilename := path.Join(backupTmpDir, packageName+".tar")
	err = backupStorage.RestoreBackupInstanceTo(ctx, packageFilename, backupInfo.PackageName)
	if err != nil {
		return nil, nil, errors.Wrap(err, "RestoreBackupInstanceTo")
	}

	// untar to temp dir
	packagePath := path.Join(backupTmpDir, packageName)
	log.Infof("unpack to %s", packagePath)
	untarArgs := []string{
		"-xf", packageFilename, "-C", backupTmpDir,
	}
	if metadataOnly {
		untarArgs = append(untarArgs, fmt.Sprintf("%s/metadata", packageName))
	} else {
		untarArgs = append(untarArgs, packageName)
	}
	if output, err := procutils.NewCommand("tar", untarArgs...).Output(); err != nil {
		log.Errorf("unable to 'tar -xf %s -C %s %s': %s", packageFilename, backupTmpDir, packageName, output)
		return nil, nil, errors.Wrap(err, "unable to untar")
	}

	// unpack metadata
	packageMetadataPath := path.Join(packagePath, PackageMetadataFilename)
	metadataBytes, err := ioutil.ReadFile(packageMetadataPath)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to read metadata file")
	}
	metadataJson, err := jsonutils.Parse(metadataBytes)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to parse string to json")
	}
	metadata := &api.InstanceBackupPackMetadata{}
	err = metadataJson.Unmarshal(metadata)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unmarshal backup metadata")
	}

	// copy disk files only if !metadataOnly
	backupIds := make([]string, len(metadata.DiskMetadatas))
	if !metadataOnly {
		for i := 0; i < len(metadata.DiskMetadatas); i++ {
			backupId := db.DefaultUUIDGenerator()
			backupIds[i] = backupId
			packageDiskPath := path.Join(packagePath, fmt.Sprintf("%s_%d", PackageDiskFilename, i))
			err := backupStorage.SaveBackupFrom(ctx, packageDiskPath, backupId)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "SaveBackupFrom %s %s", packageDiskPath, backupId)
			}
		}
	}

	return backupIds, metadata, nil
}
