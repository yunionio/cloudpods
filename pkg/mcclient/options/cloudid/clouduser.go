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

type ClouduserListOptions struct {
	options.BaseListOptions
	CloudaccountId  string `help:"Cloudaccount Id"`
	CloudproviderId string `help:"Cloudprovider Id"`
	CloudpolicyId   string `help:"filter cloudusers by cloudpolicy"`
	CloudgroupId    string `help:"filter cloudusers by cloudgroup"`
}

func (opts *ClouduserListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type ClouduserCreateOptions struct {
	NAME           string   `help:"Clouduser name"`
	MANAGER_ID     string   `help:"Cloudprovider Id"`
	OwnerId        string   `help:"Owner Id"`
	CloudpolicyIds []string `help:"cloudpolicy ids"`
	CloudgroupIds  []string `help:"cloudgroup ids"`
	Email          string   `help:"email address"`
	MobilePhone    string   `help:"phone number"`
	IsConsoleLogin *bool    `help:"is console login"`
	Password       string   `help:"clouduser password"`
	Notify         *bool    `help:"Notify user which set email when clouduser created"`
}

func (opts *ClouduserCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type ClouduserIdOption struct {
	ID string `help:"Clouduser Id or name"`
}

func (opts *ClouduserIdOption) GetId() string {
	return opts.ID
}

func (opts *ClouduserIdOption) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type ClouduserSyncOptions struct {
	ClouduserIdOption
	PolicyOnly bool `help:"Ony sync clouduser policies for cloud"`
}

func (opts *ClouduserSyncOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]bool{"policy_only": opts.PolicyOnly}), nil
}

type ClouduserPolicyOptions struct {
	ClouduserIdOption
	CLOUDPOLICY_ID string `help:"cloudpolicy Id"`
}

func (opts *ClouduserPolicyOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{
		"cloudpolicy_id": opts.CLOUDPOLICY_ID,
	}), nil
}

type ClouduserPasswordOptions struct {
	ClouduserIdOption
	Password string `help:"clouduser password"`
}

func (opts *ClouduserPasswordOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"password": opts.Password}), nil
}

type ClouduserChangeOwnerOptions struct {
	ClouduserIdOption
	UserId string `help:"local user id"`
}

func (opts *ClouduserChangeOwnerOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"user_id": opts.UserId}), nil
}

type ClouduserGroupOptions struct {
	ClouduserIdOption
	CLOUDGROUP_ID string `help:"cloudgroup id" json:"cloudgroup_id"`
}

func (opts *ClouduserGroupOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"cloudgroup_id": opts.CLOUDGROUP_ID}), nil
}

type ClouduserResetPasswordOptions struct {
	ClouduserIdOption
	Password string `help:"password"`
}

func (opts *ClouduserResetPasswordOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"password": opts.Password}), nil
}

type ClouduserCreateAccessKeyInput struct {
	ClouduserIdOption
	Name string `json:"Name"`
}

func (opts *ClouduserCreateAccessKeyInput) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"id": opts.ID, "name": opts.Name}), nil
}

type ClouduserDeleteAccessKeyInput struct {
	ClouduserIdOption
	AccessKey string `json:"access_key"`
}

func (opts *ClouduserDeleteAccessKeyInput) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"access_key": opts.AccessKey}), nil
}

type ClouduserListAccessKeyInput struct {
	ClouduserIdOption
}

func (opts *ClouduserListAccessKeyInput) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}
