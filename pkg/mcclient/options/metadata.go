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
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
)

type MetadataListOptions struct {
	Resources []string `help:"list of resource e.g server、disk、eip、snapshot, empty will show all metadata"`
	Service   string   `help:"service type"`

	SysMeta   *bool `help:"Show sys metadata only"`
	CloudMeta *bool `help:"Show cloud metadata olny"`
	UserMeta  *bool `help:"Show user metadata olny"`

	Admin *bool `help:"Show all metadata"`

	WithSysMeta   *bool `help:"Show sys metadata"`
	WithCloudMeta *bool `help:"Show cloud metadata"`
	WithUserMeta  *bool `help:"Show user metadata"`

	Key   []string `help:"key"`
	Value []string `help:"value"`

	Limit  *int `help:"limit"`
	Offset *int `help:"offset"`

	KeyOnly *bool `help:"show key only"`
}

type TagListOptions MetadataListOptions

type TagValuePairsOptions struct {
	KeyOnly bool `help:"show key only"`

	SysMeta bool `help:"show system tags only"`

	CloudMeta bool `help:"show cloud tags only"`

	UserMeta bool `help:"show user tags only"`
}

func (opts *TagValuePairsOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	if opts.KeyOnly {
		params.Add(jsonutils.JSONTrue, "key_only")
	}
	if opts.UserMeta {
		params.Add(jsonutils.JSONTrue, "user_meta")
	}
	if opts.CloudMeta {
		params.Add(jsonutils.JSONTrue, "cloud_meta")
	}
	if opts.SysMeta {
		params.Add(jsonutils.JSONTrue, "sys_meta")
	}
	return params, nil
}

func (opts *TagValuePairsOptions) Property() string {
	return "tag-value-pairs"
}

type ProjectTagValuePairsOptions struct {
	TagValuePairsOptions
}

func (opts *ProjectTagValuePairsOptions) Property() string {
	return "project-tag-value-pairs"
}

type DomainTagValuePairsOptions struct {
	TagValuePairsOptions
}

func (opts *DomainTagValuePairsOptions) Property() string {
	return "domain-tag-value-pairs"
}

type TagValueTreeOptions struct {
	Key     []string `help:"the sequence of key to construct the tree"`
	ShowMap bool     `help:"show map only"`
}

func (opts *TagValueTreeOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	if len(opts.Key) > 0 {
		params.Add(jsonutils.NewStringArray(opts.Key), "keys")
	}
	if opts.ShowMap {
		params.Add(jsonutils.JSONTrue, "show_map")
	}
	return params, nil
}

func (opts *TagValueTreeOptions) Property() string {
	return "tag-value-tree"
}

type ProjectTagValueTreeOptions struct {
	TagValueTreeOptions
}

func (opts *ProjectTagValueTreeOptions) Property() string {
	return "project-tag-value-tree"
}

type DomainTagValueTreeOptions struct {
	TagValueTreeOptions
}

func (opts *DomainTagValueTreeOptions) Property() string {
	return "domain-tag-value-tree"
}

type ResourceMetadataOptions struct {
	ID   string   `help:"ID or name of resources" json:"-"`
	TAGS []string `help:"Tags info, eg: hypervisor=aliyun、os_type=Linux、os_version"`
}

func (opts *ResourceMetadataOptions) GetId() string {
	return opts.ID
}

func (opts *ResourceMetadataOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	for _, tag := range opts.TAGS {
		sep := "="
		if strings.Index(tag, sep) < 0 {
			sep = ":"
		}
		info := strings.Split(tag, sep)
		if len(info) == 2 {
			if len(info[0]) == 0 {
				return nil, fmt.Errorf("invalidate tag info %s", tag)
			}
			params.Add(jsonutils.NewString(info[1]), info[0])
		} else if len(info) == 1 {
			params.Add(jsonutils.NewString(info[0]), info[0])
		} else {
			return nil, fmt.Errorf("invalidate tag info %s", tag)
		}
	}
	return params, nil
}
