package baremetal

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
)

// return bytes
func parseStrip(stripStr string, defaultSize string) int64 {
	size, _ := utils.GetSize(stripStr, defaultSize, 1024)
	return size / 1024
}

func parseRangeStr(str string) (ret []int64, err error) {
	im := utils.IsMatchInteger
	errGen := func(e string) error {
		return fmt.Errorf("Incorrect range str: %q", e)
	}
	rs := strings.Split(str, "-")
	if len(rs) != 2 {
		err = errGen(str)
		return
	}

	bs, es := rs[0], rs[1]
	if !im(bs) {
		err = errGen(str)
		return
	}
	if !im(es) {
		err = errGen(str)
		return
	}

	begin, _ := strconv.ParseInt(bs, 10, 64)
	end, _ := strconv.ParseInt(es, 10, 64)

	if begin > end {
		begin, end = end, begin
	}

	for i := begin; i <= end; i++ {
		ret = append(ret, i)
	}
	return
}

// range string should be: "1-3", "3"
func _parseRange(str string) (ret []int64, err error) {
	if len(str) == 0 {
		return
	}

	// exclude "," symbol
	if len(str) == 1 && !utils.IsMatchInteger(str) {
		return
	}

	// add int string
	if utils.IsMatchInteger(str) {
		i, _ := strconv.ParseInt(str, 10, 64)
		ret = append(ret, i)
		return
	}

	// add rang like string, "2-10" etc.
	ret, err = parseRangeStr(str)
	return
}

func ParseRange(rangeStr string) (ret []int64, err error) {
	rss := regexp.MustCompile(`[\s,]+`).Split(rangeStr, -1)
	intSet := sets.NewInt64()

	for _, rs := range rss {
		r, err1 := _parseRange(rs)
		if err1 != nil {
			err = err1
			return
		}
		intSet.Insert(r...)
	}
	ret = intSet.List()
	return
}

func ParseDiskConfig(desc string) (bdc BaremetalDiskConfig, err error) {
	bdc.Type = DISK_TYPE_HYBRID
	bdc.Conf = DISK_CONF_NONE
	bdc.Count = 0

	desc = strings.ToLower(desc)
	if desc == "" {
		return
	}

	parts := strings.Split(desc, ":")
	drvMap := make(map[string]string)
	for _, drv := range DISK_DRIVERS.List() {
		drvMap[strings.ToLower(drv)] = drv
	}
	for _, p := range parts {
		if len(p) == 0 {
			continue
		} else if DISK_TYPES.Has(p) {
			bdc.Type = p
		} else if DISK_CONFS.Has(p) {
			bdc.Conf = p
		} else if drv, ok := drvMap[p]; ok {
			bdc.Driver = drv
		} else if utils.IsMatchInteger(p) {
			bdc.Count, _ = strconv.ParseInt(p, 0, 0)
		} else if len(p) > 2 && p[0] == '[' && p[len(p)-1] == ']' {
			rg, err1 := ParseRange(p[1:(len(p) - 1)])
			if err1 != nil {
				err = err1
				return
			}
			bdc.Range = rg
		} else if len(p) > 2 && p[0] == '(' && p[len(p)-1] == ')' {
			bdc.Splits = p[1 : len(p)-1]
		} else if utils.HasPrefix(p, "strip") {
			strip := parseStrip(p[len("strip"):], "k")
			bdc.Strip = &strip
		} else if utils.HasPrefix(p, "adapter") {
			ada, _ := strconv.ParseInt(p[len("adapter"):], 0, 64)
			pada := int(ada)
			bdc.Adapter = &pada
		} else if p == "ra" {
			hasRA := true
			bdc.RA = &hasRA
		} else if p == "nora" {
			noRA := false
			bdc.RA = &noRA
		} else if p == "wt" {
			wt := true
			bdc.WT = &wt
		} else if p == "wb" {
			wt := false
			bdc.WT = &wt
		} else if p == "direct" {
			direct := true
			bdc.Direct = &direct
		} else if p == "cached" {
			direct := false
			bdc.Direct = &direct
		} else if p == "cachedbadbbu" {
			cached := true
			bdc.Cachedbadbbu = &cached
		} else if p == "nocachedbadbbu" {
			cached := false
			bdc.Cachedbadbbu = &cached
		} else {
			err = fmt.Errorf("ParseDiskConfig unkown option %q", p)
			return
		}
	}

	return
}

func isDiskConfigStorageMatch(
	config *BaremetalDiskConfig,
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

func RetrieveStorages(diskConfig *BaremetalDiskConfig, storages []*BaremetalStorage) (selected, rest []*BaremetalStorage) {
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

	for _, storage := range storages {
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
	Disks []*BaremetalStorage  `json:"disks"`
	Conf  *BaremetalDiskConfig `json:"conf"`
	Size  int64                `json:"size"`
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
	conf *BaremetalDiskConfig,
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
		if maxStripSize != 0 && minStripSize != 0 {
			size := *conf.Strip
			if size > maxStripSize || size < minStripSize {
				return fmt.Errorf("%q input strip size out of range(%d, %d)", DISK_DRIVER_MEGARAID, minStripSize, maxStripSize)
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

func CalculateLayout(confs []*BaremetalDiskConfig, storages []*BaremetalStorage) (layouts []Layout, err error) {
	var confIdx = 0
	for len(storages) > 0 {
		var conf *BaremetalDiskConfig
		if confIdx < len(confs) {
			conf = confs[confIdx]
			confIdx += 1
		} else {
			noneConf, _ := ParseDiskConfig("none")
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

func CheckDisksAllocable(layouts []Layout, disks []*Disk) bool {
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
		if disk.Size > 0 {
			if storeFreeSize >= disk.Size {
				storeFreeSize -= disk.Size
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
		return false
	}
	return true
}

func NewBaremetalDiskConfigs(dss ...string) ([]*BaremetalDiskConfig, error) {
	ret := make([]*BaremetalDiskConfig, 0)
	for _, ds := range dss {
		r, err := ParseDiskConfig(ds)
		if err != nil {
			return nil, err
		}
		ret = append(ret, &r)
	}
	return ret, nil
}

type SpecSizeCount map[string]int
type DiskSpec map[string]SpecSizeCount

func GetDiskSpec(storages []*BaremetalStorage) DiskSpec {
	diskSpec := make(map[string]SpecSizeCount)

	for _, s := range storages {
		var dtype string
		if s.Rotate {
			dtype = HDD_DISK_SPEC_TYPE
		} else {
			dtype = SSD_DISK_SPEC_TYPE
		}

		sizeStr := fmt.Sprintf("%d", s.Size)
		sc, ok := diskSpec[dtype]
		if !ok {
			sc = make(map[string]int)
			diskSpec[dtype] = sc
		}
		if _, ok := sc[sizeStr]; !ok {
			sc[sizeStr] = 0
		}
		diskSpec[dtype][sizeStr] += 1
	}
	return diskSpec
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

type DiskAdapterSpecs map[string]DiskSpec

func getSpec(storages []*BaremetalStorage) DiskAdapterSpecs {
	ret := make(map[string]DiskSpec)
	for adapterKey, newStorages := range groupByAdapter(storages) {
		if len(newStorages) == 0 {
			continue
		}
		ret[adapterKey] = GetDiskSpec(newStorages)
	}
	return ret
}

type DiskDriverSpecs map[string]DiskAdapterSpecs

func GetDiskSpecV2(storages []*BaremetalStorage) DiskDriverSpecs {
	spec := make(map[string]DiskAdapterSpecs)
	for _, driver := range DISK_DRIVERS.List() {
		driverStorages := getStoragesByDriver(driver, storages)
		if len(driverStorages) == 0 {
			continue
		}
		spec[driver] = getSpec(storages)
	}
	return spec
}
