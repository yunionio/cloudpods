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

package detect_storages

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/baremetal/utils/raid"
	"yunion.io/x/onecloud/pkg/baremetal/utils/raid/drivers"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

func GetRaidDevices(drv raid.IRaidDriver) []*baremetal.BaremetalStorage {
	devs := make([]*baremetal.BaremetalStorage, 0)
	for _, ada := range drv.GetAdapters() {
		devs = append(devs, ada.GetDevices()...)
	}
	return devs
}

func GetRaidLogicVolumes(drv raid.IRaidDriver) ([]*raid.RaidLogicalVolume, error) {
	lvs := []*raid.RaidLogicalVolume{}
	for _, adapter := range drv.GetAdapters() {
		lv, err := adapter.GetLogicVolumes()
		if err != nil {
			return nil, err
		}
		lvs = append(lvs, lv...)
	}
	return lvs, nil
}

func DetectStorageInfo(term raid.IExecTerm, wait bool) ([]*baremetal.BaremetalStorage, []*baremetal.BaremetalStorage, []*baremetal.BaremetalStorage, error) {
	raidDiskInfo := make([]*baremetal.BaremetalStorage, 0)
	lvDiskInfo := make([]*raid.RaidLogicalVolume, 0)

	raidDrivers := []string{}
	for _, drv := range drivers.GetDrivers(term) {
		if err := drv.ParsePhyDevs(); err != nil {
			log.Warningf("Raid driver %s ParsePhyDevs: %v", drv.GetName(), err)
			continue
		}
		raidDiskInfo = append(raidDiskInfo, GetRaidDevices(drv)...)
		if drv.GetName() == baremetal.DISK_DRIVER_MARVELRAID {
			lvs, err := GetRaidLogicVolumes(drv)
			if err != nil {
				log.Errorf("GetRaidLogicVolumes: %v", err)
			} else {
				lvDiskInfo = append(lvDiskInfo, lvs...)
			}
		}
		raidDrivers = append(raidDrivers, drv.GetName())
	}

	log.Infof("Get Raid drivers: %v, collecting disks info ...", raidDrivers)
	pcieRet, err := term.Run("/lib/mos/lsdisk --pcie")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Fail to retrieve PCIE DISK info")
	}
	pcieDiskInfo := sysutils.ParsePCIEDiskInfo(pcieRet)

	maxTries := 6
	sleep := 10 * time.Second
	nonRaidDiskInfo := []*types.SDiskInfo{}
	for tried := 0; len(nonRaidDiskInfo) <= len(lvDiskInfo) && tried < maxTries; tried++ {
		ret, err := term.Run("/lib/mos/lsdisk --nonraid")
		if err != nil {
			return nil, nil, nil, fmt.Errorf("Fail to retrieve Non-Raid SCSI DISK info")
		}
		nonRaidDiskInfo = sysutils.ParseSCSIDiskInfo(ret)
		if wait {
			time.Sleep(sleep)
		} else {
			break
		}
	}
	log.Infof("RaidDiskInfo: %s, NonRaidSCSIDiskInfo: %s, PCIEDiskInfo: %s", jsonutils.Marshal(raidDiskInfo), jsonutils.Marshal(nonRaidDiskInfo), jsonutils.Marshal(pcieDiskInfo))
	if len(nonRaidDiskInfo) < len(lvDiskInfo) {
		return nil, nil, nil, fmt.Errorf("Fail to retrieve disk info")
	}
	if len(lvDiskInfo) > 0 {
		if len(lvDiskInfo) >= len(nonRaidDiskInfo) {
			nonRaidDiskInfo = nil
		} else {
			nonRaidDiskInfo = nonRaidDiskInfo[:len(nonRaidDiskInfo)-len(lvDiskInfo)]
		}
	}

	return raidDiskInfo, convertDiskInfos(nonRaidDiskInfo), convertDiskInfos(pcieDiskInfo), nil
}

func convertDiskInfos(infos []*types.SDiskInfo) []*baremetal.BaremetalStorage {
	ret := make([]*baremetal.BaremetalStorage, 0)
	for _, info := range infos {
		ret = append(ret, convertDiskInfo(info))
	}
	return ret
}

func convertDiskInfo(info *types.SDiskInfo) *baremetal.BaremetalStorage {
	return &baremetal.BaremetalStorage{
		Driver:     info.Driver,
		Size:       int64(info.Size),
		Rotate:     info.Rotate,
		Dev:        info.Dev,
		Sector:     info.Sector,
		Block:      info.Block,
		ModuleInfo: info.ModuleInfo,
		Kernel:     info.Kernel,
		PCIClass:   info.PCIClass,
	}
}
