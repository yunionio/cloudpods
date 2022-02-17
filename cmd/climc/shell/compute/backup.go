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

package compute

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {
	bsCmd := shell.NewResourceCmd(&modules.BackupStorages)
	bsCmd.List(&compute.BackupStorageListOptions{})
	bsCmd.Show(&compute.BackupStorageIdOptions{})
	bsCmd.Create(&compute.BackupStorageCreateOptions{})
	bsCmd.Delete(&compute.BackupStorageIdOptions{})
	bsCmd.Perform("public", &options.BasePublicOptions{})
	bsCmd.Perform("private", &options.BaseIdOptions{})

	dbCmd := shell.NewResourceCmd(&modules.DiskBackups)
	dbCmd.List(&compute.DiskBackupListOptions{})
	dbCmd.Show(&compute.DiskBackupIdOptions{})
	dbCmd.Delete(&compute.DiskBackupIdOptions{})
	dbCmd.Create(&compute.DiskBackupCreateOptions{})
	dbCmd.Perform("recovery", &compute.DiskBackupRecoveryOptions{})
	dbCmd.Perform("syncstatus", &compute.DiskBackupSyncstatusOptions{})

	ibCmd := shell.NewResourceCmd(&modules.InstanceBackups)
	ibCmd.List(&compute.InstanceBackupListOptions{})
	ibCmd.Show(&compute.InstanceBackupIdOptions{})
	ibCmd.Delete(&compute.InstanceBackupIdOptions{})
	ibCmd.Perform("recovery", &compute.InstanceBackupRecoveryOptions{})
}
