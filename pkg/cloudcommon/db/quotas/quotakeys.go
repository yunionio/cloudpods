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

package quotas

import (
	"strings"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SBaseQuotaKeys struct {
	DomainId  string `width:"64" charset:"ascii" nullable:"false" primary:"true" list:"user"`
	ProjectId string `name:"tenant_id" width:"64" charset:"ascii" nullable:"false" primary:"true" list:"user"`
}

type SCloudResourceKeys struct {
	SBaseQuotaKeys
	// provider
	Provider string `width:"32" charset:"ascii" nullable:"false" primary:"true" list:"user"`
	// brand
	Brand string `width:"32" charset:"ascii" nullable:"false" primary:"true" list:"user"`
	// Env
	CloudEnv string `width:"32" charset:"ascii" nullable:"false" primary:"true" list:"user"`
	// cloudaccount
	AccountId string `width:"64" charset:"ascii" nullable:"false" primary:"true" list:"user"`
	// cloudprovider
	ManagerId string `width:"64" charset:"ascii" nullable:"false" primary:"true" list:"user"`
}

type SRegionalCloudResourceKeys struct {
	SCloudResourceKeys
	// region
	RegionId string `width:"64" charset:"ascii" nullable:"false" primary:"true" list:"user"`
}

type SZonalCloudResourceKeys struct {
	SRegionalCloudResourceKeys
	// zone
	ZoneId string `width:"64" charset:"ascii" nullable:"false" primary:"true" list:"user"`
}

func (k SBaseQuotaKeys) Fields() []string {
	return []string{
		"domain_id",
		"tenant_id",
	}
}

func (k SCloudResourceKeys) Fields() []string {
	return append(k.SBaseQuotaKeys.Fields(),
		"provider",
		"brand",
		"cloud_env",
		"account_id",
		"manager_id",
	)
}

func (k SRegionalCloudResourceKeys) Fields() []string {
	return append(k.SCloudResourceKeys.Fields(),
		"region_id",
	)
}

func (k SZonalCloudResourceKeys) Fields() []string {
	return append(k.SRegionalCloudResourceKeys.Fields(),
		"zone_id",
	)
}

func (k SBaseQuotaKeys) Values() []string {
	return []string{
		k.DomainId,
		k.ProjectId,
	}
}

func (k SCloudResourceKeys) Values() []string {
	return append(k.SBaseQuotaKeys.Values(),
		k.Provider,
		k.Brand,
		k.CloudEnv,
		k.AccountId,
		k.ManagerId,
	)
}

func (k SRegionalCloudResourceKeys) Values() []string {
	return append(k.SCloudResourceKeys.Values(),
		k.RegionId,
	)
}

func (k SZonalCloudResourceKeys) Values() []string {
	return append(k.SRegionalCloudResourceKeys.Values(),
		k.ZoneId,
	)
}

func (k1 SBaseQuotaKeys) Compare(ik IQuotaKeys) int {
	k2 := ik.(SBaseQuotaKeys)
	if k1.DomainId < k2.DomainId {
		return -1
	} else if k1.DomainId > k2.DomainId {
		return 1
	}
	if k1.ProjectId < k2.ProjectId {
		return -1
	} else if k1.ProjectId > k2.ProjectId {
		return 1
	}
	return 0
}

func (k1 SCloudResourceKeys) Compare(ik IQuotaKeys) int {
	k2 := ik.(SCloudResourceKeys)
	r := k1.SBaseQuotaKeys.Compare(k2.SBaseQuotaKeys)
	if r != 0 {
		return r
	}
	if k1.CloudEnv < k2.CloudEnv {
		return -1
	} else if k1.CloudEnv > k2.CloudEnv {
		return 1
	}
	if k1.Provider < k2.Provider {
		return -1
	} else if k1.Provider > k2.Provider {
		return 1
	}
	if k1.Brand < k2.Brand {
		return -1
	} else if k1.Brand > k2.Brand {
		return 1
	}
	return 0
}

func (k1 SRegionalCloudResourceKeys) Compare(ik IQuotaKeys) int {
	k2 := ik.(SRegionalCloudResourceKeys)
	r := k1.SCloudResourceKeys.Compare(k2.SCloudResourceKeys)
	if r != 0 {
		return r
	}
	if k1.RegionId < k2.RegionId {
		return -1
	} else if k1.RegionId > k2.RegionId {
		return 1
	}
	return 0
}

func (k1 SZonalCloudResourceKeys) Compare(ik IQuotaKeys) int {
	k2 := ik.(SZonalCloudResourceKeys)
	r := k1.SRegionalCloudResourceKeys.Compare(k2.SRegionalCloudResourceKeys)
	if r != 0 {
		return r
	}
	if k1.ZoneId < k2.ZoneId {
		return -1
	} else if k1.ZoneId > k2.ZoneId {
		return 1
	}
	return 0
}

func QuotaKeyWeight(k IQuotaKeys) uint64 {
	w := uint64(0)
	for i, v := range k.Values() {
		if len(v) > 0 {
			w += (uint64(1) << uint(i))
		}
	}
	return w
}

func (k SBaseQuotaKeys) Scope() rbacutils.TRbacScope {
	if len(k.DomainId) > 0 && len(k.ProjectId) > 0 {
		return rbacutils.ScopeProject
	} else if len(k.DomainId) > 0 && len(k.ProjectId) == 0 {
		return rbacutils.ScopeDomain
	} else if len(k.DomainId) == 0 && len(k.ProjectId) == 0 {
		return rbacutils.ScopeSystem
	} else {
		return rbacutils.ScopeNone
	}
}

func (k SBaseQuotaKeys) OwnerId() mcclient.IIdentityProvider {
	return &db.SOwnerId{
		DomainId:  k.DomainId,
		ProjectId: k.ProjectId,
	}
}

func QuotaKeyString(k IQuotaKeys) string {
	return strings.Join(k.Values(), "-")
}

func OwnerIdQuotaKeys(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider) SBaseQuotaKeys {
	switch scope {
	case rbacutils.ScopeDomain:
		return SBaseQuotaKeys{
			DomainId: ownerId.GetProjectDomainId(),
		}
	case rbacutils.ScopeProject:
		return SBaseQuotaKeys{
			DomainId:  ownerId.GetProjectDomainId(),
			ProjectId: ownerId.GetProjectId(),
		}
	}
	return SBaseQuotaKeys{}
}

type TQuotaKeysRelation string

const (
	QuotaKeysContain = TQuotaKeysRelation("contain")
	QuotaKeysBelong  = TQuotaKeysRelation("belong")
	QuotaKeysEqual   = TQuotaKeysRelation("equal")
	QuotaKeysExclude = TQuotaKeysRelation("exclude")
)

func stringRelation(s1, s2 string) TQuotaKeysRelation {
	if s1 == s2 {
		return QuotaKeysEqual
	} else if len(s1) == 0 {
		return QuotaKeysContain
	} else if len(s2) == 0 {
		return QuotaKeysBelong
	} else {
		return QuotaKeysExclude
	}
}

func relation(k1, k2 IQuotaKeys) TQuotaKeysRelation {
	a1 := k1.Values()
	a2 := k2.Values()
	relationMap := make(map[TQuotaKeysRelation]int, 0)
	for i := 0; i < len(a1); i += 1 {
		rel := stringRelation(a1[i], a2[i])
		if _, ok := relationMap[rel]; ok {
			relationMap[rel] = relationMap[rel] + 1
		} else {
			relationMap[rel] = 1
		}
	}
	switch len(relationMap) {
	case 1:
		for k := range relationMap {
			return k
		}
	case 2:
		_, equalExist := relationMap[QuotaKeysEqual]
		if equalExist {
			if _, ok := relationMap[QuotaKeysContain]; ok {
				return QuotaKeysContain
			}
			if _, ok := relationMap[QuotaKeysBelong]; ok {
				return QuotaKeysBelong
			}
			return QuotaKeysExclude
		}
	}
	return QuotaKeysExclude
}

type TQuotaList []IQuota

func (a TQuotaList) Len() int      { return len(a) }
func (a TQuotaList) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a TQuotaList) Less(i, j int) bool {
	iKeys := a[i].GetKeys()
	jKeys := a[j].GetKeys()
	relation := relation(iKeys, jKeys)
	switch relation {
	case QuotaKeysContain:
		return true
	case QuotaKeysBelong:
		return false
	}
	iw := QuotaKeyWeight(iKeys)
	jw := QuotaKeyWeight(jKeys)
	if iw < jw {
		return true
	} else if iw > jw {
		return false
	}
	return iKeys.Compare(jKeys) < 0
}
