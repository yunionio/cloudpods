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
	"context"
	"net"

	"yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/baremetal/pxe"
	baremetaltypes "yunion.io/x/onecloud/pkg/baremetal/types"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

type IBaremetal interface {
	Keyword() string

	GetId() string
	GetZoneId() string
	GetStorageCacheId() string
	GetTaskQueue() *TaskQueue
	GetSSHConfig() (*types.SSHConfig, error)
	TestSSHConfig() bool
	GetAdminNic() *types.SNic
	GetName() string
	GetClientSession() *mcclient.ClientSession
	SaveDesc(ctx context.Context, desc jsonutils.JSONObject) error
	GetNics() []types.SNic
	GetNicByMac(net.HardwareAddr) *types.SNic
	GetRawIPMIConfig() *types.SIPMIInfo
	GetIPMINic(mac net.HardwareAddr) *types.SNic
	SetExistingIPMIIPAddr(ipAddr string)
	GetServer() baremetaltypes.IBaremetalServer

	SyncStatus(ctx context.Context, status, reason string)
	AutoSyncStatus(ctx context.Context)
	SyncAllStatus(ctx context.Context, status string)
	AutoSyncAllStatus(ctx context.Context)

	GetPowerStatus() (string, error)
	DoPowerShutdown(soft bool) error
	DoPXEBoot() error
	// DoDiskBoot() error

	DoRedfishPowerOn() error
	GetAccessIp() string
	EnablePxeBoot() bool
	GenerateBootISO() error
	SendNicInfo(ctx context.Context, nic *types.SNicDevInfo, idx int, nicType compute.TNicType, reset bool, ipAddr string, reserve bool) error
	DoNTPConfig() error
	GetImageUrl(needImageCache bool) string

	RemoveServer()
	InitializeServer(session *mcclient.ClientSession, name string) error
	SaveSSHConfig(remoteAddr string, key string) error
	ServerLoadDesc(ctx context.Context) error
	GetDHCPServerIP() (net.IP, error)

	HasBMC() bool
	SSHReachable() (bool, error)
	SSHReboot(ctx context.Context) error
	SSHShutdown() error
	AdjustUEFICurrentBootOrder(ctx context.Context, cli *ssh.Client) error
}

type IBmManager interface {
	GetZoneId() string
	AddBaremetal(context.Context, jsonutils.JSONObject) (pxe.IBaremetalInstance, error)
	GetClientSession() *mcclient.ClientSession
}
