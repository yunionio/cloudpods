package sysutils

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
)

func valueOfKeyword(line string, key string) *string {
	lo := strings.ToLower(line)
	ko := strings.ToLower(key)
	pos := strings.Index(lo, ko)
	if pos >= 0 {
		val := strings.TrimSpace(line[pos+len(key):])
		return &val
	}
	return nil
}

func DumpMapToObject(data map[string]string, obj interface{}) error {
	return jsonutils.Marshal(data).Unmarshal(obj)
}

func ParseDMISysinfo(lines []string) (*types.DMISystemInfo, error) {
	if len(lines) == 0 {
		return nil, fmt.Errorf("Empty input")
	}
	keys := map[string]string{
		"manufacture": "Manufacturer:",
		"model":       "Product Name:",
		"version":     "Version:",
		"sn":          "Serial Number:",
	}
	ret := make(map[string]string)
	for _, line := range lines {
		for key, keyword := range keys {
			val := valueOfKeyword(line, keyword)
			if val != nil {
				ret[key] = *val
			}
		}
	}
	info := types.DMISystemInfo{}
	err := DumpMapToObject(ret, &info)
	if err != nil {
		return nil, err
	}
	if strings.ToLower(info.Version) == "none" {
		info.Version = ""
	}
	return &info, nil
}

func ParseCPUInfo(lines []string) (*types.CPUInfo, error) {
	cnt := 0
	var (
		model string
		freq  string
		cache string
	)
	lv := func(line string) string {
		return strings.TrimSpace(line[strings.Index(line, ":")+1:])
	}
	for _, line := range lines {
		if len(model) == 0 && strings.HasPrefix(line, "model name") {
			model = lv(line)
		}
		if len(freq) == 0 && strings.HasPrefix(line, "cpu MHz") {
			freq = lv(line)
		}
		if len(cache) == 0 && strings.HasPrefix(line, "cache size") {
			cache = strings.TrimSpace(line[strings.Index(line, ":")+1 : strings.Index(line, " KB")])
		}
		if strings.HasPrefix(line, "processor") {
			cnt += 1
		}
	}
	if len(model) == 0 {
		return nil, fmt.Errorf("Not found model name")
	}
	if len(freq) == 0 {
		return nil, fmt.Errorf("Not found cpu MHz")
	}
	if len(cache) == 0 {
		return nil, fmt.Errorf("Not found cache size")
	}
	model = strings.TrimSpace(model)
	info := &types.CPUInfo{
		Count: cnt,
		Model: model,
	}
	info.Cache, _ = strconv.Atoi(cache)
	freqF, _ := strconv.ParseFloat(freq, 32)
	info.Freq = int(freqF)
	return info, nil
}

func ParseDMICPUInfo(lines []string) *types.DMICPUInfo {
	cnt := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "Processor Information") {
			cnt += 1
		}
	}
	return &types.DMICPUInfo{
		Nodes: cnt,
	}
}

func ParseDMIMemInfo(lines []string) *types.DMIMemInfo {
	size := 0
	for _, line := range lines {
		val := valueOfKeyword(line, "Size:")
		if val == nil {
			continue
		}
		value := strings.ToLower(*val)
		if strings.HasSuffix(value, " mb") {
			sizeMb, err := strconv.Atoi(strings.TrimSuffix(value, " mb"))
			if err != nil {
				log.Errorf("parse MB error: %v", err)
				continue
			}
			size += sizeMb
		} else if strings.HasSuffix(value, " gb") {
			sizeGb, err := strconv.Atoi(strings.TrimSuffix(value, " gb"))
			if err != nil {
				log.Errorf("parse GB error: %v", err)
				continue
			}
			size += sizeGb * 1024
		}
	}
	return &types.DMIMemInfo{Total: size}
}

func ParseDMIIPMIInfo(lines []string) bool {
	for _, line := range lines {
		val := valueOfKeyword(line, "Interface Type:")
		if val != nil {
			return true
		}
	}
	return false
}

func ParseNicInfo(lines []string) []*types.NicDevInfo {
	ret := make([]*types.NicDevInfo, 0)
	for _, line := range lines {
		dat := strings.Split(line, " ")
		if len(dat) > 4 {
			dev := dat[0]
			mac, _ := net.ParseMAC(dat[1])
			speed, _ := strconv.Atoi(dat[2])
			up := false
			if dat[3] == "1" {
				up = true
			}
			mtu, _ := strconv.Atoi(dat[4])
			ret = append(ret, &types.NicDevInfo{
				Dev:   dev,
				Mac:   mac,
				Speed: speed,
				Up:    up,
				Mtu:   mtu,
			})
		}
	}
	return ret
}

func ParseDiskInfo(lines []string, driver string) []*types.DiskInfo {
	ret := make([]*types.DiskInfo, 0)
	for _, line := range lines {
		data := strings.Split(line, " ")
		if len(data) <= 6 {
			continue
		}
		dev := data[0]
		sector, _ := strconv.Atoi(data[1])
		size := sector * 512 / 1024 / 1024
		block, _ := strconv.Atoi(data[2])
		rotate := false
		if data[3] == "1" {
			rotate = true
		}
		kernel := data[4]
		pciCls := data[5]
		modinfo := strings.Join(data[6:], " ")
		ret = append(ret, &types.DiskInfo{
			Dev:        dev,
			Sector:     int64(sector),
			Block:      int64(block),
			Size:       int64(size),
			Rotate:     rotate,
			ModuleInfo: modinfo,
			Kernel:     kernel,
			PCIClass:   pciCls,
			Driver:     driver,
		})
	}
	return ret
}

func ParsePCIEDiskInfo(lines []string) []*types.DiskInfo {
	return ParseDiskInfo(lines, baremetal.DISK_DRIVER_PCIE)
}

func ParseSCSIDiskInfo(lines []string) []*types.DiskInfo {
	return ParseDiskInfo(lines, baremetal.DISK_DRIVER_LINUX)
}

func GetSerialPorts(lines []string) []string {
	ret := []string{}
	for _, l := range lines {
		if strings.Contains(l, "CTS") {
			pos := strings.Index(l, ":")
			if pos < 0 {
				continue
			}
			idx := l[0:pos]
			ret = append(ret, fmt.Sprintf("ttyS%s", idx))
		}
	}
	return ret
}
