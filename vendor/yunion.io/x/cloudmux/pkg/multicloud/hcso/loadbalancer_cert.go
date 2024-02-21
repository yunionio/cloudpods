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

package hcso

import (
	"crypto/sha1"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/cloudmux/pkg/apis"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei"
)

type SElbCert struct {
	multicloud.SResourceBase
	huawei.HuaweiTags
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
	return apis.STATUS_AVAILABLE
}

func (self *SElbCert) Refresh() error {
	cert, err := self.region.GetLoadBalancerCertificateById(self.GetId())
	if err != nil {
		return err
	}

	cert.region = self.region
	err = jsonutils.Update(self, cert)
	if err != nil {
		return err
	}

	return nil
}

func (self *SElbCert) GetProjectId() string {
	return ""
}

func (self *SElbCert) Delete() error {
	return DoDelete(self.region.ecsClient.ElbCertificates.Delete, self.GetId(), nil, nil)
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
