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

type ExternalProjectListOptions struct {
	options.BaseListOptions
}

func (opts *ExternalProjectListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type ExternalProjectIdOption struct {
	ID string `help:"external project Id"`
}

func (opts *ExternalProjectIdOption) GetId() string {
	return opts.ID
}

func (opts *ExternalProjectIdOption) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type ExterProjectChagneProjectOptions struct {
	ExternalProjectIdOption
	PROJECT string
}

func (opts *ExterProjectChagneProjectOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"project": opts.PROJECT}), nil
}

type ExternalProjectCreateOptions struct {
	NAME            string
	CLOUDACCOUNT_ID string
	ManagerId       string
	Project         string
}

func (opts *ExternalProjectCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type ExternalProjectUpdateOptions struct {
	options.BaseUpdateOptions
	Priority *int
}

func (opts *ExternalProjectUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}
