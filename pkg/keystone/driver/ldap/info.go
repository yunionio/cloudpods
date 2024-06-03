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

package ldap

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/keystone/options"
)

const (
	ErrEmptyDN      = errors.Error("empty DN")
	ErrEmptyId      = errors.Error("empty id")
	ErrEmptyName    = errors.Error("empty name")
	ErrDisabledUser = errors.Error("disabled user")
)

type SDomainInfo struct {
	DN   string
	Id   string
	Name string
}

type SUserInfo struct {
	SDomainInfo
	Enabled bool
	Extra   map[string]string
}

type SGroupInfo struct {
	SDomainInfo
	Members []string
}

func (info SDomainInfo) isValid() error {
	if len(info.DN) == 0 {
		return ErrEmptyDN
	}
	if len(info.Id) == 0 {
		return ErrEmptyId
	}
	if len(info.Name) == 0 {
		return ErrEmptyName
	}
	return nil
}

func (info SUserInfo) isValid() error {
	err := info.SDomainInfo.isValid()
	if err != nil {
		return err
	}
	// regarding disabled LDAP user as invalid
	if !options.Options.LdapSyncDisabledUsers && !info.Enabled {
		return ErrDisabledUser
	}
	return nil
}
