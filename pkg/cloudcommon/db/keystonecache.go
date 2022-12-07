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

package db

import (
	"time"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
)

type SKeystoneCacheObjectManager struct {
	SStandaloneResourceBaseManager
}

type SKeystoneCacheObject struct {
	SStandaloneResourceBase

	DomainId string `width:"128" charset:"ascii" nullable:"true"`
	Domain   string `width:"128" charset:"utf8" nullable:"true"`

	LastCheck time.Time `nullable:"true"`
	Lang      string    `width:"8" charset:"ascii" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
}

func NewKeystoneCacheObjectManager(dt interface{}, tableName string, keyword string, keywordPlural string) SKeystoneCacheObjectManager {
	return SKeystoneCacheObjectManager{SStandaloneResourceBaseManager: NewStandaloneResourceBaseManager(dt, tableName, keyword, keywordPlural)}
}

func NewKeystoneCacheObject(id string, name string, domainId string, domain string) SKeystoneCacheObject {
	obj := SKeystoneCacheObject{}
	obj.Id = id
	obj.Name = name
	obj.Domain = domain
	obj.DomainId = domainId
	return obj
}

func (t *SKeystoneCacheObject) IsExpired() bool {
	if t.LastCheck.IsZero() {
		return true
	}
	now := time.Now().UTC()
	if t.LastCheck.Add(consts.GetTenantCacheExpireSeconds()).Before(now) {
		return true
	}
	return false
}
