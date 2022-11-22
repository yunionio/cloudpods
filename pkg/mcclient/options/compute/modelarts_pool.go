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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ModelartsPoolListOptions struct {
	options.BaseListOptions
}

func (opts *ModelartsPoolListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type ModelartsPoolIdOption struct {
	ID string `help:"ModelartsPool Id"`
}

func (opts *ModelartsPoolIdOption) GetId() string {
	return opts.ID
}

func (opts *ModelartsPoolIdOption) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type ModelartsPoolCreateOption struct {
	Name         string `help:"Name"`
	ManagerId    string `help:"Manager Id"`
	InstanceType string `help:"Instance Type"`
	WorkType     string `help:"Work Type"`
	CpuArch      string `help:"Cpu Arch"`
	NodeCount    int    `help:"Node Count"`
	Cidr         string `help:"Network Cidr"`

	CloudregionId string `help:"Cloud Region ID"`
}

func (opts *ModelartsPoolCreateOption) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type ModelartsPoolUpdateOption struct {
	ID       string `help:"Id"`
	WorkType string `help:"Work Type"`
}

func (opts *ModelartsPoolUpdateOption) GetId() string {
	return opts.ID
}

func (opts *ModelartsPoolUpdateOption) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opts)
}

type ModelartsPoolSyncstatusOption struct {
	ID string `help:"Id"`
}

func (opts *ModelartsPoolSyncstatusOption) GetId() string {
	return opts.ID
}

func (opts *ModelartsPoolSyncstatusOption) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opts)
}

type ModelartsPoolChangeConfigOption struct {
	ID        string `help:"Id"`
	NodeCount int
}

func (opts *ModelartsPoolChangeConfigOption) GetId() string {
	return opts.ID
}

func (opts *ModelartsPoolChangeConfigOption) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opts)
}
