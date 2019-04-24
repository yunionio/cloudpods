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

type SLoadbalancerListener struct {
	Name                    string
	LoadbalancerID          string
	ListenerType            string
	ListenerPort            int
	BackendGroupType        string
	BackendGroupID          string
	Scheduler               string
	AccessControlListStatus string
	AccessControlListType   string
	AccessControlListID     string
	EnableHTTP2             bool
	CertificateID           string
	EgressMbps              int
	Description             string
	EstablishedTimeout      int

	HealthCheck         string
	HealthCheckTimeout  int
	HealthCheckDomain   string
	HealthCheckHttpCode string
	HealthCheckURI      string
	HealthCheckInterval int

	HealthCheckRise int
	HealthCheckFail int

	StickySession              string
	StickySessionCookie        string
	StickySessionType          string
	StickySessionCookieTimeout int

	BackendServerPort int
	XForwardedFor     bool
	Gzip              bool

	TLSCipherPolicy string
}

type SLoadbalancerListenerRule struct {
	Name             string
	Domain           string
	Path             string
	BackendGroupID   string
	BackendGroupType string
}
