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

package options

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/cmdline"
)

type DiskCreateOptions struct {
	Region string `help:"Preferred region where virtual server should be created" json:"prefer_region"`
	Zone   string `help:"Preferred zone where virtual server should be created" json:"prefer_zone"`
	Wire   string `help:"Preferred wire where virtual server should be created" json:"prefer_wire"`
	Host   string `help:"Preferred host where virtual server should be created" json:"prefer_host"`

	NAME       string   `help:"Name of the disk"`
	DISKDESC   string   `help:"Image size or size of virtual disk"`
	Desc       string   `help:"Description" metavar:"Description"`
	Storage    string   `help:"ID or name of storage where the disk is created"`
	Hypervisor string   `help:"Hypervisor of this disk, used by schedule"`
	Schedtag   []string `help:"Schedule policy, key = aggregate name, value = require|exclude|prefer|avoid" metavar:"<KEY:VALUE>"`
	TaskNotify bool     `help:"Setup task notify"`
}

func (o DiskCreateOptions) Params() (*api.DiskCreateInput, error) {
	config, err := cmdline.ParseDiskConfig(o.DISKDESC, 0)
	if err != nil {
		return nil, err
	}
	for _, desc := range o.Schedtag {
		tag, err := cmdline.ParseSchedtagConfig(desc)
		if err != nil {
			return nil, err
		}
		config.Schedtags = append(config.Schedtags, tag)
	}
	params := &api.DiskCreateInput{
		PreferRegion: o.Region,
		PreferZone:   o.Zone,
		PreferWire:   o.Wire,
		PreferHost:   o.Host,
		DiskConfig:   config,
		Hypervisor:   o.Hypervisor,
	}
	params.Description = o.Desc
	params.Name = o.NAME
	if o.Storage != "" {
		params.Storage = o.Storage
	}
	return params, nil
}
