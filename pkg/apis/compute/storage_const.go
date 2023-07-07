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
	"yunion.io/x/cloudmux/pkg/apis/compute"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	STORAGE_LOCAL     = compute.STORAGE_LOCAL
	STORAGE_BAREMETAL = "baremetal"
	STORAGE_SHEEPDOG  = "sheepdog"
	STORAGE_RBD       = compute.STORAGE_RBD
	STORAGE_DOCKER    = "docker"
	STORAGE_NAS       = compute.STORAGE_NAS
	STORAGE_VSAN      = "vsan"
	STORAGE_NFS       = compute.STORAGE_NFS
	STORAGE_GPFS      = "gpfs"
	STORAGE_CIFS      = compute.STORAGE_CIFS
	STORAGE_NVME_PT   = "nvme_pt" // nvme passthrough
	STORAGE_NVME      = "nvme"    // nvme sriov
	STORAGE_LVM       = "lvm"

	STORAGE_PUBLIC_CLOUD     = compute.STORAGE_PUBLIC_CLOUD
	STORAGE_CLOUD_EFFICIENCY = compute.STORAGE_CLOUD_EFFICIENCY
	STORAGE_CLOUD_SSD        = compute.STORAGE_CLOUD_SSD
	STORAGE_CLOUD_ESSD_PL0   = compute.STORAGE_CLOUD_ESSD_PL0 // 增强型(Enhanced)SSD 云盘, 单盘最高随机读写IOPS 1万
	STORAGE_CLOUD_ESSD       = compute.STORAGE_CLOUD_ESSD     // 增强型(Enhanced)SSD 云盘, 单盘最高随机读写IOPS 5万
	STORAGE_CLOUD_ESSD_PL2   = compute.STORAGE_CLOUD_ESSD_PL2 // 单盘最高随机读写IOPS 10万
	STORAGE_CLOUD_ESSD_PL3   = compute.STORAGE_CLOUD_ESSD_PL3 // 单盘最高随机读写IOPS 100万
	STORAGE_CLOUD_AUTO       = compute.STORAGE_CLOUD_AUTO     // ESSD AutoPL
	STORAGE_EPHEMERAL_SSD    = compute.STORAGE_EPHEMERAL_SSD  // 单块本地SSD盘, 容量最大不能超过800 GiB
	STORAGE_LOCAL_HDD_PRO    = compute.STORAGE_LOCAL_HDD_PRO  // 实例规格族d1ne和d1搭载的SATA HDD本地盘
	STORAGE_LOCAL_SSD_PRO    = compute.STORAGE_LOCAL_SSD_PRO  // 实例规格族i2、i2g、i1、ga1和gn5等搭载的NVMe

	//Azure hdd and ssd storagetype
	STORAGE_STANDARD_LRS          = compute.STORAGE_STANDARD_LRS
	STORAGE_STANDARDSSD_LRS       = compute.STORAGE_STANDARDSSD_LRS
	STORAGE_PREMIUM_LRS           = compute.STORAGE_PREMIUM_LRS
	STORAGE_AZURE_BASIC           = compute.STORAGE_AZURE_BASIC
	STORAGE_AZURE_GENERAL_PURPOSE = compute.STORAGE_AZURE_GENERAL_PURPOSE

	// aws storage type
	STORAGE_GP2_SSD      = compute.STORAGE_GP2_SSD      // aws general purpose ssd
	STORAGE_GP3_SSD      = compute.STORAGE_GP3_SSD      // aws General Purpose SSD (gp3)
	STORAGE_IO1_SSD      = compute.STORAGE_IO1_SSD      // aws Provisioned IOPS SSD
	STORAGE_IO2_SSD      = compute.STORAGE_IO2_SSD      // aws Provisioned IOPS 2 SSD
	STORAGE_ST1_HDD      = compute.STORAGE_ST1_HDD      // aws Throughput Optimized HDD
	STORAGE_SC1_HDD      = compute.STORAGE_SC1_HDD      // aws Cold HDD
	STORAGE_STANDARD_HDD = compute.STORAGE_STANDARD_HDD // aws Magnetic volumes

	// qcloud storage type
	// STORAGE_CLOUD_SSD ="cloud_ssd"
	STORAGE_LOCAL_BASIC   = compute.STORAGE_LOCAL_BASIC
	STORAGE_LOCAL_SSD     = compute.STORAGE_LOCAL_SSD
	STORAGE_LOCAL_PRO     = compute.STORAGE_LOCAL_PRO
	STORAGE_CLOUD_BASIC   = compute.STORAGE_CLOUD_BASIC
	STORAGE_CLOUD_PREMIUM = compute.STORAGE_CLOUD_PREMIUM //高性能云硬盘
	STORAGE_CLOUD_HSSD    = compute.STORAGE_CLOUD_HSSD    //增强型SSD云硬盘

	// huawei storage type
	STORAGE_HUAWEI_SSD   = compute.STORAGE_HUAWEI_SSD   // 超高IO云硬盘
	STORAGE_HUAWEI_SAS   = compute.STORAGE_HUAWEI_SAS   // 高IO云硬盘
	STORAGE_HUAWEI_SATA  = compute.STORAGE_HUAWEI_SATA  // 普通IO云硬盘
	STORAGE_HUAWEI_GPSSD = compute.STORAGE_HUAWEI_GPSSD // 通用型SSD
	STORAGE_HUAWEI_ESSD  = compute.STORAGE_HUAWEI_ESSD  // 急速型SSD

	// openstack
	STORAGE_OPENSTACK_ISCSI = compute.STORAGE_OPENSTACK_ISCSI
	STORAGE_OPENSTACK_NOVA  = compute.STORAGE_OPENSTACK_NOVA

	// Ucloud storage type
	STORAGE_UCLOUD_CLOUD_NORMAL         = compute.STORAGE_UCLOUD_CLOUD_NORMAL         // 普通云盘
	STORAGE_UCLOUD_CLOUD_SSD            = compute.STORAGE_UCLOUD_CLOUD_SSD            // SSD云盘
	STORAGE_UCLOUD_LOCAL_NORMAL         = compute.STORAGE_UCLOUD_LOCAL_NORMAL         // 普通本地盘
	STORAGE_UCLOUD_LOCAL_SSD            = compute.STORAGE_UCLOUD_LOCAL_SSD            // SSD本地盘
	STORAGE_UCLOUD_EXCLUSIVE_LOCAL_DISK = compute.STORAGE_UCLOUD_EXCLUSIVE_LOCAL_DISK // 独享本地盘

	// Zstack storage type
	STORAGE_ZSTACK_LOCAL_STORAGE = compute.STORAGE_ZSTACK_LOCAL_STORAGE
	STORAGE_ZSTACK_CEPH          = compute.STORAGE_ZSTACK_CEPH

	// Google storage type
	STORAGE_GOOGLE_LOCAL_SSD   = compute.STORAGE_GOOGLE_LOCAL_SSD   //本地SSD暂存盘 (最多8个)
	STORAGE_GOOGLE_PD_STANDARD = compute.STORAGE_GOOGLE_PD_STANDARD //标准永久性磁盘
	STORAGE_GOOGLE_PD_SSD      = compute.STORAGE_GOOGLE_PD_SSD      //SSD永久性磁盘
	STORAGE_GOOGLE_PD_BALANCED = compute.STORAGE_GOOGLE_PD_BALANCED //平衡永久性磁盘

	// ctyun storage type
	STORAGE_CTYUN_SSD  = compute.STORAGE_CTYUN_SSD  // 超高IO云硬盘
	STORAGE_CTYUN_SAS  = compute.STORAGE_CTYUN_SAS  // 高IO云硬盘
	STORAGE_CTYUN_SATA = compute.STORAGE_CTYUN_SATA // 普通IO云硬盘

	// jd cloud storage type
	STORAGE_JDCLOUD_GP1 = compute.STORAGE_JDCLOUD_GP1 // 通用型SSD云硬盘
	STORAGE_JDCLOUD_IO1 = compute.STORAGE_JDCLOUD_IO1 // 性能型SSD云硬盘
	STORAGE_JDCLOUD_STD = compute.STORAGE_JDCLOUD_STD // 容量型HDD云硬盘
	STORAGE_JDCLOUD_SSD = compute.STORAGE_JDCLOUD_SSD // SSD云硬盘
	STORAGE_JDCLOUD_PHD = compute.STORAGE_JDCLOUD_PHD // HDD云硬盘

	STORAGE_ECLOUD_CAPEBS = compute.STORAGE_ECLOUD_CAPEBS // 容量盘
	STORAGE_ECLOUD_EBS    = compute.STORAGE_ECLOUD_EBS    // 性能盘
	STORAGE_ECLOUD_SSD    = compute.STORAGE_ECLOUD_SSD    // 高性能盘
	STORAGE_ECLOUD_SSDEBS = compute.STORAGE_ECLOUD_SSDEBS // 性能优化盘
	STORAGE_ECLOUD_SYSTEM = compute.STORAGE_ECLOUD_SYSTEM // 系统盘
)

const (
	STORAGE_ENABLED = "enabled"
	// STORAGE_DISABLED = "disabled"
	STORAGE_OFFLINE = compute.STORAGE_OFFLINE // 离线
	STORAGE_ONLINE  = compute.STORAGE_ONLINE  // 在线
	STORAGE_UNMOUNT = "unmount"               // 待挂载

	DISK_TYPE_ROTATE = compute.DISK_TYPE_ROTATE
	DISK_TYPE_SSD    = compute.DISK_TYPE_SSD
	DISK_TYPE_HYBRID = compute.DISK_TYPE_HYBRID
)

const (
	RBD_DEFAULT_MON_TIMEOUT   = 5       //5 seconds 连接超时时间
	RBD_DEFAULT_OSD_TIMEOUT   = 20 * 60 //20 minute 操作超时时间
	RBD_DEFAULT_MOUNT_TIMEOUT = 2 * 60  //CephFS挂载超时时间, 目前未使用
)

var (
	DISK_TYPES          = []string{DISK_TYPE_ROTATE, DISK_TYPE_SSD, DISK_TYPE_HYBRID}
	STORAGE_LOCAL_TYPES = []string{
		STORAGE_LOCAL, STORAGE_NVME_PT, STORAGE_NVME, STORAGE_LVM,
		STORAGE_BAREMETAL, STORAGE_UCLOUD_LOCAL_NORMAL, STORAGE_UCLOUD_LOCAL_SSD, STORAGE_UCLOUD_EXCLUSIVE_LOCAL_DISK,
		STORAGE_EPHEMERAL_SSD, STORAGE_LOCAL_BASIC, STORAGE_LOCAL_SSD, STORAGE_LOCAL_PRO, STORAGE_OPENSTACK_NOVA,
		STORAGE_ZSTACK_LOCAL_STORAGE, STORAGE_GOOGLE_LOCAL_SSD,
	}
	STORAGE_SUPPORT_TYPES = STORAGE_LOCAL_TYPES
	STORAGE_ALL_TYPES     = []string{
		STORAGE_LOCAL, STORAGE_BAREMETAL, STORAGE_SHEEPDOG,
		STORAGE_RBD, STORAGE_DOCKER, STORAGE_NAS, STORAGE_VSAN,
		STORAGE_NFS, STORAGE_GPFS, STORAGE_CIFS, STORAGE_NVME_PT, STORAGE_NVME, STORAGE_LVM,
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
		STORAGE_NVME_PT, STORAGE_NVME, STORAGE_LVM,
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

	// filter storages which server can change disks to
	ServerId string `json:"server_id"`

	// filter storages of baremetal host
	IsBaremetal *bool `json:"is_baremetal"`
}
