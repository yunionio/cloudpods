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

package shell

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type ServerSkusListOptions struct {
		options.BaseListOptions
		Cloudregion string  `help:"region Id or name"`
		Usable      bool    `help:"Filter usable sku"`
		Zone        string  `help:"zone Id or name"`
		City        *string `help:"city name,eg. BeiJing"`
		Cpu         *int    `help:"Cpu core count" json:"cpu_core_count"`
		Mem         *int    `help:"Memory size in MB" json:"memory_size_mb"`
		Name        string  `help:"Name of Sku"`
	}

	R(&ServerSkusListOptions{}, "server-sku-list", "List all avaiable Server SKU", func(s *mcclient.ClientSession, args *ServerSkusListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		results, err := modules.ServerSkus.List(s, params)
		if err != nil {
			return err
		}
		printList(results, modules.ServerSkus.GetColumns(s))
		return nil
	})

	type ServerSkusIdOptions struct {
		ID string `help:"ID or Name of SKU to show"`
	}
	R(&ServerSkusIdOptions{}, "server-sku-show", "show details of a avaiable Server SKU", func(s *mcclient.ClientSession, args *ServerSkusIdOptions) error {
		result, err := modules.ServerSkus.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ServerSkusIdOptions{}, "server-sku-enable", "Enable Server SKU", func(s *mcclient.ClientSession, args *ServerSkusIdOptions) error {
		result, err := modules.ServerSkus.PerformAction(s, args.ID, "enable", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ServerSkusIdOptions{}, "server-sku-disable", "Disable Server SKU", func(s *mcclient.ClientSession, args *ServerSkusIdOptions) error {
		result, err := modules.ServerSkus.PerformAction(s, args.ID, "disable", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerSkusCacheOptions struct {
		REGION string `help:"Private cloud region"`
		ID     string `help:"ServerSku Id"`
	}

	R(&ServerSkusCacheOptions{}, "server-sku-cache", "Cache Server SKU for private cloud", func(s *mcclient.ClientSession, args *ServerSkusCacheOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.REGION), "cloudregion")
		result, err := modules.ServerSkus.PerformAction(s, args.ID, "cache-sku", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerSkusCreateOptions struct {
		Name         string `help:"ServerSku name"`
		CpuCoreCount int    `help:"Cpu Count" required:"true" positional:"true"`
		MemorySizeMB int    `help:"Memory MB" required:"true" positional:"true"`

		OsName               *string `help:"OS name/type" choices:"Linux|Windows|Any" default:"Any"`
		InstanceTypeCategory *string `help:"instance type category" choices:"general_purpose|compute_optimized|memory_optimized|storage_optimized|hardware_accelerated|high_memory|high_storage"`

		SysDiskResizable *bool   `help:"system disk is resizable"`
		SysDiskType      *string `help:"system disk type" default:"local" choices:"local"`
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

		Zone        *string `help:"Zone ID or name"`
		Cloudregion *string `help:"Cloudregion ID or name"`
		Provider    string  `help:"provider"`
		Brand       string  `help:"brand"`
	}
	R(&ServerSkusCreateOptions{}, "server-sku-create", "Create a server sku record", func(s *mcclient.ClientSession, args *ServerSkusCreateOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.ServerSkus.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerSkusUpdateOptions struct {
		ID string `help:"Name or ID of SKU" json:"-"`

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
	R(&ServerSkusUpdateOptions{}, "server-sku-update", "Update server sku attributes", func(s *mcclient.ClientSession, args *ServerSkusUpdateOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.ServerSkus.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerSkusDeleteOptions struct {
		ID    string `help:"Id or name of server sku"`
		Purge bool   `help:"purge sku"`
	}
	R(&ServerSkusDeleteOptions{}, "server-sku-delete", "Delete a server sku", func(s *mcclient.ClientSession, args *ServerSkusDeleteOptions) error {
		params := jsonutils.NewDict()
		if args.Purge {
			params.Add(jsonutils.JSONTrue, "purge")
		}
		result, err := modules.ServerSkus.Delete(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ServerSkusListOptions{}, "server-sku-specs-list", "List all avaiable Server SKU specifications", func(s *mcclient.ClientSession, args *ServerSkusListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.ServerSkus.Get(s, "instance-specs", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
