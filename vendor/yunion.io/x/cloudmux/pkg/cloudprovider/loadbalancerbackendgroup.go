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

package cloudprovider

type SLoadbalancerBackendGroup struct {
	Name      string
	GroupType string
	Backends  []SLoadbalancerBackend

	// huawei
	Scheduler string
	Protocol  string

	// aws
	ListenPort int    // 后端端口
	VpcId      string // vpc id
}

type SLoadbalancerHealthCheck struct {
	HealthCheckType string
	HealthCheckReq  string
	HealthCheckExp  string

	HealthCheck         string
	HealthCheckTimeout  int
	HealthCheckDomain   string
	HealthCheckHttpCode string
	HealthCheckURI      string
	HealthCheckInterval int

	HealthCheckRise int
	HealthCheckFail int
}

type SLoadbalancerStickySession struct {
	StickySession              string
	StickySessionCookie        string
	StickySessionType          string
	StickySessionCookieTimeout int
}
