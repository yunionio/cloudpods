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

package aliyun

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/apis"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SubjectAlternativeNames struct {
	SubjectAlternativeName []string
}

type SLoadbalancerServerCertificate struct {
	multicloud.SResourceBase
	AliyunTags
	region *SRegion

	ServerCertificateId     string                  //	服务器证书ID。
	ServerCertificateName   string                  //	服务器证书名称。
	Fingerprint             string                  //	服务器证书的指纹。
	CreateTime              string                  //	服务器证书上传的时间。
	CreateTimeStamp         uint64                  // 服务器证书上传的时间戳。
	IsAliCloudCertificate   int                     // 是否是阿里云证书。0代表不是阿里云证书。
	AliCloudCertificateName string                  //	阿里云证书名称。
	AliCloudCertificateId   string                  //	阿里云证书ID。
	ExpireTime              time.Time               //	过期时间。
	ExpireTimeStamp         uint64                  //	过期时间戳。
	CommonName              string                  //	域名，对应证书的CommonName字段。
	SubjectAlternativeNames SubjectAlternativeNames // 数组格式，返回证书的备用域名列表，对应证书的Subject Alternative Name字段，详情请参见SubjectAlternativeNames。
	ResourceGroupId         string                  //	实例的企业资源组ID
	RegionId                string                  //	负载均衡实例的地域。
}

func (certificate *SLoadbalancerServerCertificate) GetPublickKey() string {
	return ""
}

func (certificate *SLoadbalancerServerCertificate) GetPrivateKey() string {
	return ""
}

func (certificate *SLoadbalancerServerCertificate) GetName() string {
	return certificate.ServerCertificateName
}

func (certificate *SLoadbalancerServerCertificate) GetId() string {
	return certificate.ServerCertificateId
}

func (certificate *SLoadbalancerServerCertificate) GetGlobalId() string {
	return certificate.GetId()
}

func (certificate *SLoadbalancerServerCertificate) GetStatus() string {
	return apis.STATUS_AVAILABLE
}

func (certificate *SLoadbalancerServerCertificate) GetCommonName() string {
	return certificate.CommonName
}

func (certificate *SLoadbalancerServerCertificate) GetSubjectAlternativeNames() string {
	return strings.Join(certificate.SubjectAlternativeNames.SubjectAlternativeName, ",")
}

func (certificate *SLoadbalancerServerCertificate) GetFingerprint() string {
	return fmt.Sprintf("sha1:%s", strings.Replace(certificate.Fingerprint, ":", "", -1))
}

func (certificate *SLoadbalancerServerCertificate) GetExpireTime() time.Time {
	return certificate.ExpireTime
}

func (certificate *SLoadbalancerServerCertificate) Refresh() error {
	return nil
}

func (region *SRegion) UpdateServerCertificateName(certId, name string) error {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["ServerCertificateId"] = certId
	params["ServerCertificateName"] = name
	_, err := region.lbRequest("SetServerCertificateName", params)
	return err
}

func (certificate *SLoadbalancerServerCertificate) Delete() error {
	return certificate.region.DeleteServerCertificate(certificate.ServerCertificateId)
}

func (region *SRegion) DeleteServerCertificate(certId string) error {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["ServerCertificateId"] = certId
	_, err := region.lbRequest("DeleteServerCertificate", params)
	return err
}

func (region *SRegion) GetLoadbalancerServerCertificates() ([]SLoadbalancerServerCertificate, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	body, err := region.lbRequest("DescribeServerCertificates", params)
	if err != nil {
		return nil, err
	}
	certificates := []SLoadbalancerServerCertificate{}
	return certificates, body.Unmarshal(&certificates, "ServerCertificates", "ServerCertificate")
}

func (certificate *SLoadbalancerServerCertificate) GetProjectId() string {
	return certificate.ResourceGroupId
}
