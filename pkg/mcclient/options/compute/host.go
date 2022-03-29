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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type HostListOptions struct {
	Schedtag  string `help:"List hosts in schedtag"`
	Zone      string `help:"List hosts in zone"`
	Region    string `help:"List hosts in region"`
	Wire      string `help:"List hosts in wire"`
	Image     string `help:"List hosts cached images" json:"cachedimage"`
	Storage   string `help:"List hosts attached to storages"`
	Baremetal string `help:"List hosts that is managed by baremetal system" choices:"true|false"`
	Empty     bool   `help:"show empty host" json:"-"`
	Occupied  bool   `help:"show occupid host" json:"-"`
	Enabled   bool   `help:"Show enabled host only" json:"-"`
	Disabled  bool   `help:"Show disabled host only" json:"-"`
	HostType  string `help:"Host type filter" choices:"baremetal|hypervisor|esxi|kubelet|hyperv|aliyun|azure|qcloud|aws|huawei|ucloud|google|ctyun"`
	AnyMac    string `help:"Mac matches one of the host's interface"`
	AnyIp     string `help:"IP matches one of the host's interface"`

	IsBaremetal *bool `help:"filter host list by is_baremetal=true|false"`

	ResourceType string `help:"Resource type" choices:"shared|prepaid|dedicated"`

	Usable *bool `help:"List all zones that is usable"`

	Hypervisor string `help:"filter hosts by hypervisor"`

	StorageNotAttached bool `help:"List hosts not attach specified storage"`

	Uuid string `help:"find host with given system uuid"`

	CdromBoot *bool `help:"filter hosts list by cdrom_boot=true|false"`

	Sn string `help:"find host by sn"`

	OrderByServerCount string `help:"Order by server count" choices:"desc|asc"`
	OrderByStorage     string `help:"Order by host storage" choices:"desc|asc"`

	options.BaseListOptions
}

func (opts *HostListOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.ListStructToParams(opts)
	if err != nil {
		return nil, err
	}
	if opts.Empty {
		params.Add(jsonutils.JSONTrue, "is_empty")
	} else if opts.Occupied {
		params.Add(jsonutils.JSONFalse, "is_empty")
	}
	if opts.Enabled {
		params.Add(jsonutils.NewInt(1), "enabled")
	} else if opts.Disabled {
		params.Add(jsonutils.NewInt(0), "enabled")
	}
	if len(opts.Uuid) > 0 {
		params.Add(jsonutils.NewString(opts.Uuid), "uuid")
	}
	if len(opts.Sn) > 0 {
		params.Add(jsonutils.NewString(opts.Sn), "sn")
	}
	return params, nil
}

type HostStatusStatisticsOptions struct {
	HostListOptions
	options.StatusStatisticsOptions
}
