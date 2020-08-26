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

package options

type ProxySettingCreateOptions struct {
	NAME string

	HTTPProxy  string
	HTTPSProxy string
	NoProxy    string
}

type ProxySettingGetOptions struct {
	ID string `json:"-"`
}

type ProxySettingUpdateOptions struct {
	ID   string `json:"-"`
	Name string

	HTTPProxy  string
	HTTPSProxy string
	NoProxy    string
}

type ProxySettingDeleteOptions struct {
	ID string `json:"-"`
}

type ProxySettingTestOptions struct {
	ID string `json:"-"`
}

type ProxySettingListOptions struct {
	BaseListOptions
}

type ProxySettingPublicOptions struct {
	ProxySettingGetOptions
	Scope        string   `json:"scope" help:"share scope" choices:"domain|system"`
	SharedDomain []string `json:"share"`
}

type ProxySettingPrivateOptions struct {
	ProxySettingGetOptions
}
