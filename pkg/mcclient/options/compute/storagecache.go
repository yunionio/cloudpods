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

type StoragecacheListOptions struct {
	options.BaseListOptions

	CloudregionId string `help:"cloudregion id"`
}

func (opts *StoragecacheListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type StorageCacheImageActionOptions struct {
	options.BaseIdOptions
	IMAGE  string `help:"ID or name of image"`
	Force  bool   `help:"Force refresh cache, even if the image exists in cache"`
	Format string `help:"Image force" choices:"iso|vmdk|qcow2|vhd|tgz"`
}

func (opts *StorageCacheImageActionOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opts)
}

type StorageUncacheImageActionOptions struct {
	options.BaseIdOptions
	IMAGE string `help:"ID or name of image"`
	Force bool   `help:"Force uncache, even if the image exists in cache"`
}

func (opts *StorageUncacheImageActionOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opts)
}
