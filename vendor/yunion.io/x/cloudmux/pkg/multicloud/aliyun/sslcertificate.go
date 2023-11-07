package aliyun

import (
	"fmt"
	"strconv"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/errors"
)

type SSSLCertificate struct {
	multicloud.SVirtualResourceBase
	AliyunTags
	region *SRegion

	Sans        string    // 证书的SAN（Subject Alternative Name）扩展属性，表示证书关联的其他域名、IP地址等
	Id          int       // 证书ID
	StartDate   time.Time // 证书签发日期
	Province    string    // 购买证书的用户所属的公司或组织所在的省
	Common      string    // 证书绑定的主域名
	Country     string    // 购买证书的用户所属的公司或组织所在的国家或地区
	Issuer      string    // 证书颁发机构
	BuyInAliyun bool      // 是否在阿里云购买了证书
	Expired     bool      // 证书是否过期
	EndDate     time.Time // 证书到期日期
	Name        string    // 证书名称
	Fingerprint string    // 证书名称
	City        string    // 购买证书的用户所属的公司或组织所在的城市
	OrgName     string    // 购买证书的用户所属的公司或组织的名称

	// certificate details
	detailsInitd bool
	Cert         string `json:"Cert"` // 证书内容
	Key          string `json:"Key"`  // 证书私钥
}

func (s *SSSLCertificate) GetSans() string {
	return s.Sans
}

func (s *SSSLCertificate) GetStartDate() time.Time {
	return s.StartDate
}

func (s *SSSLCertificate) GetProvince() string {
	return s.Province
}

func (s *SSSLCertificate) GetCommon() string {
	return s.Common
}

func (s *SSSLCertificate) GetCountry() string {
	return s.Country
}

func (s *SSSLCertificate) GetIssuer() string {
	return s.Issuer
}

func (s *SSSLCertificate) GetExpired() bool {
	return s.Expired
}

func (s *SSSLCertificate) GetEndDate() time.Time {
	return s.EndDate
}

func (s *SSSLCertificate) GetFingerprint() string {
	return s.Fingerprint
}

func (s *SSSLCertificate) GetCity() string {
	return s.City
}

func (s *SSSLCertificate) GetOrgName() string {
	return s.OrgName
}

func (s *SSSLCertificate) GetId() string {
	return strconv.Itoa(s.Id)
}

func (s *SSSLCertificate) GetName() string {
	return s.Name
}

func (s *SSSLCertificate) GetGlobalId() string {
	return strconv.Itoa(s.Id)
}

func (s *SSSLCertificate) GetStatus() string {
	if s.Expired {
		return "expired"
	} else {
		return "normal"
	}
}

func (s *SSSLCertificate) GetIsUpload() bool {
	return false
}

func (s *SSSLCertificate) GetCert() string {
	s.GetDetails()
	return s.Cert
}

func (s *SSSLCertificate) GetKey() string {
	s.GetDetails()
	return s.Key
}

func (s *SSSLCertificate) GetDetails() (*SSSLCertificate, error) {
	if !s.detailsInitd {
		cert, err := s.region.GetCertificate(s.GetId())
		if err != nil {
			return nil, err
		}
		s.detailsInitd = true
		s.Cert = cert.Cert
		s.Key = cert.Key
	}
	return s, nil
}

// GetISSLCertificates 获取证书资源列表
func (r *SRegion) GetISSLCertificates() ([]cloudprovider.ICloudSSLCertificate, error) {
	ret := make([]SSSLCertificate, 0)

	for {
		part, total, err := r.GetSSLCertificates(100, len(ret)/100+1)
		if err != nil {
			return nil, errors.Wrapf(err, "GetSSLCertificates")
		}

		ret = append(ret, part...)
		if len(ret) >= total {
			break
		}
	}

	result := make([]cloudprovider.ICloudSSLCertificate, 0)
	for i := range ret {
		if !ret[i].BuyInAliyun {
			continue
		}
		ret[i].region = r
		result = append(result, &ret[i])
	}
	return result, nil
}

func (r *SRegion) GetSSLCertificates(size, page int) ([]SSSLCertificate, int, error) {
	if size < 1 || size > 100 {
		size = 100
	}
	if page < 1 {
		page = 1
	}

	params := map[string]string{
		"ShowSize":    fmt.Sprintf("%d", size),
		"CurrentPage": fmt.Sprintf("%d", page),
	}
	resp, err := r.scRequest("DescribeUserCertificateList", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeUserCertificateList")
	}

	ret := make([]SSSLCertificate, 0)
	err = resp.Unmarshal(&ret, "CertificateList")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}

	totalCount, _ := resp.Int("TotalCount")
	return ret, int(totalCount), nil
}

func (r *SRegion) GetCertificate(certId string) (*SSSLCertificate, error) {
	params := map[string]string{
		"CertId": fmt.Sprintf("%s", certId),
	}
	resp, err := r.scRequest("DescribeUserCertificateDetail", params)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeUserCertificateDetail")
	}

	cert := &SSSLCertificate{}
	err = resp.Unmarshal(cert)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	cert.region = r

	return cert, nil
}
