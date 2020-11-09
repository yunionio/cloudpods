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

type LoadbalancerAclDetails struct {
	apis.SharableVirtualResourceDetails
	ManagedResourceInfo
	CloudregionResourceInfo

	SLoadbalancerAcl

	LbListenerCount int `json:"lb_listener_count"`
}

type LoadbalancerAclResourceInfo struct {
	// 负载均衡ACL名称
	Acl string `json:"acl"`
}

type LoadbalancerAclResourceInput struct {
	// ACL名称或ID
	AclId string `json:"acl_id"`

	// swagger:ignore
	// Deprecated
	Acl string `json:"acl" yunion-deprecated-by:"acl_id"`
}

type LoadbalancerAclFilterListInput struct {
	LoadbalancerAclResourceInput

	// 以ACL名称排序
	OrderByAcl string `json:"order_by_acl"`
}
