package detect_storages

import (
	"fmt"
	"time"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/baremetal/utils/raid"
	"yunion.io/x/onecloud/pkg/baremetal/utils/raid/drivers"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/util/ssh"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

func GetRaidDevices(drv raid.IRaidDriver) []*baremetal.BaremetalStorage {
	devs := make([]*baremetal.BaremetalStorage, 0)
	for _, ada := range drv.GetAdapters() {
		devs = append(devs, ada.GetDevices()...)
	}
	return devs
}

func GetRaidLogicVolumes(drv raid.IRaidDriver) ([]int, error) {
	lvs := []int{}
	for _, adapter := range drv.GetAdapters() {
		lv, err := adapter.GetLogicVolumes()
		if err != nil {
			return nil, err
		}
		lvs = append(lvs, lv...)
	}
	return lvs, nil
}

func DetectStorageInfo(term *ssh.Client, wait bool) ([]*baremetal.BaremetalStorage, []*baremetal.BaremetalStorage, []*baremetal.BaremetalStorage, error) {
	raidDiskInfo := make([]*baremetal.BaremetalStorage, 0)
	lvDiskInfo := make([]int, 0)

	raidDrivers := []string{}
	for _, drv := range drivers.GetDrivers(term) {
		if err := drv.ParsePhyDevs(); err != nil {
			log.V(2).Warningf("ParsePhyDevs: %v", err)
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

	log.Infof("Get Raid drivers: %v", raidDrivers)

	pcieRet, err := term.Run("/lib/mos/lsdisk --pcie")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Fail to retrieve PCIE DISK info")
	}
	pcieDiskInfo := sysutils.ParsePCIEDiskInfo(pcieRet)

	maxTries := 6
	sleep := 10 * time.Second
	nonRaidDiskInfo := []*types.DiskInfo{}
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
	log.Infof("RaidDiskInfo: %#v, NonRaidSCSIDiskInfo: %#v, PCIEDiskInfo: %#v", raidDiskInfo, nonRaidDiskInfo, pcieDiskInfo)
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

func convertDiskInfos(infos []*types.DiskInfo) []*baremetal.BaremetalStorage {
	ret := make([]*baremetal.BaremetalStorage, 0)
	for _, info := range infos {
		ret = append(ret, convertDiskInfo(info))
	}
	return ret
}

func convertDiskInfo(info *types.DiskInfo) *baremetal.BaremetalStorage {
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
