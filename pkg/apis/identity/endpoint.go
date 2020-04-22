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

package identity

import "yunion.io/x/onecloud/pkg/apis"

type EndpointDetails struct {
	apis.StandaloneResourceDetails
	SEndpoint
	CertificateDetails

	// 服务名称,例如keystone, glance, region等
	ServiceName string `json:"service_name"`

	// 服务类型,例如identity, image, compute等
	ServiceType string `json:"service_type"`
}

type CertificateDetails struct {
	apis.SCertificateResourceBase
	CertName string `json:"cert_name"`
	CertId   string `json:"cert_id"`

	CaCertificate string `json:"ca_certificate"`
	CaPrivateKey  string `json:"ca_private_key"`
}
