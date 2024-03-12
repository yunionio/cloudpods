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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type ListOptions struct {
		options.BaseListOptions
		Model    string `help:"Specified model specs" choices:"hosts|isolated_devices|guests"`
		HostType string `help:"Host type filter" choices:"baremetal|hypervisor|esxi|kubelet|hyperv"`
		Gpu      bool   `help:"Only show gpu devices"`
		Zone     string `help:"Filter by zone id or name"`
		Occupied bool   `help:"show occupid host" json:"-"`
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
		if args.Zone != "" {
			params.Add(jsonutils.NewString(args.Zone), "zone")
		}
		if args.Occupied {
			params.Add(jsonutils.JSONFalse, "is_empty")
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
		HostType    string   `help:"Host type filter" choices:"baremetal|hypervisor|esxi|kubelet|hyperv"`
		Ncpu        int64    `help:"#CPU count of host" metavar:"<CPU_COUNT>"`
		MemSize     int64    `help:"Memory MB size"`
		DiskSpec    []string `help:"Disk spec string, like 'Linux_adapter0_HDD_111Gx4'"`
		Nic         int64    `help:"#Nics count of host" metavar:"<NIC_COUNT>"`
		GpuModel    []string `help:"GPU model, like 'GeForce GTX 1050 Ti'"`
		Occupied    bool     `help:"Show occupid host" json:"-"`
		Manufacture string   `help:"Manufacture of host"`
		Model       string   `help:"Model of host"`
	}
	R(&HostsQueryOptions{}, "spec-hosts-list", "List hosts according by specs", func(s *mcclient.ClientSession, args *HostsQueryOptions) error {
		newHostSpecKeys := func() []string {
			keys := []string{}
			if args.Ncpu > 0 {
				keys = append(keys, fmt.Sprintf("cpu:%d", args.Ncpu))
			}
			if args.MemSize > 0 {
				keys = append(keys, fmt.Sprintf("mem:%dM", args.MemSize))
			}
			if args.Nic > 0 {
				keys = append(keys, fmt.Sprintf("nic:%d", args.Nic))
			}
			if args.Manufacture != "" {
				keys = append(keys, fmt.Sprintf("manufacture:%s", args.Manufacture))
			}
			if args.Model != "" {
				keys = append(keys, fmt.Sprintf("model:%s", args.Model))
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
		if args.Occupied {
			params.Add(jsonutils.JSONFalse, "is_empty")
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
		data := &printutils.ListResult{Data: hosts, Total: len(hosts)}
		printList(data, []string{"ID", "Name", "MEM", "CPU", "Storage_Info"})
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
		data := &printutils.ListResult{Data: devs, Total: len(devs)}
		printList(data, []string{"ID", "Addr", "Dev_Type", "Model", "Vendor_Device_Id"})
		return nil
	})
}
