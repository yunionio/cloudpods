package shell

import (
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient"
	"github.com/yunionio/mcclient/modules"
)

type GeneralUsageOptions struct {
	HostType []string `help:"Host types" choices:"hypervisor|baremetal|esxi|xen|kubelet|hyperv"`
}

func fetchHostTypeOptions(args *GeneralUsageOptions) *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	if len(args.HostType) > 0 {
		params.Add(jsonutils.NewStringArray(args.HostType), "host_type")
	}
	return params
}

func init() {
	R(&GeneralUsageOptions{}, "usage", "Show general usage", func(s *mcclient.ClientSession, args *GeneralUsageOptions) error {
		params := fetchHostTypeOptions(args)
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

	R(&ResourceUsageOptions{}, "vcenter-usage", "Show general usage of vcenter", func(s *mcclient.ClientSession, args *ResourceUsageOptions) error {
		params := fetchHostTypeOptions(&args.GeneralUsageOptions)
		params.Add(jsonutils.NewString("vcenters"), "range_type")
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
}
