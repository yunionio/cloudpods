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
	"time"

	"yunion.io/x/onecloud/pkg/apis"
)

type LoadbalancerCertificateDetails struct {
	apis.SharableVirtualResourceDetails
	SLoadbalancerCertificate
	ManagedResourceInfo
	CloudregionResourceInfo

	LoadbalancerCertificateUsage
}

type LoadbalancerCertificateUsage struct {
	ListenerCount int `json:"lb_listener_count"`
}

type LoadbalancerCertificateResourceInfo struct {
	// 负载均衡证书名称
	Certificate string `json:"certificate"`
}

type LoadbalancerCertificateResourceInput struct {
	// 证书名称或ID
	Certificate string `json:"certificate"`

	// swagger:ignore
	// Deprecated
	CertificateId string `json:"certificate_id" yunion-deprecated-by:"certificate"`
}

type LoadbalancerCertificateFilterListInput struct {
	LoadbalancerCertificateResourceInput

	// 以证书名称排序
	OrderByCertificate string `json:"order_by_certificate"`
}

type LoadbalancerCertificateUpdateInput struct {
	apis.SharableVirtualResourceBaseUpdateInput
}

type LoadbalancerCertificateListInput struct {
	apis.SharableVirtualResourceListInput
	apis.ExternalizedResourceBaseListInput

	UsableResourceListInput
	RegionalFilterListInput
	ManagedResourceListInput

	CommonName              []string `json:"common_name"`
	SubjectAlternativeNames []string `json:"subject_alternative_names"`
}

type LoadbalancerCertificateCreateInput struct {
	apis.SharableVirtualResourceCreateInput

	CloudregionResourceInput
	CloudproviderResourceInput

	Certificate string `json:"certificate"`
	PrivateKey  string `json:"private_key"`
	// swagger: ignore
	Fingerprint string `json:"fingerprint"`
	// swagger: ignore
	PublicKeyAlgorithm string `json:"public_key_algorithm"`
	// swagger: ignore
	PublicKeyBitLen int `json:"public_key_bit_len"`
	// swagger: ignore
	SignatureAlgorithm string `json:"signature_algorithm"`
	// swagger: ignore
	NotBefore time.Time `json:"not_before"`
	// swagger: ignore
	NotAfter time.Time `json:"not_after"`
	// swagger: ignore
	CommonName string `json:"common_name"`
	// swagger: ignore
	SubjectAlternativeNames string `json:"subject_alternative_names"`
}
