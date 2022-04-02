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

package backupstorage

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

const BackupStoragePath = "/opt/cloud/workspace/backupstorage"

var ErrorBackupStorageOffline error = errors.Error(api.BackupStorageOffline)

type SNFSBackupStorage struct {
	BackupStorageId string
	Path            string
	NfsHost         string
	NfsSharedDir    string
	lock            *sync.Mutex
	userNumber      int
}

func NewNFSBackupStorage(backupStorageId, nfsHost, nfsSharedDir string) *SNFSBackupStorage {
	return &SNFSBackupStorage{
		BackupStorageId: backupStorageId,
		NfsHost:         nfsHost,
		NfsSharedDir:    nfsSharedDir,
		Path:            path.Join(BackupStoragePath, backupStorageId),
		lock:            &sync.Mutex{},
	}
}

func (s *SNFSBackupStorage) getBackupDir() string {
	return path.Join(s.Path, "backups")
}

func (s *SNFSBackupStorage) getPackageDir() string {
	return path.Join(s.Path, "backuppacks")
}

func (s *SNFSBackupStorage) checkAndMount() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if !fileutils2.Exists(s.Path) {
		output, err := procutils.NewCommand("mkdir", "-p", s.Path).Output()
		if err != nil {
			log.Errorf("mkdir %s failed: %s", s.Path, output)
			return errors.Wrapf(err, "mkdir %s failed: %s", s.Path, output)
		}
	}
	if err := procutils.NewRemoteCommandAsFarAsPossible("mountpoint", s.Path).Run(); err == nil {
		s.userNumber++
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := procutils.NewRemoteCommandContextAsFarAsPossible(ctx,
		"mount", "-t", "nfs", fmt.Sprintf("%s:%s", s.NfsHost, s.NfsSharedDir), s.Path).Run()
	if err != nil {
		return errors.Wrap(ErrorBackupStorageOffline, err.Error())
	}
	backupDir := s.getBackupDir()
	if !fileutils2.Exists(backupDir) {
		output, err := procutils.NewCommand("mkdir", "-p", backupDir).Output()
		if err != nil {
			log.Errorf("mkdir %s failed: %s", backupDir, output)
			return errors.Wrapf(err, "mkdir %s failed: %s", backupDir, output)
		}
	}
	packageDir := s.getPackageDir()
	if !fileutils2.Exists(packageDir) {
		output, err := procutils.NewCommand("mkdir", "-p", packageDir).Output()
		if err != nil {
			log.Errorf("mkdir %s failed: %s", packageDir, output)
			return errors.Wrapf(err, "mkdir %s failed: %s", packageDir, output)
		}
	}
	s.userNumber++
	return nil
}

func (s *SNFSBackupStorage) unMount() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.userNumber--
	if s.userNumber > 0 {
		return nil
	}
	out, err := procutils.NewRemoteCommandAsFarAsPossible("umount", s.Path).Output()
	if err != nil {
		return errors.Wrapf(err, "umount %s failed %s", s.Path, out)
	}
	return nil
}

func (s *SNFSBackupStorage) CopyBackupFrom(srcFilename string, backupId string) error {
	err := s.checkAndMount()
	if err != nil {
		return errors.Wrap(err, "unable to checkAndMount")
	}
	defer s.unMount()
	backupDir := s.getBackupDir()
	targetFilename := path.Join(backupDir, backupId)
	if output, err := procutils.NewCommand("cp", srcFilename, targetFilename).Output(); err != nil {
		log.Errorf("unable to cp %s to %s: %s", srcFilename, targetFilename, output)
		return errors.Wrapf(err, "cp %s to %s failed and output is %q", srcFilename, targetFilename, output)
	}
	return nil
}

func (s *SNFSBackupStorage) CopyBackupTo(targetFilename string, backupId string) error {
	err := s.checkAndMount()
	if err != nil {
		return errors.Wrap(err, "unable to checkAndMount")
	}
	defer s.unMount()
	backupDir := s.getBackupDir()
	srcFilename := path.Join(backupDir, backupId)
	if output, err := procutils.NewCommand("cp", srcFilename, targetFilename).Output(); err != nil {
		log.Errorf("unable to cp %s to %s: %s", srcFilename, targetFilename, output)
		return errors.Wrapf(err, "cp %s to %s failed and output is %q", srcFilename, targetFilename, output)
	}
	return nil
}

const (
	PackageDiskFilename     = "disk"
	PackageMetadataFilename = "metadata"
)

func (s *SNFSBackupStorage) Pack(backupId string, packageName string, metadata jsonutils.JSONObject) error {
	err := s.checkAndMount()
	if err != nil {
		return errors.Wrap(err, "unable to checkAndMount")
	}
	defer s.unMount()
	lockman.LockRawObject(context.Background(), "package", packageName)
	defer lockman.ReleaseRawObject(context.Background(), "package", packageName)
	backupDir := s.getBackupDir()
	backupPath := path.Join(backupDir, backupId)
	packageDir := s.getPackageDir()
	packagePath := path.Join(packageDir, packageName)
	packageFilename := path.Join(packageDir, packageName+".tar")
	if fileutils2.Exists(packageFilename) {
		return errors.Error("A package with the same name already exists")
	}
	if fileutils2.Exists(packagePath) {
		// delete residual data
		if output, err := procutils.NewCommand("rm", "-rf", packagePath).Output(); err != nil {
			log.Errorf("unable to rm %s: %s", packagePath, output)
			return errors.Wrapf(err, "rm %s failed and output is %q", packagePath, output)
		}
	}
	output, err := procutils.NewCommand("mkdir", "-p", packagePath).Output()
	if err != nil {
		log.Errorf("mkdir %s failed: %s", packagePath, output)
		return errors.Wrapf(err, "mkdir %s failed: %s", packageDir, output)
	}
	defer func() {
		if output, err := procutils.NewCommand("rm", "-rf", packagePath).Output(); err != nil {
			log.Errorf("unable to rm %s: %s", packagePath, output)
		}
	}()
	packageDiskPath := path.Join(packagePath, PackageDiskFilename)
	if output, err := procutils.NewCommand("cp", backupPath, packageDiskPath).Output(); err != nil {
		log.Errorf("unable to cp %s to %s: %s", backupPath, packageDiskPath, output)
		return errors.Wrapf(err, "cp %s to %s failed and output is %q", backupPath, packageDiskPath, output)
	}
	packageMetadataPath := path.Join(packagePath, PackageMetadataFilename)
	err = ioutil.WriteFile(packageMetadataPath, []byte(metadata.PrettyString()), 0644)
	if err != nil {
		return errors.Wrapf(err, "unable to write to %s", packageMetadataPath)
	}
	// tar
	if output, err := procutils.NewCommand("tar", "-cf", packageFilename, packagePath).Output(); err != nil {
		log.Errorf("unable to 'tar -cf %s %s': %s", packageFilename, packagePath, output)
		return errors.Wrap(err, "unable to tar")
	}
	return nil
}

func (s *SNFSBackupStorage) UnPack(backupId string, packageName string) (jsonutils.JSONObject, error) {
	err := s.checkAndMount()
	if err != nil {
		return nil, errors.Wrap(err, "unable to checkAndMount")
	}
	defer s.unMount()
	lockman.LockRawObject(context.Background(), "package", packageName)
	defer lockman.ReleaseRawObject(context.Background(), "package", packageName)
	backupDir := s.getBackupDir()
	backupPath := path.Join(backupDir, backupId)
	packageDir := s.getPackageDir()
	packagePath := path.Join(packageDir, packageName)
	packageFilename := path.Join(packageDir, packageName+".tar")
	if !fileutils2.Exists(packageFilename) {
		return nil, errors.Wrapf(err, "package %s does not exists", packageName)
	}
	if fileutils2.Exists(packagePath) {
		// delete residual data
		if output, err := procutils.NewCommand("rm", "-rf", packagePath).Output(); err != nil {
			log.Errorf("unable to rm %s: %s", packagePath, output)
			return nil, errors.Wrapf(err, "rm %s failed and output is %q", packagePath, output)
		}
	}
	// untar
	if output, err := procutils.NewCommand("tar", "-xf", packageFilename, packagePath).Output(); err != nil {
		log.Errorf("unable to 'tar -xf %s %s': %s", packageFilename, packagePath, output)
		return nil, errors.Wrap(err, "unable to untar")
	}
	defer func() {
		if output, err := procutils.NewCommand("rm", "-rf", packagePath).Output(); err != nil {
			log.Errorf("unable to rm %s: %s", packagePath, output)
		}
	}()
	packageMetadataPath := path.Join(packagePath, PackageMetadataFilename)
	metadataBytes, err := ioutil.ReadFile(packageMetadataPath)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read metadata file")
	}
	metadata, err := jsonutils.ParseQueryString(string(metadataBytes))
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse string to json")
	}
	packageDiskPath := path.Join(packagePath, PackageDiskFilename)
	if output, err := procutils.NewCommand("mv", packageDiskPath, backupPath).Output(); err != nil {
		return nil, errors.Wrapf(err, "mv %s to %s failed and output is %q", packageDiskPath, backupPath, output)
	}
	return metadata, nil
}

func (s *SNFSBackupStorage) InstancePack(packageName string, backupIds []string, metadata *api.InstanceBackupPackMetadata) error {
	err := s.checkAndMount()
	if err != nil {
		return errors.Wrap(err, "unable to checkAndMount")
	}
	defer s.unMount()
	lockman.LockRawObject(context.Background(), "package", packageName)
	defer lockman.ReleaseRawObject(context.Background(), "package", packageName)
	backupDir := s.getBackupDir()
	packageDir := s.getPackageDir()
	packagePath := path.Join(packageDir, packageName)
	packageFilename := path.Join(packageDir, packageName+".tar")
	if fileutils2.Exists(packageFilename) {
		return errors.Error("A package with the same name already exists")
	}
	if fileutils2.Exists(packagePath) {
		// delete residual data
		if output, err := procutils.NewCommand("rm", "-rf", packagePath).Output(); err != nil {
			log.Errorf("unable to rm %s: %s", packagePath, output)
			return errors.Wrapf(err, "rm %s failed and output is %q", packagePath, output)
		}
	}
	output, err := procutils.NewCommand("mkdir", "-p", packagePath).Output()
	if err != nil {
		log.Errorf("mkdir %s failed: %s", packagePath, output)
		return errors.Wrapf(err, "mkdir %s failed: %s", packageDir, output)
	}
	defer func() {
		if output, err := procutils.NewCommand("rm", "-rf", packagePath).Output(); err != nil {
			log.Errorf("unable to rm %s: %s", packagePath, output)
		}
	}()
	for i, backupId := range backupIds {
		packageDiskPath := path.Join(packagePath, fmt.Sprintf("%s_%d", PackageDiskFilename, i))
		backupPath := path.Join(backupDir, backupId)
		if output, err := procutils.NewCommand("cp", backupPath, packageDiskPath).Output(); err != nil {
			log.Errorf("unable to cp %s to %s: %s", backupPath, packageDiskPath, output)
			return errors.Wrapf(err, "cp %s to %s failed and output is %q", backupPath, packageDiskPath, output)
		}
	}
	packageMetadataPath := path.Join(packagePath, PackageMetadataFilename)
	err = ioutil.WriteFile(packageMetadataPath, []byte(jsonutils.Marshal(metadata).PrettyString()), 0644)
	if err != nil {
		return errors.Wrapf(err, "unable to write to %s", packageMetadataPath)
	}
	// tar
	if output, err := procutils.NewCommand("tar", "-cf", packageFilename, "-C", packageDir, packageName).Output(); err != nil {
		log.Errorf("unable to 'tar -cf %s -C %s %s': %s", packageFilename, packageDir, packageName, output)
		return errors.Wrap(err, "unable to tar")
	}
	return nil
}

func (s *SNFSBackupStorage) InstanceUnpack(packageName string) ([]string, *api.InstanceBackupPackMetadata, error) {
	err := s.checkAndMount()
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to checkAndMount")
	}
	defer s.unMount()
	lockman.LockRawObject(context.Background(), "package", packageName)
	defer lockman.ReleaseRawObject(context.Background(), "package", packageName)
	backupDir := s.getBackupDir()
	packageDir := s.getPackageDir()
	packagePath := path.Join(packageDir, packageName)
	packageFilename := path.Join(packageDir, packageName+".tar")
	if !fileutils2.Exists(packageFilename) {
		return nil, nil, errors.Wrapf(err, "package %s does not exists", packageName)
	}
	if fileutils2.Exists(packagePath) {
		// delete residual data
		if output, err := procutils.NewCommand("rm", "-rf", packagePath).Output(); err != nil {
			log.Errorf("unable to rm %s: %s", packagePath, output)
			return nil, nil, errors.Wrapf(err, "rm %s failed and output is %q", packagePath, output)
		}
	}
	// untar
	if output, err := procutils.NewCommand("tar", "-xf", packageFilename, "-C", packageDir, packageName).Output(); err != nil {
		log.Errorf("unable to 'tar -xf %s -C %s %s': %s", packageFilename, packageDir, packageName, output)
		return nil, nil, errors.Wrap(err, "unable to untar")
	}
	defer func() {
		if output, err := procutils.NewCommand("rm", "-rf", packagePath).Output(); err != nil {
			log.Errorf("unable to rm %s: %s", packagePath, output)
		}
	}()
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
		return nil, nil, err
	}
	backupIds := make([]string, len(metadata.DiskMetadatas))
	for i := 0; i < len(metadata.DiskMetadatas); i++ {
		backupId := db.DefaultUUIDGenerator()
		backupIds[i] = backupId
		backupPath := path.Join(backupDir, backupId)
		packageDiskPath := path.Join(packagePath, fmt.Sprintf("%s_%d", PackageDiskFilename, i))
		if output, err := procutils.NewCommand("mv", packageDiskPath, backupPath).Output(); err != nil {
			return nil, nil, errors.Wrapf(err, "mv %s to %s failed and output is %q", packageDiskPath, backupPath, output)
		}
	}
	return backupIds, metadata, nil
}

func (s *SNFSBackupStorage) ConvertFrom(srcPath string, format qemuimg.TImageFormat, backupId string) (int, error) {
	err := s.checkAndMount()
	if err != nil {
		return 0, errors.Wrap(err, "unable to checkAndMount")
	}
	defer s.unMount()
	backupDir := s.getBackupDir()
	destPath := path.Join(backupDir, backupId)
	srcInfo := qemuimg.SConvertInfo{
		Path:     srcPath,
		Format:   format,
		IoLevel:  qemuimg.IONiceNone,
		Password: "",
	}
	destInfo := qemuimg.SConvertInfo{
		Path:     destPath,
		Format:   qemuimg.QCOW2,
		IoLevel:  qemuimg.IONiceNone,
		Password: "",
	}
	err = qemuimg.Convert(srcInfo, destInfo, true, nil)
	if err != nil {
		return 0, err
	}
	newImage, err := qemuimg.NewQemuImage(destPath)
	if err != nil {
		return 0, err
	}
	return newImage.GetActualSizeMB(), nil
}

func (s *SNFSBackupStorage) ConvertTo(destPath string, format qemuimg.TImageFormat, backupId string) error {
	err := s.checkAndMount()
	if err != nil {
		return errors.Wrap(err, "unable to checkAndMount")
	}
	defer s.unMount()
	backupDir := s.getBackupDir()
	srcPath := path.Join(backupDir, backupId)
	srcInfo := qemuimg.SConvertInfo{
		Path:     srcPath,
		Format:   qemuimg.QCOW2,
		IoLevel:  qemuimg.IONiceNone,
		Password: "",
	}
	destInfo := qemuimg.SConvertInfo{
		Path:     destPath,
		Format:   format,
		IoLevel:  qemuimg.IONiceNone,
		Password: "",
	}
	var workerOpts []string
	if options.HostOptions.RestrictQemuImgConvertWorker {
		workerOpts = nil
	} else {
		workerOpts = []string{"-W", "-m", "16"}
	}
	return qemuimg.Convert(srcInfo, destInfo, false, workerOpts)
}

func (s *SNFSBackupStorage) GetBackupPath(backupId string) string {
	backupDir := s.getBackupDir()
	return path.Join(backupDir, backupId)
}

func (s *SNFSBackupStorage) RemoveBackup(backupId string) error {
	err := s.checkAndMount()
	if err != nil {
		return errors.Wrap(err, "unable to checkAndMount")
	}
	defer s.unMount()
	backupDir := s.getBackupDir()
	filename := path.Join(backupDir, backupId)
	if !fileutils2.Exists(filename) {
		return nil
	}
	if output, err := procutils.NewCommand("rm", filename).Output(); err != nil {
		log.Errorf("unable to rm %s: %s", filename, output)
		return errors.Wrapf(err, "rm %s failed and output is %q", filename, output)
	}
	return nil
}

func (s *SNFSBackupStorage) IsExists(backupId string) (bool, error) {
	err := s.checkAndMount()
	if err != nil {
		return false, errors.Wrap(err, "unable to checkAndMount")
	}
	defer s.unMount()
	backupDir := s.getBackupDir()
	filename := path.Join(backupDir, backupId)
	return fileutils2.Exists(filename), nil
}

func (s *SNFSBackupStorage) IsOnline() (bool, string, error) {
	err := s.checkAndMount()
	if errors.Cause(err) == ErrorBackupStorageOffline {
		return false, err.Error(), nil
	}
	if err != nil {
		return false, "", err
	}
	s.unMount()
	return true, "", nil
}
