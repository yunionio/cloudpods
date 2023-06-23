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

type OrganizationListOptions struct {
	options.BaseListOptions
}

func (opts *OrganizationListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type OrganizationCreateOptions struct {
	NAME string `help:"name of organization" json:"name"`

	TYPE string `help:"type of organization" choices:"project|domain|object" json:"type"`

	Desc string `help:"description" json:"description"`

	Key []string `help:"tag keys" json:"key"`
}

func (opts *OrganizationCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type OrganizationIdOptions struct {
	ID string `help:"ID or name of organization" json:"-"`
}

func (opts *OrganizationIdOptions) GetId() string {
	return opts.ID
}

func (opts *OrganizationIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type OrganizationSyncOptions struct {
	OrganizationIdOptions

	Reset bool `help:"reset organization nodes"`
}

func (opts *OrganizationSyncOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type OrganizationShowNodesOptions struct {
	OrganizationIdOptions
}

type OrganizationAddLevelOptions struct {
	OrganizationIdOptions

	api.OrganizationPerformAddLevelsInput
}

func (opts *OrganizationAddLevelOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type OrganizationAddNodeOptions struct {
	OrganizationIdOptions

	api.OrganizationPerformAddNodeInput
}

func (opts *OrganizationAddNodeOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}
