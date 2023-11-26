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
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/storageman/backupstorage"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type sNfsBackupStorageFactory struct{}

func (factory *sNfsBackupStorageFactory) NewBackupStore(backupStroageId string, backupStorageAccessInfo *jsonutils.JSONDict) (backupstorage.IBackupStorage, error) {
	accessInfo := api.SBackupStorageAccessInfo{}
	err := backupStorageAccessInfo.Unmarshal(&accessInfo)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal access info")
	}
	if len(accessInfo.NfsHost) == 0 {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "need nfs_host in backup_storage_access_info")
	}
	if len(accessInfo.NfsSharedDir) == 0 {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "need nfs_shared_dir in backup_storage_access_info")
	}
	return newNFSBackupStorage(backupStroageId, accessInfo.NfsHost, accessInfo.NfsSharedDir), nil
}

func init() {
	backupstorage.RegisterFactory(&sNfsBackupStorageFactory{})
}
