package shell

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type ListOptions struct {
		options.BaseListOptions
		Model    string `help:"Specified model specs" choices:"hosts|isolated_devices|guests"`
		HostType string `help:"Host type filter" choices:"baremetal|hypervisor|esxi|kubelet|hyperv"`
		Gpu      bool   `help:"Only show gpu devices"`
	}
	R(&ListOptions{}, "spec", "List all kinds of model specs", func(s *mcclient.ClientSession, args *ListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		model := ""
		if len(args.Model) != 0 {
			model = args.Model
		}
		if len(args.HostType) > 0 {
			params.Add(jsonutils.NewString(args.HostType), "host_type")
		}
		if args.Gpu {
			params.Add(jsonutils.JSONTrue, "gpu")
		}
		result, err := modules.Specs.GetModelSpecs(s, model, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type HostsQueryOptions struct {
		options.BaseListOptions
		HostType string   `help:"Host type filter" choices:"baremetal|hypervisor|esxi|kubelet|hyperv"`
		Ncpu     int64    `help:"#CPU count of host" metavar:"<CPU_COUNT>"`
		MemSize  string   `help:"Memory GB size"`
		DiskSpec []string `help:"Disk spec string, like 'Linux_adapter0_HDD_111Gx4'"`
		Nic      int64    `help:"#Nics count of host" metavar:"<NIC_COUNT>"`
		GpuModel []string `help:"GPU model, like 'GeForce GTX 1050 Ti'"`
	}
	R(&HostsQueryOptions{}, "spec-hosts-list", "List hosts according by specs", func(s *mcclient.ClientSession, args *HostsQueryOptions) error {
		newHostSpecKeys := func() []string {
			keys := []string{}
			if args.Ncpu > 0 {
				keys = append(keys, fmt.Sprintf("cpu:%d", args.Ncpu))
			}
			if len(args.MemSize) != 0 {
				keys = append(keys, fmt.Sprintf("mem:%s", args.MemSize))
			}
			for _, gm := range args.GpuModel {
				keys = append(keys, fmt.Sprintf("gpu_model:%s", gm))
			}
			for _, ds := range args.DiskSpec {
				keys = append(keys, fmt.Sprintf("disk:%s", ds))
			}
			return keys
		}
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		if len(args.HostType) > 0 {
			params.Add(jsonutils.NewString(args.HostType), "host_type")
		}
		specKeys := newHostSpecKeys()
		resp, err := modules.Specs.SpecsQueryModelObjects(s, "hosts", specKeys, params)
		if err != nil {
			return err
		}
		hosts, err := resp.GetArray()
		if err != nil {
			return err
		}
		data := &modules.ListResult{Data: hosts, Total: len(hosts)}
		printList(data, []string{"ID", "Name", "Mem", "CPU", "Storage_Info"})
		return nil
	})

	type IsoDevQueryOptions struct {
		options.BaseListOptions
		Model  string `help:"Device model name"`
		Vendor string `help:"Device vendor name"`
	}
	R(&IsoDevQueryOptions{}, "spec-isolated-devices-list", "List isolated devices according by specs", func(s *mcclient.ClientSession, args *IsoDevQueryOptions) error {
		newSpecKeys := func() []string {
			keys := []string{}
			if len(args.Vendor) != 0 {
				keys = append(keys, fmt.Sprintf("vendor:%s", args.Vendor))
			}
			if len(args.Model) != 0 {
				keys = append(keys, fmt.Sprintf("model:%s", args.Model))
			}
			return keys
		}
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		resp, err := modules.Specs.SpecsQueryModelObjects(s, "isolated_devices", newSpecKeys(), params)
		if err != nil {
			return err
		}
		devs, err := resp.GetArray()
		if err != nil {
			return err
		}
		data := &modules.ListResult{Data: devs, Total: len(devs)}
		printList(data, []string{"ID", "Addr", "Dev_Type", "Model", "Vendor_Device_Id"})
		return nil
	})
}
