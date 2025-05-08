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
	"os"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type HostFileListOptions struct {
	Type []string `help:"Type of host file"`

	options.BaseListOptions
}

func (opts *HostFileListOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.ListStructToParams(opts)
	if err != nil {
		return nil, err
	}
	if len(opts.Type) > 0 {
		params.Add(jsonutils.NewStringArray(opts.Type), "type")
	}
	return params, nil
}

type HostFileShowOptions struct {
	options.BaseShowOptions
}

func (o *HostFileShowOptions) Params() (jsonutils.JSONObject, error) {
	// NOTE: host show only request with base options
	return jsonutils.Marshal(o.BaseShowOptions), nil
}

type HostFileUpdateOptions struct {
	options.BaseIdOptions

	computeapi.HostFileUpdateInput

	File string `json:"file"`
}

func (opts *HostFileUpdateOptions) Params() (jsonutils.JSONObject, error) {
	if len(opts.File) > 0 {
		cont, err := os.ReadFile(opts.File)
		if err != nil {
			return nil, err
		}
		opts.Content = string(cont)
	}
	return jsonutils.Marshal(opts), nil
}

type HostFileCreateOptions struct {
	computeapi.HostFileCreateInput

	File string `json:"file"`
}

func (opts *HostFileCreateOptions) Params() (jsonutils.JSONObject, error) {
	if len(opts.File) > 0 {
		cont, err := os.ReadFile(opts.File)
		if err != nil {
			return nil, err
		}
		opts.Content = string(cont)
	}
	return jsonutils.Marshal(opts), nil
}

type HostFileDeleteOptions struct {
	options.BaseIdOptions
}

func (opts *HostFileDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type HostFileEditOptions struct {
	options.BaseIdOptions
}

func (opts *HostFileEditOptions) EditType() shell.EditType {
	return shell.EditTypeText
}

func (opts *HostFileEditOptions) EditFields() []string {
	return []string{"content"}
}
