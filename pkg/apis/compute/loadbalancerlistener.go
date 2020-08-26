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

type LoadbalancerListenerDetails struct {
	apis.VirtualResourceDetails
	LoadbalancerResourceInfo
	LoadbalancerAclResourceInfo
	LoadbalancerCertificateResourceInfo

	SLoadbalancerListener

	BackendGroup        string `json:"backend_group"`
	CertificateName     string `json:"certificate_name"`
	OriginCertificateId string `json:"origin_certificate_id"`
}

type LoadbalancerListenerResourceInfo struct {
	// 负载均衡监听器名称
	Listener string `json:"listener"`

	// 负载均衡ID
	LoadbalancerId string `json:"loadbalancer_id"`

	LoadbalancerResourceInfo
}

type LoadbalancerListenerResourceInput struct {
	// 负载均衡监听器
	ListenerId string `json:"listener_id"`

	// 负载均衡监听器ID
	// swagger:ignore
	// Deprecated
	Listener string `json:"listener" yunion-deprecated-by:"listener_id"`
}

type LoadbalancerListenerFilterListInput struct {
	LoadbalancerFilterListInput

	LoadbalancerListenerResourceInput

	// 以负载均衡监听器名称排序
	OrderByListener string `json:"order_by_listener"`
}
