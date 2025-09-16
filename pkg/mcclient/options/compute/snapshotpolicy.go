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

type SnapshotPolicyListOptions struct {
	options.BaseListOptions

	OrderByBindDiskCount string
}

func (opts *SnapshotPolicyListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type SnapshotPolicyCreateOptions struct {
	options.BaseCreateOptions
	CloudregionId string
	ManagerId     string

	RetentionDays  int   `help:"snapshot retention days" default:"-1"`
	RepeatWeekdays []int `help:"snapshot create days on week"`
	TimePoints     []int `help:"snapshot create time points on one day"`
}

func (opts *SnapshotPolicyCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type SnapshotPolicyDisksOptions struct {
	options.BaseIdOptions
	Disks []string `help:"ids of disk"`
}

func (opts *SnapshotPolicyDisksOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]interface{}{"disks": opts.Disks}), nil
}
