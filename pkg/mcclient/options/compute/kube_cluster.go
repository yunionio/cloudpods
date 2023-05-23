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

type KubeClusterListOptions struct {
	options.BaseListOptions
}

func (opts *KubeClusterListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type KubeClusterIdOption struct {
	ID string `help:"KubeCluster Id"`
}

func (opts *KubeClusterIdOption) GetId() string {
	return opts.ID
}

func (opts *KubeClusterIdOption) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type KubeClusterConfigOptions struct {
	KubeClusterIdOption
	Private       bool
	ExpireMinutes int
}

func (opts *KubeClusterConfigOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type KubeClusterCreateOptions struct {
	options.BaseCreateOptions
	Version       string
	VpcId         string
	NetworkIds    []string
	RoleName      string
	PrivateAccess bool
	PublicAccess  bool
}

func (opts *KubeClusterCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}
