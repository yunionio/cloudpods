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

type CloudproviderListOptions struct {
	options.BaseListOptions

	Usable bool `help:"Vpc & Network usable"`

	HasObjectStorage bool     `help:"filter cloudproviders that has object storage" negative:"no-object-storage"`
	Capability       []string `help:"capability filter" choices:"project|compute|network|loadbalancer|objectstore|rds|cache|event"`
	Cloudregion      string   `help:"filter cloudproviders by cloudregion"`

	ReadOnly *bool `help:"filter read only account" negative:"no-read-only"`

	HostSchedtagId string `help:"filter by host schedtag"`
	ZoneId         string
}

func (opts *CloudproviderListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type CloudproviderUpdateOptions struct {
	options.BaseIdOptions
	Name      string `help:"New name to update"`
	AccessUrl string `help:"New access url"`
	Desc      string `help:"Description"`
}

func (opts *CloudproviderUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	if len(opts.Name) > 0 {
		params.Add(jsonutils.NewString(opts.Name), "name")
	}
	if len(opts.AccessUrl) > 0 {
		params.Add(jsonutils.NewString(opts.AccessUrl), "access_url")
	}
	if len(opts.Desc) > 0 {
		params.Add(jsonutils.NewString(opts.Desc), "description")
	}
	return params, nil
}

type CloudproviderChangeProjectOptions struct {
	options.BaseIdOptions
	TENANT string `help:"ID or Name of tenant"`
}

func (opts *CloudproviderChangeProjectOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"project": opts.TENANT}), nil
}

type CloudproviderSyncOptions struct {
	options.BaseIdOptions
	Force       bool     `help:"Force sync no matter what"`
	FullSync    bool     `help:"Synchronize everything"`
	ProjectSync bool     `help:"Auto sync project info"`
	Region      []string `help:"region to sync"`
	Zone        []string `help:"region to sync"`
	Host        []string `help:"region to sync"`
}

func (opts *CloudproviderSyncOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	if opts.Force {
		params.Add(jsonutils.JSONTrue, "force")
	}
	if opts.FullSync {
		params.Add(jsonutils.JSONTrue, "full_sync")
	}
	if opts.ProjectSync {
		params.Add(jsonutils.JSONTrue, "project_sync")
	}
	if len(opts.Region) > 0 {
		params.Add(jsonutils.NewStringArray(opts.Region), "region")
	}
	if len(opts.Zone) > 0 {
		params.Add(jsonutils.NewStringArray(opts.Zone), "zone")
	}
	if len(opts.Host) > 0 {
		params.Add(jsonutils.NewStringArray(opts.Host), "host")
	}
	return params, nil
}

type CloudproviderStorageClassesOptions struct {
	options.BaseIdOptions
	Cloudregion string `help:"cloud region name or Id"`
}

func (opts *CloudproviderStorageClassesOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opts)
}

type ClouproviderProjectMappingOptions struct {
	options.BaseIdOptions
	ProjectMappingId string `json:"project_mapping_id" help:"project mapping id"`
}

func (opts *ClouproviderProjectMappingOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"project_mapping_id": opts.ProjectMappingId}), nil
}

type ClouproviderSetSyncingOptions struct {
	options.BaseIdOptions
	Enabled        bool     `help:"Enable or disable sync"`
	CloudregionIds []string `help:"Cloudregion ids for sync"`
}

func (opts *ClouproviderSetSyncingOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]interface{}{
		"enabled":         opts.Enabled,
		"cloudregion_ids": opts.CloudregionIds,
	}), nil
}
