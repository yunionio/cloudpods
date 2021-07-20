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

type CommonAlertListOptions struct {
	options.BaseListOptions
	// 报警类型
	AlertType string `help:"common alert type" choices:"normal|system"`
	Level     string `help:"common alert notify level" choices:"normal|important|fatal"`
}

func (o *CommonAlertListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type CommonAlertShowOptions struct {
	ID string `help:"ID of alart " json:"-"`
}

func (o *CommonAlertShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *CommonAlertShowOptions) GetId() string {
	return o.ID
}

type CommonAlertUpdateOptions struct {
	ID         string `help:"ID of alart " json:"-"`
	Period     string `help:"exec period of alert" json:"period"`
	Comparator string `help:"Alarm policy threshold comparison method" json:"comparator" `
	Threshold  string `help:"Alarm policy threshold" json:"threshold"`
}

func (o *CommonAlertUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *CommonAlertUpdateOptions) GetId() string {
	return o.ID
}

type CommonAlertDeleteOptions struct {
	ID    []string `help:"ID of alart"`
	Force bool     `help:"force to delete alert"`
}

func (o *CommonAlertDeleteOptions) GetIds() []string {
	return o.ID
}

func (o *CommonAlertDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}
