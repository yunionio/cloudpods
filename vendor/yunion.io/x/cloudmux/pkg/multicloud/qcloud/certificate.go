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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

var CERT_STATUS_MAP = map[int]string{
	0: "pending",
	1: "normal",
	2: "deleted",
	3: "expired",
	4: "normal",
	5: "pending",
	6: "deleted",
	7: "deleted",
	8: "pending",
}

type projectInfo struct {
	ProjectID  string `json:"projectId"`
	OwnerUin   int64  `json:"ownerUin"`
	Name       string `json:"name"`
	CreatorUin int64  `json:"creatorUin"`
	CreateTime string `json:"createTime"`
	Info       string `json:"info"`
}

// https://cloud.tencent.com/document/api/400/13675
type SCertificate struct {
	multicloud.SResourceBase
	QcloudTags
	client *SQcloudClient

	CertificateID       string      `json:"CertificateId"`
	CertificateType     string      `json:"CertificateType"`
	Deployable          bool        `json:"Deployable"`
	RenewAble           bool        `json:"RenewAble"`
	OwnerUin            string      `json:"ownerUin"`
	ProjectID           string      `json:"projectId"`
	From                string      `json:"from"`
	ProductZhName       string      `json:"productZhName"`
	Domain              string      `json:"domain"`
	Alias               string      `json:"alias"`
	Status              int         `json:"status"`
	VulnerabilityStatus string      `json:"vulnerability_status"`
	CERTBeginTime       time.Time   `json:"certBeginTime"`
	CERTEndTime         time.Time   `json:"certEndTime"`
	ValidityPeriod      string      `json:"validityPeriod"`
	InsertTime          string      `json:"insertTime"`
	ProjectInfo         projectInfo `json:"projectInfo"`
	StatusName          string      `json:"status_name"`
	IsVip               bool        `json:"is_vip"`
	IsDv                bool        `json:"is_dv"`
	IsWildcard          bool        `json:"is_wildcard"`
	IsVulnerability     bool        `json:"is_vulnerability"`

	// certificate details
	detailsInitd          bool     `json:"details_initd"`
	SubjectAltName        []string `json:"subjectAltName"`
	CertificatePrivateKey string   `json:"CertificatePrivateKey"`
	CertificatePublicKey  string   `json:"CertificatePublicKey"`
	CertFingerprint       string   `json:"CertFingerprint"`
}

func (self *SCertificate) GetDetails() (*SCertificate, error) {
	if !self.detailsInitd {
		var (
			cert *SCertificate
			err  error
		)
		cert, err = self.client.GetCertificate(self.GetId())
		if err != nil {
			return nil, err
		}
		self.detailsInitd = true
		self.SubjectAltName = cert.SubjectAltName
		self.CertificatePrivateKey = cert.CertificatePrivateKey
		self.CertificatePublicKey = cert.CertificatePublicKey
		self.CertFingerprint = cert.CertFingerprint
	}
	return self, nil
}

func (self *SCertificate) GetPublickKey() string {
	self.GetDetails()
	return self.CertificatePublicKey
}

func (self *SCertificate) GetPrivateKey() string {
	self.GetDetails()
	return self.CertificatePrivateKey
}

func (self *SCertificate) GetCert() string {
	self.GetDetails()
	return self.CertificatePublicKey
}

func (self *SCertificate) GetKey() string {
	self.GetDetails()
	return self.CertificatePrivateKey
}

// 证书不能修改
func (self *SCertificate) Sync(name, privateKey, publickKey string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SCertificate) Delete() error {
	return self.client.DeleteCertificate(self.GetId())
}

func (self *SCertificate) GetId() string {
	return self.CertificateID
}

func (self *SCertificate) GetName() string {
	if len(self.Alias) > 0 {
		return self.Alias
	} else {
		return self.Domain
	}
}

func (self *SCertificate) GetGlobalId() string {
	return self.CertificateID
}

// todo: 貌似目前onecloud没有记录状态
func (self *SCertificate) GetStatus() string {
	if _, ok := CERT_STATUS_MAP[self.Status]; !ok {
		return "unknown"
	}
	return CERT_STATUS_MAP[self.Status]
}

func (self *SCertificate) Refresh() error {
	cert, err := self.client.GetCertificate(self.GetId())
	if err != nil {
		return errors.Wrap(err, "GetCertificate")
	}

	return jsonutils.Update(self, cert)
}

func (self *SCertificate) IsEmulated() bool {
	return false
}

func (self *SCertificate) GetCommonName() string {
	return self.Domain
}

func (self *SCertificate) GetSubjectAlternativeNames() string {
	self.GetDetails()
	return self.CertFingerprint
}

func (self *SCertificate) GetFingerprint() string {
	self.GetDetails()
	_fp := sha1.Sum([]byte(self.CertificatePublicKey))
	fp := fmt.Sprintf("sha1:% x", _fp)
	return strings.Replace(fp, " ", ":", -1)
}

func (self *SCertificate) GetExpireTime() time.Time {
	return self.CERTEndTime
}

func (self *SCertificate) GetProjectId() string {
	return self.ProjectID
}

func (self *SCertificate) GetSans() string {
	return strings.Join(self.SubjectAltName, ",")
}

func (self *SCertificate) GetStartDate() time.Time {
	return self.CERTBeginTime
}

func (self *SCertificate) GetProvince() string {
	return ""
}

func (self *SCertificate) GetCommon() string {
	return self.Domain
}

func (self *SCertificate) GetCountry() string {
	return ""
}

func (self *SCertificate) GetIssuer() string {
	return self.ProductZhName
}

func (self *SCertificate) GetExpired() bool {
	if self.Status == 3 {
		return true
	} else {
		return false
	}
}

func (self *SCertificate) GetEndDate() time.Time {
	return self.CERTEndTime
}

func (self *SCertificate) GetCity() string {
	return ""
}

func (self *SCertificate) GetOrgName() string {
	return ""
}

func (self *SCertificate) GetIsUpload() bool {
	if self.From == "upload" {
		return true
	}
	return false
}

// https://cloud.tencent.com/document/product/400/41674
func (self *SQcloudClient) GetCertificate(certId string) (*SCertificate, error) {
	params := map[string]string{
		"CertificateId": certId,
	}

	resp, err := self.sslRequest("DescribeCertificateDetail", params)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeCertificateDetail")
	}

	cert := &SCertificate{client: self}
	err = resp.Unmarshal(cert)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}

	return cert, nil
}

// https://cloud.tencent.com/document/product/400/41665
// 返回证书ID
func (self *SQcloudClient) CreateCertificate(projectId, publicKey, privateKey, certType, desc string) (string, error) {
	params := map[string]string{
		"CertificatePublicKey": publicKey,
		"CertificateType":      certType,
		"Alias":                desc,
	}

	if len(privateKey) > 0 {
		params["CertificatePrivateKey"] = privateKey
	} else {
		if certType == "SVR" {
			return "", fmt.Errorf("certificate private key required while certificate type is SVR")
		}
	}

	if len(projectId) > 0 {
		params["ProjectId"] = projectId
	}

	resp, err := self.sslRequest("UploadCertificate", params)
	if err != nil {
		return "", err
	}

	return resp.GetString("CertificateId")
}

// https://cloud.tencent.com/document/product/400/41675
func (self *SQcloudClient) DeleteCertificate(id string) error {
	if len(id) == 0 {
		return fmt.Errorf("DelteCertificate certificate id should not be empty")
	}

	params := map[string]string{"CertificateId": id}
	resp, err := self.sslRequest("DeleteCertificate", params)
	if err != nil {
		return errors.Wrap(err, "DeleteCertificate")
	}

	if deleted, _ := resp.Bool("DeleteResult"); deleted {
		return nil
	}
	return fmt.Errorf("DeleteCertificate %s", resp)
}

func (self *SRegion) GetILoadBalancerCertificates() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
	certs, err := self.client.GetCertificates("", "", "")
	if err != nil {
		return nil, errors.Wrap(err, "GetCertificates")
	}

	icerts := make([]cloudprovider.ICloudLoadbalancerCertificate, len(certs))
	for i := range certs {
		certs[i].client = self.client
		icerts[i] = &certs[i]
	}

	return icerts, nil
}

func (self *SRegion) GetILoadBalancerCertificateById(certId string) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	cert, err := self.client.GetCertificate(certId)
	if err != nil {
		return nil, errors.Wrap(err, "GetCertificate")
	}

	return cert, nil
}

// todo:目前onecloud端只能指定服务器端证书。需要兼容客户端证书？
// todo:支持指定Project。
// todo: 已过期的证书不能上传也不能关联资源
func (self *SRegion) CreateILoadBalancerCertificate(input *cloudprovider.SLoadbalancerCertificate) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	certId, err := self.client.CreateCertificate("", input.Certificate, input.PrivateKey, "SVR", input.Name)
	if err != nil {
		return nil, err
	}

	cert, err := self.client.GetCertificate(certId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetCertificate")
	}
	return cert, nil
}

func (self *SQcloudClient) GetCertificates(projectId, certificateStatus, searchKey string) ([]SCertificate, error) {
	params := map[string]string{}
	params["Limit"] = "100"
	if len(projectId) > 0 {
		params["ProjectId"] = projectId
	}

	if len(certificateStatus) > 0 {
		params["CertificateStatus.0"] = certificateStatus
	}

	if len(searchKey) > 0 {
		params["SearchKey"] = searchKey
	}

	certs := []SCertificate{}
	offset := 0
	total := 100
	for total > offset {
		params["Offset"] = strconv.Itoa(offset)
		resp, err := self.sslRequest("DescribeCertificates", params)
		if err != nil {
			return nil, errors.Wrap(err, "DescribeCertificates")
		}

		_certs := []SCertificate{}
		err = resp.Unmarshal(&certs, "Certificates")
		if err != nil {
			return nil, errors.Wrap(err, "Unmarshal.Certificates")
		}

		err = resp.Unmarshal(&total, "TotalCount")
		if err != nil {
			return nil, errors.Wrap(err, "Unmarshal.TotalCount")
		}

		for i := range _certs {
			_certs[i].client = self
			certs = append(certs, _certs[i])
		}

		offset += 100
	}

	return certs, nil
}
