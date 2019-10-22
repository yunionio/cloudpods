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

package k8s

import (
	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/fileutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/cmdline"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type MachineListOptions struct {
	options.BaseListOptions
	Cluster string `help:"Filter by cluster"`
}

func (o MachineListOptions) Params() (*jsonutils.JSONDict, error) {
	return options.ListStructToParams(&o)
}

type MachineCreateOptions struct {
	CLUSTER    string `help:"Cluster id"`
	ROLE       string `help:"Machine role" choices:"node|controlplane"`
	Type       string `help:"Resource type" choices:"vm|baremetal" json:"resource_type"`
	Instance   string `help:"VM or host instance id" json:"resource_id"`
	Name       string `help:"Name of node"`
	Disk       string `help:"VM root disk size, e.g. 100G"`
	Net        string `help:"VM network config"`
	Cpu        int    `help:"VM cpu count"`
	Memory     string `help:"VM memory size, e.g. 1G"`
	Hypervisor string `help:"VM hypervisor"`
}

func (o MachineCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	if o.Name != "" {
		params.Add(jsonutils.NewString(o.Name), "name")
	}
	params.Add(jsonutils.NewString(o.CLUSTER), "cluster")
	if o.ROLE != "" {
		params.Add(jsonutils.NewString(o.ROLE), "role")
	}
	if o.Instance != "" {
		params.Add(jsonutils.NewString(o.Instance), "resource_id")
	}
	if o.Type != "" {
		params.Add(jsonutils.NewString(o.Type), "resource_type")
	}
	if o.Type != "vm" {
		return params, nil
	}
	vmConfig := jsonutils.NewDict()
	if len(o.Disk) != 0 {
		diskConf, err := cmdline.ParseDiskConfig(o.Disk, 0)
		if err != nil {
			return nil, errors.Wrapf(err, "Parse disk %s", o.Disk)
		}
		vmConfig.Add(jsonutils.NewArray(diskConf.JSON(diskConf)), "disks")
	}
	if len(o.Net) != 0 {
		netConf, err := cmdline.ParseNetworkConfig(o.Net, 0)
		if err != nil {
			return nil, errors.Wrapf(err, "Parse network %s", o.Net)
		}
		vmConfig.Add(jsonutils.NewArray(netConf.JSON(netConf)), "nets")
	}
	if len(o.Memory) != 0 {
		memSize, err := fileutils.GetSizeMb(o.Memory, 'M', 1024)
		if err != nil {
			return nil, errors.Wrapf(err, "Parse memory %s", o.Memory)
		}
		vmConfig.Add(jsonutils.NewInt(int64(memSize)), "vmem_size")
	}
	if o.Cpu > 0 {
		vmConfig.Add(jsonutils.NewInt(int64(o.Cpu)), "vcpu_count")
	}
	vmConfig.Add(jsonutils.NewString(o.Hypervisor), "hypervisor")
	params.Add(vmConfig, "config", "vm")
	return params, nil
}
