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
	"context"
	"strconv"
	"strings"

	"gopkg.in/ldap.v3"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/ldaputils"
)

type SLDAPDriver struct {
	driver.SBaseIdentityDriver

	ldapConfig *api.SLDAPIdpConfigOptions
}

func NewLDAPDriver(idpId, idpName, template, targetDomainId string, autoCreateProject bool, conf api.TConfigs) (driver.IIdentityBackend, error) {
	base, err := driver.NewBaseIdentityDriver(idpId, idpName, template, targetDomainId, autoCreateProject, conf)
	if err != nil {
		return nil, errors.Wrap(err, "NewBaseIdentityDriver")
	}
	drv := SLDAPDriver{SBaseIdentityDriver: base}
	drv.SetVirtualObject(&drv)
	err = drv.prepareConfig()
	if err != nil {
		return nil, errors.Wrap(err, "prepareConfig")
	}
	return &drv, nil
}

func (self *SLDAPDriver) prepareConfig() error {
	if self.ldapConfig == nil {
		conf := api.SLDAPIdpConfigOptions{}
		switch self.Template {
		case api.IdpTemplateMSSingleDomain:
			conf = MicrosoftActiveDirectorySingleDomainTemplate
		case api.IdpTemplateMSMultiDomain:
			conf = MicrosoftActiveDirectoryMultipleDomainTemplate
		case api.IdpTemplateOpenLDAPSingleDomain:
			conf = OpenLdapSingleDomainTemplate
		}
		confJson := jsonutils.Marshal(self.Config["ldap"])
		err := confJson.Unmarshal(&conf)
		if err != nil {
			return errors.Wrap(err, "json.Unmarshal")
		}
		log.Debugf("%s %s %#v", self.Config, confJson, self.ldapConfig)
		self.ldapConfig = &conf
	}
	return nil
}

func queryScope(scope string) int {
	if scope == api.QueryScopeOne {
		return ldap.ScopeSingleLevel
	} else {
		return ldap.ScopeWholeSubtree
	}
}

func (self *SLDAPDriver) userQueryScope() int {
	scope := self.ldapConfig.UserQueryScope
	if len(scope) == 0 {
		scope = self.ldapConfig.QueryScope
	}
	return queryScope(scope)
}

func (self *SLDAPDriver) groupQueryScope() int {
	scope := self.ldapConfig.GroupQueryScope
	if len(scope) == 0 {
		scope = self.ldapConfig.QueryScope
	}
	return queryScope(scope)
}

func (self *SLDAPDriver) domainQueryScope() int {
	scope := self.ldapConfig.DomainQueryScope
	if len(scope) == 0 {
		scope = self.ldapConfig.QueryScope
	}
	return queryScope(scope)
}

func (self *SLDAPDriver) userAttributeList() []string {
	attrs := []string{
		"dn",
		self.ldapConfig.UserIdAttribute,
		self.ldapConfig.UserNameAttribute,
		self.ldapConfig.UserEnabledAttribute,
	}
	for _, m := range self.ldapConfig.UserAdditionalAttribute {
		parts := strings.Split(m, ":")
		if len(parts) == 2 {
			attrs = append(attrs, parts[0])
		}
	}
	return attrs
}

func (self *SLDAPDriver) groupAttributeList() []string {
	return []string{
		"dn",
		self.ldapConfig.GroupIdAttribute,
		self.ldapConfig.GroupNameAttribute,
		self.ldapConfig.GroupMemberAttribute,
	}
}

func (self *SLDAPDriver) domainAttributeList() []string {
	return []string{
		"dn",
		self.ldapConfig.DomainIdAttribute,
		self.ldapConfig.DomainNameAttribute,
	}
}

func (self *SLDAPDriver) entry2Domain(entry *ldap.Entry) SDomainInfo {
	info := SDomainInfo{}
	info.DN = entry.DN
	info.Id = ldaputils.GetAttributeValue(entry, self.ldapConfig.DomainIdAttribute)
	info.Name = ldaputils.GetAttributeValue(entry, self.ldapConfig.DomainNameAttribute)
	return info
}

func (self *SLDAPDriver) entry2Group(entry *ldap.Entry) SGroupInfo {
	info := SGroupInfo{}
	info.DN = entry.DN
	info.Id = ldaputils.GetAttributeValue(entry, self.ldapConfig.GroupIdAttribute)
	info.Name = ldaputils.GetAttributeValue(entry, self.ldapConfig.GroupNameAttribute)
	info.Members = ldaputils.GetAttributeValues(entry, self.ldapConfig.GroupMemberAttribute)
	return info
}

func (self *SLDAPDriver) entry2User(entry *ldap.Entry) SUserInfo {
	info := SUserInfo{}
	info.DN = entry.DN
	info.Id = ldaputils.GetAttributeValue(entry, self.ldapConfig.UserIdAttribute)
	info.Name = ldaputils.GetAttributeValue(entry, self.ldapConfig.UserNameAttribute)
	enabledStr := ldaputils.GetAttributeValue(entry, self.ldapConfig.UserEnabledAttribute)
	if len(enabledStr) == 0 {
		enabledStr = self.ldapConfig.UserEnabledDefault
	}
	if self.ldapConfig.UserEnabledMask > 0 {
		enabled, _ := strconv.ParseInt(enabledStr, 0, 64)
		if (enabled & self.ldapConfig.UserEnabledMask) != 0 {
			info.Enabled = true
		}
	} else {
		info.Enabled = utils.ToBool(enabledStr)
	}
	if self.ldapConfig.UserEnabledInvert {
		info.Enabled = !info.Enabled
	}
	info.Extra = make(map[string]string)
	for _, m := range self.ldapConfig.UserAdditionalAttribute {
		parts := strings.Split(m, ":")
		if len(parts) == 2 {
			info.Extra[parts[1]] = ldaputils.GetAttributeValue(entry, parts[0])
		}
	}
	return info
}

func (self *SLDAPDriver) getClient() (*ldaputils.SLDAPClient, error) {
	cli := ldaputils.NewLDAPClient(
		self.ldapConfig.Url,
		self.ldapConfig.User,
		self.ldapConfig.Password,
		self.ldapConfig.Suffix,
		false,
	)
	err := cli.Connect()
	if err != nil {
		return nil, errors.Wrap(err, "Connect")
	}
	return cli, nil
}

func (self *SLDAPDriver) getDomainTreeDN() string {
	if len(self.ldapConfig.DomainTreeDN) > 0 {
		return self.ldapConfig.DomainTreeDN
	}
	return self.ldapConfig.Suffix
}

func (self *SLDAPDriver) getUserTreeDN() string {
	if len(self.ldapConfig.UserTreeDN) > 0 {
		return self.ldapConfig.UserTreeDN
	}
	return self.ldapConfig.Suffix
}

func (self *SLDAPDriver) getGroupTreeDN() string {
	if len(self.ldapConfig.GroupTreeDN) > 0 {
		return self.ldapConfig.GroupTreeDN
	}
	return self.ldapConfig.Suffix
}

func (self *SLDAPDriver) Authenticate(ctx context.Context, ident mcclient.SAuthenticationIdentity) (*api.SUserExtended, error) {
	cli, err := self.getClient()
	if err != nil {
		return nil, errors.Wrap(err, "getClient")
	}
	defer cli.Close()

	usrExt, err := models.UserManager.FetchUserExtended(
		ident.Password.User.Id,
		ident.Password.User.Name,
		ident.Password.User.Domain.Id,
		ident.Password.User.Domain.Name,
	)
	if err != nil {
		return nil, errors.Wrap(err, "UserManager.FetchUserExtended")
	}

	var userTreeDN string
	if len(self.ldapConfig.DomainTreeDN) > 0 {
		// import domains
		idMap, err := models.IdmappingManager.FetchEntity(usrExt.DomainId, api.IdMappingEntityDomain)
		if err != nil {
			return nil, errors.Wrap(err, "IdmappingManager.FetchEntity for domain")
		}
		entries, err := self.searchDomainEntries(cli, idMap.IdpEntityId)
		if err != nil {
			return nil, errors.Wrap(err, "self.searchDomainEntries")
		}
		if len(entries) == 0 {
			return nil, errors.Error("fail to find domain DN")
		}
		userTreeDN = entries[0].DN
	} else {
		userTreeDN = self.getUserTreeDN()
	}

	usrIdmap, err := models.IdmappingManager.FetchEntity(usrExt.Id, api.IdMappingEntityUser)
	if err != nil {
		return nil, errors.Wrap(err, "IdmappingManager.FetchEntity for user")
	}
	username := usrIdmap.IdpEntityId
	password := ident.Password.User.Password

	_, err = cli.Authenticate(
		userTreeDN,
		self.ldapConfig.UserObjectclass,
		self.ldapConfig.UserIdAttribute,
		username,
		password,
		self.ldapConfig.UserFilter,
		nil,
		self.userQueryScope(),
	)
	if err != nil {
		log.Errorf("LDAP AUTH error: %s", err)
		return nil, errors.Wrap(err, "Authenticate error")
	}

	return usrExt, nil
}
