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

type DiskIdOptions struct {
	ID string `help:"ID or Name of disk"`
}

func (o *DiskIdOptions) GetId() string {
	return o.ID
}

func (o *DiskIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type DiskMigrateOptions struct {
	DiskIdOptions

	TargetStorageId string `help:"Disk migrate target storage id or name" json:"target_storage_id"`
}

func (o *DiskMigrateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type DiskChangeStorageTypeOptions struct {
	DiskIdOptions

	StorageType string `help:"Disk migrate target storage type" json:"storage_type"`
}

func (o *DiskChangeStorageTypeOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type DiskResetTemplateOptions struct {
	DiskIdOptions

	TemplateId string `help:"reset disk tempalte id" json:"template_id"`
}

func (o *DiskResetTemplateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type DiskRebuildOptions struct {
	options.ResourceIdOptions
	BackupId                                  string `help:"disk backup id" json:"backup_id"`
	TemplateId                                string `help:"disk template id" json:"template_id"`
	Size                                      string `help:"disk size in MB" json:"size"`
	Fs                                        string `help:"disk fs type"`
	FsFeatureF2fsCaseInsensitive              *bool  `help:"f2fs enable CaseInsensitive" json:"-"`
	FsFeatureF2fsOverprovisionRatioPercentage *int   `help:"f2fs OverprovisionRatioPercentage" json:"-"`
	FsFeatureExt4CaseInsensitive              *bool  `help:"ext4 enable CaseInsensitive" json:"-"`
	FsFeatureExt4ReservedBlocksPercentage     *int   `help:"ext4 ReservedBlocksPercentage" json:"-"`
}

func (o *DiskRebuildOptions) Params() (jsonutils.JSONObject, error) {
	res := api.DiskRebuildInput{}
	if o.BackupId != "" {
		res.BackupId = &o.BackupId
	}
	if o.TemplateId != "" {
		res.TemplateId = &o.TemplateId
	}
	if o.Size != "" {
		res.TemplateId = &o.Size
	}
	if o.Fs != "" {
		res.Fs = &o.Fs
	}

	if o.FsFeatureExt4CaseInsensitive != nil || o.FsFeatureExt4ReservedBlocksPercentage != nil {
		if res.FsFeatures == nil {
			res.FsFeatures = &api.DiskFsFeatures{}
		}
		res.FsFeatures.Ext4 = &api.DiskFsExt4Features{}
		if o.FsFeatureExt4CaseInsensitive != nil {
			res.FsFeatures.Ext4.CaseInsensitive = *o.FsFeatureExt4CaseInsensitive
		}
		if o.FsFeatureExt4ReservedBlocksPercentage != nil {
			res.FsFeatures.Ext4.ReservedBlocksPercentage = *o.FsFeatureExt4ReservedBlocksPercentage
		}
	}
	if o.FsFeatureF2fsCaseInsensitive != nil || o.FsFeatureF2fsOverprovisionRatioPercentage != nil {
		if res.FsFeatures == nil {
			res.FsFeatures = &api.DiskFsFeatures{}
		}
		res.FsFeatures.F2fs = &api.DiskFsF2fsFeatures{}
		if o.FsFeatureF2fsCaseInsensitive != nil {
			res.FsFeatures.F2fs.CaseInsensitive = *o.FsFeatureF2fsCaseInsensitive
		}
		if o.FsFeatureF2fsOverprovisionRatioPercentage != nil {
			res.FsFeatures.F2fs.OverprovisionRatioPercentage = *o.FsFeatureF2fsOverprovisionRatioPercentage
		}
	}
	return jsonutils.Marshal(res), nil
}

type DiskListOptions struct {
	options.BaseListOptions
	Unused        *bool    `help:"Show unused disks"`
	Share         *bool    `help:"Show Share storage disks"`
	Local         *bool    `help:"Show Local storage disks"`
	ServerId      []string `help:"Guest ID or name"`
	GuestStatus   string   `help:"Guest Status"`
	OrderByServer string   `help:"Order By Server"`
	Storage       string   `help:"Storage ID or name"`
	Type          string   `help:"Disk type" choices:"sys|data|swap|volume"`
	CloudType     string   `help:"Public cloud or private cloud" choices:"Public|Private"`

	OrderByGuestCount string `help:"Order By Guest Count"`

	BillingType string `help:"billing type" choices:"postpaid|prepaid"`

	SnapshotpolicyId string `help:"snapshotpolicy id"`

	StorageHostId               string `help:"filter disk by host"`
	BindingServerSnapshotpolicy *bool  `help:"filter disk by binding server snapshotpolicy" negative:"no-binding-server-snapshotpolicy"`
	BindingSnapshotpolicy       *bool  `help:"filter disk by binding snapshotpolicy" negative:"no-binding-snapshotpolicy"`
}

func (opts *DiskListOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.ListStructToParams(opts)
	if err != nil {
		return nil, err
	}
	if len(opts.CloudType) > 0 {
		if opts.CloudType == "Public" {
			params.Add(jsonutils.JSONTrue, "public_cloud")
		} else if opts.CloudType == "Private" {
			params.Add(jsonutils.JSONTrue, "private_cloud")
		}
	}
	return params, nil
}

type DiskChangeBillingTypeOptions struct {
	DiskIdOptions
	BillingType string `choices:"prepaid|postpaid"`
}

func (o *DiskChangeBillingTypeOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"billing_type": o.BillingType}), nil
}
