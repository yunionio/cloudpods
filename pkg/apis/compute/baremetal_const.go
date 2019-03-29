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
	"yunion.io/x/pkg/util/sets"
)

const (
	DISK_CONF_RAID0  = "raid0"
	DISK_CONF_RAID1  = "raid1"
	DISK_CONF_RAID5  = "raid5"
	DISK_CONF_RAID10 = "raid10"
	DISK_CONF_NONE   = "none"

	DEFAULT_DISK_CONF = DISK_CONF_NONE

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
