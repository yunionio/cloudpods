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

type AppListOptions struct {
	options.BaseListOptions
	TechStack string
}

func (opts *AppListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type AppIdOptions struct {
	ID string `help:"App Id"`
}

func (opts *AppIdOptions) GetId() string {
	return opts.ID
}

func (opts *AppIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type AppEnvironmentListOptions struct {
	options.BaseListOptions
	AppId        string `help:"App Id"`
	InstanceType string `help:"Instance Type"`
}

func (opts *AppEnvironmentListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type AppEnvironmentIdOption struct {
	ID string `help:"AppEnvironment ID"`
}

func (opts *AppEnvironmentIdOption) GetId() string {
	return opts.ID
}

func (opts *AppEnvironmentIdOption) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}
