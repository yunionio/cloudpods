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
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/cmdline"
)

func ParseDiskConfig(desc string) (api.BaremetalDiskConfig, error) {
	conf, err := cmdline.ParseBaremetalDiskConfig(desc)
	if err != nil {
		return api.BaremetalDiskConfig{}, err
	}
	return *conf, nil
}

func isDiskConfigStorageMatch(
	config *api.BaremetalDiskConfig,
	confDriver *string,
	confAdapter *int,
	storage *BaremetalStorage,
	selected []*BaremetalStorage,
) bool {
	isRotate := storage.Rotate
	adapter := storage.Adapter
	index := storage.Index
	driver := storage.Driver

	typeIsHybrid := config.Type == DISK_TYPE_HYBRID
	typeIsRotate := config.Type == DISK_TYPE_ROTATE && isRotate
	typeIsSSD := config.Type == DISK_TYPE_SSD && !isRotate
	rangeIsNoneAndCountZero := len(config.Range) == 0 && config.Count == 0
	rangeIsNotNoneAndIndexInRange := len(config.Range) != 0 && sets.NewInt64(config.Range...).Has(index)
	rangeIsNoneAndSmallThanCount := len(config.Range) == 0 && int64(len(selected)) < config.Count
	adapterIsEqual := (confAdapter == nil || *confAdapter == adapter) &&
		(confDriver == nil || *confDriver == driver)

	log.V(10).Debugf("typeIsHybrid: %v, typeIsRotate: %v, typeIsSSD: %v, rangeIsNoneAndCountZero: %v, rangeIsNotNoneAndIndexInRange: %v, rangeIsNoneAndSmallThanCount: %v, adapterIsEqual: %v", typeIsHybrid, typeIsRotate, typeIsSSD, rangeIsNoneAndCountZero, rangeIsNotNoneAndIndexInRange, rangeIsNoneAndSmallThanCount, adapterIsEqual)

	if (typeIsHybrid || typeIsRotate || typeIsSSD) &&
		(rangeIsNoneAndCountZero || rangeIsNotNoneAndIndexInRange || rangeIsNoneAndSmallThanCount) &&
		adapterIsEqual {
		return true
	}
	return false
}

func RetrieveStorages(diskConfig *api.BaremetalDiskConfig, storages []*BaremetalStorage) (selected, rest []*BaremetalStorage) {
	var confDriver *string = nil
	var confAdapter *int = nil

	if diskConfig.Adapter != nil {
		confAdapter = diskConfig.Adapter
	}
	if diskConfig.Driver != "" {
		confDriver = &diskConfig.Driver
	}

	selected = make([]*BaremetalStorage, 0)
	rest = make([]*BaremetalStorage, 0)
	idx := 0
	curAdapter := 0
	adapterChange := false

	for _, storage := range storages {
		if storage.Adapter != curAdapter {
			adapterChange = true
			curAdapter = storage.Adapter
		}
		if adapterChange {
			idx = 0
			adapterChange = false
		}
		if storage.Index == 0 {
			storage.Index = int64(idx)
		}

		if isDiskConfigStorageMatch(diskConfig, confDriver, confAdapter, storage, selected) {
			if confDriver == nil {
				confDriver = &storage.Driver
			}
			if confAdapter == nil {
				confAdapter = &storage.Adapter
			}
			selected = append(selected, storage)
		} else {
			rest = append(rest, storage)
		}
		idx++
	}
	return
}

func GetMinDiskRequirement(diskConfig string) int {
	minDisk := 1
	if diskConfig == DISK_CONF_RAID1 {
		minDisk = 2
	}
	if diskConfig == DISK_CONF_RAID5 {
		minDisk = 3
	} else if diskConfig == DISK_CONF_RAID10 {
		minDisk = 4
	}
	return minDisk
}

func RequireEvenDisks(diskConfig string) bool {
	if sets.NewString(
		DISK_CONF_RAID10,
		DISK_CONF_RAID1,
	).Has(diskConfig) {
		return true

	}
	return false
}

type Layout struct {
	Disks []*BaremetalStorage      `json:"disks"`
	Conf  *api.BaremetalDiskConfig `json:"conf"`
	Size  int64                    `json:"size"`
}

func (l Layout) String() string {
	bytes, _ := json.MarshalIndent(l, "", "  ")
	return string(bytes)
}

func RetrieveStorageDrivers(storages []*BaremetalStorage) sets.String {
	ret := sets.NewString()
	for _, s := range storages {
		if !ret.Has(s.Driver) {
			ret.Insert(s.Driver)
		}
	}
	return ret
}

func MeetConfig(
	conf *api.BaremetalDiskConfig,
	storages []*BaremetalStorage,
) error {
	storageDrvs := RetrieveStorageDrivers(storages)
	if len(storageDrvs.List()) > 1 {
		return fmt.Errorf("%v more than 1 storages drivers", storageDrvs)
	}
	driver := storageDrvs.List()[0]
	if conf.Conf != DISK_CONF_NONE && !DISK_DRIVERS_RAID.Has(driver) {
		return fmt.Errorf("BaremetalStorage driver %s not support RAID", driver)
	}

	minDisk := GetMinDiskRequirement(conf.Conf)
	if len(storages) < minDisk {
		return fmt.Errorf("%q requires at least %d disks", conf.Conf, minDisk)
	}

	if RequireEvenDisks(conf.Conf) && (len(storages)%2) != 0 {
		return fmt.Errorf("%q requires event number of disks", conf.Conf)
	}

	if len(conf.Splits) > 0 &&
		sets.NewString(
			DISK_CONF_NONE,
			DISK_DRIVER_MPT2SAS).Has(conf.Conf) {
		return fmt.Errorf("Cannot divide a normal disk into splits")
	}

	if driver == DISK_DRIVER_MPT2SAS {
		if conf.Conf == DISK_CONF_RAID5 {
			return fmt.Errorf("%q not support RAID5", DISK_DRIVER_MPT2SAS)
		}
		if conf.Conf == DISK_CONF_RAID0 && len(storages) < 2 {
			return fmt.Errorf("%q %q requires at least 2 disks", DISK_DRIVER_MPT2SAS, DISK_CONF_RAID0)
		}
		if conf.Conf == DISK_CONF_RAID10 && len(storages) > 10 {
			return fmt.Errorf("%q %q only support no more than 10 disks", DISK_DRIVER_MPT2SAS, DISK_CONF_RAID10)
		}
	}

	if driver == DISK_DRIVER_MEGARAID && conf.Strip != nil {
		minStripSize := storages[0].MinStripSize
		maxStripSize := storages[0].MaxStripSize
		if maxStripSize != -1 && minStripSize != -1 {
			size := *conf.Strip
			if size > maxStripSize || size < minStripSize {
				return fmt.Errorf("%q input strip size out of range(%d, %d), input: %d", DISK_DRIVER_MEGARAID, minStripSize, maxStripSize, size)
			}
		}
	}

	return nil
}

func GetStoragesMinSize(ss []*BaremetalStorage) int64 {
	minSize := int64(-1)
	for _, s := range ss {
		if minSize < 0 || minSize > s.Size {
			minSize = s.Size
		}
	}
	return minSize
}

func CalculateSize(conf string, storages []*BaremetalStorage) int64 {
	if conf == "" {
		conf = DEFAULT_DISK_CONF
	}
	var size int64

	if conf == DISK_CONF_RAID5 {
		size = GetStoragesMinSize(storages) * int64(len(storages)-1)
	} else if sets.NewString(DISK_CONF_RAID10, DISK_CONF_RAID1).Has(conf) {
		size = GetStoragesMinSize(storages) * int64((len(storages) / 2))
	} else {
		for _, s := range storages {
			size += s.Size
		}
	}
	return size
}

func GetSplitSizes(size int64, splitConf string) []int64 {
	ssizes := strings.Split(splitConf, ",")
	isizes := make([]int64, len(ssizes))
	leftoverIdx := -1
	subtotal := int64(0)
	for index := range ssizes {
		if strings.HasSuffix(ssizes[index], "%") {
			ssizeFloat64, err := strconv.ParseFloat(ssizes[index][:len(ssizes[index])-1], 64)
			if err != nil {
				log.Errorf("GetSplitSizes ParseFloat err: %v", err)
				continue
			}
			isizes[index] = int64(ssizeFloat64 / float64(100) * float64(size))
			subtotal += isizes[index]
		} else if ssizes[index] != "" {
			isizes[index], _ = utils.GetSizeMB(ssizes[index], "M")
			subtotal += isizes[index]
		} else {
			if leftoverIdx >= 0 {
				log.Errorf("%v", ErrMoreThanOneSizeUnspecificSplit)
				return []int64{}
			}
			leftoverIdx = index
		}
	}
	if leftoverIdx >= 0 {
		isizes[leftoverIdx] = size - subtotal
		if isizes[leftoverIdx] <= 0 {
			log.Errorf("%v", ErrNoMoreSpaceForUnspecificSplit)
			return []int64{}
		}
	} else {
		if subtotal > size {
			log.Errorf("%v", ErrSubtotalOfSplitExceedsDiskSize)
			return []int64{}
		}
	}
	return isizes
}

func ExpandNoneConf(layouts []Layout) (ret []Layout) {
	for _, layout := range layouts {
		if layout.Conf.Conf == DISK_CONF_NONE && len(layout.Disks) >= 1 {
			conf := layout.Conf
			conf.Count = 1
			for _, disk := range layout.Disks {
				ret = append(ret, Layout{Disks: []*BaremetalStorage{disk}, Conf: conf, Size: disk.Size})
			}
		} else {
			ret = append(ret, layout)
		}
	}
	return ret
}

func GetLayoutRaidConfig(layouts []Layout) []*api.BaremetalDiskConfig {
	var disk []*BaremetalStorage
	ret := make([]*api.BaremetalDiskConfig, 0)
	for _, layout := range layouts {
		if layout.Conf.Conf == DISK_CONF_NONE &&
			sets.NewString(DISK_DRIVER_LINUX, DISK_DRIVER_PCIE).Has(layout.Disks[0].Driver) {
			continue
		}
		if !reflect.DeepEqual(disk, layout.Disks) {
			ret = append(ret, layout.Conf)
			disk = layout.Disks
			lastConf := ret[len(ret)-1]
			/*
				if 'size' in ret[-1]:
				    ret[-1]['size'] = [ret[-1]['size']]
			*/
			if lastConf.Size == nil {
				lastConf.Size = make([]int64, 0)
			}
		} else {
			lastConf := ret[len(ret)-1]
			lastConf.Size = append(lastConf.Size, layout.Conf.Size...)
		}
	}
	return ret
}

func CalculateLayout(confs []*api.BaremetalDiskConfig, storages []*BaremetalStorage) (layouts []Layout, err error) {
	var confIdx = 0
	for len(storages) > 0 {
		var conf *api.BaremetalDiskConfig
		if confIdx < len(confs) {
			conf = confs[confIdx]
			confIdx += 1
		} else {
			noneConf, _ := ParseDiskConfig(DISK_CONF_NONE)
			conf = &noneConf
		}
		selected, storage1 := RetrieveStorages(conf, storages)
		storages = storage1
		if len(selected) == 0 {
			err = fmt.Errorf("Not found matched storages by config: %#v", conf)
			return
		}
		resultErr := MeetConfig(conf, selected)
		if resultErr != nil {
			err = fmt.Errorf("selected storages %#v not meet baremetal dick config: %#v, err: %v", selected, conf, resultErr)
			return
		}
		sz := CalculateSize(conf.Conf, selected)
		if len(conf.Splits) == 0 {
			layouts = append(layouts, Layout{
				Disks: selected,
				Conf:  conf,
				Size:  sz,
			})
		} else {
			splitSizes := GetSplitSizes(sz, conf.Splits)
			conf.Size = splitSizes
			layouts = append(layouts, Layout{
				Disks: selected,
				Conf:  conf,
				Size:  sz,
			})
		}
	}
	if confIdx < len(confs) {
		err = fmt.Errorf("Not enough disks to meet configuration")
	}
	return
}

func expandLayoutSplits(layouts []Layout) []Layout {
	ret := make([]Layout, 0)
	for _, l := range layouts {
		splitSizes := GetSplitSizes(l.Size, l.Conf.Splits)
		if len(splitSizes) <= 0 {
			ret = append(ret, l)
		} else {
			for _, ssz := range splitSizes {
				subLayout := l
				subLayout.Size = ssz
				ret = append(ret, subLayout)
			}
		}
	}
	return ret
}

func IsDisksAllocable(layouts []Layout, disks []*api.DiskConfig) bool {
	allocable, _ := CheckDisksAllocable(layouts, disks)
	return allocable
}

func CheckDisksAllocable(layouts []Layout, disks []*api.DiskConfig) (bool, []*api.DiskConfig) {
	layouts = ExpandNoneConf(layouts)
	layouts = expandLayoutSplits(layouts)
	storeIndex := 0
	storeFreeSize := int64(-1)
	diskIndex := 0
	layoutLen := len(layouts)
	for _, disk := range disks {
		if storeIndex >= layoutLen {
			break
		}
		if storeFreeSize < 0 {
			storeFreeSize = layouts[storeIndex].Size - 2 // start, end space
		}
		if disk.SizeMb > 0 {
			if storeFreeSize >= int64(disk.SizeMb) {
				storeFreeSize -= int64(disk.SizeMb)
				diskIndex++
				if storeFreeSize == 0 {
					storeIndex++
					storeFreeSize = -1
				}
			} else {
				storeIndex++
				storeFreeSize = -1
			}
		} else {
			diskIndex++
			storeIndex++
			storeFreeSize = -1
		}
	}
	if diskIndex < len(disks) {
		return false, nil
	}
	extraDisks := make([]*api.DiskConfig, 0)
	for ; storeIndex < layoutLen; storeIndex += 1 {
		disk := api.DiskConfig{SizeMb: -1}
		extraDisks = append(extraDisks, &disk)
	}
	return true, extraDisks
}

func NewBaremetalDiskConfigs(dss ...string) ([]*api.BaremetalDiskConfig, error) {
	ret := make([]*api.BaremetalDiskConfig, 0)
	for _, ds := range dss {
		r, err := ParseDiskConfig(ds)
		if err != nil {
			return nil, err
		}
		ret = append(ret, &r)
	}
	return ret, nil
}

func getStorageDiskType(isRotate bool) string {
	if isRotate {
		return HDD_DISK_SPEC_TYPE
	}
	return SSD_DISK_SPEC_TYPE
}

func NewDiskSpec(s *BaremetalStorage, index int) *api.DiskSpec {
	return &api.DiskSpec{
		Type:       getStorageDiskType(s.Rotate),
		Size:       s.Size,
		StartIndex: index,
		EndIndex:   index,
		Count:      1,
	}
}

func IsDiskSpecSameAs(ds *api.DiskSpec, s *BaremetalStorage, index int) bool {
	if ds.Size != s.Size {
		return false
	}
	dType := getStorageDiskType(s.Rotate)
	if ds.Type != dType {
		return false
	}
	// discontinuity check
	if ds.EndIndex != (index - 1) {
		return false
	}
	return true
}

func addDiskSpecStorage(ds *api.DiskSpec, s *BaremetalStorage, index int) {
	ds.Count++
	if s.Index != 0 {
		ds.EndIndex = index
	} else {
		ds.EndIndex++
	}
}

func GetDiskSpecs(storages []*BaremetalStorage) []*api.DiskSpec {
	diskSpecs := make([]*api.DiskSpec, 0)

	for idx, s := range storages {
		var lastSpec *api.DiskSpec
		if len(diskSpecs) != 0 {
			lastSpec = diskSpecs[len(diskSpecs)-1]
		}
		if lastSpec == nil || !IsDiskSpecSameAs(lastSpec, s, idx) {
			ds := NewDiskSpec(s, idx)
			diskSpecs = append(diskSpecs, ds)
		} else {
			addDiskSpecStorage(lastSpec, s, idx)
		}
	}
	return diskSpecs
}

func getStoragesByDriver(driver string, storages []*BaremetalStorage) []*BaremetalStorage {
	ret := make([]*BaremetalStorage, 0)
	for _, s := range storages {
		if s.Driver == driver {
			ret = append(ret, s)
		}
	}
	return ret
}

func groupByAdapter(storages []*BaremetalStorage) map[string][]*BaremetalStorage {
	ret := make(map[string][]*BaremetalStorage)
	for _, storage := range storages {
		adapter := storage.Adapter
		adapterKey := fmt.Sprintf("adapter%d", adapter)
		oldStorages := ret[adapterKey]
		if len(oldStorages) == 0 {
			ret[adapterKey] = []*BaremetalStorage{storage}
		} else {
			oldStorages = append(oldStorages, storage)
			ret[adapterKey] = oldStorages
		}
	}
	return ret
}

func getSpec(storages []*BaremetalStorage) api.DiskAdapterSpec {
	ret := make(map[string][]*api.DiskSpec)
	for adapterKey, newStorages := range groupByAdapter(storages) {
		if len(newStorages) == 0 {
			continue
		}
		ret[adapterKey] = GetDiskSpecs(newStorages)
	}
	return ret
}

func GetDiskSpecV2(storages []*BaremetalStorage) api.DiskDriverSpec {
	spec := make(map[string]api.DiskAdapterSpec)
	for _, driver := range DISK_DRIVERS.List() {
		driverStorages := getStoragesByDriver(driver, storages)
		if len(driverStorages) == 0 {
			continue
		}
		spec[driver] = getSpec(storages)
	}
	return spec
}

type DiskConfiguration struct {
	Driver     string
	Adapter    int
	RaidConfig string
	Block      int64
	Size       int64
}

func GetDiskConfigurations(layouts []Layout) []DiskConfiguration {
	disks := make([]DiskConfiguration, 0)
	for _, rr := range layouts {
		driver := rr.Disks[0].Driver
		adapter := rr.Disks[0].Adapter
		block := rr.Disks[0].GetBlock()
		raidConf := rr.Conf.Conf
		if raidConf == DISK_CONF_NONE {
			for _, d := range rr.Disks {
				disks = append(disks, DiskConfiguration{
					Driver:     driver,
					Adapter:    adapter,
					RaidConfig: raidConf,
					Block:      block,
					Size:       d.Size,
				})
			}
		} else {
			if len(rr.Conf.Size) != 0 {
				for _, sz := range rr.Conf.Size {
					disks = append(disks, DiskConfiguration{
						Driver:     driver,
						Adapter:    adapter,
						RaidConfig: raidConf,
						Block:      block,
						Size:       sz,
					})
				}
			} else {
				disks = append(disks, DiskConfiguration{
					Driver:     driver,
					Adapter:    adapter,
					RaidConfig: raidConf,
					Block:      block,
					Size:       rr.Size,
				})
			}
		}
	}
	return disks
}

type DriverAdapterDiskConfig struct {
	Driver  string
	Adapter int
	Configs []*api.BaremetalDiskConfig
}

func GroupLayoutResultsByDriverAdapter(layouts []Layout) []*DriverAdapterDiskConfig {
	ret := make([]*DriverAdapterDiskConfig, 0)
	tbl := make(map[string]*DriverAdapterDiskConfig)
	for _, layout := range layouts {
		driver := layout.Disks[0].Driver
		adapter := layout.Disks[0].Adapter
		key := fmt.Sprintf("%s.%d", driver, adapter)
		if item, ok := tbl[key]; ok {
			item.Configs = append(item.Configs, layout.Conf)
		} else {
			item := &DriverAdapterDiskConfig{
				Driver:  driver,
				Adapter: adapter,
				Configs: []*api.BaremetalDiskConfig{layout.Conf},
			}
			ret = append(ret, item)
			tbl[key] = item
		}
	}
	return ret
}

func ValidateDiskConfigs(confs []*api.BaremetalDiskConfig) error {
	if len(confs) == 0 {
		return nil
	}
	for idx, conf := range confs {
		if conf.Conf != DISK_CONF_NONE {
			// raid validate
			if idx > 0 {
				preConf := confs[idx-1]
				if preConf.Conf == DISK_CONF_NONE {
					return fmt.Errorf("Raid config %d must before none raid config", idx)
				}
			}
		} else {
			// none raid validate
			if idx+1 == len(confs) {
				return nil
			}
			restConfs := confs[idx+1:]
			hasRaidConf := false
			for _, restConf := range restConfs {
				if restConf.Conf != DISK_CONF_NONE {
					hasRaidConf = true
				}
			}
			if hasRaidConf {
				return fmt.Errorf("Raid config after none raid config %d", idx)
			}
		}
	}
	return nil
}
