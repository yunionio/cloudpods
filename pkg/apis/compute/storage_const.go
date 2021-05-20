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
	"yunion.io/x/onecloud/pkg/apis"
)

const (
	STORAGE_LOCAL     = "local"
	STORAGE_BAREMETAL = "baremetal"
	STORAGE_SHEEPDOG  = "sheepdog"
	STORAGE_RBD       = "rbd"
	STORAGE_DOCKER    = "docker"
	STORAGE_NAS       = "nas"
	STORAGE_VSAN      = "vsan"
	STORAGE_NFS       = "nfs"
	STORAGE_GPFS      = "gpfs"
	STORAGE_CIFS      = "cifs"

	STORAGE_PUBLIC_CLOUD     = "cloud"
	STORAGE_CLOUD_EFFICIENCY = "cloud_efficiency"
	STORAGE_CLOUD_SSD        = "cloud_ssd"
	STORAGE_CLOUD_ESSD       = "cloud_essd"     //增强型(Enhanced)SSD 云盘, 单盘最高随机读写IOPS 5万
	STORAGE_CLOUD_ESSD_PL2   = "cloud_essd_pl2" //单盘最高随机读写IOPS 10万
	STORAGE_CLOUD_ESSD_PL3   = "cloud_essd_pl3" //单盘最高随机读写IOPS 100万
	STORAGE_EPHEMERAL_SSD    = "ephemeral_ssd"  //单块本地SSD盘, 容量最大不能超过800 GiB

	//Azure hdd and ssd storagetype
	STORAGE_STANDARD_LRS          = "standard_lrs"
	STORAGE_STANDARDSSD_LRS       = "standardssd_lrs"
	STORAGE_PREMIUM_LRS           = "premium_lrs"
	STORAGE_AZURE_BASIC           = "basic_storage"
	STORAGE_AZURE_GENERAL_PURPOSE = "general_purpose_storage"

	// aws storage type
	STORAGE_GP2_SSD      = "gp2"      // aws general purpose ssd
	STORAGE_GP3_SSD      = "gp3"      // aws General Purpose SSD (gp3)
	STORAGE_IO1_SSD      = "io1"      // aws Provisioned IOPS SSD
	STORAGE_IO2_SSD      = "io2"      // aws Provisioned IOPS 2 SSD
	STORAGE_ST1_HDD      = "st1"      // aws Throughput Optimized HDD
	STORAGE_SC1_HDD      = "sc1"      // aws Cold HDD
	STORAGE_STANDARD_HDD = "standard" // aws Magnetic volumes

	// qcloud storage type
	// STORAGE_CLOUD_SSD ="cloud_ssd"
	STORAGE_LOCAL_BASIC   = "local_basic"
	STORAGE_LOCAL_SSD     = "local_ssd"
	STORAGE_LOCAL_PRO     = "local_pro"
	STORAGE_CLOUD_BASIC   = "cloud_basic"
	STORAGE_CLOUD_PREMIUM = "cloud_premium" //高性能云硬盘
	STORAGE_CLOUD_HSSD    = "cloud_hssd"    //增强型SSD云硬盘

	// huawei storage type
	STORAGE_HUAWEI_SSD   = "SSD"   // 超高IO云硬盘
	STORAGE_HUAWEI_SAS   = "SAS"   // 高IO云硬盘
	STORAGE_HUAWEI_SATA  = "SATA"  // 普通IO云硬盘
	STORAGE_HUAWEI_GPSSD = "GPSSD" // 通用型SSD

	// openstack
	STORAGE_OPENSTACK_ISCSI = "iscsi"
	STORAGE_OPENSTACK_NOVA  = "nova"

	// Ucloud storage type
	STORAGE_UCLOUD_CLOUD_NORMAL         = "CLOUD_NORMAL"         // 普通云盘
	STORAGE_UCLOUD_CLOUD_SSD            = "CLOUD_SSD"            // SSD云盘
	STORAGE_UCLOUD_LOCAL_NORMAL         = "LOCAL_NORMAL"         // 普通本地盘
	STORAGE_UCLOUD_LOCAL_SSD            = "LOCAL_SSD"            // SSD本地盘
	STORAGE_UCLOUD_EXCLUSIVE_LOCAL_DISK = "EXCLUSIVE_LOCAL_DISK" // 独享本地盘

	// Zstack storage type
	STORAGE_ZSTACK_LOCAL_STORAGE = "localstorage"
	STORAGE_ZSTACK_CEPH          = "ceph"

	// Google storage type
	STORAGE_GOOGLE_LOCAL_SSD   = "local-ssd"   //本地SSD暂存盘 (最多8个)
	STORAGE_GOOGLE_PD_STANDARD = "pd-standard" //标准永久性磁盘
	STORAGE_GOOGLE_PD_SSD      = "pd-ssd"      //SSD永久性磁盘
	STORAGE_GOOGLE_PD_BALANCED = "pd-balanced" //平衡永久性磁盘

	// ctyun storage type
	STORAGE_CTYUN_SSD  = "SSD"  // 超高IO云硬盘
	STORAGE_CTYUN_SAS  = "SAS"  // 高IO云硬盘
	STORAGE_CTYUN_SATA = "SATA" // 普通IO云硬盘

	STORAGE_ECLOUD_CAPEBS = "capebs" // 容量盘
	STORAGE_ECLOUD_EBS    = "ebs"    // 性能盘
	STORAGE_ECLOUD_SSD    = "ssd"    // 高性能盘
	STORAGE_ECLOUD_SSDEBS = "ssdebs" // 性能优化盘
	STORAGE_ECLOUD_SYSTEM = "system" // 系统盘
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

const (
	RBD_DEFAULT_MON_TIMEOUT   = 5       //5 seconds 连接超时时间
	RBD_DEFAULT_OSD_TIMEOUT   = 20 * 60 //20 minute 操作超时时间
	RBD_DEFAULT_MOUNT_TIMEOUT = 2 * 60  //CephFS挂载超时时间, 目前未使用
)

var (
	DISK_TYPES          = []string{DISK_TYPE_ROTATE, DISK_TYPE_SSD, DISK_TYPE_HYBRID}
	STORAGE_LOCAL_TYPES = []string{STORAGE_LOCAL, STORAGE_BAREMETAL, STORAGE_UCLOUD_LOCAL_NORMAL, STORAGE_UCLOUD_LOCAL_SSD, STORAGE_UCLOUD_EXCLUSIVE_LOCAL_DISK,
		STORAGE_EPHEMERAL_SSD, STORAGE_LOCAL_BASIC, STORAGE_LOCAL_SSD, STORAGE_LOCAL_PRO, STORAGE_OPENSTACK_NOVA,
		STORAGE_ZSTACK_LOCAL_STORAGE, STORAGE_GOOGLE_LOCAL_SSD}
	STORAGE_SUPPORT_TYPES = STORAGE_LOCAL_TYPES
	STORAGE_ALL_TYPES     = []string{
		STORAGE_LOCAL, STORAGE_BAREMETAL, STORAGE_SHEEPDOG,
		STORAGE_RBD, STORAGE_DOCKER, STORAGE_NAS, STORAGE_VSAN,
		STORAGE_NFS, STORAGE_GPFS, STORAGE_CIFS,
	}
	STORAGE_TYPES = []string{STORAGE_LOCAL, STORAGE_BAREMETAL, STORAGE_SHEEPDOG,
		STORAGE_RBD, STORAGE_DOCKER, STORAGE_NAS, STORAGE_VSAN, STORAGE_NFS,
		STORAGE_PUBLIC_CLOUD, STORAGE_CLOUD_SSD, STORAGE_CLOUD_ESSD, STORAGE_CLOUD_ESSD_PL2, STORAGE_CLOUD_ESSD_PL3,
		STORAGE_EPHEMERAL_SSD, STORAGE_CLOUD_EFFICIENCY,
		STORAGE_STANDARD_LRS, STORAGE_STANDARDSSD_LRS, STORAGE_PREMIUM_LRS,
		STORAGE_GP2_SSD, STORAGE_GP3_SSD, STORAGE_IO1_SSD, STORAGE_ST1_HDD, STORAGE_SC1_HDD, STORAGE_STANDARD_HDD,
		STORAGE_LOCAL_BASIC, STORAGE_LOCAL_SSD, STORAGE_CLOUD_BASIC, STORAGE_CLOUD_PREMIUM,
		STORAGE_HUAWEI_SSD, STORAGE_HUAWEI_SAS, STORAGE_HUAWEI_SATA,
		STORAGE_OPENSTACK_ISCSI, STORAGE_UCLOUD_CLOUD_NORMAL, STORAGE_UCLOUD_CLOUD_SSD,
		STORAGE_UCLOUD_LOCAL_NORMAL, STORAGE_UCLOUD_LOCAL_SSD, STORAGE_UCLOUD_EXCLUSIVE_LOCAL_DISK,
		STORAGE_ZSTACK_LOCAL_STORAGE, STORAGE_ZSTACK_CEPH, STORAGE_GPFS, STORAGE_CIFS,
	}

	HOST_STORAGE_LOCAL_TYPES = []string{STORAGE_LOCAL, STORAGE_BAREMETAL, STORAGE_ZSTACK_LOCAL_STORAGE, STORAGE_OPENSTACK_NOVA}

	STORAGE_LIMITED_TYPES = []string{STORAGE_LOCAL, STORAGE_BAREMETAL, STORAGE_NAS, STORAGE_RBD, STORAGE_NFS, STORAGE_GPFS, STORAGE_VSAN, STORAGE_CIFS}

	SHARED_FILE_STORAGE = []string{STORAGE_NFS, STORAGE_GPFS}
	FIEL_STORAGE        = []string{STORAGE_LOCAL, STORAGE_NFS, STORAGE_GPFS}

	// 目前来说只支持这些
	SHARED_STORAGE = []string{STORAGE_NFS, STORAGE_GPFS, STORAGE_RBD}
)

func IsDiskTypeMatch(t1, t2 string) bool {
	switch t1 {
	case DISK_TYPE_ROTATE:
		if t2 == DISK_TYPE_SSD {
			return false
		} else {
			return true
		}
	case DISK_TYPE_SSD:
		if t2 == DISK_TYPE_ROTATE {
			return false
		} else {
			return true
		}
	default:
		return true
	}
}

type StorageResourceInput struct {
	// 存储（ID或Name）
	StorageId string `json:"storage_id"`
	// swagger:ignore
	// Deprecated
	// filter by storage_id
	Storage string `json:"storage" yunion-deprecated-by:"storage_id"`
}

type StorageFilterListInputBase struct {
	StorageResourceInput

	// 以存储名称排序
	// pattern:asc|desc
	OrderByStorage string `json:"order_by_storage"`
}

type StorageFilterListInput struct {
	StorageFilterListInputBase

	StorageShareFilterListInput

	ZonalFilterListInput
	ManagedResourceListInput
}

type StorageShareFilterListInput struct {
	// filter shared storage
	Share *bool `json:"share"`
	// filter local storage
	Local *bool `json:"local"`
}

type StorageListInput struct {
	apis.EnabledStatusInfrasResourceBaseListInput
	apis.ExternalizedResourceBaseListInput
	SchedtagResourceInput

	ManagedResourceListInput
	ZonalFilterListInput

	UsableResourceListInput
	StorageShareFilterListInput

	// filter by host schedtag
	HostSchedtagId string `json:"host_schedtag_id"`

	// filter by cachedimage
	ImageId string `json:"image_id"`

	// filter storages which attached the specified host
	HostId string `json:"host_id"`
}
