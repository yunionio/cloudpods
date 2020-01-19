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

package models

import (
	"time"

	"yunion.io/x/jsonutils"
)

type Host struct {
	EnabledStatusStandaloneResource

	Rack  string
	Slots string

	AccessMac  string
	AccessIp   string
	ManagerUri string

	SysInfo jsonutils.JSONObject
	SN      string

	CpuCount    int
	NodeCount   int8
	CpuDesc     string
	CpuMhz      int
	CpuCache    int
	CpuReserved int
	CpuCmtbound float32

	MemSize     int
	MemReserved int
	MemCmtbound float32

	StorageSize   int
	StorageType   string
	StorageDriver string
	StorageInfo   jsonutils.JSONObject

	IpmiInfo jsonutils.JSONObject

	HostStatus string

	ZoneId string

	HostType string

	Version string

	IsBaremetal bool

	IsMaintenance bool

	LastPingAt time.Time

	ResourceType string

	RealExternalId string

	IsImport bool
}
