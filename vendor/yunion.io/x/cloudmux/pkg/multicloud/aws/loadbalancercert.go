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

	"yunion.io/x/cloudmux/pkg/apis"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SElbCertificate struct {
	multicloud.SResourceBase
	AwsTags
	region *SRegion
	cert   *x509.Certificate

	Path                  string    `json:"Path"`
	ServerCertificateName string    `json:"ServerCertificateName"`
	ServerCertificateID   string    `json:"ServerCertificateId"`
	Arn                   string    `json:"Arn"`
	UploadDate            time.Time `json:"UploadDate"`
	Expiration            time.Time `json:"Expiration"`
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
	return apis.STATUS_AVAILABLE
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

func (self *SElbCertificate) GetProjectId() string {
	return ""
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
	return ret.ServerCertificate.CertificateBody, self.client.iamRequest("GetServerCertificate", params, &ret)
}

func (self *SRegion) deleteElbCertificate(certName string) error {
	params := map[string]string{
		"ServerCertificateName": certName,
	}
	ret := struct{}{}
	return self.client.iamRequest("DeleteServerCertificate", params, &ret)
}

func (self *SRegion) CreateLoadbalancerCertifacate(opts *cloudprovider.SLoadbalancerCertificate) (string, error) {
	params := map[string]string{
		"ServerCertificateName": opts.Name,
		"PrivateKey":            opts.PrivateKey,
		"CertificateBody":       opts.Certificate,
	}
	ret := struct {
		ServerCertificateMetadata struct {
			Arn string `xml:"Arn"`
		} `xml:"ServerCertificateMetadata"`
	}{}

	err := self.client.iamRequest("UploadServerCertificate", params, &ret)
	if err != nil {
		return "", errors.Wrapf(err, "UploadServerCertificate")
	}
	return ret.ServerCertificateMetadata.Arn, nil
}

func (self *SRegion) CreateILoadBalancerCertificate(opts *cloudprovider.SLoadbalancerCertificate) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	arn, err := self.CreateLoadbalancerCertifacate(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateLoadbalancerCertifacate")
	}

	// wait upload cert success
	err = cloudprovider.Wait(5*time.Second, 30*time.Second, func() (bool, error) {
		_, err := self.GetILoadBalancerCertificateById(arn)
		if err == nil {
			return true, nil
		}

		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return false, nil
		} else {
			return false, err
		}
	})
	if err != nil {
		return nil, errors.Wrap(err, "region.CreateILoadBalancerCertificate.Wait")
	}
	return self.GetILoadBalancerCertificateById(arn)
}

func (self *SRegion) ListServerCertificates() ([]SElbCertificate, error) {
	params := map[string]string{}
	ret := []SElbCertificate{}
	for {
		part := struct {
			IsTruncated                   bool              `xml:"IsTruncated"`
			ServerCertificateMetadataList []SElbCertificate `xml:"ServerCertificateMetadataList>member"`
			Marker                        string            `xml:"Marker"`
		}{}
		err := self.client.iamRequest("ListServerCertificates", params, &part)
		if err != nil {
			return nil, errors.Wrapf(err, "ListServerCertificates")
		}
		ret = append(ret, part.ServerCertificateMetadataList...)
		if len(part.Marker) == 0 || len(part.ServerCertificateMetadataList) == 0 {
			break
		}
		params["Marker"] = part.Marker
	}
	return ret, nil
}

func (self *SElbCertificate) GetDescription() string {
	return self.AwsTags.GetDescription()
}
