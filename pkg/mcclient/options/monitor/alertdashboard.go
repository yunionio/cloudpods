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

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type AlertDashBoardCreateOptions struct {
	apis.ScopedResourceCreateInput
	NAME    string `help:"Name of bashboard"`
	Refresh string `help:"dashboard query refresh priod e.g. 1m|5m"`
}

func (o *AlertDashBoardCreateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type AlertDashBoardListOptions struct {
	options.BaseListOptions
}

func (o *AlertDashBoardListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type AlertDashBoardShowOptions struct {
	ID string `help:"ID of Metric " json:"-"`
}

func (o *AlertDashBoardShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *AlertDashBoardShowOptions) GetId() string {
	return o.ID
}

type AlertDashBoardDeleteOptions struct {
	ID string `json:"-"`
}

func (o *AlertDashBoardDeleteOptions) GetId() string {
	return o.ID
}

func (o *AlertDashBoardDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type AlertClonePanelOptions struct {
	ID             string `help:"ID of alart " json:"-"`
	PanelId        string `help:"ID of alertPanel" json:"panel_id"`
	ClonePanelName string `help:"new panel name" json:"clone_panel_name"`
}

func (o *AlertClonePanelOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *AlertClonePanelOptions) GetId() string {
	return o.ID
}

type AlertCloneDashboardOptions struct {
	ID        string `help:"ID of alart " json:"-"`
	CloneName string `json:"clone_name" help:"new dashboard name"`
}

func (o *AlertCloneDashboardOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *AlertCloneDashboardOptions) GetId() string {
	return o.ID
}
