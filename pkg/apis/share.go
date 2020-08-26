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

package apis

import (
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
	CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN  = "account_domain"
	CLOUD_ACCOUNT_SHARE_MODE_SYSTEM          = "system"
	CLOUD_ACCOUNT_SHARE_MODE_PROVIDER_DOMAIN = "provider_domain"
)

type SAccountShareInfo struct {
	IsPublic      bool
	PublicScope   rbacutils.TRbacScope
	ShareMode     string
	SharedDomains []string
}

type SShareInfo struct {
	IsPublic       bool
	PublicScope    rbacutils.TRbacScope
	SharedDomains  []string
	SharedProjects []string
}

func (i SAccountShareInfo) GetProjectShareInfo() SShareInfo {
	ret := SShareInfo{}
	switch i.ShareMode {
	case CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN:
		ret.IsPublic = true
		ret.PublicScope = rbacutils.ScopeDomain
	case CLOUD_ACCOUNT_SHARE_MODE_PROVIDER_DOMAIN:
		ret.IsPublic = true
		ret.PublicScope = rbacutils.ScopeDomain
	case CLOUD_ACCOUNT_SHARE_MODE_SYSTEM:
		ret.IsPublic = true
		if i.IsPublic && i.PublicScope == rbacutils.ScopeSystem {
			ret.PublicScope = rbacutils.ScopeSystem
		} else {
			ret.PublicScope = rbacutils.ScopeDomain
			ret.SharedDomains = i.SharedDomains
		}
	}
	return ret
}

func (i SAccountShareInfo) GetDomainShareInfo() SShareInfo {
	ret := SShareInfo{}
	switch i.ShareMode {
	case CLOUD_ACCOUNT_SHARE_MODE_ACCOUNT_DOMAIN:
		ret.IsPublic = false
		ret.PublicScope = rbacutils.ScopeNone
	case CLOUD_ACCOUNT_SHARE_MODE_PROVIDER_DOMAIN:
		ret.IsPublic = false
		ret.PublicScope = rbacutils.ScopeNone
	case CLOUD_ACCOUNT_SHARE_MODE_SYSTEM:
		if i.IsPublic && i.PublicScope == rbacutils.ScopeSystem {
			ret.IsPublic = true
			ret.PublicScope = rbacutils.ScopeSystem
		} else if len(i.SharedDomains) > 0 {
			ret.IsPublic = true
			ret.PublicScope = rbacutils.ScopeDomain
			ret.SharedDomains = i.SharedDomains
		} else {
			ret.IsPublic = false
			ret.PublicScope = rbacutils.ScopeNone
		}
	}
	return ret
}

func (i SShareInfo) IsViolate(i2 SShareInfo) bool {
	if i.IsPublic && !i2.IsPublic {
		return true
	} else if !i.IsPublic && i2.IsPublic {
		return false
	}
	// is_public equals
	if i.PublicScope.HigherThan(i2.PublicScope) {
		return true
	} else if i2.PublicScope.HigherThan(i.PublicScope) {
		return false
	}
	// public_scope equals
	aNoB, _, bNoA := stringutils2.Split(stringutils2.NewSortedStrings(i.SharedDomains), stringutils2.NewSortedStrings(i2.SharedDomains))
	if len(aNoB) > 0 {
		return true
	} else if len(bNoA) > 0 {
		return false
	}
	// shared_domains equals
	aNoB, _, bNoA = stringutils2.Split(stringutils2.NewSortedStrings(i.SharedProjects), stringutils2.NewSortedStrings(i2.SharedProjects))
	if len(aNoB) > 0 {
		return true
	} else if len(bNoA) > 0 {
		return false
	}
	return false
}

func (i SShareInfo) Intersect(i2 SShareInfo) SShareInfo {
	if i.IsPublic && !i2.IsPublic {
		return i2
	} else if !i.IsPublic && i2.IsPublic {
		return i
	}
	// is_public equals
	if i.PublicScope.HigherThan(i2.PublicScope) {
		return i2
	} else if i2.PublicScope.HigherThan(i.PublicScope) {
		return i
	}
	// public_scope equals
	_, domains, _ := stringutils2.Split(stringutils2.NewSortedStrings(i.SharedDomains), stringutils2.NewSortedStrings(i2.SharedDomains))
	_, projs, _ := stringutils2.Split(stringutils2.NewSortedStrings(i.SharedProjects), stringutils2.NewSortedStrings(i2.SharedProjects))
	ret := SShareInfo{
		IsPublic:       i.IsPublic,
		PublicScope:    i.PublicScope,
		SharedDomains:  domains,
		SharedProjects: projs,
	}
	if ret.PublicScope == rbacutils.ScopeProject && len(ret.SharedProjects) == 0 {
		ret.IsPublic = false
		ret.PublicScope = rbacutils.ScopeNone
	}
	return ret
}

func (i SShareInfo) Equals(i2 SShareInfo) bool {
	if !i.IsViolate(i2) && !i2.IsViolate(i) {
		return true
	} else {
		return false
	}
}

func (i *SShareInfo) FixProjectShare() {
	if i.PublicScope == rbacutils.ScopeProject && len(i.SharedProjects) == 0 {
		i.IsPublic = false
		i.PublicScope = rbacutils.ScopeNone
	}
}

func (i *SShareInfo) FixDomainShare() {
	if i.PublicScope == rbacutils.ScopeProject {
		i.IsPublic = false
		i.PublicScope = rbacutils.ScopeNone
		i.SharedProjects = nil
	} else if i.PublicScope == rbacutils.ScopeDomain && len(i.SharedDomains) == 0 {
		i.IsPublic = false
		i.PublicScope = rbacutils.ScopeNone
	}
}
