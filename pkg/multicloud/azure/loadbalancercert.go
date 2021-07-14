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

package azure

import (
	"bytes"
	"crypto/sha1"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SLoadbalancerCert struct {
	lb        *SLoadbalancer
	cert      *x509.Certificate
	Name      string `json:"name"`
	ID        string `json:"id"`
	PublicKey string `json:"public_key"`
}

func (self *SLoadbalancerCert) GetId() string {
	return self.ID
}

func (self *SLoadbalancerCert) GetName() string {
	return self.Name
}

func (self *SLoadbalancerCert) GetGlobalId() string {
	return strings.ToLower(self.GetId())
}

func (self *SLoadbalancerCert) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SLoadbalancerCert) Refresh() error {
	cert, err := self.lb.GetILoadBalancerCertificateById(self.GetId())
	if err != nil {
		return errors.Wrap(err, "GetILoadBalancerCertificateById")
	}

	err = jsonutils.Update(self, cert)
	if err != nil {
		return errors.Wrap(err, "Update")
	}

	return nil
}

func (self *SLoadbalancerCert) IsEmulated() bool {
	return false
}

func (self *SLoadbalancerCert) GetSysTags() map[string]string {
	return nil
}

func (self *SLoadbalancerCert) GetTags() (map[string]string, error) {
	return nil, nil
}

func (self *SLoadbalancerCert) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

func (self *SLoadbalancerCert) GetProjectId() string {
	return getResourceGroup(self.GetId())
}

func (self *SLoadbalancerCert) Sync(name, privateKey, publickKey string) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Sync")
}

func (self *SLoadbalancerCert) Delete() error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Delete")
}

func (self *SLoadbalancerCert) ParsePublicKey() (*x509.Certificate, error) {
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

func (self *SLoadbalancerCert) GetCommonName() string {
	cert, err := self.ParsePublicKey()
	if err != nil {
		return ""
	}

	return cert.Issuer.CommonName
}

func (self *SLoadbalancerCert) GetSubjectAlternativeNames() string {
	_, err := self.ParsePublicKey()
	if err != nil {
		return ""
	}

	return ""
}

func (self *SLoadbalancerCert) GetFingerprint() string {
	publicKey := self.GetPublickKey()
	if len(publicKey) == 0 {
		return ""
	}

	_fp := sha1.Sum([]byte(publicKey))
	fp := fmt.Sprintf("sha1:% x", _fp)
	return strings.Replace(fp, " ", ":", -1)
}

func (self *SLoadbalancerCert) GetExpireTime() time.Time {
	cert, err := self.ParsePublicKey()
	if err != nil {
		return time.Time{}
	}

	return cert.NotAfter
}

func (self *SLoadbalancerCert) GetPublickKey() string {
	if len(self.PublicKey) > 0 {
		var pk bytes.Buffer
		pk.WriteString("-----BEGIN CERTIFICATE-----\r\n")
		content := bytes.NewBufferString(self.PublicKey)
		for {
			l := content.Next(64)
			if len(l) == 64 {
				pk.WriteString(fmt.Sprintf("%s\r\n", l))
			} else {
				pk.WriteString(fmt.Sprintf("%s\r\n", l))
				break
			}
		}
		pk.WriteString("-----END CERTIFICATE-----")
		return pk.String()
	}

	return ""
}

func (self *SLoadbalancerCert) GetPrivateKey() string {
	return ""
}
