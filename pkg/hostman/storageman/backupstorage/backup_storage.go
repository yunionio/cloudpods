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
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type IBackupStorageFactory interface {
	NewBackupStore(storeId string, backupStorageAccessInfo *jsonutils.JSONDict) (IBackupStorage, error)
}

type IBackupStorage interface {
	// 从指定路径拷贝磁盘文件到备份存储
	SaveBackupFrom(ctx context.Context, srcFilename string, bakcupId string) error
	// 将备份backupId对应的备份文件拷贝到指定的文件路径
	RestoreBackupTo(ctx context.Context, targetFilename string, backupId string) error
	// 删除备份
	RemoveBackup(ctx context.Context, backupId string) error
	// 备份是否存在
	IsBackupExists(backupId string) (bool, error)

	// 从指定路径拷贝主机备份文件到备份存储
	SaveBackupInstanceFrom(ctx context.Context, srcFilename string, bakcupInstanceId string) error
	// 将备份backupId对应的备份文件拷贝到指定的文件路径
	RestoreBackupInstanceTo(ctx context.Context, targetFilename string, backupInstanceId string) error
	// 删除备份
	RemoveBackupInstance(ctx context.Context, backupInstanceId string) error
	// 备份是否存在
	IsBackupInstanceExists(backupInstanceId string) (bool, error)

	// ConvertTo(destPath string, format qemuimgfmt.TImageFormat, backupId string) error
	// ConvertFrom(srcPath string, format qemuimgfmt.TImageFormat, backupId string) (int, error)
	// InstancePack(ctx context.Context, packageName string, backupIds []string, metadata *api.InstanceBackupPackMetadata) (string, error)
	// InstanceUnpack(ctx context.Context, packageName string, metadataOnly bool) ([]string, *api.InstanceBackupPackMetadata, error)

	// 存储是否在线
	IsOnline() (bool, string, error)
}

var factories []IBackupStorageFactory
var backupStoragePool map[string]IBackupStorage
var backupStorageLock *sync.Mutex

func init() {
	backupStorageLock = &sync.Mutex{}
	backupStoragePool = make(map[string]IBackupStorage)
}

func RegisterFactory(factory IBackupStorageFactory) {
	factories = append(factories, factory)
}

func newBackupStorage(backupStroageId string, backupStorageAccessInfo *jsonutils.JSONDict) (IBackupStorage, error) {
	errs := make([]error, 0)
	for _, factory := range factories {
		store, err := factory.NewBackupStore(backupStroageId, backupStorageAccessInfo)
		if err == nil {
			return store, nil
		} else {
			errs = append(errs, err)
		}
	}
	return nil, errors.NewAggregate(errs)
}

func GetBackupStorage(backupStroageId string, backupStorageAccessInfo *jsonutils.JSONDict) (IBackupStorage, error) {
	backupStorageLock.Lock()
	defer backupStorageLock.Unlock()

	if ibs, ok := backupStoragePool[backupStroageId]; !ok {
		bs, err := newBackupStorage(backupStroageId, backupStorageAccessInfo)
		if err != nil {
			return nil, errors.Wrap(err, "newBackupStorage")
		}
		backupStoragePool[backupStroageId] = bs
		return bs, nil
	} else {
		return ibs, nil
	}
}
