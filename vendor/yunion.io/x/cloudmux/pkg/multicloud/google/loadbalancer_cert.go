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

package google

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/apis"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SLoadbalancerCertificate struct {
	region *SRegion
	SResourceBase
	cert *x509.Certificate

	ID                string      `json:"id"`
	CreationTimestamp string      `json:"creationTimestamp"`
	Certificate       string      `json:"certificate"`
	SelfManaged       SelfManaged `json:"selfManaged"`
	Type              string      `json:"type"`
	ExpireTime        time.Time   `json:"expireTime"`
	Region            string      `json:"region"`
	Kind              string      `json:"kind"`
}

type SelfManaged struct {
	Certificate string `json:"certificate"`
}

func (self *SLoadbalancerCertificate) GetStatus() string {
	return apis.STATUS_AVAILABLE
}

func (self *SLoadbalancerCertificate) Refresh() error {
	return nil
}

func (self *SLoadbalancerCertificate) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SLoadbalancerCertificate) GetSysTags() map[string]string {
	return nil
}

func (self *SLoadbalancerCertificate) GetTags() (map[string]string, error) {
	return nil, nil
}

func (self *SLoadbalancerCertificate) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadbalancerCertificate) GetProjectId() string {
	return self.region.GetProjectId()
}

func (self *SLoadbalancerCertificate) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadbalancerCertificate) GetCommonName() string {
	c := self.getCert()
	if c == nil {
		return ""
	}
	return c.Subject.CommonName
}

func (self *SLoadbalancerCertificate) GetSubjectAlternativeNames() string {
	c := self.getCert()
	if c == nil {
		return ""
	}

	names := []string{}
	for i := range c.Extensions {
		names = append(names, string(c.Extensions[i].Value))
	}

	return strings.Join(names, ",")
}

func (self *SLoadbalancerCertificate) getCert() *x509.Certificate {
	if self.cert != nil {
		return self.cert
	}

	p, _ := pem.Decode([]byte(self.Certificate))
	c, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		log.Errorf("get certificate %s(%s): %s", self.Name, self.GetId(), err)
		return nil
	}

	self.cert = c
	return c
}

func (self *SLoadbalancerCertificate) GetFingerprint() string {
	c := self.getCert()
	if c == nil {
		return ""
	}
	d := sha256.Sum256(c.Raw)
	return api.LB_TLS_CERT_FINGERPRINT_ALGO_SHA256 + ":" + hex.EncodeToString(d[:])
}

func (self *SLoadbalancerCertificate) GetExpireTime() time.Time {
	return self.ExpireTime
}

func (self *SLoadbalancerCertificate) GetPublickKey() string {
	return ""
}

func (self *SLoadbalancerCertificate) GetPrivateKey() string {
	return ""
}

func (self *SRegion) GetILoadBalancerCertificates() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
	certs, err := self.GetRegionalSslCertificates("")
	if err != nil {
		return nil, errors.Wrap(err, "GetRegionalSslCertificates")
	}

	icerts := make([]cloudprovider.ICloudLoadbalancerCertificate, len(certs))
	for i := range certs {
		icerts[i] = &certs[i]
	}

	return icerts, nil
}

func (self *SRegion) GetILoadBalancerCertificateById(certId string) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	ret := SLoadbalancerCertificate{}
	err := self.GetBySelfId(certId, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "Get")
	}
	ret.region = self
	return &ret, nil
}
