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

package hcs

import (
	"crypto/sha1"
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SElbCert struct {
	multicloud.SResourceBase
	multicloud.HcsTags
	region *SRegion

	Certificate  string    `json:"certificate"`
	CreateTime   time.Time `json:"create_time"`
	ExpireTime   time.Time `json:"expire_time"`
	Description  string    `json:"description"`
	Domain       string    `json:"domain"`
	ID           string    `json:"id"`
	AdminStateUp bool      `json:"admin_state_up"`
	TenantID     string    `json:"tenant_id"`
	Name         string    `json:"name"`
	PrivateKey   string    `json:"private_key"`
	Type         string    `json:"type"`
	UpdateTime   time.Time `json:"update_time"`
}

func (self *SElbCert) GetPublickKey() string {
	return self.Certificate
}

func (self *SElbCert) GetPrivateKey() string {
	return self.PrivateKey
}

func (self *SElbCert) GetId() string {
	return self.ID
}

func (self *SElbCert) GetName() string {
	return self.Name
}

func (self *SElbCert) GetGlobalId() string {
	return self.GetId()
}

func (self *SElbCert) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SElbCert) Refresh() error {
	cert, err := self.region.GetLoadBalancerCertificate(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, cert)
}

func (self *SElbCert) IsEmulated() bool {
	return false
}

func (self *SElbCert) GetProjectId() string {
	return ""
}

func (self *SElbCert) Sync(name, privateKey, publickKey string) error {
	params := map[string]interface{}{
		"name":        name,
		"private_key": privateKey,
		"certificate": publickKey,
	}
	return self.region.lbUpdate("lbaas/certificates/"+self.GetId(), params)
}

func (self *SElbCert) Delete() error {
	return self.region.lbDelete("lbaas/certificates/" + self.GetId())
}

func (self *SElbCert) GetCommonName() string {
	return self.Domain
}

func (self *SElbCert) GetSubjectAlternativeNames() string {
	return self.Domain
}

func (self *SElbCert) GetFingerprint() string {
	_fp := sha1.Sum([]byte(self.Certificate))
	fp := fmt.Sprintf("sha1:% x", _fp)
	return strings.Replace(fp, " ", ":", -1)
}

func (self *SElbCert) GetExpireTime() time.Time {
	return self.ExpireTime
}

func (self *SRegion) GetLoadBalancerCertificate(id string) (*SElbCert, error) {
	ret := &SElbCert{region: self}
	err := self.lbGet("lbaas/certificates/"+id, ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// https://support.huaweicloud.com/api-elb/elb_qy_zs_0001.html
func (self *SRegion) CreateLoadBalancerCertificate(cert *cloudprovider.SLoadbalancerCertificate) (*SElbCert, error) {
	params := map[string]interface{}{
		"name":        cert.Name,
		"private_key": cert.PrivateKey,
		"certificate": cert.Certificate,
	}
	ret := &SElbCert{region: self}
	err := self.lbCreate("lbaas/certificates", params, ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SRegion) GetLoadBalancerCertificates() ([]SElbCert, error) {
	ret := []SElbCert{}
	err := self.lbList("lbaas/certificates", url.Values{}, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SRegion) GetILoadBalancerCertificates() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
	ret, err := self.GetLoadBalancerCertificates()
	if err != nil {
		return nil, err
	}

	iret := make([]cloudprovider.ICloudLoadbalancerCertificate, len(ret))
	for i := range ret {
		ret[i].region = self
		iret[i] = &ret[i]
	}
	return iret, nil
}

func (self *SRegion) CreateILoadBalancerCertificate(cert *cloudprovider.SLoadbalancerCertificate) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	ret, err := self.CreateLoadBalancerCertificate(cert)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SRegion) GetILoadBalancerCertificateById(certId string) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	cert, err := self.GetLoadBalancerCertificate(certId)
	if err != nil {
		return nil, err
	}
	return cert, nil
}
