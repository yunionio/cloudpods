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

	"yunion.io/x/onecloud/pkg/apis/baremetal"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type HostListOptions struct {
	Schedtag        string   `help:"List hosts in schedtag"`
	Zone            string   `help:"List hosts in zone"`
	Region          string   `help:"List hosts in region"`
	Wire            string   `help:"List hosts in wire"`
	Image           string   `help:"List hosts cached images" json:"cachedimage"`
	Storage         string   `help:"List hosts attached to storages"`
	Baremetal       string   `help:"List hosts that is managed by baremetal system" choices:"true|false"`
	Empty           bool     `help:"show empty host" json:"-"`
	Occupied        bool     `help:"show occupid host" json:"-"`
	Enabled         bool     `help:"Show enabled host only" json:"-"`
	Disabled        bool     `help:"Show disabled host only" json:"-"`
	HostType        string   `help:"Host type filter" choices:"baremetal|hypervisor|esxi|container|hyperv|aliyun|azure|qcloud|aws|huawei|ucloud|google|ctyun"`
	AnyMac          string   `help:"Mac matches one of the host's interface"`
	AnyIp           []string `help:"IP matches one of the host's interface"`
	HostStorageType []string `help:"List host in host_storage_type"`

	IsBaremetal *bool `help:"filter host list by is_baremetal=true|false"`

	ResourceType string `help:"Resource type" choices:"shared|prepaid|dedicated"`

	Usable *bool `help:"List all zones that is usable"`

	Hypervisor string `help:"filter hosts by hypervisor"`

	StorageNotAttached bool `help:"List hosts not attach specified storage"`

	Uuid string `help:"find host with given system uuid"`

	CdromBoot *bool `help:"filter hosts list by cdrom_boot=true|false"`

	Sn string `help:"find host by sn"`

	OrderByServerCount       string `help:"Order by server count" choices:"desc|asc"`
	OrderByStorage           string `help:"Order by host storage" choices:"desc|asc"`
	OrderByStorageCommitRate string `help:"Order by host storage commite rate" choices:"desc|asc"`
	OrderByCpuCommitRate     string `help:"Order by host cpu commit rate" choices:"desc|asc"`
	OrderByMemCommitRate     string `help:"Order by host meme commit rate" choices:"desc|asc"`

	OrderByStorageUsed string `help:"Order by storage used" choices:"desc|asc"`
	OrderByCpuCommit   string `help:"Order by cpu commit" choices:"desc|asc"`
	OrderByMemCommit   string `help:"Order by mem commit" choices:"desc|asc"`

	HideCpuTopoInfo *bool `help:"Host list will remove cpu_info and topology info from sysinfo and metadata"`

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

type HostShowOptions struct {
	options.BaseShowOptions
	ShowMetadata bool `help:"Show host metadata in details"`
	ShowNicInfo  bool `help:"Show host nic_info in details"`
	ShowSysInfo  bool `help:"Show host sys_info in details"`
	ShowAll      bool `help:"Show all of host details" short-token:"a"`
}

func (o *HostShowOptions) Params() (jsonutils.JSONObject, error) {
	// NOTE: host show only request with base options
	return jsonutils.Marshal(o.BaseShowOptions), nil
}

type HostReserveCpusOptions struct {
	options.BaseIdsOptions
	Cpus                    string
	Mems                    string
	DisableSchedLoadBalance bool
	ProcessesPrefix         []string `help:"Processes prefix bind reserved cpus"`
}

func (o *HostReserveCpusOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type HostAutoMigrateOnHostDownOptions struct {
	options.BaseIdsOptions
	AutoMigrateOnHostDown     string `help:"Auto migrate on host down" choices:"enable|disable" default:"disable"`
	AutoMigrateOnHostShutdown string `help:"Auto migrate on host shutdown" choices:"enable|disable" default:"disable"`
}

func (o *HostAutoMigrateOnHostDownOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type HostSetCommitBoundOptions struct {
	options.BaseIdOptions
	CpuCmtbound *float32 `help:"Cpu commit bound"`
	MemCmtBound *float32 `help:"Mem commit bound"`
}

func (o *HostSetCommitBoundOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type HostStatusStatisticsOptions struct {
	HostListOptions
	options.StatusStatisticsOptions
}

type HostValidateIPMI struct {
	IP       string `json:"ip" help:"IPMI ip address"`
	USERNAME string `json:"username" help:"IPMI username"`
	PASSWORD string `json:"password" help:"IPMI password"`
}

func (h HostValidateIPMI) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(baremetal.ValidateIPMIRequest{
		Ip:       h.IP,
		Username: h.USERNAME,
		Password: h.PASSWORD,
	}), nil
}

type HostUpdateOptions struct {
	options.BaseIdOptions
	Name        string  `help:"New name of the host"`
	Description *string `help:"New Description of the host"`
	MemReserved *string `help:"Memory reserved"`
	CpuReserved *int64  `help:"CPU reserved"`
	HostType    *string `help:"Change host type, CAUTION!!!!" choices:"hypervisor|kubelet|esxi|baremetal"`
	CpuCount    *int
	NodeCount   *int8
	CpuDesc     *string
	MemSize     *int
	StorageSize *int64
	// AccessIp          string  `help:"Change access ip, CAUTION!!!!"`
	AccessMac          *string `help:"Change baremetal access MAC, CAUTION!!!!"`
	Uuid               *string `help:"Change baremetal UUID,  CAUTION!!!!"`
	EnableNumaAllocate string  `help:"Host enable numa allocate" choices:"True|False"`

	IpmiUsername *string `help:"IPMI user"`
	IpmiPassword *string `help:"IPMI password"`
	IpmiIpAddr   *string `help:"IPMI ip_addr"`

	Sn *string `help:"serial number"`

	Hostname *string `help:"update host name"`

	PublicIp *string `help:"public_ip"`

	NoPublicIp *bool `help:"clear public ip"`
}

func (opts *HostUpdateOptions) Params() (jsonutils.JSONObject, error) {
	v := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	if opts.NoPublicIp != nil && *opts.NoPublicIp {
		v.Set("public_ip", jsonutils.NewString(""))
		v.Remove("no_public_ip")
	}
	if len(opts.EnableNumaAllocate) > 0 {
		enableNumaAllocate := false
		if opts.EnableNumaAllocate == "True" {
			enableNumaAllocate = true
		}
		v.Set("enable_numa_allocate", jsonutils.NewBool(enableNumaAllocate))
	}
	v.Remove("id")
	return v, nil
}
