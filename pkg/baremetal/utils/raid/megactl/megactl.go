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

package megactl

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/baremetal/utils/raid"
	raiddrivers "yunion.io/x/onecloud/pkg/baremetal/utils/raid"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/util/regutils2"
)

var (
	sizePattern   = regexp.MustCompile(`(?P<sector>0x[0-9a-fA-F]+)`)
	adapterPatter = regexp.MustCompile(`^Adapter #(?P<idx>[0-9]+)`)
)

type MegaRaidPhyDev struct {
	*raiddrivers.RaidBasePhyDev

	enclosure    int
	slot         int
	minStripSize int
	maxStripSize int
	sector       int64
	block        int64
}

func NewMegaRaidPhyDev() *MegaRaidPhyDev {
	return &MegaRaidPhyDev{
		RaidBasePhyDev: raiddrivers.NewRaidBasePhyDev(baremetal.DISK_DRIVER_MEGARAID),
		enclosure:      -1,
		slot:           -1,
		minStripSize:   -1,
		maxStripSize:   -1,
		sector:         -1,
		block:          512,
	}
}

func (dev *MegaRaidPhyDev) ToBaremetalStorage(index int) *baremetal.BaremetalStorage {
	s := dev.RaidBasePhyDev.ToBaremetalStorage(index)
	s.Enclosure = dev.enclosure
	s.Slot = dev.slot
	s.Size = dev.GetSize()
	s.MinStripSize = int64(dev.minStripSize)
	s.MaxStripSize = int64(dev.maxStripSize)
	s.Block = dev.block
	s.Sector = dev.sector
	return s
}

func (dev *MegaRaidPhyDev) GetSize() int64 {
	return dev.sector * dev.block / 1024 / 1024 // MB
}

func (dev *MegaRaidPhyDev) parseLine(line string) bool {
	key, val := stringutils.SplitKeyValue(line)
	if key == "" {
		return false
	}
	switch key {
	case "Media Type":
		if val == "Hard Disk Device" {
			dev.Rotate = tristate.True
		} else {
			dev.Rotate = tristate.False
		}
	case "Enclosure Device ID":
		enclosure, err := strconv.Atoi(val)
		if err == nil {
			dev.enclosure = enclosure
		}
	case "Slot Number":
		dev.slot, _ = strconv.Atoi(val)
	case "Coerced Size":
		sizeStr := regutils2.GetParams(sizePattern, val)["sector"]
		if len(sizeStr) != 0 {
			sizeStr = strings.Replace(sizeStr, "0x", "", -1)
			sector, err := strconv.ParseInt(sizeStr, 16, 64)
			if err != nil {
				log.Errorf("Parse sector %q to Int error: %v", sizeStr, err)
			}
			dev.sector = sector
		} else {
			dev.sector = 0
		}
	case "Inquiry Data":
		dev.Model = dev.convertModel(val)
	case "Firmware state":
		dev.Status = dev.convertState(val)
	case "Logical Sector Size":
		block, err := strconv.Atoi(val)
		if err != nil {
			log.Errorf("parse logical sector size error: %v", err)
			dev.block = 512
		} else if block > 0 {
			dev.block = int64(block)
		}
	default:
		return false
	}
	return true
}

func (dev *MegaRaidPhyDev) fillByStorcliPD(pd *StorcliPhysicalDrive) error {
	if pd.MediaType == "HDD" {
		dev.Rotate = tristate.True
	} else {
		dev.Rotate = tristate.False
	}

	eId, slotId := stringutils.SplitKeyValue(pd.EnclosureIdSlotNo)
	eIdInt, err := strconv.Atoi(eId)
	if err != nil {
		return errors.Errorf("Can't convert enclosureId %q", eId)
	}
	slotIdInt, err := strconv.Atoi(slotId)
	if err != nil {
		return errors.Errorf("Can't convert slotId %q", slotId)
	}
	dev.enclosure = eIdInt
	dev.slot = slotIdInt

	dev.Model = dev.convertModel(pd.Model)

	dev.Status = dev.convertState(pd.State)

	sector := strings.TrimSuffix(pd.SectorSize, "B")
	sectorInt, err := strconv.Atoi(sector)
	if err != nil {
		return errors.Errorf("Can't convert sector %q", pd.SectorSize)
	}
	// Use block not sector ...
	dev.block = int64(sectorInt)

	// parse size then fill block count
	sizeStrUnit := strings.Split(pd.Size, " ")
	if len(sizeStrUnit) != 2 {
		return errors.Errorf("Invalid size string %q", pd.Size)
	}
	sizeStr := sizeStrUnit[0]
	unit := sizeStrUnit[1]
	size, err := strconv.ParseFloat(sizeStr, 64)
	if err != nil {
		return errors.Errorf("Invalid size %q", sizeStr)
	}
	// convert size to Bytes
	switch unit {
	case "TB":
		size = size * 1024 * 1024 * 1024 * 1024
	case "GB":
		size = size * 1024 * 1024 * 1024
	case "MB":
		size = size * 1024 * 1024
	}
	blockCnt := int64(size) / int64(sectorInt)
	dev.sector = blockCnt

	return nil
}

func (dev *MegaRaidPhyDev) convertModel(val string) string {
	return strings.Join(regexp.MustCompile(`\s+`).Split(val, -1), " ")
}

func (dev *MegaRaidPhyDev) convertState(val string) string {
	state := val
	if val == "JBOD" {
		state = "jbod"
	} else if strings.Contains(strings.ToLower(val), "online") || utils.IsInStringArray(val, []string{"Onln"}) {
		state = "online"
	} else if val == "Rebuild" {
		state = "rebuild"
	} else if strings.Contains(strings.ToLower(val), "hotspare") {
		state = "hotspare"
	} else if strings.Contains(strings.ToLower(val), "copyback") {
		state = "copyback"
	} else if strings.Contains(strings.ToLower(val), "unconfigured(good)") {
		state = "unconfigured_good"
	} else {
		state = "offline"
	}
	return state
}

func (dev *MegaRaidPhyDev) isComplete() bool {
	if !dev.RaidBasePhyDev.IsComplete() {
		return false
	}
	if dev.sector < 0 {
		return false
	}
	if dev.block < 0 {
		return false
	}
	if dev.slot < 0 {
		return false
	}
	return true
}

func (dev *MegaRaidPhyDev) isJBOD() bool {
	return dev.Status == "jbod"
}

func GetSpecString(dev *baremetal.BaremetalStorage) string {
	if dev.Enclosure < 0 {
		return fmt.Sprintf(":%d", dev.Slot)
	}
	return fmt.Sprintf("%d:%d", dev.Enclosure, dev.Slot)
}

type MegaRaidAdaptor struct {
	index        int
	storcliIndex int
	raid         *MegaRaid
	devs         []*MegaRaidPhyDev
	sn           string
	name         string
	busNumber    string
	deviceNumber string
	funcNumber   string
	// used by sg_map
	hostNum int
	//channelNum int

	minStripSize int
	maxStripSize int
}

func NewMegaRaidAdaptor(index int, raid *MegaRaid) (*MegaRaidAdaptor, error) {
	adapter := &MegaRaidAdaptor{
		index:        index,
		storcliIndex: -1,
		raid:         raid,
	}
	if err := adapter.fillInfo(); err != nil {
		return adapter, errors.Wrapf(err, "%d fill info", adapter.index)
	}
	return adapter, nil
}

func NewMegaRaidAdaptorByStorcli(storAda *StorcliAdaptor, raid *MegaRaid) (*MegaRaidAdaptor, error) {
	adapter := &MegaRaidAdaptor{
		index:        storAda.Controller,
		storcliIndex: storAda.Controller,
		raid:         raid,
		sn:           storAda.sn,
		name:         storAda.name,
		busNumber:    storAda.busNumber,
		deviceNumber: storAda.deviceNumber,
		funcNumber:   storAda.funcNumber,
	}
	if err := adapter.checkPciDevice(); err != nil {
		return nil, errors.Wrap(err, "checkPciDevice")
	}
	return adapter, nil
}

func (adapter MegaRaidAdaptor) key() string {
	return adapter.name + adapter.sn
}

/*
Adapter: 0
Product Name: MegaRAID 9560-8i 4GB
Memory: 4096MB
BBU: Absent
Serial No: SKC4011564
*/
func (adapter *MegaRaidAdaptor) fillInfo() error {
	size2Int := func(sizeStr string) int {
		sz, _ := strconv.ParseFloat(strings.Fields(sizeStr)[0], 32)
		szInt := int(sz)
		if strings.Contains(sizeStr, "KB") {
			return szInt
		}
		if strings.Contains(sizeStr, "MB") {
			return szInt * 1024
		}
		return -1
	}
	cmd := GetCommand("-CfgDsply", fmt.Sprintf("-a%d", adapter.index))
	ret, err := adapter.remoteRun(cmd)
	if err != nil {
		return errors.Wrap(err, "remote get SN")
	}
	for _, l := range ret {
		key, val := stringutils.SplitKeyValue(l)
		if len(key) == 0 {
			continue
		}
		switch key {
		case "Serial No":
			adapter.sn = val
		case "Product Name":
			adapter.name = val
		case "Strip Size":
			sz := size2Int(val)
			adapter.minStripSize = sz
			adapter.maxStripSize = sz
		case "Min Strip Size":
			adapter.minStripSize = size2Int(val)
		case "Max Strip Size":
			adapter.maxStripSize = size2Int(val)
		}
	}
	if len(adapter.key()) == 0 {
		return errors.Error("Not found Serial No and Product Name")
	}
	return adapter.fillPCIInfo()
}

func (adapter *MegaRaidAdaptor) fillPCIInfo() error {
	cmd := GetCommand("-adpgetpciinfo", fmt.Sprintf("-a%d", adapter.index))
	ret, err := adapter.remoteRun(cmd)
	if err != nil {
		return errors.Wrapf(err, "%d remote run get pci info", adapter.index)
	}
	for _, l := range ret {
		key, val := stringutils.SplitKeyValue(l)
		if len(key) == 0 {
			continue
		}
		switch key {
		case "Bus Number":
			if len(val) == 1 {
				val = fmt.Sprintf("0%s", val)
			}
			if len(val) != 2 {
				return errors.Errorf("Invalid bus number: %s", val)
			}
			adapter.busNumber = val
		case "Device Number":
			if len(val) == 1 {
				val = fmt.Sprintf("0%s", val)
			}
			if len(val) != 2 {
				return errors.Errorf("Invalid device number: %s", val)
			}
			adapter.deviceNumber = val
		case "Function Number":
			if len(val) != 1 {
				return errors.Errorf("Invalid function number: %s", val)
			}
			adapter.funcNumber = val
		}
	}
	if err := adapter.checkPciDevice(); err != nil {
		return errors.Wrap(err, "checkPciDevice")
	}
	return nil
}

func (adapter *MegaRaidAdaptor) checkPciDevice() error {
	pciDir := fmt.Sprintf("/sys/bus/pci/devices/0000:%s:%s.%s/", adapter.busNumber, adapter.deviceNumber, adapter.funcNumber)
	cmd := raiddrivers.GetCommand("ls", pciDir, "|", "grep", "host")
	ret, err := adapter.remoteRun(cmd)
	if err != nil {
		return errors.Wrapf(err, "find pci host number")
	}
	if len(ret) == 0 {
		return errors.Errorf("Not find pci host dir")
	}
	hostNumStr := ret[0]
	hostNum, err := strconv.Atoi(strings.TrimLeft(hostNumStr, "host"))
	if err != nil {
		return errors.Errorf("Invalid hostNum %s", hostNumStr)
	}
	adapter.hostNum = hostNum
	pciHostDir := fmt.Sprintf("%s%s/", pciDir, hostNumStr)
	// $ ls /sys/bus/pci/devices/0000:03:00.0/host0/ | grep target | head -n 1
	// target0:2:0
	targetCmd := raiddrivers.GetCommand("ls", pciHostDir, "|", "grep", "target", "|", "head", "-n", "1")
	ret, err = adapter.remoteRun(targetCmd)
	if err != nil {
		return errors.Wrapf(err, "find target %q", targetCmd)
	}
	if len(ret) == 0 {
		return errors.Errorf("Not find target dir")
	}
	//targetStr := ret[0]
	//parts := strings.Split(targetStr, ":")
	//if len(parts) != 3 {
	//// not build raid logical volume yet
	//log.Warningf("Cmd %q invalid target string %q, skip fill logical volume info", targetCmd, targetStr)
	//return nil
	//}
	//channelNum, err := strconv.Atoi(parts[1])
	//if err != nil {
	//return errors.Errorf("Invalid channel number %s", parts[1])
	//}
	//adapter.channelNum = channelNum
	return nil
}

func (adapter *MegaRaidAdaptor) GetIndex() int {
	return adapter.index
}

func (adapter *MegaRaidAdaptor) getTerm() raid.IExecTerm {
	return adapter.raid.term
}

func (adapter *MegaRaidAdaptor) remoteRun(cmds ...string) ([]string, error) {
	return adapter.getTerm().Run(cmds...)
}

func (adapter *MegaRaidAdaptor) AddPhyDev(dev *MegaRaidPhyDev) {
	dev.Adapter = adapter.index
	adapter.devs = append(adapter.devs, dev)
}

func (adapter *MegaRaidAdaptor) GetDevices() []*baremetal.BaremetalStorage {
	ret := []*baremetal.BaremetalStorage{}
	for idx, dev := range adapter.devs {
		ret = append(ret, dev.ToBaremetalStorage(idx))
	}
	return ret
}

func (adapter *MegaRaidAdaptor) GetLogicVolumes() ([]*raiddrivers.RaidLogicalVolume, error) {
	errs := make([]error, 0)
	megaLvs, err := adapter.getMegacliLogicVolumes()
	if err != nil {
		errs = append(errs, err)
	}
	storeLvs, err := adapter.getStorcliLogicVolums()
	if err != nil {
		errs = append(errs, err)
	}
	if len(megaLvs) > 0 {
		return megaLvs, nil
	}
	if len(storeLvs) > 0 {
		return storeLvs, nil
	}
	if len(errs) == 0 {
		// no error, no volume
		return []*raiddrivers.RaidLogicalVolume{}, nil
	}
	return nil, errors.NewAggregate(errs)
}

func (adapter *MegaRaidAdaptor) getMegacliLogicVolumes() ([]*raiddrivers.RaidLogicalVolume, error) {
	cmd := GetCommand("-LDInfo", "-Lall", fmt.Sprintf("-a%d", adapter.index))
	ret, err := adapter.remoteRun(cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "remoteRun %s", cmd)
	}
	lvs, err := adapter.parseLogicVolumes(ret)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	for i := range lvs {
		lvs[i].Driver = raiddrivers.RaidDriverToolMegacli64
	}
	return lvs, nil
}

func (adapter *MegaRaidAdaptor) getStorcliLogicVolums() ([]*raiddrivers.RaidLogicalVolume, error) {
	lvs, err := adapter.getStorcliLogicVolumsV2()
	if err != nil {
		return nil, err
	}
	ret := make([]*raiddrivers.RaidLogicalVolume, len(lvs))
	for i := range lvs {
		lv := lvs[i]
		ret[i] = &raiddrivers.RaidLogicalVolume{
			Index:    lv.Index,
			Adapter:  adapter.index,
			BlockDev: lv.GetOSDevice(),
			IsSSD:    tristate.NewFromBool(lv.IsSSD()),
			Driver:   raiddrivers.RaidDriverToolStorecli,
		}
	}
	return ret, nil
}

func (adapter *MegaRaidAdaptor) getStorcliLogicVolumsV2() ([]*StorcliLogicalVolume, error) {
	cmd := GetCommand2(fmt.Sprintf("/c%d/vall", adapter.index), "show", "all", "J")
	ret, err := adapter.remoteRun(cmd)
	if err != nil {
		return nil, fmt.Errorf("getStorcliLogicVolumsV2 error: %v", err)
	}
	output := strings.Join(ret, "\n")
	lvs, err := parseStorcliLVs(output)
	if err != nil {
		return nil, errors.Wrap(err, "parseStorcliLVs")
	}
	return lvs.GetLogicalVolumes(adapter.index)
}

var storcliLVRegexp = regexp.MustCompile(`^(?P<dg>\d+)\/(?P<vd>\d+)\s+(?P<type>RAID\d+).*`)

func parseStorcliLogicalVolumes(adapter int, lines []string) ([]*raiddrivers.RaidLogicalVolume, error) {
	lvs := make([]*raiddrivers.RaidLogicalVolume, 0)
	for _, line := range lines {
		result := regutils2.GetParams(storcliLVRegexp, line)
		if len(result) == 0 {
			continue
		}
		idxStr, ok := result["vd"]
		if !ok {
			return nil, errors.Errorf("Not found virtual drive by line %q", line)
		}
		idx, _ := strconv.Atoi(idxStr)
		lvs = append(lvs, &raiddrivers.RaidLogicalVolume{
			Index:   idx,
			Adapter: adapter,
		})
	}
	return lvs, nil
}

var logicalVolumeIdRegexp = regexp.MustCompile(`.*(Target Id: (?P<idx>[0-9]+))`)

func (adapter *MegaRaidAdaptor) parseLogicVolumes(lines []string) ([]*raiddrivers.RaidLogicalVolume, error) {
	lvs := make([]*raiddrivers.RaidLogicalVolume, 0)
	for _, line := range lines {
		key, val := stringutils.SplitKeyValue(line)
		if key != "" && key == "Virtual Drive" {
			idxStr := regutils2.GetParams(logicalVolumeIdRegexp, val)["idx"]
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				return nil, errors.Errorf("index %q to int: %v", idxStr, err)
			}
			blockDev, err := getLogicVolumeDeviceById(adapter.hostNum, idx, adapter.getTerm())
			if err != nil {
				return nil, err
			}
			lvs = append(lvs, &raiddrivers.RaidLogicalVolume{
				Index:    idx,
				Adapter:  adapter.index,
				BlockDev: blockDev,
			})
		}
	}
	return lvs, nil
}

func getLogicVolumeDeviceById(hostNum, scsiId int, term raid.IExecTerm) (string, error) {
	items, err := raiddrivers.SGMap(term)
	if err != nil {
		return "", err
	}
	isMatch := func(item api.SGMapItem) bool {
		return item.HostNumber == hostNum && item.SCSIId == scsiId
	}
	for _, item := range items {
		if isMatch(item) {
			return item.LinuxDeviceName, nil
		}
	}
	return "", errors.Errorf("Not found SG item by id: %d:%d", hostNum, scsiId)
}

func (adapter *MegaRaidAdaptor) PreBuildRaid(confs []*api.BaremetalDiskConfig) error {
	adapter.clearJBODDisks()
	return nil
}

func (adapter *MegaRaidAdaptor) PostBuildRaid() error {
	// sync rotational of logical block device
	if err := adapter.storcliSyncBlockDeviceAttrs(); err != nil {
		log.Warningf("adapter %d storcliSyncBlockDeviceAttrs: %v", adapter.index, err)
	}
	return nil
}

func (adapter *MegaRaidAdaptor) storcliSyncBlockDeviceAttrs() error {
	lvs, err := adapter.getStorcliLogicVolumsV2()
	if err != nil {
		return errors.Wrap(err, "getStorcliLogicVolumsV2")
	}
	for _, lv := range lvs {
		if !lv.IsSSD() {
			continue
		}
		// e.g: echo 0 | tee /sys/block/sda/queue/rotational if logical volume is SSD
		cmd := fmt.Sprintf("echo 0 | tee %s", lv.GetSysBlockRotationalPath())
		if out, err := adapter.remoteRun(cmd); err != nil {
			return errors.Wrapf(err, "%s out %v", cmd, out)
		}
	}
	return nil
}

func conf2Params(conf *api.BaremetalDiskConfig) []string {
	params := []string{}
	if conf.WT != nil {
		if *conf.WT {
			params = append(params, "WT")
		} else {
			params = append(params, "WB")
		}
	}
	if conf.RA != nil {
		if *conf.RA {
			params = append(params, "RA")
		} else {
			params = append(params, "NORA")
		}
	}
	if conf.Direct != nil {
		if *conf.Direct {
			params = append(params, "Direct")
		} else {
			params = append(params, "Cached")
		}
	}
	if conf.Cachedbadbbu != nil {
		if *conf.Cachedbadbbu {
			params = append(params, "CachedBadBBU")
		} else {
			params = append(params, "NoCachedBadBBU")
		}
	}
	if conf.Strip != nil {
		params = append(params, fmt.Sprintf("-strpsz%d", *conf.Strip))
	}
	if len(conf.Size) > 0 {
		for _, sz := range conf.Size {
			params = append(params, fmt.Sprintf("-sz%d", sz))
		}
	}
	return params
}

func (adapter *MegaRaidAdaptor) storcliBuildRaid0(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return adapter.storcliBuildRaid(devs, conf, 0)
}

func (adapter *MegaRaidAdaptor) megacliBuildRaid0(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return adapter.megacliBuildRaid(devs, conf, 0)
}

func (adapter *MegaRaidAdaptor) storcliBuildRaid1(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return adapter.storcliBuildRaid(devs, conf, 1)
}

func (adapter *MegaRaidAdaptor) megacliBuildRaid1(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return adapter.megacliBuildRaid(devs, conf, 1)
}

func (adapter *MegaRaidAdaptor) storcliBuildRaid5(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return adapter.storcliBuildRaid(devs, conf, 5)
}

func (adapter *MegaRaidAdaptor) megacliBuildRaid5(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return adapter.megacliBuildRaid(devs, conf, 5)
}

func (adapter *MegaRaidAdaptor) storcliBuildRaid10(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return adapter.storcliBuildRaid(devs, conf, 10)
}

func (adapter *MegaRaidAdaptor) megacliBuildRaid10(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	if len(devs)%2 != 0 {
		return fmt.Errorf("Odd number of %d devs", len(devs))
	}
	devCnt := len(devs) / 2
	params := []string{}
	for i := 0; i < devCnt; i++ {
		d1 := devs[i]
		d2 := devs[i+devCnt]
		params = append(params, fmt.Sprintf("-Array%d[%s,%s]", i, GetSpecString(d1), GetSpecString(d2)))
	}
	args := []string{"-CfgSpanAdd", "-r10"}
	args = append(args, params...)
	args = append(args, conf2Params(conf)...)
	args = append(args, fmt.Sprintf("-a%d", adapter.index))
	cmd := GetCommand(args...)
	_, err := adapter.remoteRun(cmd)
	return err
}

func (adapter *MegaRaidAdaptor) storcliBuildRaid(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig, level uint) error {
	if err := storcliBuildRaid(
		adapter.GetStorcliCommand,
		adapter.getTerm(),
		devs, conf, level); err != nil {
		return err
	}
	return nil
}

func (adapter *MegaRaidAdaptor) megacliBuildRaid(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig, level uint) error {
	labels := []string{}
	for _, dev := range devs {
		labels = append(labels, GetSpecString(dev))
	}
	args := []string{"-CfgLdAdd", fmt.Sprintf("-r%d", level), fmt.Sprintf("[%s]", strings.Join(labels, ","))}
	args = append(args, conf2Params(conf)...)
	args = append(args, fmt.Sprintf("-a%d", adapter.index))
	cmd := GetCommand(args...)
	log.Infof("_megacliBuildRaid command: %s", cmd)
	_, err := adapter.remoteRun(cmd)
	return err
}

func cliBuildRaid(
	devs []*baremetal.BaremetalStorage,
	conf *api.BaremetalDiskConfig,
	funcs ...func([]*baremetal.BaremetalStorage, *api.BaremetalDiskConfig) error,
) error {
	var errs []error
	for _, f := range funcs {
		err := f(devs, conf)
		if err == nil {
			return nil
		}
		errs = append(errs, err)
	}
	return errors.NewAggregate(errs)
}

func (adapter *MegaRaidAdaptor) BuildRaid0(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return cliBuildRaid(devs, conf, adapter.megacliBuildRaid0, adapter.storcliBuildRaid0)
}

func (adapter *MegaRaidAdaptor) BuildRaid1(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return cliBuildRaid(devs, conf, adapter.megacliBuildRaid1, adapter.storcliBuildRaid1)
}

func (adapter *MegaRaidAdaptor) BuildRaid5(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return cliBuildRaid(devs, conf, adapter.megacliBuildRaid5, adapter.storcliBuildRaid5)
}

func (adapter *MegaRaidAdaptor) BuildRaid10(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return cliBuildRaid(devs, conf, adapter.megacliBuildRaid10, adapter.storcliBuildRaid10)
}

func (adapter *MegaRaidAdaptor) BuildNoneRaid(devs []*baremetal.BaremetalStorage) error {
	return cliBuildRaid(devs, nil, adapter.megacliBuildNoRaid, adapter.storcliBuildNoRaid)
}

func (raid *MegaRaid) GetStorcliAdaptor() ([]*StorcliAdaptor, map[string]*StorcliAdaptor, error) {
	ret := make(map[string]*StorcliAdaptor)
	cmd := GetCommand2("/call", "show", "|", "grep", "-iE", `'^(Controller|Product Name|Serial Number|Bus Number|Device Number|Function Number)\s='`)
	lines, err := raid.term.Run(cmd)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Get storcli adapter")
	}
	adapter := newStorcliAdaptor()
	list := make([]*StorcliAdaptor, 0)
	for _, l := range lines {
		adapter.parseLine(l)
		if adapter.isComplete() {
			ret[adapter.key()] = adapter
			list = append(list, adapter)
			adapter = newStorcliAdaptor()
		}
	}
	return list, ret, nil
}

func (adapter *MegaRaidAdaptor) storcliCtrlIndex() (int, error) {
	if adapter.storcliIndex >= 0 {
		return adapter.storcliIndex, nil
	}
	_, storcliAdaps, err := adapter.raid.GetStorcliAdaptor()
	if err != nil {
		return -1, errors.Wrap(err, "Get all Storcli adaptor")
	}
	storAdap, ok := storcliAdaps[adapter.key()]
	if !ok {
		return -1, errors.Errorf("Not found storcli adaptor by SN %q", adapter.key())
	}
	return storAdap.Controller, nil
}

func (adapter *MegaRaidAdaptor) GetStorcliCommand(args ...string) (string, error) {
	controllerIdx, err := adapter.storcliCtrlIndex()
	if err != nil {
		return "", errors.Errorf("Adapter %d get storcli controller index: %v", adapter.index, err)
	}
	nargs := []string{fmt.Sprintf("/c%d", controllerIdx)}
	nargs = append(nargs, args...)
	return GetCommand2(nargs...), nil
}

func (adapter *MegaRaidAdaptor) storcliIsJBODEnabled() bool {
	return storcliIsJBODEnabled(adapter.GetStorcliCommand, adapter.getTerm())
}

func (adapter *MegaRaidAdaptor) storcliEnableJBOD(enable bool) bool {
	val := "off"
	if enable {
		val = "on"
	}
	cmd, err := adapter.GetStorcliCommand("set", fmt.Sprintf("jbod=%s", val), "force")
	if err != nil {
		log.Errorf("get storcli controller cmd: %v", err)
		return false
	}
	_, err = adapter.remoteRun(cmd)
	if err != nil {
		log.Errorf("EnableJBOD %v fail: %v", enable, err)
		return false
	}
	return true
}

func (adapter *MegaRaidAdaptor) storcliBuildJBOD(devs []*baremetal.BaremetalStorage) error {
	return storcliBuildJBOD(adapter.GetStorcliCommand, adapter.getTerm(), devs)
}

func (adapter *MegaRaidAdaptor) storcliBuildNoRaid(devs []*baremetal.BaremetalStorage, _ *api.BaremetalDiskConfig) error {
	return storcliBuildNoRaid(adapter.GetStorcliCommand, adapter.getTerm(), devs)
}

func (adapter *MegaRaidAdaptor) megacliBuildNoRaid(devs []*baremetal.BaremetalStorage, _ *api.BaremetalDiskConfig) error {
	err := adapter.megacliBuildJBOD(devs)
	if err == nil {
		return nil
	}
	log.Errorf("Try megacli build jbod fail: %v", err)
	cmds := []string{}
	for _, dev := range devs {
		cmd := GetCommand("-CfgLdAdd", "-r0", fmt.Sprintf("[%s]", GetSpecString(dev)),
			"WT", "NORA", "Direct", "NoCachedBadBBU", fmt.Sprintf("-a%d", adapter.index))
		cmds = append(cmds, cmd)
	}
	_, err = adapter.remoteRun(cmds...)
	return err
}

func (adapter *MegaRaidAdaptor) megacliIsJBODEnabled() bool {
	cmd := GetCommand("-AdpGetProp", "-EnableJBOD", fmt.Sprintf("-a%d", adapter.index))
	pref := fmt.Sprintf("Adapter %d: JBOD: ", adapter.index)
	lines, err := adapter.remoteRun(cmd)
	if err != nil {
		log.Errorf("megacliIsJBODEnabled error: %v", err)
		return false
	}
	for _, line := range lines {
		if strings.HasPrefix(line, pref) {
			val := strings.ToLower(strings.TrimSpace(line[len(pref):]))
			if val == "disabled" {
				return false
			}
			return true
		}
	}
	return false
}

func (adapter *MegaRaidAdaptor) megacliEnableJBOD(enable bool) bool {
	var val string = "0"
	if enable {
		val = "1"
	}
	cmd := GetCommand("-AdpSetProp", "-EnableJBOD", fmt.Sprintf("-%s", val), fmt.Sprintf("-a%d", adapter.index))
	_, err := adapter.remoteRun(cmd)
	if err != nil {
		log.Errorf("enable jbod %v fail: %v", enable, err)
		return false
	}
	return true
}

func (adapter *MegaRaidAdaptor) megacliBuildJBOD(devs []*baremetal.BaremetalStorage) error {
	if !adapter.megacliIsJBODEnabled() {
		adapter.megacliEnableJBOD(true)
		adapter.megacliEnableJBOD(false)
		adapter.megacliEnableJBOD(true)
	}
	if !adapter.megacliIsJBODEnabled() {
		return fmt.Errorf("JBOD not supported")
	}
	// try clear jbod disk of devices
	if err := adapter.megacliClearJBODDisks(devs); err != nil {
		log.Warningf("try clear megaraid jbod disks before make jbod: %s", err)
	}
	devIds := []string{}
	for _, d := range devs {
		devIds = append(devIds, GetSpecString(d))
	}
	cmd := GetCommand("-PDMakeJBOD", fmt.Sprintf("-PhysDrv[%s]", strings.Join(devIds, ",")), fmt.Sprintf("-a%d", adapter.index))
	_, err := adapter.remoteRun(cmd)
	return err
}

func (adapter *MegaRaidAdaptor) RemoveLogicVolumes() error {
	lvIdx, err := adapter.GetLogicVolumes()
	if err != nil {
		return errors.Wrap(err, "GetLogicVolumes")
	}
	if len(lvIdx) == 0 {
		log.Infof("RemoveLogicVolumes: no logical volume to delete!")
		return nil
	}
	errs := make([]error, 0)
	lvIdx = raiddrivers.ReverseLogicalArray(lvIdx)
	for i := range lvIdx {
		lv := lvIdx[i]
		switch lv.Driver {
		case raiddrivers.RaidDriverToolMegacli64:
			cmd := GetCommand("-CfgLdDel", fmt.Sprintf("-L%d", lv.Index), "-Force", fmt.Sprintf("-a%d", adapter.index))
			_, err := adapter.remoteRun(cmd)
			if err != nil {
				errs = append(errs, err)
			}
		case raiddrivers.RaidDriverToolStorecli:
			cmd := GetCommand2(fmt.Sprintf("/c%d/v%d", adapter.index, lv.Index), "delete", "force")
			_, err := adapter.remoteRun(cmd)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return errors.NewAggregate(errs)
	}
	return nil
}

func (adapter *MegaRaidAdaptor) storcliClearJBODDisks() error {
	return storcliClearJBODDisks(
		adapter.GetStorcliCommand, adapter.getTerm(),
		adapter.devs,
	)
}

func (adapter *MegaRaidAdaptor) megacliClearJBODDisks(devs []*baremetal.BaremetalStorage) error {
	devIds := []string{}
	for _, dev := range devs {
		devIds = append(devIds, GetSpecString(dev))
	}
	errs := make([]error, 0)
	for _, devId := range devIds {
		cmd := GetCommand("-PDMakeGood", "-PhysDrv", fmt.Sprintf("'[%s]'", devId), "-Force", fmt.Sprintf("-a%d", adapter.index))
		if _, err := adapter.remoteRun(cmd); err != nil {
			err = errors.Wrapf(err, "PDMakeGood megacli cmd %v", cmd)
			errs = append(errs, err)
		}
	}
	return errors.NewAggregate(errs)
}

func (adapter *MegaRaidAdaptor) megacliClearAllJBODDisks() error {
	allDevs := make([]*baremetal.BaremetalStorage, 0)
	for idx, dev := range adapter.devs {
		allDevs = append(allDevs, dev.ToBaremetalStorage(idx))
	}
	return adapter.megacliClearJBODDisks(allDevs)
}

func (adapter *MegaRaidAdaptor) clearJBODDisks() {
	if err := adapter.megacliClearAllJBODDisks(); err != nil {
		log.Errorf("megacliClearAllJBODDisks error: %v", err)
		log.Infof("try storcliClearJBODDisks")
		if err := adapter.storcliClearJBODDisks(); err != nil {
			log.Errorf("storcliClearJBODDisks error: %v", err)
		}
	}
	adapter.megacliEnableJBOD(true)
	adapter.megacliEnableJBOD(false)
	adapter.megacliEnableJBOD(true)
	adapter.megacliEnableJBOD(false)
}

type MegaRaid struct {
	term       raid.IExecTerm
	adapters   []*MegaRaidAdaptor
	PhyDevsCnt int
	Capacity   int64
}

func NewMegaRaid(term raid.IExecTerm) raiddrivers.IRaidDriver {
	return &MegaRaid{
		term:     term,
		adapters: make([]*MegaRaidAdaptor, 0),
	}
}

func GetCommand(args ...string) string {
	bin := "/opt/MegaRAID/MegaCli/MegaCli64"
	return raiddrivers.GetCommand(bin, args...)
}

func GetCommand2(args ...string) string {
	bin := "/opt/MegaRAID/storcli/storcli64"
	return raiddrivers.GetCommand(bin, args...)
}

func (raid *MegaRaid) GetName() string {
	return baremetal.DISK_DRIVER_MEGARAID
}

func (raid *MegaRaid) ParsePhyDevs() error {
	if !utils.IsInStringArray(raiddrivers.MODULE_MEGARAID, raiddrivers.GetModules(raid.term)) {
		return fmt.Errorf("Not found megaraid_sas module")
	}
	if err := raid.parsePhyDevsUseMegacli(); err == nil {
		return nil
	} else {
		// try use storecli parse physical devices
		if err := raid.parsePhyDevsUseStorcli(); err != nil {
			return errors.Wrap(err, "parsePhyDevsUseStorcli")
		}
	}
	return nil
}

func (raid *MegaRaid) parsePhyDevsUseMegacli() error {
	cmd := GetCommand("-PDList", "-aALL")
	ret, err := raid.term.Run(cmd)
	if err != nil {
		return fmt.Errorf("List raid disk error: %v", err)
	}
	if raiddrivers.Debug {
		log.Debugf("-PDList -aALL: %s", ret)
	}
	err = raid.parsePhyDevs(ret)
	if err != nil {
		return fmt.Errorf("parse physical disk device error: %v", err)
	}
	return nil
}

func (raid *MegaRaid) parsePhyDevsUseStorcli() error {
	adapters, _, err := raid.GetStorcliAdaptor()
	if err != nil {
		return errors.Wrap(err, "Get storcli adapter")
	}
	raid.adapters = make([]*MegaRaidAdaptor, 0)
	for _, ada := range adapters {
		megaAda, err := NewMegaRaidAdaptorByStorcli(ada, raid)
		if err != nil {
			return errors.Wrap(err, "NewMegaRaidAdaptorByStorcli")
		}
		devs, err := ada.getMegaPhyDevs(GetCommand2, raid.term)
		if err != nil {
			return errors.Wrapf(err, "get storcli %d mega PDs", ada.Controller)
		}
		for i := range devs {
			megaAda.AddPhyDev(devs[i])
		}
		raid.adapters = append(raid.adapters, megaAda)
	}
	return nil
}

func (raid *MegaRaid) parsePhyDevs(lines []string) error {
	phyDev := NewMegaRaidPhyDev()
	var adapter *MegaRaidAdaptor
	var err error
	for _, line := range lines {
		adapterStr := regutils2.GetParams(adapterPatter, line)["idx"]
		if adapterStr != "" {
			adapterInt, _ := strconv.Atoi(adapterStr)
			adapter, err = NewMegaRaidAdaptor(adapterInt, raid)
			if err != nil {
				return errors.Wrapf(err, "New raid adapter %d", adapterInt)
			}
			raid.adapters = append(raid.adapters, adapter)
		} else if phyDev.parseLine(line) && phyDev.isComplete() {
			if adapter == nil {
				return fmt.Errorf("Adapter is nil")
			}
			adapter.AddPhyDev(phyDev)
			raid.PhyDevsCnt += 1
			raid.Capacity += phyDev.GetSize()
			phyDev = NewMegaRaidPhyDev()
		}
	}
	for _, adapter := range raid.adapters {
		if err := adapter.addPhyDevsStripSize(); err != nil {
			log.Errorf("Adapter %d fill phsical devices strip size: %v", adapter.GetIndex(), err)
		}
	}
	return nil
}

func (adapter *MegaRaidAdaptor) addPhyDevsStripSize() error {
	for _, dev := range adapter.devs {
		dev.minStripSize = adapter.minStripSize
		dev.maxStripSize = adapter.maxStripSize
	}
	return nil
}

func (raid *MegaRaid) CleanRaid() error {
	for _, adapter := range raid.adapters {
		adapter.clearJBODDisks()
		adapter.RemoveLogicVolumes()
	}
	return nil
}

func (raid *MegaRaid) PreBuildRaid(_ []*api.BaremetalDiskConfig, _ int) error {
	return raid.clearForeignState()
}

func (raid *MegaRaid) GetAdapters() []raiddrivers.IRaidAdapter {
	ret := make([]raiddrivers.IRaidAdapter, 0)
	for _, a := range raid.adapters {
		ret = append(ret, a)
	}
	return ret
}

func (raid *MegaRaid) clearForeignState() error {
	errs := make([]error, 0)
	cmd := GetCommand("-CfgForeign", "-Clear", "-aALL")
	_, err := raid.term.Run(cmd)
	if err != nil {
		errs = append(errs, err)
		cmd2 := GetCommand2("/call/fall", "delete")
		if _, err := raid.term.Run(cmd2); err == nil {
			return nil
		} else {
			errs = append(errs, err)
		}
	}
	return errors.NewAggregate(errs)
}

func (raid *MegaRaid) RemoveLogicVolumes() {
	for _, adapter := range raid.adapters {
		adapter.RemoveLogicVolumes()
	}
}
