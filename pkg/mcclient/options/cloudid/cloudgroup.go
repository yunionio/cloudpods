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

package cloudid

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type CloudgroupListOptions struct {
	options.BaseListOptions

	ClouduserId   string `json:"clouduser_id"`
	CloudpolicyId string `json:"cloudpolicy_id"`
	Usable        bool   `json:"usable"`
}

func (opts *CloudgroupListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type CloudgroupCreateOptions struct {
	NAME           string `json:"name"`
	MANAGER_ID     string
	CloudpolicyIds []string `json:"cloudpolicy_ids"`
	Desc           string   `json:"description"`
}

func (opts *CloudgroupCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type CloudgroupIdOptions struct {
	ID string `help:"Cloudgroup Id"`
}

func (opts *CloudgroupIdOptions) GetId() string {
	return opts.ID
}

func (opts *CloudgroupIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type CloudgroupPolicyOptions struct {
	CloudgroupIdOptions
	CLOUDPOLICY_ID string `help:"Cloudpolicy Id"`
}

func (opts *CloudgroupPolicyOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"cloudpolicy_id": opts.CLOUDPOLICY_ID}), nil
}

type CloudgroupUserOptions struct {
	CloudgroupIdOptions
	CLOUDUSER_ID string `help:"Clouduser Id"`
}

func (opts *CloudgroupUserOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"clouduser_id": opts.CLOUDUSER_ID}), nil
}

type CloudgroupPoliciesOptions struct {
	CloudgroupIdOptions
	CloudpolicyIds []string `json:"cloudpolicy_ids"`
}

func (opts *CloudgroupPoliciesOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string][]string{"cloudpolicy_ids": opts.CloudpolicyIds}), nil
}

type CloudgroupUsersOptions struct {
	CloudgroupIdOptions
	ClouduserIds []string `json:"clouduser_ids"`
}

func (opts *CloudgroupUsersOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string][]string{"clouduser_ids": opts.ClouduserIds}), nil
}

type CloudgroupPublicOptions struct {
	CloudgroupIdOptions
	Scope         string   `help:"sharing scope" choices:"system|domain"`
	SharedDomains []string `help:"share to domains"`
}

func (opts *CloudgroupPublicOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	if len(opts.Scope) > 0 {
		params.Add(jsonutils.NewString(opts.Scope), "scope")
	}
	if len(opts.SharedDomains) > 0 {
		params.Add(jsonutils.Marshal(opts.SharedDomains), "shared_domains")
	}
	return params, nil
}
