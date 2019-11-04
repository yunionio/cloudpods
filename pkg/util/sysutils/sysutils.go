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

package sysutils

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/compute"
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

func ParseDMISysinfo(lines []string) (*types.SDMISystemInfo, error) {
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
	info := types.SDMISystemInfo{}
	err := DumpMapToObject(ret, &info)
	if err != nil {
		return nil, err
	}
	if strings.ToLower(info.Version) == "none" {
		info.Version = ""
	}
	return &info, nil
}

func ParseCPUInfo(lines []string) (*types.SCPUInfo, error) {
	cnt := 0
	var (
		model     string
		freq      string
		cache     string
		microcode string
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
		if len(microcode) == 0 && strings.HasPrefix(line, "microcode") {
			microcode = lv(line)
		}
		if strings.HasPrefix(line, "processor") {
			cnt += 1
		}
	}
	if len(model) == 0 {
		log.Errorf("Failed to get cpu model")
	}
	if len(freq) == 0 {
		log.Errorf("Failed to get cpu MHz")
	}
	if len(cache) == 0 {
		log.Errorf("Failed to get cpu cache size")
	}
	model = strings.TrimSpace(model)
	info := &types.SCPUInfo{
		Count:     cnt,
		Model:     model,
		Microcode: microcode,
	}
	if len(cache) > 0 {
		info.Cache, _ = strconv.Atoi(cache)
	}
	if len(freq) > 0 {
		freqF, _ := strconv.ParseFloat(freq, 32)
		info.Freq = int(freqF)
	}
	return info, nil
}

func ParseDMICPUInfo(lines []string) *types.SDMICPUInfo {
	cnt := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "Processor Information") {
			cnt += 1
		}
	}
	return &types.SDMICPUInfo{
		Nodes: cnt,
	}
}

func ParseDMIMemInfo(lines []string) *types.SDMIMemInfo {
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
	return &types.SDMIMemInfo{Total: size}
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

func ParseNicInfo(lines []string) []*types.SNicDevInfo {
	ret := make([]*types.SNicDevInfo, 0)
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
			ret = append(ret, &types.SNicDevInfo{
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

func ParseDiskInfo(lines []string, driver string) []*types.SDiskInfo {
	ret := make([]*types.SDiskInfo, 0)
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
		ret = append(ret, &types.SDiskInfo{
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

func ParsePCIEDiskInfo(lines []string) []*types.SDiskInfo {
	return ParseDiskInfo(lines, baremetal.DISK_DRIVER_PCIE)
}

func ParseSCSIDiskInfo(lines []string) []*types.SDiskInfo {
	return ParseDiskInfo(lines, baremetal.DISK_DRIVER_LINUX)
}

func GetSecureTTYs(lines []string) []string {
	ttys := []string{}
	for _, l := range lines {
		if len(l) == 0 {
			continue
		}
		if strings.HasPrefix(l, "#") {
			continue
		}
		ttys = append(ttys, l)
	}
	return ttys
}

func GetSerialPorts(lines []string) []string {
	// http://wiki.networksecuritytoolkit.org/index.php/Console_Output_and_Serial_Terminals
	ret := []string{}
	for _, l := range lines {
		if strings.Contains(l, "CTS") || strings.Contains(l, "RTS") {
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

// ParseSGMap parse command 'sg_map -x' outputs:
//
//  /dev/sg1 0 2 0 0 0 /dev/sda
//  /dev/sg2 0 2 1 0 0 /dev/sdb
//  /dev/sg3 0 2 2 0 0 /dev/sdc
//
func ParseSGMap(lines []string) []compute.SGMapItem {
	ret := make([]compute.SGMapItem, 0)
	for _, l := range lines {
		items := strings.Fields(l)
		if len(items) != 7 {
			continue
		}
		hostNum, _ := strconv.Atoi(items[1])
		bus, _ := strconv.Atoi(items[2])
		scsiId, _ := strconv.Atoi(items[3])
		lun, _ := strconv.Atoi(items[4])
		typ, _ := strconv.Atoi(items[5])
		ret = append(ret, compute.SGMapItem{
			SGDeviceName:    items[0],
			HostNumber:      hostNum,
			Bus:             bus,
			SCSIId:          scsiId,
			Lun:             lun,
			Type:            typ,
			LinuxDeviceName: items[6],
		})
	}
	return ret
}

func Start(closeFd bool, args ...string) (p *os.Process, err error) {
	if args[0], err = exec.LookPath(args[0]); err == nil {
		var procAttr os.ProcAttr
		if closeFd {
			procAttr.Files = []*os.File{nil, nil, nil}
		} else {
			procAttr.Files = []*os.File{os.Stdin,
				os.Stdout, os.Stderr}
		}
		p, err := os.StartProcess(args[0], args, &procAttr)
		if err == nil {
			return p, nil
		}
	}
	return nil, err
}

func ParseIPMIUser(lines []string) []compute.IPMIUser {
	ret := make([]compute.IPMIUser, 0)
	for _, l := range lines {
		if strings.HasPrefix(l, "ID") {
			continue
		}
		fields := strings.Fields(l)
		if strings.Contains(l, "Empty User") {
			id, err := strconv.Atoi(fields[0])
			if err != nil {
				continue
			}
			ret = append(ret, compute.IPMIUser{Id: id})
			continue
		}
		if len(fields) != 6 {
			continue
		}
		id, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		ret = append(ret, compute.IPMIUser{
			Id:   id,
			Name: fields[1],
			Priv: fields[5],
		})
	}
	return ret
}

const (
	UUID_EMPTY = "00000000-0000-0000-0000-000000000000"
)

func NormalizeUuid(uuid string) string {
	uuid = strings.ToLower(uuid)
	if uuid == UUID_EMPTY {
		uuid = ""
	}
	return uuid
}
