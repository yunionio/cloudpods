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
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/qemuimgfmt"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type IBackupStorage interface {
	CopyBackupFrom(srcFilename string, bakcupId string) error
	CopyBackupTo(targetFilename string, backupId string) error
	RemoveBackup(backupId string) error
	IsExists(backupId string) (bool, error)
	ConvertTo(destPath string, format qemuimgfmt.TImageFormat, backupId string) error
	ConvertFrom(srcPath string, format qemuimgfmt.TImageFormat, backupId string) (int, error)
	InstancePack(ctx context.Context, packageName string, backupIds []string, metadata *api.InstanceBackupPackMetadata) (string, error)
	InstanceUnpack(ctx context.Context, packageName string, metadataOnly bool) ([]string, *api.InstanceBackupPackMetadata, error)
	IsOnline() (bool, string, error)
}

var backupStoragePool *sync.Map = &sync.Map{}

func NewBackupStorage(backupStroageId string, backupStorageAccessInfo *jsonutils.JSONDict) (IBackupStorage, error) {
	nfsHost, err := backupStorageAccessInfo.GetString("nfs_host")
	if err != nil {
		return nil, fmt.Errorf("need nfs_host in backup_storage_access_info")
	}
	nfsSharedDir, err := backupStorageAccessInfo.GetString("nfs_shared_dir")
	if err != nil {
		return nil, fmt.Errorf("need nfs_shared_dir in backup_storage_access_info")
	}
	return NewNFSBackupStorage(backupStroageId, nfsHost, nfsSharedDir), nil
}

func GetBackupStorage(backupStroageId string, backupStorageAccessInfo *jsonutils.JSONDict) (IBackupStorage, error) {
	bs, err := NewBackupStorage(backupStroageId, backupStorageAccessInfo)
	if err != nil {
		return nil, err
	}
	ibs, _ := backupStoragePool.LoadOrStore(backupStroageId, bs)
	return ibs.(IBackupStorage), nil
}
