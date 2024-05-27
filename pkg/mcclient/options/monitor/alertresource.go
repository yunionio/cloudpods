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

package monitor

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type AlertResourceListOptions struct {
	options.BaseListOptions
	Type string `json:"type"`
}

func (o AlertResourceListOptions) Params() (jsonutils.JSONObject, error) {
	param, err := o.BaseListOptions.Params()
	if err != nil {
		return nil, err
	}
	params := param.(*jsonutils.JSONDict)
	if o.Type != "" {
		params.Add(jsonutils.NewString(o.Type), "type")
	}
	return params, nil
}

type AlertResourceShowOptions struct {
	ID string `help:"ID or name of the alert resource" json:"-"`
}

func (o AlertResourceShowOptions) GetId() string {
	return o.ID
}

func (o AlertResourceShowOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type AlertResourceDeleteOptions struct {
	ID []string `help:"ID of alert resource to delete"`
}

func (o AlertResourceDeleteOptions) GetIds() []string {
	return o.ID
}

func (o AlertResourceDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type AlertResourceAlertListOptions struct {
	options.BaseListOptions
	Resource string `help:"ID or name of alert resource"`
	Alert    string `help:"ID or name of alert"`
}

func (o AlertResourceAlertListOptions) GetMasterOpt() string {
	return o.Resource
}

func (o AlertResourceAlertListOptions) GetSlaveOpt() string {
	return o.Alert
}

func (o AlertResourceAlertListOptions) Params() (jsonutils.JSONObject, error) {
	return o.BaseListOptions.Params()
}
