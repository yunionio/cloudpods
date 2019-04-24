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

package tasks

import (
	"net"

	"yunion.io/x/jsonutils"

	baremetaltypes "yunion.io/x/onecloud/pkg/baremetal/types"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IBaremetal interface {
	GetId() string
	GetZoneId() string
	GetTaskQueue() *TaskQueue
	GetSSHConfig() (*types.SSHConfig, error)
	TestSSHConfig() bool
	GetAdminNic() *types.SNic
	GetName() string
	GetClientSession() *mcclient.ClientSession
	SaveDesc(desc jsonutils.JSONObject) error
	GetNicByMac(net.HardwareAddr) *types.SNic
	GetRawIPMIConfig() *types.SIPMIInfo
	GetIPMINic(mac net.HardwareAddr) *types.SNic
	SetExistingIPMIIPAddr(ipAddr string)
	GetServer() baremetaltypes.IBaremetalServer

	SyncStatus(status, reason string)
	AutoSyncStatus()
	SyncAllStatus(status string)
	AutoSyncAllStatus()

	GetPowerStatus() (string, error)
	DoPowerShutdown(soft bool) error
	DoPXEBoot() error
	// DoDiskBoot() error

	RemoveServer()
}
