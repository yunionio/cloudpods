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

package aliyun

// https://help.aliyun.com/zh/es/product-overview/node-specifications
var esSpec = map[string]struct {
	VcpuCount  int
	VmemSizeGb int
}{
	// 开发测试型规格
	"elasticsearch.t6.large": {VcpuCount: 2, VmemSizeGb: 4},

	// 云盘型（1:1规格族）
	"elasticsearch.ic5.large":   {VcpuCount: 2, VmemSizeGb: 2},
	"elasticsearch.ic5.xlarge":  {VcpuCount: 4, VmemSizeGb: 4},
	"elasticsearch.ic5.2xlarge": {VcpuCount: 8, VmemSizeGb: 8},
	"elasticsearch.ic5.3xlarge": {VcpuCount: 12, VmemSizeGb: 12},
	"elasticsearch.ic5.4xlarge": {VcpuCount: 16, VmemSizeGb: 16},

	// 云盘型（1:2规格族）
	"elasticsearch.n4.small":      {VcpuCount: 1, VmemSizeGb: 2},
	"elasticsearch.sn1ne.large":   {VcpuCount: 2, VmemSizeGb: 4},
	"elasticsearch.sn1ne.xlarge":  {VcpuCount: 4, VmemSizeGb: 8},
	"elasticsearch.sn1ne.2xlarge": {VcpuCount: 8, VmemSizeGb: 16},
	"elasticsearch.sn1ne.4xlarge": {VcpuCount: 16, VmemSizeGb: 32},
	"elasticsearch.sn1ne.8xlarge": {VcpuCount: 32, VmemSizeGb: 64},

	// 云盘型（1:4规格族）
	"elasticsearch.sn2ne.large":   {VcpuCount: 2, VmemSizeGb: 8},
	"elasticsearch.sn2ne.xlarge":  {VcpuCount: 4, VmemSizeGb: 16},
	"elasticsearch.sn2ne.2xlarge": {VcpuCount: 8, VmemSizeGb: 32},
	"elasticsearch.sn2ne.4xlarge": {VcpuCount: 16, VmemSizeGb: 64},
	"elasticsearch.sn2ne.8xlarge": {VcpuCount: 32, VmemSizeGb: 128},

	// 云盘型（1:8规格族）
	"elasticsearch.r5.large":   {VcpuCount: 2, VmemSizeGb: 16},
	"elasticsearch.r5.xlarge":  {VcpuCount: 4, VmemSizeGb: 32},
	"elasticsearch.r5.2xlarge": {VcpuCount: 8, VmemSizeGb: 64},
	"elasticsearch.r6.4xlarge": {VcpuCount: 16, VmemSizeGb: 128},
	"elasticsearch.r6.8xlarge": {VcpuCount: 32, VmemSizeGb: 256},

	// 大数据型
	"elasticsearch.c6.13xlarge": {VcpuCount: 52, VmemSizeGb: 96},

	// 新一代云盘型（1:2规格族）
	"elasticsearch.sn1ne.large.new":   {VcpuCount: 2, VmemSizeGb: 4},
	"elasticsearch.sn1ne.xlarge.new":  {VcpuCount: 4, VmemSizeGb: 8},
	"elasticsearch.sn1ne.2xlarge.new": {VcpuCount: 8, VmemSizeGb: 16},
	"elasticsearch.sn1ne.4xlarge.new": {VcpuCount: 16, VmemSizeGb: 32},
	"elasticsearch.sn1ne.8xlarge.new": {VcpuCount: 32, VmemSizeGb: 64},

	// 新一代云盘型（1:4规格族）
	"elasticsearch.sn2ne.large.new":   {VcpuCount: 2, VmemSizeGb: 8},
	"elasticsearch.sn2ne.xlarge.new":  {VcpuCount: 4, VmemSizeGb: 16},
	"elasticsearch.sn2ne.2xlarge.new": {VcpuCount: 8, VmemSizeGb: 32},
	"elasticsearch.sn2ne.4xlarge.new": {VcpuCount: 16, VmemSizeGb: 64},

	// 本地SSD盘型
	"elasticsearch.i2g.2xlarge": {VcpuCount: 8, VmemSizeGb: 32},
	"elasticsearch.i2g.4xlarge": {VcpuCount: 16, VmemSizeGb: 64},
	"elasticsearch.i2.4xlarge":  {VcpuCount: 16, VmemSizeGb: 128},
	"elasticsearch.i2g.8xlarge": {VcpuCount: 32, VmemSizeGb: 128},

	// 大数据本地SATA盘型
	"elasticsearch.d1.2xlarge": {VcpuCount: 8, VmemSizeGb: 32},
	"elasticsearch.d1.4xlarge": {VcpuCount: 16, VmemSizeGb: 64},

	// OpenStore存储型（存算分离型）
	"openstore.hybrid.i2.2xlarge":  {VcpuCount: 8, VmemSizeGb: 64},
	"openstore.hybrid.i2g.4xlarge": {VcpuCount: 16, VmemSizeGb: 64},

	// AMD处理器 - 新一代云盘型（1:2规格族）
	"elasticsearch.c7a.large":   {VcpuCount: 2, VmemSizeGb: 4},
	"elasticsearch.c7a.xlarge":  {VcpuCount: 4, VmemSizeGb: 8},
	"elasticsearch.c7a.2xlarge": {VcpuCount: 8, VmemSizeGb: 16},
	"elasticsearch.c7a.4xlarge": {VcpuCount: 16, VmemSizeGb: 32},
	"elasticsearch.c7a.8xlarge": {VcpuCount: 32, VmemSizeGb: 64},

	// AMD处理器 - 新一代云盘型（1:4规格族）
	"elasticsearch.g7a.large":   {VcpuCount: 2, VmemSizeGb: 8},
	"elasticsearch.g7a.xlarge":  {VcpuCount: 4, VmemSizeGb: 16},
	"elasticsearch.g7a.2xlarge": {VcpuCount: 8, VmemSizeGb: 32},
	"elasticsearch.g7a.4xlarge": {VcpuCount: 16, VmemSizeGb: 64},
	"elasticsearch.g7a.8xlarge": {VcpuCount: 32, VmemSizeGb: 128},

	// AMD处理器 - 新一代云盘型（1:8规格族）
	"elasticsearch.r7a.large":   {VcpuCount: 2, VmemSizeGb: 16},
	"elasticsearch.r7a.xlarge":  {VcpuCount: 4, VmemSizeGb: 32},
	"elasticsearch.r7a.2xlarge": {VcpuCount: 8, VmemSizeGb: 64},
	"elasticsearch.r7a.4xlarge": {VcpuCount: 16, VmemSizeGb: 128},
	"elasticsearch.r7a.8xlarge": {VcpuCount: 32, VmemSizeGb: 256},

	// 新一代云盘型（1:2计算型-Turbo-1）
	"elasticsearch.turbo1.ca.large":    {VcpuCount: 2, VmemSizeGb: 4},
	"elasticsearch.turbo1.ca.xlarge":   {VcpuCount: 4, VmemSizeGb: 8},
	"elasticsearch.turbo1.ca.2xlarge":  {VcpuCount: 8, VmemSizeGb: 16},
	"elasticsearch.turbo1.ca.4xlarge":  {VcpuCount: 16, VmemSizeGb: 32},
	"elasticsearch.turbo1.ca.8xlarge":  {VcpuCount: 32, VmemSizeGb: 64},
	"elasticsearch.turbo1.ca.16xlarge": {VcpuCount: 64, VmemSizeGb: 128},

	// 新一代云盘型（1:4通用型-Turbo-1）
	"elasticsearch.turbo1.ga.large":   {VcpuCount: 2, VmemSizeGb: 8},
	"elasticsearch.turbo1.ga.xlarge":  {VcpuCount: 4, VmemSizeGb: 16},
	"elasticsearch.turbo1.ga.2xlarge": {VcpuCount: 8, VmemSizeGb: 32},
	"elasticsearch.turbo1.ga.4xlarge": {VcpuCount: 16, VmemSizeGb: 64},
	"elasticsearch.turbo1.ga.8xlarge": {VcpuCount: 32, VmemSizeGb: 128},

	// 新一代云盘型（1:8内存型-Turbo-1）
	"elasticsearch.turbo1.ra.large":   {VcpuCount: 2, VmemSizeGb: 16},
	"elasticsearch.turbo1.ra.xlarge":  {VcpuCount: 4, VmemSizeGb: 32},
	"elasticsearch.turbo1.ra.2xlarge": {VcpuCount: 8, VmemSizeGb: 64},
	"elasticsearch.turbo1.ra.4xlarge": {VcpuCount: 16, VmemSizeGb: 128},
	"elasticsearch.turbo1.ra.8xlarge": {VcpuCount: 32, VmemSizeGb: 256},

	// 保留原有的d2s规格（如果仍在使用）
	"elasticsearch.d2s.5xlarge": {VcpuCount: 20, VmemSizeGb: 88},
}
