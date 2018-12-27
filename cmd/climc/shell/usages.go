package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type GeneralUsageOptions struct {
	HostType []string `help:"Host types" choices:"hypervisor|baremetal|esxi|xen|kubelet|hyperv|aliyun|azure|aws|huawei|qcloud"`
	Provider []string `help:"Provider" choices:"VMware|Aliyun|Azure|Aws|Qcloud|Huawei"`
	Project  string
}

func fetchHostTypeOptions(args *GeneralUsageOptions) *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	if len(args.HostType) > 0 {
		params.Add(jsonutils.NewStringArray(args.HostType), "host_type")
	}
	if len(args.Provider) > 0 {
		params.Add(jsonutils.NewStringArray(args.Provider), "provider")
	}
	return params
}

func init() {
	R(&GeneralUsageOptions{}, "usage", "Show general usage", func(s *mcclient.ClientSession, args *GeneralUsageOptions) error {
		params := fetchHostTypeOptions(args)
		if args.Project != "" {
			params.Add(jsonutils.NewString(args.Project), "project")
		}
		result, err := modules.Usages.GetGeneralUsage(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ResourceUsageOptions struct {
		GeneralUsageOptions
		ID string `help:"ID or name of resource"`
	}
	R(&ResourceUsageOptions{}, "zone-usage", "Show general usage of zone", func(s *mcclient.ClientSession, args *ResourceUsageOptions) error {
		params := fetchHostTypeOptions(&args.GeneralUsageOptions)
		params.Add(jsonutils.NewString("zones"), "range_type")
		params.Add(jsonutils.NewString(args.ID), "range_id")
		result, err := modules.Usages.GetGeneralUsage(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ResourceUsageOptions{}, "wire-usage", "Show general usage of wire", func(s *mcclient.ClientSession, args *ResourceUsageOptions) error {
		params := fetchHostTypeOptions(&args.GeneralUsageOptions)
		params.Add(jsonutils.NewString("wires"), "range_type")
		params.Add(jsonutils.NewString(args.ID), "range_id")
		result, err := modules.Usages.GetGeneralUsage(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ResourceUsageOptions{}, "cloud-provider-usage", "Show general usage of vcenter", func(s *mcclient.ClientSession, args *ResourceUsageOptions) error {
		params := fetchHostTypeOptions(&args.GeneralUsageOptions)
		params.Add(jsonutils.NewString("cloudproviders"), "range_type")
		params.Add(jsonutils.NewString(args.ID), "range_id")
		result, err := modules.Usages.GetGeneralUsage(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ResourceUsageOptions{}, "cloud-account-usage", "Show general usage of vcenter", func(s *mcclient.ClientSession, args *ResourceUsageOptions) error {
		params := fetchHostTypeOptions(&args.GeneralUsageOptions)
		params.Add(jsonutils.NewString("cloudaccounts"), "range_type")
		params.Add(jsonutils.NewString(args.ID), "range_id")
		result, err := modules.Usages.GetGeneralUsage(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ResourceUsageOptions{}, "schedtag-usage", "Show general usage of a scheduler tag", func(s *mcclient.ClientSession, args *ResourceUsageOptions) error {
		params := fetchHostTypeOptions(&args.GeneralUsageOptions)
		params.Add(jsonutils.NewString("schedtags"), "range_type")
		params.Add(jsonutils.NewString(args.ID), "range_id")
		result, err := modules.Usages.GetGeneralUsage(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ResourceUsageOptions{}, "cloud-region-usage", "Show general usage of a cloud region", func(s *mcclient.ClientSession, args *ResourceUsageOptions) error {
		params := fetchHostTypeOptions(&args.GeneralUsageOptions)
		params.Add(jsonutils.NewString("cloudregions"), "range_type")
		params.Add(jsonutils.NewString(args.ID), "range_id")
		result, err := modules.Usages.GetGeneralUsage(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ImageUsageOptions struct {
		Project string `help:"check image usage of a project"`
	}
	R(&ImageUsageOptions{}, "image-usage", "Show general usage of images", func(s *mcclient.ClientSession, args *ImageUsageOptions) error {
		params := jsonutils.NewDict()
		if args.Project != "" {
			params.Add(jsonutils.NewString(args.Project), "project")
		}
		result, err := modules.Images.GetUsage(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
