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

type ScopedPolicyBindingListOptions struct {
	options.ExtraListOptions
	api.ScopedPolicyBindingListInput
}

func (o ScopedPolicyBindingListOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type ScopedPolicyBindingShowOptions struct {
	options.BaseShowOptions `id->help:"ID of scoped policy binding"`
}

type ScopedPolicyBindingIDOptions struct {
	ID string `json:"-" help:"ID of scoped policy binding"`
}

func (o ScopedPolicyBindingIDOptions) GetId() string {
	return o.ID
}

func (o ScopedPolicyBindingIDOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}
