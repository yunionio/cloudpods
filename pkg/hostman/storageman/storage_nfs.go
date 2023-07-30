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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func init() {
	registerStorageFactory(&SNFSStorageFactory{})
}

type SNFSStorageFactory struct {
}

func (factory *SNFSStorageFactory) NewStorage(manager *SStorageManager, mountPoint string) IStorage {
	return NewNFSStorage(manager, mountPoint)
}

func (factory *SNFSStorageFactory) StorageType() string {
	return api.STORAGE_NFS
}

type SNFSStorage struct {
	SNasStorage
}

func NewNFSStorage(manager *SStorageManager, path string) *SNFSStorage {
	ret := &SNFSStorage{}
	ret.SNasStorage = *NewNasStorage(manager, path, ret)
	if !fileutils2.Exists(path) {
		procutils.NewCommand("mkdir", "-p", path).Run()
	}
	return ret
}

func (s *SNFSStorage) newDisk(diskId string) IDisk {
	return NewNFSDisk(s, diskId)
}

func (s *SNFSStorage) StorageType() string {
	return api.STORAGE_NFS
}

func (s *SNFSStorage) SyncStorageInfo() (jsonutils.JSONObject, error) {
	if len(s.StorageId) == 0 {
		return nil, fmt.Errorf("Sync nfs storage without storage id")
	}
	content := jsonutils.NewDict()
	content.Set("capacity", jsonutils.NewInt(int64(s.GetAvailSizeMb())))
	content.Set("storage_type", jsonutils.NewString(s.StorageType()))
	content.Set("status", jsonutils.NewString(api.STORAGE_ONLINE))
	content.Set("zone", jsonutils.NewString(s.GetZoneId()))
	log.Infof("Sync storage info %s", s.StorageId)
	res, err := modules.Storages.Put(
		hostutils.GetComputeSession(context.Background()),
		s.StorageId, content)
	if err != nil {
		log.Errorf("SyncStorageInfo Failed: %s: %s", content, err)
	}
	return res, err
}

func (s *SNFSStorage) SetStorageInfo(storageId, storageName string, conf jsonutils.JSONObject) error {
	s.StorageId = storageId
	s.StorageName = storageName
	if dconf, ok := conf.(*jsonutils.JSONDict); ok {
		s.StorageConf = dconf
	}
	if err := s.checkAndMount(); err != nil {
		return errors.Errorf("Fail to mount storage to mountpoint: %s, %s", s.Path, err)
	}
	return s.BindMountStoragePath(s.Path)
}

func (s *SNFSStorage) checkAndMount() error {
	if err := procutils.NewRemoteCommandAsFarAsPossible("mountpoint", s.Path).Run(); err == nil {
		return nil
	}
	if s.StorageConf == nil {
		return fmt.Errorf("Storage conf is nil")
	}
	host, err := s.StorageConf.GetString("nfs_host")
	if err != nil {
		return fmt.Errorf("Storage conf missing nfs_host ")
	}
	sharedDir, err := s.StorageConf.GetString("nfs_shared_dir")
	if err != nil {
		return fmt.Errorf("Storage conf missing nfs_shared_dir")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = procutils.NewRemoteCommandContextAsFarAsPossible(ctx,
		"mount", "-t", "nfs", fmt.Sprintf("%s:%s", host, sharedDir), s.Path).Run()
	if err != nil {
		return err
	}
	return nil
}

func (s *SNFSStorage) Detach() error {
	if !strings.HasPrefix(s.Path, "/opt/cloud") {
		tmpPath := path.Join(TempBindMountPath, s.Path)
		out, err := procutils.NewCommand("umount", s.Path).Output()
		if err != nil {
			return errors.Wrapf(err, "1. umount %s failed %s", s.Path, out)
		}
		out, err = procutils.NewRemoteCommandAsFarAsPossible("umount", tmpPath).Output()
		if err != nil {
			return errors.Wrapf(err, "2. umount %s failed %s", tmpPath, out)
		}
	}
	out, err := procutils.NewRemoteCommandAsFarAsPossible("umount", s.Path).Output()
	if err != nil {
		return errors.Wrapf(err, "3. umount %s failed %s", s.Path, out)
	}
	return nil
}
