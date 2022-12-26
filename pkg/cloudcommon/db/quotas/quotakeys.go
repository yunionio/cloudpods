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
	"fmt"
	"strings"

	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SBaseDomainQuotaKeys struct {
	// 配额适用的域ID
	DomainId string `width:"64" charset:"ascii" nullable:"false" primary:"true" list:"user" json:"domain_id"`
}

type SBaseProjectQuotaKeys struct {
	SBaseDomainQuotaKeys

	// 配额适用的项目ID
	ProjectId string `name:"tenant_id" width:"64" charset:"ascii" nullable:"false" primary:"true" list:"user" json:"project_id"`
}

type SBaseDomainQuotaDetailKeys struct {
	// 配额适用的项目的域名称
	Domain string `json:"domain"`
}

type SBaseProjectQuotaDetailKeys struct {
	SBaseDomainQuotaDetailKeys
	// 配额适用的项目名称
	Project string `json:"project"`
}

type SCloudResourceBaseKeys struct {
	// 配额适用的平台名称，参考List接口的平台列表
	Provider string `width:"32" charset:"ascii" nullable:"false" primary:"true" list:"user" json:"provider"`
	// 配额适用的品牌名称，参考List接口的品牌列表
	Brand string `width:"32" charset:"ascii" nullable:"false" primary:"true" list:"user" json:"brand"`
	// 配额适用的云环境，参考List接口的云环境列表
	CloudEnv string `width:"32" charset:"ascii" nullable:"false" primary:"true" list:"user" json:"cloud_env"`
	// 配额适用的云账号ID
	AccountId string `width:"64" charset:"ascii" nullable:"false" primary:"true" list:"user" json:"account_id"`
	// 配额适用的云订阅ID
	ManagerId string `width:"64" charset:"ascii" nullable:"false" primary:"true" list:"user" json:"manager_id"`
}

type SCloudResourceKeys struct {
	SBaseProjectQuotaKeys
	SCloudResourceBaseKeys
}

type SCloudResourceDetailKeys struct {
	SBaseProjectQuotaDetailKeys
	SCloudResourceDetailBaseKeys
}

type SCloudResourceDetailBaseKeys struct {
	// 配额适用的云账号名称
	Account string `json:"account"`
	// 配额适用的云订阅名称
	Manager string `json:"manager"`
}

type SRegionalBaseKeys struct {
	// 配额适用的区域ID
	RegionId string `width:"64" charset:"ascii" nullable:"false" primary:"true" list:"user" json:"region_id"`
}

type SRegionalCloudResourceKeys struct {
	SCloudResourceKeys
	SRegionalBaseKeys
}

type SRegionalCloudResourceDetailKeys struct {
	SCloudResourceDetailKeys

	SRegionalCloudResourceDetailBaseKeys
}

type SRegionalCloudResourceDetailBaseKeys struct {
	// 配额适用的区域名称
	Region string `json:"region"`
}

type SDomainRegionalCloudResourceKeys struct {
	SBaseDomainQuotaKeys
	SCloudResourceBaseKeys
	SRegionalBaseKeys
}

type SDomainRegionalCloudResourceDetailKeys struct {
	SBaseDomainQuotaDetailKeys
	SCloudResourceDetailBaseKeys
	SRegionalCloudResourceDetailBaseKeys
}

type SZonalCloudResourceKeys struct {
	SRegionalCloudResourceKeys
	// 配额适用的可用区ID
	ZoneId string `width:"64" charset:"ascii" nullable:"false" primary:"true" list:"user" json:"zone_id"`
}

type SZonalCloudResourceDetailKeys struct {
	SRegionalCloudResourceDetailKeys

	// 配额适用的可用区名称
	Zone string `json:"zone"`
}

func (k SBaseDomainQuotaKeys) Fields() []string {
	return []string{
		"domain_id",
	}
}

func (k SBaseProjectQuotaKeys) Fields() []string {
	return append(k.SBaseDomainQuotaKeys.Fields(),
		"tenant_id",
	)
}

func (k SCloudResourceBaseKeys) Fields() []string {
	return []string{
		"provider",
		"brand",
		"cloud_env",
		"account_id",
		"manager_id",
	}
}

func (k SCloudResourceKeys) Fields() []string {
	ret := k.SBaseProjectQuotaKeys.Fields()
	ret = append(ret, k.SCloudResourceBaseKeys.Fields()...)
	return ret
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

func (k SDomainRegionalCloudResourceKeys) Fields() []string {
	ret := k.SBaseDomainQuotaKeys.Fields()
	ret = append(ret, k.SCloudResourceBaseKeys.Fields()...)
	ret = append(ret, "region_id")
	return ret
}

func (k SBaseDomainQuotaKeys) Values() []string {
	return []string{
		k.DomainId,
	}
}

func (k SBaseProjectQuotaKeys) Values() []string {
	return append(k.SBaseDomainQuotaKeys.Values(),
		k.ProjectId,
	)
}

func (k SCloudResourceBaseKeys) Values() []string {
	return []string{
		k.Provider,
		k.Brand,
		k.CloudEnv,
		k.AccountId,
		k.ManagerId,
	}
}

func (k SCloudResourceKeys) Values() []string {
	ret := k.SBaseProjectQuotaKeys.Values()
	ret = append(ret, k.SCloudResourceBaseKeys.Values()...)
	return ret
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

func (k SDomainRegionalCloudResourceKeys) Values() []string {
	ret := k.SBaseDomainQuotaKeys.Values()
	ret = append(ret, k.SCloudResourceBaseKeys.Values()...)
	ret = append(ret, k.RegionId)
	return ret
}

func (k1 SBaseDomainQuotaKeys) Compare(ik IQuotaKeys) int {
	k2 := ik.(SBaseDomainQuotaKeys)
	if k1.DomainId < k2.DomainId {
		return -1
	} else if k1.DomainId > k2.DomainId {
		return 1
	}
	return 0
}

func (k1 SBaseProjectQuotaKeys) Compare(ik IQuotaKeys) int {
	k2 := ik.(SBaseProjectQuotaKeys)
	r := k1.SBaseDomainQuotaKeys.Compare(k2.SBaseDomainQuotaKeys)
	if r != 0 {
		return r
	}
	if k1.ProjectId < k2.ProjectId {
		return -1
	} else if k1.ProjectId > k2.ProjectId {
		return 1
	}
	return 0
}

func (k1 SCloudResourceBaseKeys) compare(k2 SCloudResourceBaseKeys) int {
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

func (k1 SCloudResourceKeys) Compare(ik IQuotaKeys) int {
	k2 := ik.(SCloudResourceKeys)
	r := k1.SBaseProjectQuotaKeys.Compare(k2.SBaseProjectQuotaKeys)
	if r != 0 {
		return r
	}
	r = k1.SCloudResourceBaseKeys.compare(k2.SCloudResourceBaseKeys)
	if r != 0 {
		return r
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

func (k1 SDomainRegionalCloudResourceKeys) Compare(ik IQuotaKeys) int {
	k2 := ik.(SDomainRegionalCloudResourceKeys)
	r := k1.SBaseDomainQuotaKeys.Compare(k2.SBaseDomainQuotaKeys)
	if r != 0 {
		return r
	}
	r = k1.SCloudResourceBaseKeys.compare(k2.SCloudResourceBaseKeys)
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

func QuotaKeyWeight(k IQuotaKeys) uint64 {
	w := uint64(0)
	for i, v := range k.Values() {
		if len(v) > 0 {
			w += (uint64(1) << uint(i))
		}
	}
	return w
}

func (k SBaseDomainQuotaKeys) Scope() rbacscope.TRbacScope {
	if len(k.DomainId) > 0 {
		return rbacscope.ScopeDomain
	} else {
		return rbacscope.ScopeSystem
	}
}

func (k SBaseProjectQuotaKeys) Scope() rbacscope.TRbacScope {
	if len(k.DomainId) > 0 && len(k.ProjectId) > 0 {
		return rbacscope.ScopeProject
	} else if len(k.DomainId) > 0 && len(k.ProjectId) == 0 {
		return rbacscope.ScopeDomain
	} else if len(k.DomainId) == 0 && len(k.ProjectId) == 0 {
		return rbacscope.ScopeSystem
	} else {
		return rbacscope.ScopeNone
	}
}

func (k SBaseDomainQuotaKeys) OwnerId() mcclient.IIdentityProvider {
	return &db.SOwnerId{DomainId: k.DomainId}
}

func (k SBaseProjectQuotaKeys) OwnerId() mcclient.IIdentityProvider {
	return &db.SOwnerId{
		DomainId:  k.DomainId,
		ProjectId: k.ProjectId,
	}
}

func QuotaKeyString(k IQuotaKeys) string {
	parts := make([]string, 0)
	fields := k.Fields()
	values := k.Values()
	for i := range fields {
		if len(values[i]) > 0 {
			parts = append(parts, fmt.Sprintf("%s=%s", fields[i], values[i]))
		}
	}
	return strings.Join(parts, ",")
}

func IsBaseProjectQuotaKeys(k IQuotaKeys) bool {
	fields := k.Fields()
	values := k.Values()
	for i := range fields {
		if fields[i] != "domain_id" && fields[i] != "tenant_id" && len(values[i]) > 0 {
			return false
		}
	}
	return true
}

func IsBaseDomainQuotaKeys(k IQuotaKeys) bool {
	fields := k.Fields()
	values := k.Values()
	for i := range fields {
		if fields[i] != "domain_id" && len(values[i]) > 0 {
			return false
		}
	}
	return true
}

func OwnerIdProjectQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider) SBaseProjectQuotaKeys {
	if scope == rbacscope.ScopeDomain {
		return SBaseProjectQuotaKeys{
			SBaseDomainQuotaKeys: SBaseDomainQuotaKeys{
				DomainId: ownerId.GetProjectDomainId(),
			},
		}
	} else {
		return SBaseProjectQuotaKeys{
			SBaseDomainQuotaKeys: SBaseDomainQuotaKeys{
				DomainId: ownerId.GetProjectDomainId(),
			},
			ProjectId: ownerId.GetProjectId(),
		}
	}
}

func OwnerIdDomainQuotaKeys(ownerId mcclient.IIdentityProvider) SBaseDomainQuotaKeys {
	return SBaseDomainQuotaKeys{DomainId: ownerId.GetProjectDomainId()}
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
