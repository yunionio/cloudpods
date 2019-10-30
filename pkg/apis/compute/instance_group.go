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

import "yunion.io/x/onecloud/pkg/apis"

type InstanceGroupListInput struct {
	apis.BaseListInput

	// Filter by service type
	ServiceType string `json:"service_type"`
	// Filter by parent id
	ParentId string `json:"parent_id"`
	// Filter by zone id
	ZoneId string `json:"zone_id"`
	// Filter by guest id or name
	Guest string `json:"guest"`
}

type InstanceGroupDetail struct {
	apis.Meta
	SGroup
	GuestCount int64 `json:"guest_count"`
}
