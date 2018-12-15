package aliyun

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
)

type SubjectAlternativeNames struct {
	SubjectAlternativeName []string
}

type SLoadbalancerServerCertificate struct {
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
	return ""
}

func (certificate *SLoadbalancerServerCertificate) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (certificate *SLoadbalancerServerCertificate) IsEmulated() bool {
	return false
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
