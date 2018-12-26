package sysutils

const (
	STORAGE_LOCAL     = "local"
	STORAGE_BAREMETAL = "baremetal"
	STORAGE_SHEEPDOG  = "sheepdog"
	STORAGE_RBD       = "rbd"
	STORAGE_DOCKER    = "docker"
	STORAGE_NAS       = "nas"
	STORAGE_VSAN      = "vsan"

	STORAGE_CLOUD            = "cloud"
	STORAGE_CLOUD_SSD        = "cloud_ssd"
	STORAGE_CLOUD_ESSD       = "cloud_essd" //增强型(Enhanced)SSD 云盘
	STORAGE_CLOUD_EFFICIENCY = "cloud_efficiency"

	//Azure hdd and ssd storagetype
	STORAGE_STANDARD_GRS   = "standard_grs"
	STORAGE_STANDARD_LRS   = "standard_lrs"
	STORAGE_STANDARD_RAGRS = "standard_ragrs"
	STORAGE_STANDARD_ZRS   = "standard_zrs"
	STORAGE_PREMIUM_LRS    = "premium_lrs"

	// aws storage type
	STORAGE_GP2_SSD      = "gp2"      // aws general purpose ssd
	STORAGE_IO1_SSD      = "io1"      // aws Provisioned IOPS SSD
	STORAGE_ST1_HDD      = "st1"      // aws Throughput Optimized HDD
	STORAGE_SC1_SSD      = "sc1"      // aws Cold HDD
	STORAGE_STANDARD_SSD = "standard" // aws Magnetic volumes

	// qcloud storage type
	// STORAGE_CLOUD_SSD ="cloud_ssd"
	STORAGE_LOCAL_BASIC   = "local_basic"
	STORAGE_LOCAL_SSD     = "local_ssd"
	STORAGE_CLOUD_BASIC   = "cloud_basic"
	STORAGE_CLOUD_PERMIUM = "cloud_permium"
)

var STORAGE_TYPES = []string{STORAGE_LOCAL, STORAGE_BAREMETAL, STORAGE_SHEEPDOG,
	STORAGE_RBD, STORAGE_DOCKER, STORAGE_NAS, STORAGE_VSAN,
	STORAGE_CLOUD, STORAGE_CLOUD_SSD, STORAGE_CLOUD_ESSD, STORAGE_CLOUD_EFFICIENCY,
	STORAGE_STANDARD_GRS, STORAGE_STANDARD_LRS, STORAGE_STANDARD_RAGRS, STORAGE_STANDARD_ZRS, STORAGE_PREMIUM_LRS,
	STORAGE_GP2_SSD, STORAGE_IO1_SSD, STORAGE_ST1_HDD, STORAGE_SC1_SSD, STORAGE_STANDARD_SSD,
	STORAGE_LOCAL_BASIC, STORAGE_LOCAL_SSD, STORAGE_CLOUD_BASIC, STORAGE_CLOUD_PERMIUM,
}

var LOCAL_STORAGE_TYPES = []string{STORAGE_LOCAL, STORAGE_BAREMETAL, STORAGE_LOCAL_BASIC, STORAGE_LOCAL_SSD}

// var SUPPORT = []string {support = [STORAGE_LOCAL, STORAGE_BAREMETAL, STORAGE_NAS]
