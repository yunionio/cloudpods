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
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type SnapshotPolicyListOptions struct {
	options.BaseListOptions

	OrderByBindDiskCount string
	OrderBySnapshotCount string
}

func (opts *SnapshotPolicyListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type SnapshotPolicyCreateOptions struct {
	options.BaseCreateOptions
	CloudregionId string
	ManagerId     string

	Type           string `help:"snapshot type" default:"disk" choices:"disk|server"`
	RetentionDays  int    `help:"snapshot retention days" default:"-1"`
	RetentionCount int    `help:"snapshot retention count" default:"-1"`
	RepeatWeekdays []int  `help:"snapshot create days on week"`
	TimePoints     []int  `help:"snapshot create time points on one day"`
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

type SnapshotPolicyResourcesOptions struct {
	options.BaseIdOptions
	Resources []string `help:"resource info etc: disk:id,server:id"`
}

func (opts *SnapshotPolicyResourcesOptions) Params() (jsonutils.JSONObject, error) {
	resources := []struct {
		Id   string `json:"id"`
		Type string `json:"type"`
	}{}
	for _, resource := range opts.Resources {
		parts := strings.SplitN(resource, ":", 2)
		if len(parts) != 2 {
			return nil, httperrors.NewBadRequestError("Invalid resource info: %s", resource)
		}
		resources = append(resources, struct {
			Id   string `json:"id"`
			Type string `json:"type"`
		}{
			Id:   parts[1],
			Type: parts[0],
		})
	}
	return jsonutils.Marshal(map[string]interface{}{"resources": resources}), nil
}
