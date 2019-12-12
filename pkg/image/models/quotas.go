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

package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"

	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	api "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	commonOptions "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/image/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/utils"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SQuotaManager struct {
	quotas.SQuotaBaseManager
}

var (
	ImageQuota               SQuota
	QuotaManager             *SQuotaManager
	QuotaUsageManager        *SQuotaManager
	QuotaPendingUsageManager *SQuotaManager
)

func init() {
	ImageQuota = SQuota{}

	QuotaPendingUsageManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaUsageManager(SQuota{},
			"quota_pending_usage_tbl",
			"quota_pending_usage",
			"quota_pending_usages",
		),
	}
	QuotaUsageManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaUsageManager(SQuota{},
			"quota_usage_tbl",
			"quota_usage",
			"quota_usages",
		),
	}
	QuotaManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaBaseManager(SQuota{}, "quota_tbl", QuotaPendingUsageManager, QuotaUsageManager,
			"image_quota", "image_quotas"),
	}

	quotas.Register(QuotaManager)
}

type SQuota struct {
	quotas.SQuotaBase

	SImageQuotaKeys

	Image int
}

func (self *SQuota) GetKeys() quotas.IQuotaKeys {
	return self.SImageQuotaKeys
}

func (self *SQuota) SetKeys(keys quotas.IQuotaKeys) {
	self.SImageQuotaKeys = keys.(SImageQuotaKeys)
}

func (self *SQuota) FetchSystemQuota() {
	keys := self.SImageQuotaKeys
	base := 0
	switch options.Options.DefaultQuotaValue {
	case commonOptions.DefaultQuotaUnlimit:
		base = -1
	case commonOptions.DefaultQuotaZero:
		base = 0
		if keys.Scope() == rbacutils.ScopeDomain { // domain level quota
			base = 10
		} else if keys.DomainId == identityapi.DEFAULT_DOMAIN_ID && keys.ProjectId == auth.AdminCredential().GetProjectId() {
			base = 1
		}
	case commonOptions.DefaultQuotaDefault:
		base = 1
		if keys.Scope() == rbacutils.ScopeDomain {
			base = 10
		}
	}
	defaultValue := func(def int) int {
		if base < 0 {
			return -1
		} else {
			return def * base
		}
	}
	self.Image = defaultValue(options.Options.DefaultImageQuota)
}

func (self *SQuota) FetchUsage(ctx context.Context) error {
	keys := self.SImageQuotaKeys

	scope := keys.Scope()
	ownerId := keys.OwnerId()

	var isISO tristate.TriState
	if keys.Type == string(api.ImageTypeISO) {
		isISO = tristate.True
	} else if keys.Type == string(api.ImageTypeTemplate) {
		isISO = tristate.False
	} else {
		isISO = tristate.None
	}

	count := ImageManager.count(scope, ownerId, "", isISO, false)
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
	self.Image = self.Image + quotas.NonNegative(squota.Image)
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
	err := quotas.NewOutOfQuotaError()
	sreq := request.(*SQuota)
	squota := quota.(*SQuota)
	if sreq.Image > 0 && self.Image+sreq.Image > squota.Image {
		err.Add("image", squota.Image, self.Image, sreq.Image)
	}
	if err.IsError() {
		return err
	} else {
		return nil
	}
}

func (self *SQuota) ToJSON(prefix string) jsonutils.JSONObject {
	ret := jsonutils.NewDict()
	// if self.Image > 0 {
	ret.Add(jsonutils.NewInt(int64(self.Image)), quotas.KeyName(prefix, "image"))
	// }
	return ret
}

func (manager *SQuotaManager) FetchIdNames(ctx context.Context, idMap map[string]map[string]string) (map[string]map[string]string, error) {
	for field := range idMap {
		switch field {
		case "domain_id":
			fieldIdMap, err := utils.FetchDomainNames(ctx, idMap[field])
			if err != nil {
				return nil, errors.Wrap(err, "utils.FetchDomainNames")
			}
			idMap[field] = fieldIdMap
		case "tenant_id":
			fieldIdMap, err := utils.FetchTenantNames(ctx, idMap[field])
			if err != nil {
				return nil, errors.Wrap(err, "utils.FetchTenantNames")
			}
			idMap[field] = fieldIdMap
		}
	}
	return idMap, nil
}

type SImageQuotaKeys struct {
	quotas.SBaseQuotaKeys

	Type string `width:"16" charset:"ascii" nullable:"false" primary:"true" list:"user"`
}

func (k SImageQuotaKeys) Fields() []string {
	return append(k.SBaseQuotaKeys.Fields(), "type")
}

func (k SImageQuotaKeys) Values() []string {
	return append(k.SBaseQuotaKeys.Values(), k.Type)
}

func (k1 SImageQuotaKeys) Compare(ik quotas.IQuotaKeys) int {
	k2 := ik.(SImageQuotaKeys)
	r := k1.SBaseQuotaKeys.Compare(k2.SBaseQuotaKeys)
	if r != 0 {
		return r
	}
	if k1.Type < k2.Type {
		return -1
	} else if k1.Type > k2.Type {
		return 1
	}
	return 0
}
