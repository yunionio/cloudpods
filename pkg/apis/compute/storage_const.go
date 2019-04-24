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
	STORAGE_LOCAL     = "local"
	STORAGE_BAREMETAL = "baremetal"
	STORAGE_SHEEPDOG  = "sheepdog"
	STORAGE_RBD       = "rbd"
	STORAGE_DOCKER    = "docker"
	STORAGE_NAS       = "nas"
	STORAGE_VSAN      = "vsan"
	STORAGE_NFS       = "nfs"

	STORAGE_PUBLIC_CLOUD     = "cloud"
	STORAGE_CLOUD_EFFICIENCY = "cloud_efficiency"
	STORAGE_CLOUD_SSD        = "cloud_ssd"
	STORAGE_CLOUD_ESSD       = "cloud_essd"    //增强型(Enhanced)SSD 云盘
	STORAGE_EPHEMERAL_SSD    = "ephemeral_ssd" //单块本地SSD盘, 容量最大不能超过800 GiB

	//Azure hdd and ssd storagetype
	STORAGE_STANDARD_LRS    = "standard_lrs"
	STORAGE_STANDARDSSD_LRS = "standardssd_lrs"
	STORAGE_PREMIUM_LRS     = "premium_lrs"

	// aws storage type
	STORAGE_GP2_SSD      = "gp2"      // aws general purpose ssd
	STORAGE_IO1_SSD      = "io1"      // aws Provisioned IOPS SSD
	STORAGE_ST1_HDD      = "st1"      // aws Throughput Optimized HDD
	STORAGE_SC1_HDD      = "sc1"      // aws Cold HDD
	STORAGE_STANDARD_HDD = "standard" // aws Magnetic volumes

	// qcloud storage type
	// STORAGE_CLOUD_SSD ="cloud_ssd"
	STORAGE_LOCAL_BASIC   = "local_basic"
	STORAGE_LOCAL_SSD     = "local_ssd"
	STORAGE_CLOUD_BASIC   = "cloud_basic"
	STORAGE_CLOUD_PREMIUM = "cloud_premium"

	// huawei storage type
	STORAGE_HUAWEI_SSD  = "SSD"  // 超高IO云硬盘
	STORAGE_HUAWEI_SAS  = "SAS"  // 高IO云硬盘
	STORAGE_HUAWEI_SATA = "SATA" // 普通IO云硬盘

	// openstack
	STORAGE_OPENSTACK_ISCSI = "iscsi"

	// Ucloud storage type
	STORAGE_UCLOUD_CLOUD_NORMAL         = "CLOUD_NORMAL"         // 普通云盘
	STORAGE_UCLOUD_CLOUD_SSD            = "CLOUD_SSD"            // SSD云盘
	STORAGE_UCLOUD_LOCAL_NORMAL         = "LOCAL_NORMAL"         // 普通本地盘
	STORAGE_UCLOUD_LOCAL_SSD            = "LOCAL_SSD"            // SSD本地盘
	STORAGE_UCLOUD_EXCLUSIVE_LOCAL_DISK = "EXCLUSIVE_LOCAL_DISK" // 独享本地盘
)

const (
	STORAGE_ENABLED = "enabled"
	// STORAGE_DISABLED = "disabled"
	STORAGE_OFFLINE = "offline"
	STORAGE_ONLINE  = "online"

	DISK_TYPE_ROTATE = "rotate"
	DISK_TYPE_SSD    = "ssd"
	DISK_TYPE_HYBRID = "hybrid"
)

var (
	DISK_TYPES            = []string{DISK_TYPE_ROTATE, DISK_TYPE_SSD, DISK_TYPE_HYBRID}
	STORAGE_LOCAL_TYPES   = []string{STORAGE_LOCAL, STORAGE_BAREMETAL}
	STORAGE_SUPPORT_TYPES = STORAGE_LOCAL_TYPES
	STORAGE_ALL_TYPES     = []string{
		STORAGE_LOCAL, STORAGE_BAREMETAL, STORAGE_SHEEPDOG,
		STORAGE_RBD, STORAGE_DOCKER, STORAGE_NAS, STORAGE_VSAN,
		STORAGE_NFS,
	}
	STORAGE_TYPES = []string{STORAGE_LOCAL, STORAGE_BAREMETAL, STORAGE_SHEEPDOG,
		STORAGE_RBD, STORAGE_DOCKER, STORAGE_NAS, STORAGE_VSAN, STORAGE_NFS,
		STORAGE_PUBLIC_CLOUD, STORAGE_CLOUD_SSD, STORAGE_CLOUD_ESSD, STORAGE_EPHEMERAL_SSD, STORAGE_CLOUD_EFFICIENCY,
		STORAGE_STANDARD_LRS, STORAGE_STANDARDSSD_LRS, STORAGE_PREMIUM_LRS,
		STORAGE_GP2_SSD, STORAGE_IO1_SSD, STORAGE_ST1_HDD, STORAGE_SC1_HDD, STORAGE_STANDARD_HDD,
		STORAGE_LOCAL_BASIC, STORAGE_LOCAL_SSD, STORAGE_CLOUD_BASIC, STORAGE_CLOUD_PREMIUM,
		STORAGE_HUAWEI_SSD, STORAGE_HUAWEI_SAS, STORAGE_HUAWEI_SATA,
		STORAGE_OPENSTACK_ISCSI, STORAGE_UCLOUD_CLOUD_NORMAL, STORAGE_UCLOUD_CLOUD_SSD,
		STORAGE_UCLOUD_LOCAL_NORMAL, STORAGE_UCLOUD_LOCAL_SSD, STORAGE_UCLOUD_EXCLUSIVE_LOCAL_DISK,
	}

	STORAGE_LIMITED_TYPES = []string{STORAGE_LOCAL, STORAGE_BAREMETAL, STORAGE_NAS, STORAGE_RBD, STORAGE_NFS}
)
