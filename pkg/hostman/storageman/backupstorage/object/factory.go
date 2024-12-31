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

package object

import (
	"yunion.io/x/cloudmux/pkg/multicloud/objectstore"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/storageman/backupstorage"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type sObjectBackupStorageFactory struct{}

func (factory *sObjectBackupStorageFactory) NewBackupStore(backupStroageId string, backupStorageAccessInfo *jsonutils.JSONDict) (backupstorage.IBackupStorage, error) {
	accessInfo := api.SBackupStorageAccessInfo{}
	err := backupStorageAccessInfo.Unmarshal(&accessInfo)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal access info")
	}
	if len(accessInfo.ObjectBucketUrl) == 0 {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "need object_bucket_url in backup_storage_access_info")
	}
	if len(accessInfo.ObjectAccessKey) == 0 {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "need object_access_key in backup_storage_access_info")
	}
	if len(accessInfo.ObjectSecret) == 0 {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "need object_secret in backup_storage_access_info")
	}
	return newObjectBackupStorage(backupStroageId, accessInfo.ObjectBucketUrl, accessInfo.ObjectAccessKey, accessInfo.ObjectSecret, objectstore.S3SignVersion(accessInfo.ObjectSignVer))
}

func init() {
	backupstorage.RegisterFactory(&sObjectBackupStorageFactory{})
}
