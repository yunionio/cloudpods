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

package identity

import (
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type OrganizationNodeListOptions struct {
	options.BaseListOptions

	OrgId string `help:"filter by organization Id"`

	OrgType string `help:"filter by organization type"`

	Level int `help:"filter by level"`
}

func (opts *OrganizationNodeListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type OrganizationNodeIdOptions struct {
	ID string `help:"ID or name of organization node" json:"-"`
}

func (opts *OrganizationNodeIdOptions) GetId() string {
	return opts.ID
}

func (opts *OrganizationNodeIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type OrganizationNodeUpdateOptions struct {
	OrganizationNodeIdOptions

	Weigth      int    `help:"update weight of node"`
	Description string `help:"update description"`
}

func (opts *OrganizationNodeUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type OrganizationNodeBindOptions struct {
	OrganizationNodeIdOptions

	api.OrganizationNodePerformBindInput
}

func (opts *OrganizationNodeBindOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}
