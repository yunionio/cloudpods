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

import (
	"yunion.io/x/jsonutils"
)

type DnsZoneCacheListOptions struct {
	BaseListOptions
	DnsZoneId string `help:"Dns Zone Id"`
}

func (opts *DnsZoneCacheListOptions) Params() (jsonutils.JSONObject, error) {
	return ListStructToParams(opts)
}

type DnsZoneCacheIdOptions struct {
	ID string `help:"Dns zone cache Id or Name"`
}

func (opts DnsZoneCacheIdOptions) GetId() string {
	return opts.ID
}

func (opts DnsZoneCacheIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}
