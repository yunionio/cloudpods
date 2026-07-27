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

package compute

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"

	baseoptions "yunion.io/x/onecloud/pkg/mcclient/options"
)

type ServerSkusListOptions struct {
	_ struct{} `mcp-desc:"【创建流程中的中间步骤】查规格。口语 2c2g 用 spec；公有云须 provider+cloudregion。取 name 作 instance-type 后立刻 climc_server_create"`

	baseoptions.BaseListOptions
	Cloudregion string  `help:"region Id or name" mcp:"true"`
	Usable      bool    `help:"Filter usable sku" mcp:"true"`
	Zone        string  `help:"zone Id or name" mcp:"true"`
	City        *string `help:"city name,eg. BeiJing"`
	// Spec 口语规格，如 2c2g / 2核2G / 4C8G；会解析为 cpu_core_count + memory_size_mb
	Spec                   string `help:"Human spec like 2c2g / 2核2G / 4C8G; expands to cpu+mem(MB)" json:"-" mcp:"true"`
	Cpu                    *int   `help:"Cpu core count；用户说2核时传2。也可改用 --spec 2c2g" json:"cpu_core_count" mcp:"true"`
	Mem                    *int   `help:"Memory size in MB；2G=2048。也可改用 --spec 2c2g" json:"memory_size_mb" mcp:"true"`
	Name                   string `help:"Name of Sku" mcp:"true"`
	PostpaidStatus         string `help:"Postpaid status；创建优先 available" choices:"soldout|available" mcp:"true"`
	PrepaidStatus          string `help:"Prepaid status" choices:"soldout|available"`
	CpuArch                string `help:"Cpu Arch" choices:"x86|aarch64" mcp:"true"`
	Enabled                *bool  `help:"Filter enabled skus" mcp:"true"`
	Distinct               bool   `help:"distinct sku by name"`
	OrderByTotalGuestCount string
}

func (opts *ServerSkusListOptions) GetId() string {
	return "instance-specs"
}

func (opts *ServerSkusListOptions) Params() (jsonutils.JSONObject, error) {
	if err := opts.applySpec(); err != nil {
		return nil, err
	}
	return baseoptions.ListStructToParams(opts)
}

func (opts *ServerSkusListOptions) applySpec() error {
	spec := strings.TrimSpace(opts.Spec)
	if spec == "" {
		return nil
	}
	cpu, memMB, err := ParseSkuSpec(spec)
	if err != nil {
		return err
	}
	if opts.Cpu == nil {
		opts.Cpu = &cpu
	}
	if opts.Mem == nil {
		opts.Mem = &memMB
	}
	return nil
}

// skuSpecPatterns 支持：2c2g、2C2G、4c8g、2核2G、2核2g、2vcpu2gb、2c/2g、2x2g
var skuSpecPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^\s*(\d+)\s*[cC]\s*[xX/]?\s*(\d+)\s*([gGmM])b?\s*$`),
	regexp.MustCompile(`(?i)^\s*(\d+)\s*[xX/]\s*(\d+)\s*([gGmM])b?\s*$`),
	regexp.MustCompile(`(?i)^\s*(\d+)\s*核\s*(\d+)\s*([gGmM])b?\s*$`),
	regexp.MustCompile(`(?i)^\s*(\d+)\s*v?cpu\s*[xX/]?\s*(\d+)\s*([gGmM])b?\s*$`),
}

// ParseSkuSpec 将口语规格解析为 CPU 核数与内存 MB。
func ParseSkuSpec(spec string) (cpu int, memMB int, err error) {
	s := strings.TrimSpace(spec)
	if s == "" {
		return 0, 0, fmt.Errorf("empty sku spec")
	}
	for _, re := range skuSpecPatterns {
		m := re.FindStringSubmatch(s)
		if m == nil {
			continue
		}
		cpu, err = strconv.Atoi(m[1])
		if err != nil || cpu <= 0 {
			return 0, 0, fmt.Errorf("invalid cpu in spec %q", spec)
		}
		mem, err := strconv.Atoi(m[2])
		if err != nil || mem <= 0 {
			return 0, 0, fmt.Errorf("invalid memory in spec %q", spec)
		}
		switch strings.ToLower(m[3]) {
		case "g":
			memMB = mem * 1024
		case "m":
			memMB = mem
		default:
			return 0, 0, fmt.Errorf("unsupported memory unit in spec %q", spec)
		}
		return cpu, memMB, nil
	}
	return 0, 0, fmt.Errorf("unrecognized sku spec %q, expect like 2c2g or 2核2G", spec)
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
	CpuArch      string `help:"CPU architecture" choices:"x86|aarch64"`

	OsName               *string `help:"OS name/type" choices:"Linux|Windows|Any" default:"Any"`
	InstanceTypeCategory *string `help:"instance type category" choices:"general_purpose|compute_optimized|memory_optimized|storage_optimized|hardware_accelerated|high_memory|high_storage"`

	SysDiskResizable *bool   `help:"system disk is resizable"`
	SysDiskType      *string `help:"system disk type"`
	SysDiskMaxSizeGB *int    `help:"system disk maximal size in gb"`

	AttachedDiskType   *string `help:"attached data disk type"`
	AttachedDiskSizeGB *int    `help:"attached data disk size in GB"`
	AttachedDiskCount  *int    `help:"attached data disk count"`
	DataDiskTypes      string  `help:"data disk type"`

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
	return baseoptions.StructToParams(opts)
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

	HourPrice  *float64 `help:"Hourly price"`
	MonthPrice *float64 `help:"Monthly price"`
	Currency   *string  `help:"Currency code, e.g. CNY, USD"`
}

func (opts *ServerSkusUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return baseoptions.StructToParams(opts)
}

type ServerSkusBatchUpdatePriceOptions struct {
	CloudregionId string `help:"Cloudregion ID or name"`
	Currency      string `help:"Default currency code, e.g. CNY, USD"`
	Skus          string `help:"JSON array of sku price items, e.g. '[{\"name\":\"ecs.g1.c1m1\",\"zone_id\":\"zone-a\",\"hour_price\":0.5,\"month_price\":50,\"currency\":\"CNY\"}]'"`
}

func (opts *ServerSkusBatchUpdatePriceOptions) Params() (jsonutils.JSONObject, error) {
	params, err := baseoptions.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	if len(opts.Skus) > 0 {
		skus, err := jsonutils.Parse([]byte(opts.Skus))
		if err != nil {
			return nil, err
		}
		params.Set("skus", skus)
	}
	return params, nil
}
