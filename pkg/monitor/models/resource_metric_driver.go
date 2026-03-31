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

package models

import (
	"yunion.io/x/onecloud/pkg/apis/monitor"
)

// IResourceMetricDriver 定义资源监控指标的驱动接口
type IResourceMetricDriver interface {
	// GetResType 返回资源类型标识
	GetResType() string
	// GetTagKey 返回 InfluxDB 中用于标识资源 ID 的 tag key
	GetTagKey() string
	// GetMetricSpecs 返回该资源类型的所有指标查询规格
	GetMetricSpecs() []ResourceMetricSpec
}

// ResourceMetricSpec 描述一个 measurement 下需要查询的字段及其输出映射
type ResourceMetricSpec struct {
	Measurement string
	Fields      []string
	OutputKeys  []string
}

var resourceMetricDrivers = make(map[string]IResourceMetricDriver)

// RegisterResourceMetricDriver 注册资源监控指标驱动
func RegisterResourceMetricDriver(drv IResourceMetricDriver) {
	resourceMetricDrivers[drv.GetResType()] = drv
}

// GetResourceMetricDriver 根据资源类型获取对应驱动
func GetResourceMetricDriver(resType string) IResourceMetricDriver {
	return resourceMetricDrivers[resType]
}

// SetResourceMetricValue 将查询结果赋值到 ResourceMetricValues 对应字段
func SetResourceMetricValue(rv *monitor.ResourceMetricValues, key string, val float64) {
	switch key {
	case "cpu_usage":
		rv.CpuUsage = &val
	case "mem_usage":
		rv.MemUsage = &val
	case "disk_usage":
		rv.DiskUsage = &val
	case "disk_read_rate":
		rv.DiskReadRate = &val
	case "disk_write_rate":
		rv.DiskWriteRate = &val
	case "net_in_rate":
		rv.NetInRate = &val
	case "net_out_rate":
		rv.NetOutRate = &val
	}
}

// sHostMetricDriver 宿主机监控指标驱动
type sHostMetricDriver struct{}

func (d *sHostMetricDriver) GetResType() string {
	return monitor.METRIC_RES_TYPE_HOST
}

func (d *sHostMetricDriver) GetTagKey() string {
	return monitor.MEASUREMENT_TAG_ID[monitor.METRIC_RES_TYPE_HOST]
}

func (d *sHostMetricDriver) GetMetricSpecs() []ResourceMetricSpec {
	return []ResourceMetricSpec{
		{"cpu", []string{"usage_active"}, []string{"cpu_usage"}},
		{"mem", []string{"used_percent"}, []string{"mem_usage"}},
		{"disk", []string{"used_percent"}, []string{"disk_usage"}},
		{"diskio", []string{"read_bps", "write_bps"}, []string{"disk_read_rate", "disk_write_rate"}},
		{"net", []string{"bps_recv", "bps_sent"}, []string{"net_in_rate", "net_out_rate"}},
	}
}

// sGuestMetricDriver 虚拟机监控指标驱动
type sGuestMetricDriver struct{}

func (d *sGuestMetricDriver) GetResType() string {
	return monitor.METRIC_RES_TYPE_GUEST
}

func (d *sGuestMetricDriver) GetTagKey() string {
	return monitor.MEASUREMENT_TAG_ID[monitor.METRIC_RES_TYPE_GUEST]
}

func (d *sGuestMetricDriver) GetMetricSpecs() []ResourceMetricSpec {
	return []ResourceMetricSpec{
		{"vm_cpu", []string{"usage_active"}, []string{"cpu_usage"}},
		{"vm_mem", []string{"used_percent"}, []string{"mem_usage"}},
		{"vm_disk", []string{"used_percent"}, []string{"disk_usage"}},
		{"vm_diskio", []string{"read_bps", "write_bps"}, []string{"disk_read_rate", "disk_write_rate"}},
		{"vm_netio", []string{"bps_recv", "bps_sent"}, []string{"net_in_rate", "net_out_rate"}},
	}
}

func init() {
	RegisterResourceMetricDriver(&sHostMetricDriver{})
	RegisterResourceMetricDriver(&sGuestMetricDriver{})
}
