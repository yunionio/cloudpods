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

	"yunion.io/x/onecloud/pkg/hostman/storageman"
)

type SBaseParms struct {
	Sid  string
	Body jsonutils.JSONObject
}

type SGuestDeploy struct {
	Sid    string
	Body   jsonutils.JSONObject
	IsInit bool
}

type SSrcPrepareMigrate struct {
	Sid         string
	LiveMigrate bool
}

type SDestPrepareMigrate struct {
	Sid             string
	ServerUrl       string
	QemuVersion     string
	SnapshotsUri    string
	DisksUri        string
	TargetStorageId string
	LiveMigrate     bool
	RebaseDisks     bool

	Desc             jsonutils.JSONObject
	DisksBackingFile jsonutils.JSONObject
	SrcSnapshots     jsonutils.JSONObject
}

type SLiveMigrate struct {
	Sid      string
	DestPort int
	DestIp   string
	IsLocal  bool
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
	Sid        string
	SnapshotId string
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
