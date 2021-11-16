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

package aws

import (
	"crypto/sha1"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SElbCertificate struct {
	multicloud.SResourceBase
	multicloud.AwsTags
	region *SRegion
	cert   *x509.Certificate

	Path                  string    `xml:"Path"`
	ServerCertificateName string    `xml:"ServerCertificateName"`
	ServerCertificateId   string    `xml:"ServerCertificateId"`
	Arn                   string    `xml:"Arn"`
	UploadDate            time.Time `xml:"UploadDate"`
	Expiration            time.Time `xml:"Expiration"`
	PublicKey             string
}

func (self *SElbCertificate) GetId() string {
	return self.Arn
}

func (self *SElbCertificate) GetName() string {
	return self.ServerCertificateName
}

func (self *SElbCertificate) GetGlobalId() string {
	return self.GetId()
}

func (self *SElbCertificate) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SElbCertificate) Refresh() error {
	icert, err := self.region.GetILoadBalancerCertificateById(self.GetId())
	if err != nil {
		return err
	}

	err = jsonutils.Update(self, icert)
	if err != nil {
		return err
	}

	return nil
}

func (self *SElbCertificate) IsEmulated() bool {
	return false
}

func (self *SElbCertificate) GetProjectId() string {
	return ""
}

func (self *SElbCertificate) Sync(name, privateKey, publickKey string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SElbCertificate) Delete() error {
	return self.region.deleteElbCertificate(self.GetName())
}

func (self *SElbCertificate) GetCommonName() string {
	cert, err := self.ParsePublicKey()
	if err != nil {
		return ""
	}

	return cert.Issuer.CommonName
}

func (self *SElbCertificate) GetSubjectAlternativeNames() string {
	// todo: fix me
	_, err := self.ParsePublicKey()
	if err != nil {
		return ""
	}

	return ""
}

func (self *SElbCertificate) GetFingerprint() string {
	publicKey := self.GetPublickKey()
	if len(publicKey) == 0 {
		return ""
	}

	_fp := sha1.Sum([]byte(publicKey))
	fp := fmt.Sprintf("sha1:% x", _fp)
	return strings.Replace(fp, " ", ":", -1)
}

func (self *SElbCertificate) GetExpireTime() time.Time {
	return self.Expiration
}

func (self *SElbCertificate) GetPublickKey() string {
	if self.PublicKey == "" {
		ret, err := self.region.getPublicKey(self.GetName())
		if err != nil {
			log.Errorf("GetPublickKey %s", err)
			return ""
		}

		self.PublicKey = ret
	}

	return self.PublicKey
}

func (self *SElbCertificate) GetPrivateKey() string {
	return ""
}

func (self *SElbCertificate) ParsePublicKey() (*x509.Certificate, error) {
	if self.cert != nil {
		return self.cert, nil
	}

	publicKey := self.GetPublickKey()
	if len(publicKey) == 0 {
		return nil, fmt.Errorf("SElbCertificate ParsePublicKey public key is empty")
	}

	block, _ := pem.Decode([]byte(publicKey))
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "ParseCertificate")
	}

	self.cert = cert
	return cert, nil
}

func (self *SRegion) getPublicKey(certName string) (string, error) {
	params := map[string]string{
		"ServerCertificateName": certName,
	}
	ret := struct {
		ServerCertificate struct {
			CertificateBody string `xml:"CertificateBody"`
		} `xml:"ServerCertificate"`
	}{}
	err := self.client.iamRequest("GetServerCertificate", params, &ret)
	if err != nil {
		return "", errors.Wrapf(err, "GetServerCertificate")
	}
	return ret.ServerCertificate.CertificateBody, nil
}

func (self *SRegion) deleteElbCertificate(certName string) error {
	params := map[string]string{
		"ServerCertificateName": certName,
	}
	return self.client.iamRequest("DeleteServerCertificate", params, nil)
}
