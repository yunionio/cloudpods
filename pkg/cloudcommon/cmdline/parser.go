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

package cmdline

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis/compute"
)

var (
	ErrorEmptyDesc = errors.Errorf("Empty description")
)

// ParseSchedtagConfig desc format: <schedtagName>:<strategy>:<resource_type>
func ParseSchedtagConfig(desc string) (*compute.SchedtagConfig, error) {
	if len(desc) == 0 {
		return nil, ErrorEmptyDesc
	}
	parts := strings.Split(desc, ":")
	if len(parts) < 2 {
		return nil, fmt.Errorf("Invalid desc: %s", desc)
	}
	strategy := parts[1]
	if !utils.IsInStringArray(strategy, compute.STRATEGY_LIST) {
		return nil, fmt.Errorf("Invalid strategy: %s", strategy)
	}
	conf := &compute.SchedtagConfig{
		Id:       parts[0],
		Strategy: parts[1],
	}
	if len(parts) == 3 {
		conf.ResourceType = parts[2]
	}
	return conf, nil
}

// ParseResourceSchedtagConfig desc format: <idx>:<schedtagName>:<strategy>
func ParseResourceSchedtagConfig(desc string) (int, *compute.SchedtagConfig, error) {
	if len(desc) == 0 {
		return 0, nil, ErrorEmptyDesc
	}
	parts := strings.Split(desc, ":")
	if len(parts) != 3 {
		return 0, nil, fmt.Errorf("Invalid desc: %s", desc)
	}
	idx, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, nil, err
	}
	tag, err := ParseSchedtagConfig(fmt.Sprintf("%s:%s", parts[1], parts[2]))
	if err != nil {
		return 0, nil, err
	}
	return idx, tag, nil
}

func ParseDiskConfig(diskStr string, idx int) (*compute.DiskConfig, error) {
	if len(diskStr) == 0 {
		return nil, ErrorEmptyDesc
	}
	diskConfig := new(compute.DiskConfig)
	diskConfig.Index = idx

	// default backend and medium type
	diskConfig.Backend = "" // STORAGE_LOCAL
	diskConfig.Medium = ""
	diskConfig.SizeMb = -1

	oldPart, newPart := []string{}, []string{}
	for _, d0 := range strings.Split(diskStr, ",") {
		for _, d1 := range strings.Split(d0, ":") {
			if len(d1) == 0 {
				continue
			}
			if strings.Contains(d1, "=") {
				newPart = append(newPart, d1)
				continue
			}
			oldPart = append(oldPart, d1)
		}
	}
	for _, p := range oldPart {
		if regutils.MatchSize(p) {
			diskConfig.SizeMb, _ = fileutils.GetSizeMb(p, 'M', 1024)
		} else if utils.IsInStringArray(p, osprofile.FS_TYPES) {
			diskConfig.Fs = p
		} else if utils.IsInStringArray(p, osprofile.IMAGE_FORMAT_TYPES) {
			diskConfig.Format = p
		} else if utils.IsInStringArray(p, osprofile.DISK_DRIVERS) {
			diskConfig.Driver = p
		} else if utils.IsInStringArray(p, osprofile.DISK_CACHE_MODES) {
			diskConfig.Cache = p
		} else if utils.IsInStringArray(p, compute.DISK_TYPES) {
			diskConfig.Medium = p
		} else if utils.IsInStringArray(p, []string{compute.DISK_TYPE_VOLUME}) {
			diskConfig.DiskType = p
		} else if p[0] == '/' {
			diskConfig.Mountpoint = p
		} else if p == "autoextend" {
			diskConfig.SizeMb = -1
		} else if utils.IsInStringArray(p, compute.STORAGE_TYPES) {
			diskConfig.Backend = p
		} else if len(p) > 0 {
			diskConfig.ImageId = p
		}
	}
	for _, p := range newPart {
		info := strings.Split(p, "=")
		if len(info) != 2 {
			return nil, errors.Errorf("invalid disk description %s", p)
		}
		var err error
		desc, str := info[0], info[1]
		switch desc {
		case "size":
			diskConfig.SizeMb, err = fileutils.GetSizeMb(str, 'M', 1024)
			if err != nil {
				return nil, errors.Errorf("invalid disk size %s", str)
			}
		case "fs":
			if !utils.IsInStringArray(str, osprofile.FS_TYPES) {
				return nil, errors.Errorf("invalid disk fs %s, allow choices: %s", str, osprofile.FS_TYPES)
			}
			diskConfig.Fs = str
		case "format":
			if !utils.IsInStringArray(str, osprofile.IMAGE_FORMAT_TYPES) {
				return nil, errors.Errorf("invalid disk format %s, allow choices: %s", str, osprofile.IMAGE_FORMAT_TYPES)
			}
			diskConfig.Format = str
		case "driver":
			if !utils.IsInStringArray(str, osprofile.DISK_DRIVERS) {
				return nil, errors.Errorf("invalid disk driver %s, allow choices: %s", str, osprofile.DISK_DRIVERS)
			}
			diskConfig.Driver = str
		case "cache", "cache_mode":
			if !utils.IsInStringArray(str, osprofile.DISK_CACHE_MODES) {
				return nil, errors.Errorf("invalid disk cache mode %s, allow choices: %s", str, osprofile.DISK_CACHE_MODES)
			}
			diskConfig.Cache = str
		case "medium":
			if !utils.IsInStringArray(str, compute.DISK_TYPES) {
				return nil, errors.Errorf("invalid disk medium type %s, allow choices: %s", str, compute.DISK_TYPES)
			}
			diskConfig.Medium = str
		case "type", "disk_type":
			diskTypes := []string{compute.DISK_TYPE_SYS, compute.DISK_TYPE_DATA}
			if !utils.IsInStringArray(str, diskTypes) {
				return nil, errors.Errorf("invalid disk type %s, allow choices: %s", str, diskTypes)
			}
			diskConfig.DiskType = str
		case "mountpoint":
			diskConfig.Mountpoint = str
		case "storage_type", "backend":
			diskConfig.Backend = str
		case "snapshot", "snapshot_id":
			diskConfig.SnapshotId = str
		case "disk", "disk_id":
			diskConfig.DiskId = str
		case "storage", "storage_id":
			diskConfig.Storage = str
		case "image", "image_id":
			diskConfig.ImageId = str
		case "existing_path":
			diskConfig.ExistingPath = str
		case "boot_index":
			bootIndex, err := strconv.Atoi(str)
			if err != nil {
				return nil, errors.Wrapf(err, "parse disk boot index %s", str)
			}
			bootIndex8 := int8(bootIndex)
			diskConfig.BootIndex = &bootIndex8
		case "nvme-device-id":
			diskConfig.NVMEDevice = &compute.IsolatedDeviceConfig{
				Id: str,
			}
		case "nvme-device-model":
			diskConfig.NVMEDevice = &compute.IsolatedDeviceConfig{
				Model: str,
			}
		case "iops":
			diskConfig.Iops, _ = strconv.Atoi(str)
			if err != nil {
				return nil, errors.Wrapf(err, "parse disk iops %s", str)
			}
		case "throughput":
			diskConfig.Throughput, _ = strconv.Atoi(str)
			if err != nil {
				return nil, errors.Wrapf(err, "parse disk iops %s", str)
			}
		case "preallocation":
			if !utils.IsInStringArray(str, compute.DISK_PREALLOCATIONS) {
				return nil, errors.Errorf("invalid preallocation %s, allow choices: %s", str, compute.DISK_PREALLOCATIONS)
			}
			diskConfig.Preallocation = str
		default:
			return nil, errors.Errorf("invalid disk description %s", p)
		}
	}
	return diskConfig, nil
}

func ParseNetworkConfigByJSON(desc jsonutils.JSONObject, idx int) (*compute.NetworkConfig, error) {
	if _, ok := desc.(*jsonutils.JSONString); ok {
		descStr, _ := desc.GetString()
		return ParseNetworkConfig(descStr, idx)
	}
	conf := new(compute.NetworkConfig)
	conf.Index = idx
	err := desc.Unmarshal(conf)
	return conf, err
}

func ParseNetworkConfig(desc string, idx int) (*compute.NetworkConfig, error) {
	if len(desc) == 0 {
		return nil, ErrorEmptyDesc
	}
	parts := strings.Split(desc, ":")
	netConfig := new(compute.NetworkConfig)
	netConfig.Index = idx
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		if regutils.MatchIP4Addr(p) {
			netConfig.Address = p
		} else if regutils.MatchIP6Addr(p) {
			netConfig.Address6 = p
		} else if regutils.MatchCompactMacAddr(p) {
			netConfig.Mac = netutils.MacUnpackHex(p)
		} else if strings.HasPrefix(p, "wire=") {
			netConfig.Wire = p[len("wire="):]
		} else if strings.HasPrefix(p, "macs=") {
			macSegs := strings.Split(p[len("macs="):], ",")
			macs := make([]string, len(macSegs))
			for i := range macSegs {
				macs[i] = netutils.MacUnpackHex(macSegs[i])
			}
			netConfig.Macs = macs
		} else if strings.HasPrefix(p, "ips=") {
			netConfig.Addresses = strings.Split(p[len("ips="):], ",")
		} else if strings.HasPrefix(p, "ip6s=") {
			netConfig.Addresses6 = strings.Split(p[len("ip6s="):], ",")
		} else if p == "[require_designated_ip]" {
			netConfig.RequireDesignatedIP = true
		} else if p == "[random_exit]" {
			netConfig.Exit = true
		} else if p == "[random]" {
			netConfig.Exit = false
		} else if p == "[private]" {
			netConfig.Private = true
		} else if p == "[reserved]" {
			netConfig.Reserved = true
		} else if p == "[teaming]" {
			netConfig.RequireTeaming = true
		} else if p == "[try-teaming]" {
			netConfig.TryTeaming = true
		} else if strings.HasPrefix(p, "standby-port=") {
			netConfig.StandbyPortCount, _ = strconv.Atoi(p[len("standby-port="):])
		} else if strings.HasPrefix(p, "standby-addr=") {
			netConfig.StandbyAddrCount, _ = strconv.Atoi(p[len("standby-addr="):])
		} else if utils.IsInStringArray(p, []string{"virtio", "e1000", "vmxnet3"}) {
			netConfig.Driver = p
		} else if strings.HasPrefix(p, "num-queues=") {
			netConfig.NumQueues, _ = strconv.Atoi(p[len("num-queues="):])
		} else if regutils.MatchSize(p) {
			bw, err := fileutils.GetSizeMb(p, 'M', 1000)
			if err != nil {
				return nil, err
			}
			netConfig.BwLimit = bw
		} else if p == "[vip]" {
			netConfig.Vip = true
		} else if strings.HasPrefix(p, "sriov-nic-id=") {
			netConfig.SriovDevice = &compute.IsolatedDeviceConfig{
				Id: p[len("sriov-nic-id="):],
			}
		} else if strings.HasPrefix(p, "sriov-nic-model=") {
			netConfig.SriovDevice = &compute.IsolatedDeviceConfig{
				Model: p[len("sriov-nic-model="):],
			}
		} else if strings.HasPrefix(p, "rx-traffic-limit=") {
			var err error
			netConfig.RxTrafficLimit, err = strconv.ParseInt(p[len("rx-traffic-limit="):], 10, 0)
			if err != nil {
				return nil, errors.Wrap(err, "parse rx-traffic-limit")
			}
		} else if strings.HasPrefix(p, "tx-traffic-limit=") {
			var err error
			netConfig.TxTrafficLimit, err = strconv.ParseInt(p[len("tx-traffic-limit="):], 10, 0)
			if err != nil {
				return nil, errors.Wrap(err, "parse tx-traffic-limit")
			}
		} else if utils.IsInStringArray(p, compute.ALL_NETWORK_TYPES) {
			netConfig.NetType = p
		} else {
			netConfig.Network = p
		}
	}
	return netConfig, nil
}

func ParseIsolatedDevice(desc string, idx int) (*compute.IsolatedDeviceConfig, error) {
	if len(desc) == 0 {
		return nil, ErrorEmptyDesc
	}
	if idx < 0 {
		return nil, fmt.Errorf("Invalid index: %d", idx)
	}
	dev := new(compute.IsolatedDeviceConfig)
	parts := strings.Split(desc, ":")
	for _, p := range parts {
		if regutils.MatchUUIDExact(p) {
			dev.Id = p
		} else if utils.IsInStringArray(p, compute.VALID_PASSTHROUGH_TYPES) {
			dev.DevType = p
		} else if strings.HasPrefix(p, "vendor=") {
			dev.Vendor = p[len("vendor="):]
		} else {
			dev.Model = p
		}
	}
	return dev, nil
}

func ParseBaremetalDiskConfig(desc string) (*compute.BaremetalDiskConfig, error) {
	bdc := new(compute.BaremetalDiskConfig)
	bdc.Type = compute.DISK_TYPE_HYBRID
	bdc.Conf = compute.DISK_CONF_NONE
	bdc.Count = 0
	desc = strings.ToLower(desc)
	if len(desc) == 0 {
		return bdc, nil
	}

	parts := strings.Split(desc, ":")
	drvMap := make(map[string]string)
	for _, drv := range compute.DISK_DRIVERS.List() {
		drvMap[strings.ToLower(drv)] = drv
	}

	for _, p := range parts {
		if len(p) == 0 {
			continue
		} else if utils.IsInStringArray(p, compute.DISK_TYPES) {
			bdc.Type = p
		} else if compute.DISK_CONFS.Has(p) {
			bdc.Conf = p
		} else if drv, ok := drvMap[p]; ok {
			bdc.Driver = drv
		} else if utils.IsMatchInteger(p) {
			bdc.Count, _ = strconv.ParseInt(p, 0, 0)
		} else if len(p) > 2 && p[0] == '[' && p[len(p)-1] == ']' {
			rg, err1 := ParseRange(p[1:(len(p) - 1)])
			if err1 != nil {
				return nil, err1
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
			return nil, fmt.Errorf("ParseDiskConfig unkown option %q", p)
		}
	}

	return bdc, nil
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

// return KB
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
