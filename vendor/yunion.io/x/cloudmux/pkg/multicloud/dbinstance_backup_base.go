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

package multicloud

import (
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SDBInstanceBackupBase struct {
	SResourceBase
}

func (backup *SDBInstanceBackupBase) GetBackMode() string {
	return api.BACKUP_MODE_AUTOMATED
}

func (backup *SDBInstanceBackupBase) Delete() error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "Delete")
}

func (backup *SDBInstanceBackupBase) GetProjectId() string {
	return ""
}

func (backup *SDBInstanceBackupBase) CreateICloudDBInstance(opts *cloudprovider.SManagedDBInstanceCreateConfig) (cloudprovider.ICloudDBInstance, error) {
	return nil, errors.Wrap(cloudprovider.ErrNotImplemented, "CreateICloudDBInstance")
}

func (backup *SDBInstanceBackupBase) GetBackupMethod() cloudprovider.TBackupMethod {
	return cloudprovider.BackupMethodUnknown
}
