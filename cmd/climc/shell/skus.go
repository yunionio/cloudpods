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
		Provider string `help:"provider" choices:"all|kvm|esxi|xen|hyperv|aliyun|azure|aws|qcloud|huawei"`
		Region   string `help:"region Id or name"`
		Zone     string `help:"zone Id or name"`
		Cpu      int    `help:"Cpu core count"`
		Mem      int    `help:"Memory size in MB"`
		Name     string `help:"Name of Sku"`
	}
	R(&ServerSkusListOptions{}, "server-sku-list", "List all avaiable Server SKU", func(s *mcclient.ClientSession, args *ServerSkusListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		if args.Cpu > 0 {
			params.Add(jsonutils.NewInt(int64(args.Cpu)), "cpu_core_count")
		}
		if args.Mem > 0 {
			params.Add(jsonutils.NewInt(int64(args.Mem)), "memory_size_mb")
		}
		results, err := modules.ServerSkus.List(s, params)
		if err != nil {
			return err
		}
		printList(results, modules.ServerSkus.GetColumns(s))
		return nil
	})

	type ServerSkusShowOptions struct {
		ID string `help:"ID or Name of SKU to show"`
	}
	R(&ServerSkusShowOptions{}, "server-sku-show", "show details of a avaiable Server SKU", func(s *mcclient.ClientSession, args *ServerSkusShowOptions) error {
		result, err := modules.ServerSkus.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerSkusCreateOptions struct {
		CpuCoreCount int    `help:"Cpu Count" required:"true" positional:"true"`
		MemorySizeMB int    `help:"Memory MB" required:"true" positional:"true"`
		Provider     string `help:"Provider name" choices:"all|kvm|esxi"`

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

		Zone   *string `help:"Zone ID or name"`
		Region *string `help:"Region ID or name"`
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

		CpuCoreCount *int `help:"Cpu Count"`
		MemorySizeMB *int `help:"Memory MB"`

		Provider             string  `help:"Provider name" choices:"all|kvm|esxi"`
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
		ID string `help:"Id or name of server sku"`
	}
	R(&ServerSkusDeleteOptions{}, "server-sku-delete", "Delete a server sku", func(s *mcclient.ClientSession, args *ServerSkusDeleteOptions) error {
		result, err := modules.ServerSkus.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
