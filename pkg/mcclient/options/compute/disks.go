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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/cmdline"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type DiskCreateOptions struct {
	Manager string `help:"Preferred manager where virtual server should be created" json:"prefer_manager_id"`
	Region  string `help:"Preferred region where virtual server should be created" json:"prefer_region_id"`
	Zone    string `help:"Preferred zone where virtual server should be created" json:"prefer_zone_id"`
	Wire    string `help:"Preferred wire where virtual server should be created" json:"prefer_wire_id"`
	Host    string `help:"Preferred host where virtual server should be created" json:"prefer_host_id"`
	Count   int    `help:"Count to create" json:"count"`

	NAME       string   `help:"Name of the disk"`
	DISKDESC   string   `help:"Image size or size of virtual disk"`
	Desc       string   `help:"Description" metavar:"Description"`
	Storage    string   `help:"ID or name of storage where the disk is created"`
	Hypervisor string   `help:"Hypervisor of this disk, used by schedule"`
	Backend    string   `help:"Backend of this disk"`
	Schedtag   []string `help:"Schedule policy, key = aggregate name, value = require|exclude|prefer|avoid" metavar:"<KEY:VALUE>"`
	TaskNotify bool     `help:"Setup task notify"`
	SnapshotId string   `help:"snapshot id"`
	BackupId   string   `help:"Backup id"`

	Project string `help:"Owner project"`
}

func (o DiskCreateOptions) Params() (*api.DiskCreateInput, error) {
	config, err := cmdline.ParseDiskConfig(o.DISKDESC, 0)
	if err != nil {
		return nil, err
	}
	if len(o.Backend) > 0 {
		config.Backend = o.Backend
	}
	for _, desc := range o.Schedtag {
		tag, err := cmdline.ParseSchedtagConfig(desc)
		if err != nil {
			return nil, err
		}
		config.Schedtags = append(config.Schedtags, tag)
	}
	params := &api.DiskCreateInput{
		PreferManager: o.Manager,
		PreferRegion:  o.Region,
		PreferZone:    o.Zone,
		PreferWire:    o.Wire,
		PreferHost:    o.Host,
		DiskConfig:    config,
		Hypervisor:    o.Hypervisor,
	}
	params.Description = o.Desc
	params.Name = o.NAME
	params.ProjectId = o.Project
	if o.Storage != "" {
		params.Storage = o.Storage
	}
	params.BackupId = o.BackupId
	return params, nil
}

type DiskMigrateOptions struct {
	ID string `help:"ID of the server" json:"-"`

	TargetStorageId string `help:"Disk migrate target storage id or name" json:"target_storage_id"`
}

func (o *DiskMigrateOptions) GetId() string {
	return o.ID
}

func (o *DiskMigrateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}
