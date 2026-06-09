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

package ucloud

import (
	"fmt"
	"sort"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/jsonutils"
)

// SAvailableInstanceType 对应 DescribeAvailableInstanceTypes 返回的机型信息。
type SAvailableInstanceType struct {
	Zone          string
	InstanceType  string
	Name          string
	Status        string
	ParentType    string
	Description   string
	MachineClass  string
	MachineSizes  []SMachineSize
	UHostFamilies []SUHostFamily
	Disks         []SDiskCategory
}

type SDiskCategory struct {
	Name     string
	BootDisk []SBootDiskInfo
	DataDisk []SDataDiskInfo
}

type SBootDiskInfo struct {
	Name          string
	InstantResize bool
	MaximalSize   int
	Features      []string
}

type SDataDiskInfo struct {
	Name        string
	MinimalSize int
	MaximalSize int
	Features    []string
}

type SUHostFamily struct {
	Name         string
	CpuFrequency string
}

type SMachineSize struct {
	Gpu        int
	Collection []SInstanceCollection
}

type SInstanceCollection struct {
	Cpu                int
	MemoryGB           []int
	MinimalCpuPlatform []string
}

// SInstancePackage 展平后的云主机套餐规格，Spec 格式与 instance-create 的 INSTANCETYPE 一致。
type SInstancePackage struct {
	Zone        string
	Name        string
	UHostType   string
	Status      string
	CPU         int
	MemoryMB    int
	GPU         int
	Spec        string
	Description string
}

func formatInstanceSpec(hostType string, cpu, memoryGB, gpu int) string {
	spec := fmt.Sprintf("%s.c%d.m%d", hostType, cpu, memoryGB)
	if gpu > 0 {
		spec = fmt.Sprintf("%s.g%d", spec, gpu)
	}
	return spec
}

func parseMemoryGB(obj jsonutils.JSONObject) []int {
	val, err := obj.Get("Memory")
	if err != nil || val == nil {
		return nil
	}
	switch v := val.(type) {
	case *jsonutils.JSONArray:
		items, _ := v.GetArray()
		ret := make([]int, 0, len(items))
		for _, item := range items {
			if n, err := item.Int(); err == nil {
				ret = append(ret, int(n))
			}
		}
		return ret
	default:
		if n, err := val.Int(); err == nil {
			// 兼容 Memory 为单个整数的情况：小于 512 视为 GB，否则视为 MB。
			if n >= 512 {
				return []int{int(n / 1024)}
			}
			return []int{int(n)}
		}
	}
	return nil
}

func parseStringArray(obj jsonutils.JSONObject, key string) []string {
	arr, err := obj.GetArray(key)
	if err != nil {
		return nil
	}
	ret := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, err := item.GetString(); err == nil {
			ret = append(ret, s)
		}
	}
	return ret
}

func parseInstanceCollection(obj jsonutils.JSONObject) SInstanceCollection {
	cpu, _ := obj.Int("Cpu")
	return SInstanceCollection{
		Cpu:                int(cpu),
		MemoryGB:           parseMemoryGB(obj),
		MinimalCpuPlatform: parseStringArray(obj, "MinimalCpuPlatform"),
	}
}

func parseMachineSize(obj jsonutils.JSONObject) SMachineSize {
	gpu, _ := obj.Int("Gpu")
	size := SMachineSize{Gpu: int(gpu)}
	cols, err := obj.GetArray("Collection")
	if err != nil {
		return size
	}
	for _, col := range cols {
		size.Collection = append(size.Collection, parseInstanceCollection(col))
	}
	return size
}

func parseBootDiskInfo(obj jsonutils.JSONObject) SBootDiskInfo {
	ret := SBootDiskInfo{}
	ret.Name, _ = obj.GetString("Name")
	ret.InstantResize, _ = obj.Bool("InstantResize")
	maxSize, _ := obj.Int("MaximalSize")
	ret.MaximalSize = int(maxSize)
	ret.Features = parseStringArray(obj, "Features")
	return ret
}

func parseDataDiskInfo(obj jsonutils.JSONObject) SDataDiskInfo {
	ret := SDataDiskInfo{}
	ret.Name, _ = obj.GetString("Name")
	minSize, _ := obj.Int("MinimalSize")
	ret.MinimalSize = int(minSize)
	maxSize, _ := obj.Int("MaximalSize")
	ret.MaximalSize = int(maxSize)
	ret.Features = parseStringArray(obj, "Features")
	return ret
}

func parseDiskCategory(obj jsonutils.JSONObject) SDiskCategory {
	ret := SDiskCategory{}
	ret.Name, _ = obj.GetString("Name")
	bootDisks, err := obj.GetArray("BootDisk")
	if err == nil {
		for _, boot := range bootDisks {
			ret.BootDisk = append(ret.BootDisk, parseBootDiskInfo(boot))
		}
	}
	dataDisks, err := obj.GetArray("DataDisk")
	if err == nil {
		for _, data := range dataDisks {
			ret.DataDisk = append(ret.DataDisk, parseDataDiskInfo(data))
		}
	}
	return ret
}

func parseAvailableInstanceType(obj jsonutils.JSONObject) SAvailableInstanceType {
	ret := SAvailableInstanceType{}
	ret.Zone, _ = obj.GetString("Zone")
	ret.InstanceType, _ = obj.GetString("InstanceType")
	ret.Name, _ = obj.GetString("Name")
	ret.Status, _ = obj.GetString("Status")
	ret.ParentType, _ = obj.GetString("ParentType")
	ret.Description, _ = obj.GetString("Description")
	ret.MachineClass, _ = obj.GetString("MachineClass")

	sizes, err := obj.GetArray("MachineSizes")
	if err == nil {
		for _, size := range sizes {
			ret.MachineSizes = append(ret.MachineSizes, parseMachineSize(size))
		}
	}
	families, err := obj.GetArray("UHostFamilies")
	if err == nil {
		for _, family := range families {
			f := SUHostFamily{}
			f.Name, _ = family.GetString("Name")
			f.CpuFrequency, _ = family.GetString("CpuFrequency")
			ret.UHostFamilies = append(ret.UHostFamilies, f)
		}
	}
	disks, err := obj.GetArray("Disks")
	if err == nil {
		for _, disk := range disks {
			ret.Disks = append(ret.Disks, parseDiskCategory(disk))
		}
	}
	return ret
}

func parseAvailableInstanceTypes(resp jsonutils.JSONObject) ([]SAvailableInstanceType, error) {
	arr, err := resp.GetArray("AvailableInstanceTypes")
	if err != nil {
		return nil, err
	}
	ret := make([]SAvailableInstanceType, 0, len(arr))
	for _, item := range arr {
		ret = append(ret, parseAvailableInstanceType(item))
	}
	return ret, nil
}

func flattenInstancePackages(types []SAvailableInstanceType) []SInstancePackage {
	ret := make([]SInstancePackage, 0)
	for i := range types {
		hostType := types[i].ParentType
		if len(hostType) == 0 {
			hostType = types[i].Name
		}
		for _, size := range types[i].MachineSizes {
			for _, col := range size.Collection {
				if col.Cpu <= 0 || len(col.MemoryGB) == 0 {
					continue
				}
				for _, memGB := range col.MemoryGB {
					if memGB <= 0 {
						continue
					}
					ret = append(ret, SInstancePackage{
						Zone:        types[i].Zone,
						Name:        types[i].Name,
						UHostType:   hostType,
						Status:      types[i].Status,
						CPU:         col.Cpu,
						MemoryMB:    memGB * 1024,
						GPU:         size.Gpu,
						Spec:        formatInstanceSpec(hostType, col.Cpu, memGB, size.Gpu),
						Description: types[i].Description,
					})
				}
			}
		}
	}
	return ret
}

var ucloudStorageTypeOrder = []string{
	api.STORAGE_UCLOUD_CLOUD_NORMAL,
	api.STORAGE_UCLOUD_CLOUD_SSD,
	api.STORAGE_UCLOUD_CLOUD_ESSD,
	api.STORAGE_UCLOUD_CLOUD_RSSD,
	api.STORAGE_UCLOUD_LOCAL_NORMAL,
	api.STORAGE_UCLOUD_LOCAL_SSD,
	api.STORAGE_UCLOUD_EXCLUSIVE_LOCAL_DISK,
}

func collectStorageTypesFromInstanceTypes(types []SAvailableInstanceType, zoneId string) []string {
	typeSet := map[string]struct{}{}
	for i := range types {
		if len(zoneId) > 0 && len(types[i].Zone) > 0 && types[i].Zone != zoneId {
			continue
		}
		for _, disk := range types[i].Disks {
			for _, boot := range disk.BootDisk {
				if len(boot.Name) > 0 {
					typeSet[boot.Name] = struct{}{}
				}
			}
			for _, data := range disk.DataDisk {
				if len(data.Name) > 0 {
					typeSet[data.Name] = struct{}{}
				}
			}
		}
	}
	ret := make([]string, 0, len(typeSet))
	for _, storageType := range ucloudStorageTypeOrder {
		if _, ok := typeSet[storageType]; ok {
			ret = append(ret, storageType)
			delete(typeSet, storageType)
		}
	}
	others := make([]string, 0, len(typeSet))
	for storageType := range typeSet {
		others = append(others, storageType)
	}
	sort.Strings(others)
	ret = append(ret, others...)
	return ret
}

// GetZoneStorageTypes 从 DescribeAvailableInstanceTypes 的 Disks 信息获取可用区存储类型。
func (self *SRegion) GetZoneStorageTypes(zoneId string) ([]string, error) {
	types, err := self.GetAvailableInstanceTypes(zoneId)
	if err != nil {
		return nil, err
	}
	return collectStorageTypesFromInstanceTypes(types, zoneId), nil
}

// GetAvailableInstanceTypes 获取地域/可用区下可售机型信息。
func (self *SRegion) GetAvailableInstanceTypes(zoneId string) ([]SAvailableInstanceType, error) {
	params := NewUcloudParams()
	if len(zoneId) > 0 {
		params.Set("Zone", zoneId)
	}
	params.Set("Region", self.GetId())
	params = self.client.commonParams(params)
	params.SetAction("DescribeAvailableInstanceTypes")
	resp, err := jsonRequest(self.client, params)
	if err != nil {
		return nil, err
	}
	return parseAvailableInstanceTypes(resp)
}

// GetInstancePackages 获取展平后的云主机套餐列表。
func (self *SRegion) GetInstancePackages(zoneId string) ([]SInstancePackage, error) {
	types, err := self.GetAvailableInstanceTypes(zoneId)
	if err != nil {
		return nil, err
	}
	return flattenInstancePackages(types), nil
}
