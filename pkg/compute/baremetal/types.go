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

package baremetal

import (
	"errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

const (
	DISK_CONF_RAID0  = api.DISK_CONF_RAID0
	DISK_CONF_RAID1  = api.DISK_CONF_RAID1
	DISK_CONF_RAID5  = api.DISK_CONF_RAID5
	DISK_CONF_RAID10 = api.DISK_CONF_RAID10
	DISK_CONF_NONE   = api.DISK_CONF_NONE

	DEFAULT_DISK_CONF = api.DEFAULT_DISK_CONF

	DISK_TYPE_ROTATE = api.DISK_TYPE_ROTATE
	DISK_TYPE_SSD    = api.DISK_TYPE_SSD
	DISK_TYPE_HYBRID = api.DISK_TYPE_HYBRID

	DEFAULT_DISK_TYPE = api.DEFAULT_DISK_TYPE

	DISK_DRIVER_MEGARAID   = api.DISK_DRIVER_MEGARAID
	DISK_DRIVER_LINUX      = api.DISK_DRIVER_LINUX
	DISK_DRIVER_HPSARAID   = api.DISK_DRIVER_HPSARAID
	DISK_DRIVER_MPT2SAS    = api.DISK_DRIVER_MPT2SAS
	DISK_DRIVER_MARVELRAID = api.DISK_DRIVER_MARVELRAID
	DISK_DRIVER_PCIE       = api.DISK_DRIVER_PCIE

	HDD_DISK_SPEC_TYPE = api.HDD_DISK_SPEC_TYPE
	SSD_DISK_SPEC_TYPE = api.SSD_DISK_SPEC_TYPE
)

var (
	BaremetalDefaultDiskConfig = api.BaremetalDefaultDiskConfig

	DISK_CONFS = api.DISK_CONFS

	DISK_TYPES = api.DISK_TYPES

	DISK_DRIVERS_RAID = api.DISK_DRIVERS_RAID

	DISK_DRIVERS = api.DISK_DRIVERS
)

var (
	ErrMoreThanOneSizeUnspecificSplit = errors.New(`more than 1 size unspecific split`)
	ErrNoMoreSpaceForUnspecificSplit  = errors.New(`no more space for an unspecific split`)
	ErrSubtotalOfSplitExceedsDiskSize = errors.New(`subtotal of split exceeds disk size`)
)

type BaremetalStorage struct {
	Size         int64  `json:"size,allowzero"`
	Driver       string `json:"driver"`
	Rotate       bool   `json:"rotate,allowfalse"`
	Dev          string `json:"dev,omitempty"`
	Sector       int64  `json:"sector,omitempty"`
	Block        int64  `json:"block,omitempty"`
	ModuleInfo   string `json:"module,omitempty"`
	Kernel       string `json:"kernel,omitempty"`
	PCIClass     string `json:"pci_class,omitempty"`
	Slot         int    `json:"slot,allowzero"`
	Status       string `json:"status,omitempty"`
	Adapter      int    `json:"adapter,allowzero"`
	Model        string `json:"model,omitempty"`
	Enclosure    int    `json:"enclousure,allowzero"`
	MinStripSize int64  `json:"min_strip_size,omitempty"`
	MaxStripSize int64  `json:"max_strip_size,omitempty"`
	Index        int64  `json:"index,allowzero"`
	Addr         string `json:"addr,omitempty"`
}

func (s BaremetalStorage) GetBlock() int64 {
	if s.Block <= 0 {
		return 512
	}
	return s.Block
}

/*type Disk struct {
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
}*/
