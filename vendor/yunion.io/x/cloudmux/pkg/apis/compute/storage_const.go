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
	STORAGE_LOCAL = "local"
	STORAGE_RBD   = "rbd"
	STORAGE_NAS   = "nas"
	STORAGE_VSAN  = "vsan"
	STORAGE_NFS   = "nfs"
	STORAGE_CIFS  = "cifs"

	STORAGE_PUBLIC_CLOUD     = "cloud"
	STORAGE_CLOUD_EFFICIENCY = "cloud_efficiency"
	STORAGE_CLOUD_SSD        = "cloud_ssd"
	STORAGE_CLOUD_ESSD_PL0   = "cloud_essd_pl0" // 增强型(Enhanced)SSD 云盘, 单盘最高随机读写IOPS 1万
	STORAGE_CLOUD_ESSD       = "cloud_essd"     // 增强型(Enhanced)SSD 云盘, 单盘最高随机读写IOPS 5万
	STORAGE_CLOUD_ESSD_PL2   = "cloud_essd_pl2" // 单盘最高随机读写IOPS 10万
	STORAGE_CLOUD_ESSD_PL3   = "cloud_essd_pl3" // 单盘最高随机读写IOPS 100万
	STORAGE_CLOUD_AUTO       = "cloud_auto"     // ESSD AutoPL
	STORAGE_EPHEMERAL_SSD    = "ephemeral_ssd"  // 单块本地SSD盘, 容量最大不能超过800 GiB
	STORAGE_LOCAL_HDD_PRO    = "local_hdd_pro"  // 实例规格族d1ne和d1搭载的SATA HDD本地盘
	STORAGE_LOCAL_SSD_PRO    = "local_ssd_pro"  // 实例规格族i2、i2g、i1、ga1和gn5等搭载的NVMe

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
	STORAGE_HUAWEI_ESSD  = "ESSD"  // 急速型SSD

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

	// jd cloud storage type
	STORAGE_JDCLOUD_GP1 = "ssd.gp1"     // 通用型SSD云硬盘
	STORAGE_JDCLOUD_IO1 = "ssd.io1"     // 性能型SSD云硬盘
	STORAGE_JDCLOUD_STD = "hdd.std1"    // 容量型HDD云硬盘
	STORAGE_JDCLOUD_SSD = "ssd"         // SSD云硬盘
	STORAGE_JDCLOUD_PHD = "premium-hdd" // HDD云硬盘

	STORAGE_ECLOUD_CAPEBS = "capebs" // 容量盘
	STORAGE_ECLOUD_EBS    = "ebs"    // 性能盘
	STORAGE_ECLOUD_SSD    = "ssd"    // 高性能盘
	STORAGE_ECLOUD_SSDEBS = "ssdebs" // 性能优化盘
	STORAGE_ECLOUD_SYSTEM = "system" // 系统盘
)

const (
	STORAGE_OFFLINE = "offline" // 离线
	STORAGE_ONLINE  = "online"  // 在线

	DISK_TYPE_ROTATE = "rotate"
	DISK_TYPE_SSD    = "ssd"
	DISK_TYPE_HYBRID = "hybrid"
)
