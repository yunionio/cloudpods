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
	"net/url"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/apis"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/errors"
)

type SLoadbalancerCert struct {
	multicloud.SResourceBase
	AzureTags
	region     *SRegion
	Name       string `json:"name"`
	Id         string `json:"id"`
	Properties struct {
		PublicCertData string
		HttpListeners  []struct {
			Id string
		}
	}
}

func (self *SLoadbalancerCert) GetId() string {
	return self.Id
}

func (self *SLoadbalancerCert) GetName() string {
	return self.Name
}

func (self *SLoadbalancerCert) GetGlobalId() string {
	return strings.ToLower(self.GetId())
}

func (self *SLoadbalancerCert) GetStatus() string {
	return apis.STATUS_AVAILABLE
}

func (self *SLoadbalancerCert) GetProjectId() string {
	return getResourceGroup(self.GetId())
}

func (self *SLoadbalancerCert) Delete() error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Delete")
}

func (self *SLoadbalancerCert) ParsePublicKey() (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(self.GetPublickKey()))
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "ParseCertificate")
	}
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
	if len(self.Properties.PublicCertData) > 0 {
		var pk bytes.Buffer
		pk.WriteString("-----BEGIN CERTIFICATE-----\r\n")
		content := bytes.NewBufferString(self.Properties.PublicCertData)
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

func (self *SRegion) GetLoadbalancerCertificates() ([]SLoadbalancerCert, error) {
	params := url.Values{}
	resp, err := self.list_resources("Microsoft.Network/applicationGateways", "2023-09-01", params)
	if err != nil {
		return nil, err
	}
	ret := []SLoadbalancer{}
	err = resp.Unmarshal(&ret, "value")
	if err != nil {
		return nil, err
	}
	result := []SLoadbalancerCert{}
	for i := range ret {
		if ret[i].Location != self.Name {
			continue
		}
		result = append(result, ret[i].Properties.SSLCertificates...)
	}
	return result, nil
}
