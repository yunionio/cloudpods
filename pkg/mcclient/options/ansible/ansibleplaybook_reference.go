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

package ansible

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type APRListOptions struct {
	options.BaseListOptions
}

func (ao *APRListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(ao)
}

type APROptions struct {
	ID string `help:"id or name of ansible playbook reference"`
}

func (ao *APROptions) GetId() string {
	return ao.ID
}

func (ao *APROptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type aprRunOptions struct {
	ServerName      string
	ServerIp        string
	ServerUser      string
	Args            map[string]interface{}
	ProxyEndpoingId string
}

type APRRunOptions struct {
	APROptions
	aprRunOptions
}

func (ao *APRRunOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(ao.aprRunOptions), nil
}

type aprStopOptions struct {
	AnsiblePlaybookInstanceId string
}

type APRStopOptions struct {
	APROptions
	aprStopOptions
}

func (ao *APRStopOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(ao.aprStopOptions), nil
}

type APIListOptions struct {
	options.BaseListOptions
	AnsiblePlayboookReferenceId string
}

func (ao *APIListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(ao)
}

type APIOptions struct {
	ID string
}

func (ao *APIOptions) GetId() string {
	return ao.ID
}

func (ao *APIOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}
