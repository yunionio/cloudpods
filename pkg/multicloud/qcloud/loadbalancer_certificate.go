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

package qcloud

import (
	"crypto/sha1"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
)

type SLBCertificate struct {
	region *SRegion

	SCertificate
}

func (self *SLBCertificate) GetPublickKey() string {
	return ""
}

func (self *SLBCertificate) GetPrivateKey() string {
	return ""
}

// 证书不能修改
func (self *SLBCertificate) Sync(name, privateKey, publickKey string) error {
	return nil
}

func (self *SLBCertificate) Delete() error {
	return self.region.DeleteCertificate(self.GetId())
}

func (self *SLBCertificate) GetId() string {
	return self.ID
}

func (self *SLBCertificate) GetName() string {
	return self.Alias
}

func (self *SLBCertificate) GetGlobalId() string {
	return self.ID
}

// todo: 貌似目前onecloud没有记录状态
func (self *SLBCertificate) GetStatus() string {
	return strconv.Itoa(self.Status)
}

func (self *SLBCertificate) Refresh() error {
	certs, total, err := self.region.GetCertificates(self.GetId(), true, 0, 0)
	if err != nil {
		return err
	}

	if total != 1 {
		return fmt.Errorf("Expecting 1 certificate, got %d", total)
	}

	cert := SLBCertificate{region: self.region, SCertificate: certs[0]}
	return jsonutils.Update(self, cert)
}

func (self *SLBCertificate) IsEmulated() bool {
	return false
}

func (self *SLBCertificate) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SLBCertificate) GetCommonName() string {
	return self.Domain
}

func (self *SLBCertificate) GetSubjectAlternativeNames() string {
	return strings.Join(self.SubjectAltName, ",")
}

func (self *SLBCertificate) GetFingerprint() string {
	_fp := sha1.Sum([]byte(self.Cert))
	fp := fmt.Sprintf("sha1:% x", _fp)
	return strings.Replace(fp, " ", ":", -1)
}

func (self *SLBCertificate) GetExpireTime() time.Time {
	return self.CERTEndTime
}

func (self *SLBCertificate) GetProjectId() string {
	return self.ProjectID
}
