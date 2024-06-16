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

	baseoptions "yunion.io/x/onecloud/pkg/mcclient/options"
)

type DBInstanceSkuListOption struct {
	baseoptions.BaseListOptions
	Engine        string
	EngineVersion string
	Category      string
	StorageType   string
	Cloudregion   string
	VcpuCount     *int
	VmemSizeMb    *int
}

func (opts *DBInstanceSkuListOption) Params() (jsonutils.JSONObject, error) {
	return baseoptions.ListStructToParams(opts)
}

type DBInstanceSkuIdOption struct {
	ID string `help:"DBInstance Id or name"`
}

func (opts *DBInstanceSkuIdOption) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

func (opts *DBInstanceSkuIdOption) GetId() string {
	return opts.ID
}
