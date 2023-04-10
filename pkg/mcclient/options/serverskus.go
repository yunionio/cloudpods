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

package options

import "yunion.io/x/jsonutils"

type ServerSkusListOptions struct {
	BaseListOptions
	Cloudregion            string  `help:"region Id or name"`
	Usable                 bool    `help:"Filter usable sku"`
	Zone                   string  `help:"zone Id or name"`
	City                   *string `help:"city name,eg. BeiJing"`
	Cpu                    *int    `help:"Cpu core count" json:"cpu_core_count"`
	Mem                    *int    `help:"Memory size in MB" json:"memory_size_mb"`
	Name                   string  `help:"Name of Sku"`
	PostpaidStatus         string  `help:"Postpaid status" choices:"soldout|available"`
	PrepaidStatus          string  `help:"Prepaid status" choices:"soldout|available"`
	Enabled                *bool   `help:"Filter enabled skus"`
	Distinct               bool    `help:"distinct sku by name"`
	OrderByTotalGuestCount string
}

func (opts *ServerSkusListOptions) GetId() string {
	return "instance-specs"
}

func (opts *ServerSkusListOptions) Params() (jsonutils.JSONObject, error) {
	return ListStructToParams(opts)
}

type ServerSkusIdOptions struct {
	ID string `help:"ID or Name of SKU to show"`
}

func (opts *ServerSkusIdOptions) GetId() string {
	return opts.ID
}

func (opts *ServerSkusIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type ServerSkusCreateOptions struct {
	Name         string `help:"ServerSku name"`
	CpuCoreCount int    `help:"Cpu Count" required:"true" positional:"true"`
	MemorySizeMB int    `help:"Memory MB" required:"true" positional:"true"`

	OsName               *string `help:"OS name/type" choices:"Linux|Windows|Any" default:"Any"`
	InstanceTypeCategory *string `help:"instance type category" choices:"general_purpose|compute_optimized|memory_optimized|storage_optimized|hardware_accelerated|high_memory|high_storage"`

	SysDiskResizable *bool   `help:"system disk is resizable"`
	SysDiskType      *string `help:"system disk type" choices:"local"`
	SysDiskMaxSizeGB *int    `help:"system disk maximal size in gb"`

	AttachedDiskType   *string `help:"attached data disk type"`
	AttachedDiskSizeGB *int    `help:"attached data disk size in GB"`
	AttachedDiskCount  *int    `help:"attached data disk count"`

	MaxDataDiskCount *int `help:"maximal allowed data disk count"`

	NicType     *string `help:"nic type"`
	MaxNicCount *int    `help:"maximal nic count"`

	GPUSpec       *string `help:"GPU spec"`
	GPUCount      *int    `help:"GPU count"`
	GPUAttachable *bool   `help:"Allow attach GPU"`

	ZoneId        string `help:"Zone ID or name"`
	CloudregionId string `help:"Cloudregion ID or name"`
	Provider      string `help:"provider"`
	Brand         string `help:"brand"`
}

func (opts *ServerSkusCreateOptions) Params() (jsonutils.JSONObject, error) {
	return StructToParams(opts)
}

type ServerSkusUpdateOptions struct {
	ServerSkusIdOptions

	PostpaidStatus *string `help:"skus available status for postpaid instance" choices:"available|soldout"`
	PrepaidStatus  *string `help:"skus available status for prepaid instance"  choices:"available|soldout"`
	CpuCoreCount   *int    `help:"Cpu Count"`
	MemorySizeMB   *int    `help:"Memory MB"`

	InstanceTypeCategory *string `help:"instance type category" choices:"general_purpose|compute_optimized|memory_optimized|storage_optimized|hardware_accelerated|high_memory|high_storage"`

	SysDiskResizable *bool `help:"system disk is resizable"`
	SysDiskMaxSizeGB *int  `help:"system disk maximal size in gb"`

	AttachedDiskType   *string `help:"attached data disk type"`
	AttachedDiskSizeGB *int    `help:"attached data disk size in GB"`
	AttachedDiskCount  *int    `help:"attached data disk count"`

	MaxDataDiskCount *int `help:"maximal allowed data disk count"`

	NicType     *string `help:"nic type"`
	MaxNicCount *int    `help:"maximal nic count"`

	GPUSpec       *string `help:"GPU spec"`
	GPUCount      *int    `help:"GPU count"`
	GPUAttachable *bool   `help:"Allow attach GPU"`

	Zone   *string `help:"Zone ID or name"`
	Region *string `help:"Region ID or name"`
}

func (opts *ServerSkusUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return StructToParams(opts)
}
