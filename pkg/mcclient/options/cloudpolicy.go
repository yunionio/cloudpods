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
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type CloudpolicyListOptions struct {
	BaseListOptions

	CloudproviderId string `json:"cloudprovider_id"`
	ClouduserId     string `json:"clouduser_id"`
	CloudgroupId    string `json:"cloudgroup_id"`
	PolicyType      string `help:"Filter cloudpolicy by policy type" choices:"system|custom"`
}

func (opts *CloudpolicyListOptions) Params() (jsonutils.JSONObject, error) {
	return ListStructToParams(opts)
}

type CloudpolicyIdOptions struct {
	ID string `help:"Cloudpolicy Id"`
}

func (opts *CloudpolicyIdOptions) GetId() string {
	return opts.ID
}

func (opts *CloudpolicyIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type CloudpolicyGroupOptions struct {
	CloudpolicyIdOptions
	CLOUDGROUP_ID string `help:"Cloudgroup Id" json:"cloudgroup_id"`
}

func (opts *CloudpolicyGroupOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"cloudgroup_id": opts.CLOUDGROUP_ID}), nil
}

type CloudpolicyUpdateOption struct {
	CloudpolicyIdOptions
	Name           string
	Description    string
	PolicyDocument string
}

func (opts *CloudpolicyUpdateOption) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	params.Remove("policy_document")
	if len(opts.PolicyDocument) > 0 {
		document, err := jsonutils.Parse([]byte(opts.PolicyDocument))
		if err != nil {
			return nil, errors.Wrapf(err, "invalid policy document")
		}
		params.Add(document, "document")
	}
	return params, nil
}

type CloudpolicyCreateOption struct {
	NAME            string
	PROVIDER        string `choices:"Aliyun|Google|Aws|Azure|Huawei"`
	Descritpion     string
	POLICY_DOCUMENT string
}

func (opts *CloudpolicyCreateOption) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	params.Remove("policy_document")
	document, err := jsonutils.Parse([]byte(opts.POLICY_DOCUMENT))
	if err != nil {
		return nil, errors.Wrapf(err, "invalid policy document")
	}
	params.Add(document, "document")
	return params, nil
}
