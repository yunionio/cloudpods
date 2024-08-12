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
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SDiskCreateByDiskinfo struct {
	DiskId   string
	Disk     IDisk
	DiskInfo api.DiskAllocateInput

	Storage IStorage
}

func (i *SDiskCreateByDiskinfo) String() string {
	return fmt.Sprintf("disk_id: %s, disk_info: %s", i.DiskId, jsonutils.Marshal(i.DiskInfo))
}

type SDiskMigrate struct {
	DiskId             string
	Disk               IDisk
	SrcStorageId       string
	TemplateId         string
	OutChainSnaps      []jsonutils.JSONObject
	SnapsChain         []jsonutils.JSONObject
	DiskBackingFile    string
	SnapshotsUri       string
	DiskUri            string
	SysDiskHasTemplate bool

	Storage IStorage
}

type SDiskReset struct {
	SnapshotId    string
	BackingDiskId string
	Input         jsonutils.JSONObject
}

type SDiskCleanupSnapshots struct {
	ConvertSnapshots []jsonutils.JSONObject
	DeleteSnapshots  []jsonutils.JSONObject
}

type SStorageDeleteSnapshots struct {
	DiskId      string
	SnapshotIds []string
}

type SStorageDeleteSnapshot struct {
	DiskId          string
	SnapshotId      string
	ConvertSnapshot string
	BlockStream     bool
	EncryptInfo     apis.SEncryptInfo
}

type SDiskBackup struct {
	SnapshotId              string              `json:"snapshot_id"`
	SnapshotLocation        string              `json:"snapshot_location"`
	BackupId                string              `json:"backup_id"`
	BackupStorageId         string              `json:"backup_storage_id"`
	BackupStorageAccessInfo *jsonutils.JSONDict `json:"backup_storage_access_info"`

	EncryptKeyId string `json:"encrypt_key_id"`

	UserCred mcclient.TokenCredential
}

type SStorageBackup struct {
	BackupId                string
	BackupLocalPath         string
	BackupStorageId         string
	BackupStorageAccessInfo *jsonutils.JSONDict
}

type SStoragePackBackup struct {
	PackageName             string
	BackupId                string
	BackupStorageId         string
	BackupStorageAccessInfo *jsonutils.JSONDict
	Metadata                api.DiskBackupPackMetadata
}

type SStoragePackInstanceBackup struct {
	PackageName             string
	BackupStorageId         string
	BackupStorageAccessInfo *jsonutils.JSONDict
	BackupIds               []string
	Metadata                api.InstanceBackupPackMetadata
}

type SStorageUnpackInstanceBackup struct {
	PackageName             string
	BackupStorageId         string
	BackupStorageAccessInfo *jsonutils.JSONDict
	MetadataOnly            *bool
}

type SStorageSaveToGlanceInfo struct {
	UserCred mcclient.TokenCredential
	DiskInfo *jsonutils.JSONDict
}
