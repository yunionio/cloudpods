package baremetal

import (
	"errors"

	"yunion.io/x/pkg/util/sets"
)

const (
	DISK_CONF_RAID0  = "raid0"
	DISK_CONF_RAID1  = "raid1"
	DISK_CONF_RAID5  = "raid5"
	DISK_CONF_RAID10 = "raid10"
	DISK_CONF_NONE   = "none"

	DEFAULT_DISK_CONF = DISK_CONF_NONE

	DISK_TYPE_ROTATE = "rotate"
	DISK_TYPE_SSD    = "ssd"
	DISK_TYPE_HYBRID = "hybrid"

	DEFAULT_DISK_TYPE = DISK_TYPE_ROTATE

	DISK_DRIVER_MEGARAID   = "MegaRaid"
	DISK_DRIVER_LINUX      = "Linux"
	DISK_DRIVER_HPSARAID   = "HPSARaid"
	DISK_DRIVER_MPT2SAS    = "Mpt2SAS"
	DISK_DRIVER_MARVELRAID = "MarvelRaid"
	DISK_DRIVER_PCIE       = "PCIE"

	HDD_DISK_SPEC_TYPE = "HDD"
	SSD_DISK_SPEC_TYPE = "SSD"
)

var (
	BaremetalDefaultDiskConfig = BaremetalDiskConfig{
		Type:  DISK_TYPE_HYBRID,
		Conf:  DISK_CONF_NONE,
		Count: 0,
	}

	DISK_CONFS = sets.NewString(
		DISK_CONF_RAID0,
		DISK_CONF_RAID1,
		DISK_CONF_RAID5,
		DISK_CONF_RAID10,
		DISK_CONF_NONE,
	)

	DISK_TYPES = sets.NewString(
		DISK_TYPE_ROTATE,
		DISK_TYPE_SSD,
		DISK_TYPE_HYBRID,
	)

	DISK_DRIVERS_RAID = sets.NewString(
		DISK_DRIVER_MEGARAID,
		DISK_DRIVER_HPSARAID,
		DISK_DRIVER_MPT2SAS,
		DISK_DRIVER_MARVELRAID,
	)

	DISK_DRIVERS = sets.NewString(
		DISK_DRIVER_LINUX,
		DISK_DRIVER_PCIE).Union(DISK_DRIVERS_RAID)
)

var (
	ErrMoreThanOneSizeUnspecificSplit = errors.New(`more than 1 size unspecific split`)
	ErrNoMoreSpaceForUnspecificSplit  = errors.New(`no more space for an unspecific split`)
	ErrSubtotalOfSplitExceedsDiskSize = errors.New(`subtotal of split exceeds disk size`)
)

type BaremetalStorage struct {
	Slot         int    `json:"slot"`
	Status       string `json:"status"`
	Rotate       bool   `json:"rotate"`
	Adapter      int    `json:"adapter"`
	Driver       string `json:"driver"`
	Model        string `json:"model"`
	Enclosure    int    `json:"enclousure"`
	Size         int64  `json:"size"`
	MinStripSize int64  `json:"min_strip_size,omitempty"`
	MaxStripSize int64  `json:"max_strip_size,omitempty"`
	Index        int64  `json:"index"`
}

type BaremetalDiskConfig struct {
	// disk type
	Type string `json:"type"`
	// raid config
	Conf         string  `json:"conf"`
	Count        int64   `json:"count"`
	Range        []int64 `json:"range"`
	Splits       string  `json:"splits"`
	Adapter      *int    `json:"adapter"`
	Driver       string  `json:"driver"`
	Cachedbadbbu bool    `json:"cachedbadbbu"`
	Strip        int64   `json:"strip"`
	RA           bool    `json:"ra"`
	WT           bool    `json:"wt"`
	Direct       bool    `json:"direct"`
}

type Disk struct {
	Backend         string  `json:"backend"`
	ImageID         string  `json:"image_id"`
	Fs              *string `json:"fs"`
	Os              string  `json:"os"`
	OSDistribution  string  `json:"os_distribution"`
	OsVersion       string  `json:"os_version"`
	Format          string  `json:"format"`
	MountPoint      *string `json:"mountpoint"`
	Driver          *string `json:"driver"`
	Cache           *string `json:"cache"`
	ImageDiskFormat string  `json:"image_disk_format"`
	Size            int64   `json:"size"`
	Storage         *string `json:"storage"`
}
