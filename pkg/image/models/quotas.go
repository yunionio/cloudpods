package models

import (
	"context"
	"errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/tristate"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/image/options"
)

var QuotaManager *quotas.SQuotaManager

func init() {
	dbStore := quotas.NewDBQuotaStore()
	pendingStore := quotas.NewMemoryQuotaStore()

	QuotaManager = quotas.NewQuotaManager("quotas", SQuota{}, dbStore, pendingStore)
}

var (
	ErrOutOfImage = errors.New("out of image quota")
)

type SQuota struct {
	Image int
}

func (self *SQuota) FetchSystemQuota() {
	self.Image = options.Options.DefaultImageQuota
}

func (self *SQuota) FetchUsage(ctx context.Context, projectId string) error {
	count := ImageManager.count(projectId, "", tristate.None, false)
	self.Image = int(count["total"].Count)
	return nil
}

func (self *SQuota) IsEmpty() bool {
	if self.Image > 0 {
		return false
	}
	return true
}

func (self *SQuota) Add(quota quotas.IQuota) {
	squota := quota.(*SQuota)
	self.Image = self.Image + squota.Image
}

func (self *SQuota) Sub(quota quotas.IQuota) {
	squota := quota.(*SQuota)
	self.Image = quotas.NonNegative(self.Image - squota.Image)
}

func (self *SQuota) Update(quota quotas.IQuota) {
	squota := quota.(*SQuota)
	if squota.Image > 0 {
		self.Image = squota.Image
	}
}

func (self *SQuota) Exceed(request quotas.IQuota, quota quotas.IQuota) error {
	sreq := request.(*SQuota)
	squota := quota.(*SQuota)
	if sreq.Image > 0 && self.Image > squota.Image {
		return ErrOutOfImage
	}
	return nil
}

func (self *SQuota) ToJSON(prefix string) jsonutils.JSONObject {
	ret := jsonutils.NewDict()
	if self.Image > 0 {
		ret.Add(jsonutils.NewInt(int64(self.Image)), quotas.KeyName(prefix, "image"))
	}
	return ret
}
