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

package types

import (
	"net"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/baremetal/utils/disktool"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

type IBaremetalServer interface {
	GetName() string
	GetId() string
	RemoveDesc()
	DoDiskUnconfig(term *ssh.Client) error
	DoDiskConfig(term *ssh.Client) error
	DoEraseDisk(term *ssh.Client) error
	DoPartitionDisk(term *ssh.Client) ([]*disktool.Partition, error)
	DoRebuildRootDisk(term *ssh.Client) ([]*disktool.Partition, error)
	SyncPartitionSize(term *ssh.Client, parts []*disktool.Partition) ([]jsonutils.JSONObject, error)
	DoDeploy(term *ssh.Client, data jsonutils.JSONObject, isInit bool) (jsonutils.JSONObject, error)
	SaveDesc(desc jsonutils.JSONObject) error
	GetNicByMac(mac net.HardwareAddr) *types.SNic

	GetRootTemplateId() string
}
