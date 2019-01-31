package qcloud

import (
	"strconv"
	"time"
)

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
	OwnerUin            string      `json:"ownerUin"`
	ProjectID           string      `json:"projectId"`
	From                string      `json:"from"`
	Type                int         `json:"type"`
	Cert                string      `json:"cert"`
	CERTType            string      `json:"certType"`
	ProductZhName       string      `json:"productZhName"`
	Domain              string      `json:"domain"`
	Alias               string      `json:"alias"`
	Status              int         `json:"status"`
	VulnerabilityStatus string      `json:"vulnerability_status"`
	VerifyType          string      `json:"verifyType"`
	CERTBeginTime       time.Time   `json:"certBeginTime"`
	CERTEndTime         time.Time   `json:"certEndTime"`
	ValidityPeriod      string      `json:"validityPeriod"`
	InsertTime          string      `json:"insertTime"`
	ProjectInfo         projectInfo `json:"projectInfo"`
	ID                  string      `json:"id"` // 证书Id
	SubjectAltName      []string    `json:"subjectAltName"`
	TypeName            string      `json:"type_name"`
	StatusName          string      `json:"status_name"`
	IsVip               bool        `json:"is_vip"`
	IsDv                bool        `json:"is_dv"`
	IsWildcard          bool        `json:"is_wildcard"`
	IsVulnerability     bool        `json:"is_vulnerability"`
}

func (self *SRegion) GetCertificates(id string, withCert bool, limit int, page int) ([]SCertificate, int, error) {
	params := map[string]string{}
	if withCert {
		params["withCert"] = "1"
	}

	if len(id) > 0 {
		params["id"] = id
	}

	if limit > 0 {
		params["count"] = strconv.Itoa(limit)
	}

	if page > 0 {
		params["page"] = strconv.Itoa(page)
	}

	resp, err := self.wssRequest("CertGetList", params)
	if err != nil {
		return nil, 0, err
	}

	certs := []SCertificate{}
	err = resp.Unmarshal(&certs, "list")
	if err != nil {
		return nil, 0, err
	}

	total, err := resp.Float("totalNum")
	if err != nil {
		return nil, 0, err
	}

	return certs, int(total), nil
}
