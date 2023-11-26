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

package nfs

import (
	"context"
	"fmt"
	"path"
	"sync"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

var ErrorBackupStorageOffline error = errors.Error(api.BackupStorageOffline)

type SNFSBackupStorage struct {
	BackupStorageId string
	Path            string
	NfsHost         string
	NfsSharedDir    string
	lock            *sync.Mutex
	userNumber      int
}

func newNFSBackupStorage(backupStorageId, nfsHost, nfsSharedDir string) *SNFSBackupStorage {
	return &SNFSBackupStorage{
		BackupStorageId: backupStorageId,
		NfsHost:         nfsHost,
		NfsSharedDir:    nfsSharedDir,
		Path:            path.Join(options.HostOptions.LocalBackupStoragePath, backupStorageId),
		lock:            &sync.Mutex{},
	}
}

func (s *SNFSBackupStorage) getBackupDir() string {
	return path.Join(s.Path, "backups")
}

func (s *SNFSBackupStorage) getBackupDiskPath(backupId string) string {
	return path.Join(s.getBackupDir(), backupId)
}

func (s *SNFSBackupStorage) getPackageDir() string {
	return path.Join(s.Path, "backuppacks")
}

func (s *SNFSBackupStorage) getBackupInstancePath(backupInstanceId string) string {
	return path.Join(s.getPackageDir(), backupInstanceId)
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

func (s *SNFSBackupStorage) SaveBackupFrom(ctx context.Context, srcFilename string, backupId string) error {
	return s.saveFile(ctx, srcFilename, backupId, s.getBackupDiskPath)
}

func (s *SNFSBackupStorage) SaveBackupInstanceFrom(ctx context.Context, srcFilename string, backupId string) error {
	return s.saveFile(ctx, srcFilename, backupId, s.getBackupDiskPath)
}

func (s *SNFSBackupStorage) saveFile(ctx context.Context, srcFilename string, id string, getPathFunc func(string) string) error {
	err := s.checkAndMount()
	if err != nil {
		return errors.Wrap(err, "unable to checkAndMount")
	}
	defer s.unMount()

	targetFilename := getPathFunc(id)
	if output, err := procutils.NewCommand("cp", srcFilename, targetFilename).Output(); err != nil {
		log.Errorf("unable to cp %s to %s: %s", srcFilename, targetFilename, output)
		return errors.Wrapf(err, "cp %s to %s failed and output is %q", srcFilename, targetFilename, output)
	}
	return nil
}

func (s *SNFSBackupStorage) RestoreBackupTo(ctx context.Context, targetFilename string, backupId string) error {
	return s.restoreFile(ctx, targetFilename, backupId, s.getBackupDiskPath)
}

func (s *SNFSBackupStorage) RestoreBackupInstanceTo(ctx context.Context, targetFilename string, backupId string) error {
	return s.restoreFile(ctx, targetFilename, backupId, s.getBackupInstancePath)
}

func (s *SNFSBackupStorage) restoreFile(ctx context.Context, targetFilename string, id string, getPathFunc func(string) string) error {
	err := s.checkAndMount()
	if err != nil {
		return errors.Wrap(err, "unable to checkAndMount")
	}
	defer s.unMount()

	srcFilename := getPathFunc(id)
	if output, err := procutils.NewCommand("cp", srcFilename, targetFilename).Output(); err != nil {
		log.Errorf("unable to cp %s to %s: %s", srcFilename, targetFilename, output)
		return errors.Wrapf(err, "cp %s to %s failed and output is %q", srcFilename, targetFilename, output)
	}
	return nil
}

func (s *SNFSBackupStorage) RemoveBackup(ctx context.Context, backupId string) error {
	return s.removeFile(ctx, backupId, s.getBackupDiskPath)
}

func (s *SNFSBackupStorage) RemoveBackupInstance(ctx context.Context, backupId string) error {
	return s.removeFile(ctx, backupId, s.getBackupInstancePath)
}

func (s *SNFSBackupStorage) removeFile(ctx context.Context, id string, getPathFunc func(id string) string) error {
	err := s.checkAndMount()
	if err != nil {
		return errors.Wrap(err, "unable to checkAndMount")
	}
	defer s.unMount()

	filename := getPathFunc(id)
	if !fileutils2.Exists(filename) {
		return nil
	}
	if output, err := procutils.NewCommand("rm", filename).Output(); err != nil {
		log.Errorf("unable to rm %s: %s", filename, output)
		return errors.Wrapf(err, "rm %s failed and output is %q", filename, output)
	}
	return nil
}

func (s *SNFSBackupStorage) IsBackupExists(backupId string) (bool, error) {
	return s.isFileExists(backupId, s.getBackupDiskPath)
}

func (s *SNFSBackupStorage) IsBackupInstanceExists(backupId string) (bool, error) {
	return s.isFileExists(backupId, s.getBackupInstancePath)
}

func (s *SNFSBackupStorage) isFileExists(id string, getPathFunc func(id string) string) (bool, error) {
	err := s.checkAndMount()
	if err != nil {
		return false, errors.Wrap(err, "unable to checkAndMount")
	}
	defer s.unMount()

	filename := getPathFunc(id)
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
