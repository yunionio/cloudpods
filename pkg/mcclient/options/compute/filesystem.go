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

type FileSystemListOptions struct {
	options.BaseListOptions
}

func (opts *FileSystemListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type FileSystemIdOption struct {
	ID string `help:"File system Id"`
}

func (opts *FileSystemIdOption) GetId() string {
	return opts.ID
}

func (opts *FileSystemIdOption) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type FileSystemCreateOptions struct {
	NAME           string
	Protocol       string `choices:"NFS|SMB|CPFS|CephFS"`
	FileSystemType string
	Capacity       int64  `json:"capacity"`
	NetworkId      string `json:"network_id"`
	StorageType    string `json:"storage_type"`
	ZoneId         string `json:"zone_id"`
	ManagerId      string `json:"manager_id"`
}

func (opts *FileSystemCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type FileSystemSetQuotaOption struct {
	FileSystemIdOption
	MaxGb    int64
	MaxFiles int64
}

func (opts *FileSystemSetQuotaOption) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}
