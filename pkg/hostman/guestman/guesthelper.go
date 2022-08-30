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

package guestman

import (
	"yunion.io/x/jsonutils"

	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/multicloud/esxi/vcenter"
)

type SBaseParms struct {
	Sid  string
	Body jsonutils.JSONObject
}

type SGuestDeploy struct {
	UserCred mcclient.TokenCredential

	Sid    string
	Body   jsonutils.JSONObject
	IsInit bool
}

type SSrcPrepareMigrate struct {
	Sid               string
	LiveMigrate       bool
	LiveMigrateUseTLS bool
}

type SDestPrepareMigrate struct {
	Sid               string
	ServerUrl         string
	QemuVersion       string
	SourceQemuCmdline string
	MigrateCerts      map[string]string
	EnableTLS         bool
	SnapshotsUri      string
	DisksUri          string
	// TargetStorageId string
	TargetStorageIds []string
	LiveMigrate      bool
	RebaseDisks      bool

	Desc             jsonutils.JSONObject
	DisksBackingFile jsonutils.JSONObject
	SrcSnapshots     jsonutils.JSONObject

	MemorySnapshotsUri string
	SrcMemorySnapshots []string

	UserCred mcclient.TokenCredential
}

type SLiveMigrate struct {
	Sid       string
	DestPort  int
	DestIp    string
	IsLocal   bool
	EnableTLS bool
}

type SDriverMirror struct {
	Sid          string
	NbdServerUri string
	Desc         jsonutils.JSONObject
}

type SGuestHotplugCpuMem struct {
	Sid         string
	AddCpuCount int64
	AddMemSize  int64
}

type SReloadDisk struct {
	Sid  string
	Disk storageman.IDisk
}

type SDiskSnapshot struct {
	UserCred   mcclient.TokenCredential
	Sid        string
	SnapshotId string
	Disk       storageman.IDisk
}

type SMemorySnapshot struct {
	*hostapi.GuestMemorySnapshotRequest
	Sid string
}

type SMemorySnapshotReset struct {
	*hostapi.GuestMemorySnapshotResetRequest
	Sid string
}

type SMemorySnapshotDelete struct {
	*hostapi.GuestMemorySnapshotDeleteRequest
}

type SDiskBackup struct {
	Sid        string
	SnapshotId string
	BackupId   string
	Disk       storageman.IDisk
}

type SDeleteDiskSnapshot struct {
	Sid             string
	DeleteSnapshot  string
	Disk            storageman.IDisk
	ConvertSnapshot string
	PendingDelete   bool
}

type SLibvirtServer struct {
	Uuid  string
	MacIp map[string]string
}

type SLibvirtDomainImportConfig struct {
	LibvritDomainXmlDir string
	Servers             []SLibvirtServer
}

type SGuestCreateFromLibvirt struct {
	Sid         string
	MonitorPath string
	GuestDesc   *jsonutils.JSONDict
	DisksPath   *jsonutils.JSONDict
}

type SGuestIoThrottle struct {
	Sid  string
	BPS  int64
	IOPS int64
}

type SGuestCreateFromEsxi struct {
	Sid            string
	GuestDesc      *jsonutils.JSONDict
	EsxiAccessInfo SEsxiAccessInfo
}

type SEsxiAccessInfo struct {
	Datastore  vcenter.SVCenterAccessInfo
	HostIp     string
	GuestExtId string
}

type SQgaGuestSetPassword struct {
	*hostapi.GuestSetPasswordRequest
	Sid string
}
