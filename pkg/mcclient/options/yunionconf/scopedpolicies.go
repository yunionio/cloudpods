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

package yunionconf

import (
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/yunionconf"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ScopedPolicyListOptions struct {
	options.ExtraListOptions
	api.ScopedPolicyListInput
}

func (o ScopedPolicyListOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type ScopedPolicyShowOptions struct {
	options.BaseShowOptions `id->help:"ID or name of scoped policy"`
}

type ScopedPolicyIDOptions struct {
	ID string `help:"ID or name of scoped policy" json:"-"`
}

func (o ScopedPolicyIDOptions) GetId() string {
	return o.ID
}

func (o ScopedPolicyIDOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type ScopedPolicyCreateOptions struct {
	api.ScopedPolicyCreateInput
}

func (o ScopedPolicyCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type ScopedPolicyUpdateOptions struct {
	ScopedPolicyIDOptions
	api.ScopedPolicyUpdateInput
}

func (o ScopedPolicyUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type ScopedPolicyBindOptions struct {
	ScopedPolicyIDOptions
	api.ScopedPolicyBindInput
}
