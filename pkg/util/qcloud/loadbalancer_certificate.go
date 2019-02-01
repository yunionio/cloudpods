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

func (self *SLBCertificate) Sync(name, privateKey, publickKey string) error {
	panic("implement me")
}

func (self *SLBCertificate) Delete() error {
	return self.region.DeleteCertificate(self.GetId())
}

func (self *SLBCertificate) GetId() string {
	return self.ID
}

// todo: ??
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
		return fmt.Errorf("%d Certificate found", total)
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

// todo: ??
func (self *SLBCertificate) GetCommonName() string {
	return self.Domain
}

func (self *SLBCertificate) GetSubjectAlternativeNames() string {
	return strings.Join(self.SubjectAltName, ",")
}

// todo: ??
func (self *SLBCertificate) GetFingerprint() string {
	_fp := sha1.Sum([]byte(self.Cert))
	fp := fmt.Sprintf("sha1:% x", _fp)
	return strings.Replace(fp, " ", ":", -1)
}

func (self *SLBCertificate) GetExpireTime() time.Time {
	return self.CERTEndTime
}
