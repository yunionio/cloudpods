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

package cloudid

import "yunion.io/x/onecloud/pkg/apis"

const (
	CLOUD_POLICY_CACHE_STATUS_CACHING = "caching"
	CLOUD_POLICY_CACHE_STATUS_READY   = "ready"
)

type CloudpolicycacheListInput struct {
	apis.StatusStandaloneResourceListInput

	// 根据权限过滤
	CloudpolicyId string `json:"cloudpolicy_id"`

	// 根据云账号过滤
	CloudaccountId string `json:"cloudaccount_id"`
}

type CloudpolicycacheDetails struct {
	apis.StatusStandaloneResourceDetails
	CloudaccountResourceDetails
	CloudproviderResourceDetails
}
