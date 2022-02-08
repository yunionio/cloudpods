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
	"path"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

const BackupStoragePath = "/opt/cloud/workspace/backupstorage"

type SNFSBackupStorage struct {
	BackupStorageId string
	Path            string
	NfsHost         string
	NfsSharedDir    string
}

func NewNFSBackupStorage(backupStorageId, nfsHost, nfsSharedDir string) *SNFSBackupStorage {
	return &SNFSBackupStorage{
		BackupStorageId: backupStorageId,
		NfsHost:         nfsHost,
		NfsSharedDir:    nfsSharedDir,
		Path:            path.Join(BackupStoragePath, backupStorageId),
	}
}

func (s *SNFSBackupStorage) getBackupDir() string {
	return path.Join(s.Path, "backups")
}

func (s *SNFSBackupStorage) checkAndMount() error {
	lockman.LockRawObject(context.Background(), "backupstorage", s.BackupStorageId)
	defer lockman.ReleaseRawObject(context.Background(), "backupstorage", s.BackupStorageId)
	if !fileutils2.Exists(s.Path) {
		output, err := procutils.NewCommand("mkdir", "-p", s.Path).Output()
		if err != nil {
			log.Errorf("mkdir %s failed: %s", s.Path, output)
			return errors.Wrapf(err, "mkdir %s failed: %s", s.Path, output)
		}
	}
	if err := procutils.NewRemoteCommandAsFarAsPossible("mountpoint", s.Path).Run(); err == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := procutils.NewRemoteCommandContextAsFarAsPossible(ctx,
		"mount", "-t", "nfs", fmt.Sprintf("%s:%s", s.NfsHost, s.NfsSharedDir), s.Path).Run()
	if err != nil {
		return err
	}
	backupDir := s.getBackupDir()
	if !fileutils2.Exists(backupDir) {
		output, err := procutils.NewCommand("mkdir", "-p", backupDir).Output()
		if err != nil {
			log.Errorf("mkdir %s failed: %s", backupDir, output)
			return errors.Wrapf(err, "mkdir %s failed: %s", backupDir, output)
		}
	}
	return nil
}

func (s *SNFSBackupStorage) unMount() error {
	lockman.LockRawObject(context.Background(), "backupstorage", s.BackupStorageId)
	defer lockman.ReleaseRawObject(context.Background(), "backupstorage", s.BackupStorageId)
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
		Format:   qemuimg.RAW,
		IoLevel:  qemuimg.IONiceNone,
		Password: "",
	}
	destInfo := qemuimg.SConvertInfo{
		Path:     destPath,
		Format:   qemuimg.QCOW2,
		IoLevel:  qemuimg.IONiceNone,
		Password: "",
	}
	err = qemuimg.Convert(srcInfo, destInfo, nil, true, nil)
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
	var opts []string
	if format == qemuimg.QCOW2 {
		opts = qemuimg.Qcow2SparseOptions()
	}
	var workerOpts []string
	if options.HostOptions.RestrictQemuImgConvertWorker {
		workerOpts = nil
	} else {
		workerOpts = []string{"-W", "-m", "16"}
	}
	return qemuimg.Convert(srcInfo, destInfo, opts, false, workerOpts)
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
