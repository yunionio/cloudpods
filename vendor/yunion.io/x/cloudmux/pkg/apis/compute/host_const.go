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

const (
	HOST_TYPE_ESXI = "esxi" // # VMWare vSphere ESXi

	HOST_TYPE_ALIYUN         = "aliyun"
	HOST_TYPE_APSARA         = "apsara"
	HOST_TYPE_AWS            = "aws"
	HOST_TYPE_QCLOUD         = "qcloud"
	HOST_TYPE_AZURE          = "azure"
	HOST_TYPE_HUAWEI         = "huawei"
	HOST_TYPE_HCSO           = "hcso"
	HOST_TYPE_HCS            = "hcs"
	HOST_TYPE_HCSOP          = "hcsop"
	HOST_TYPE_OPENSTACK      = "openstack"
	HOST_TYPE_UCLOUD         = "ucloud"
	HOST_TYPE_ZSTACK         = "zstack"
	HOST_TYPE_GOOGLE         = "google"
	HOST_TYPE_CTYUN          = "ctyun"
	HOST_TYPE_ECLOUD         = "ecloud"
	HOST_TYPE_JDCLOUD        = "jdcloud"
	HOST_TYPE_CLOUDPODS      = "cloudpods"
	HOST_TYPE_NUTANIX        = "nutanix"
	HOST_TYPE_BINGO_CLOUD    = "bingocloud"
	HOST_TYPE_INCLOUD_SPHERE = "incloudsphere"
	HOST_TYPE_PROXMOX        = "proxmox"
	HOST_TYPE_REMOTEFILE     = "remotefile"
	HOST_TYPE_H3C            = "h3c"

	// # possible status
	HOST_ONLINE  = "online"
	HOST_OFFLINE = "offline"

	NIC_TYPE_IPMI  = "ipmi"
	NIC_TYPE_ADMIN = "admin"
	// #NIC_TYPE_NORMAL = 'normal'

	BAREMETAL_READY   = "ready"
	BAREMETAL_RUNNING = "running"
	BAREMETAL_UNKNOWN = "unknown"

	HOST_STATUS_RUNNING = BAREMETAL_RUNNING
	HOST_STATUS_READY   = BAREMETAL_READY
	HOST_STATUS_UNKNOWN = BAREMETAL_UNKNOWN
)
