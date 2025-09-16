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
	"database/sql"
	"strconv"
	"strings"

	"gopkg.in/ldap.v3"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/ldaputils"
)

type SLDAPDriver struct {
	driver.SBaseIdentityDriver

	ldapConfig *api.SLDAPIdpConfigOptions
}

func NewLDAPDriver(idpId, idpName, template, targetDomainId string, conf api.TConfigs) (driver.IIdentityBackend, error) {
	base, err := driver.NewBaseIdentityDriver(idpId, idpName, template, targetDomainId, conf)
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

func (drv *SLDAPDriver) prepareConfig() error {
	if drv.ldapConfig == nil {
		conf := api.SLDAPIdpConfigOptions{}
		switch drv.Template {
		case api.IdpTemplateMSSingleDomain:
			conf = MicrosoftActiveDirectorySingleDomainTemplate
		case api.IdpTemplateMSMultiDomain:
			conf = MicrosoftActiveDirectoryMultipleDomainTemplate
		case api.IdpTemplateOpenLDAPSingleDomain:
			conf = OpenLdapSingleDomainTemplate
		}
		confJson := jsonutils.Marshal(drv.Config["ldap"])
		err := confJson.Unmarshal(&conf)
		if err != nil {
			return errors.Wrap(err, "json.Unmarshal")
		}
		log.Debugf("%s %s %#v", drv.Config, confJson, drv.ldapConfig)
		drv.ldapConfig = &conf
	}
	return nil
}

func (ldap *SLDAPDriver) GetSsoRedirectUri(ctx context.Context, callbackUrl, state string) (string, error) {
	return "", errors.Wrap(httperrors.ErrNotSupported, "not a SSO driver")
}

func queryScope(scope string) int {
	if scope == api.QueryScopeOne {
		return ldap.ScopeSingleLevel
	} else {
		return ldap.ScopeWholeSubtree
	}
}

func (drv *SLDAPDriver) userQueryScope() int {
	scope := drv.ldapConfig.UserQueryScope
	if len(scope) == 0 {
		scope = drv.ldapConfig.QueryScope
	}
	return queryScope(scope)
}

func (drv *SLDAPDriver) groupQueryScope() int {
	scope := drv.ldapConfig.GroupQueryScope
	if len(scope) == 0 {
		scope = drv.ldapConfig.QueryScope
	}
	return queryScope(scope)
}

func (drv *SLDAPDriver) domainQueryScope() int {
	scope := drv.ldapConfig.DomainQueryScope
	if len(scope) == 0 {
		scope = drv.ldapConfig.QueryScope
	}
	return queryScope(scope)
}

func (drv *SLDAPDriver) userAttributeList() []string {
	attrs := []string{
		"dn",
		drv.ldapConfig.UserIdAttribute,
		drv.ldapConfig.UserNameAttribute,
		drv.ldapConfig.UserEnabledAttribute,
	}
	for _, m := range drv.ldapConfig.UserAdditionalAttribute {
		parts := strings.Split(m, ":")
		if len(parts) == 2 && !utils.IsInArray(parts[0], attrs) {
			attrs = append(attrs, parts[0])
		}
	}
	return attrs
}

func (drv *SLDAPDriver) groupAttributeList() []string {
	return []string{
		"dn",
		drv.ldapConfig.GroupIdAttribute,
		drv.ldapConfig.GroupNameAttribute,
		drv.ldapConfig.GroupMemberAttribute,
	}
}

func (drv *SLDAPDriver) domainAttributeList() []string {
	return []string{
		"dn",
		drv.ldapConfig.DomainIdAttribute,
		drv.ldapConfig.DomainNameAttribute,
	}
}

func (drv *SLDAPDriver) entry2Domain(entry *ldap.Entry) SDomainInfo {
	info := SDomainInfo{}
	info.DN = entry.DN
	info.Id = ldaputils.GetAttributeValue(entry, drv.ldapConfig.DomainIdAttribute)
	info.Name = ldaputils.GetAttributeValue(entry, drv.ldapConfig.DomainNameAttribute)
	return info
}

func (drv *SLDAPDriver) entry2Group(entry *ldap.Entry) SGroupInfo {
	info := SGroupInfo{}
	info.DN = entry.DN
	info.Id = ldaputils.GetAttributeValue(entry, drv.ldapConfig.GroupIdAttribute)
	info.Name = ldaputils.GetAttributeValue(entry, drv.ldapConfig.GroupNameAttribute)
	info.Members = ldaputils.GetAttributeValues(entry, drv.ldapConfig.GroupMemberAttribute)
	return info
}

func (drv *SLDAPDriver) entry2User(entry *ldap.Entry) SUserInfo {
	info := SUserInfo{}
	info.DN = entry.DN
	info.Id = ldaputils.GetAttributeValue(entry, drv.ldapConfig.UserIdAttribute)
	info.Name = ldaputils.GetAttributeValue(entry, drv.ldapConfig.UserNameAttribute)
	enabledStr := ldaputils.GetAttributeValue(entry, drv.ldapConfig.UserEnabledAttribute)
	if len(enabledStr) == 0 {
		enabledStr = drv.ldapConfig.UserEnabledDefault
	}
	if drv.ldapConfig.UserEnabledMask > 0 {
		enabled, _ := strconv.ParseInt(enabledStr, 0, 64)
		if (enabled & drv.ldapConfig.UserEnabledMask) != 0 {
			info.Enabled = true
		}
	} else {
		info.Enabled = utils.ToBool(enabledStr)
	}
	if drv.ldapConfig.UserEnabledInvert {
		info.Enabled = !info.Enabled
	}
	info.Extra = make(map[string]string)
	for _, m := range drv.ldapConfig.UserAdditionalAttribute {
		parts := strings.Split(m, ":")
		if len(parts) == 2 {
			info.Extra[parts[1]] = ldaputils.GetAttributeValue(entry, parts[0])
		}
	}
	return info
}

func (drv *SLDAPDriver) getClient() (*ldaputils.SLDAPClient, error) {
	cli := ldaputils.NewLDAPClient(
		drv.ldapConfig.Url,
		drv.ldapConfig.User,
		drv.ldapConfig.Password,
		drv.ldapConfig.Suffix,
		false,
	)
	err := cli.Connect()
	if err != nil {
		return nil, errors.Wrap(err, "Connect")
	}
	return cli, nil
}

func (drv *SLDAPDriver) getDomainTreeDN() string {
	if len(drv.ldapConfig.DomainTreeDN) > 0 {
		return drv.ldapConfig.DomainTreeDN
	}
	return drv.ldapConfig.Suffix
}

func (drv *SLDAPDriver) getUserTreeDN() string {
	if len(drv.ldapConfig.UserTreeDN) > 0 {
		return drv.ldapConfig.UserTreeDN
	}
	return drv.ldapConfig.Suffix
}

func (drv *SLDAPDriver) getGroupTreeDN() string {
	if len(drv.ldapConfig.GroupTreeDN) > 0 {
		return drv.ldapConfig.GroupTreeDN
	}
	return drv.ldapConfig.Suffix
}

func (drv *SLDAPDriver) Authenticate(ctx context.Context, ident mcclient.SAuthenticationIdentity) (*api.SUserExtended, error) {
	cli, err := drv.getClient()
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
	if len(drv.ldapConfig.DomainTreeDN) > 0 {
		// import domains
		idMap, err := models.IdmappingManager.FetchFirstEntity(usrExt.DomainId, api.IdMappingEntityDomain)
		if err != nil {
			return nil, errors.Wrap(err, "IdmappingManager.FetchEntity for domain")
		}
		var searchEntry *ldap.Entry
		err = drv.searchDomainEntries(cli, idMap.IdpEntityId,
			func(entry *ldap.Entry) error {
				searchEntry = entry
				return ldaputils.StopSearch
			})
		if err != nil {
			return nil, errors.Wrap(err, "drv.searchDomainEntries")
		}
		if searchEntry == nil {
			return nil, errors.Error("fail to find domain DN")
		}
		userTreeDN = searchEntry.DN
	} else {
		userTreeDN = drv.getUserTreeDN()
	}

	usrIdmaps, err := models.IdmappingManager.FetchEntities(usrExt.Id, api.IdMappingEntityUser)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, errors.Wrap(err, "IdmappingManager.FetchEntity for user")
	}
	var usrIdmap *models.SIdmapping
	for i := range usrIdmaps {
		if usrIdmaps[i].IdpId == drv.IdpId {
			usrIdmap = &usrIdmaps[i]
			break
		}
	}
	if usrIdmap == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidCredential, "user not found in identity provider")
	}
	username := usrIdmap.IdpEntityId
	password := ident.Password.User.Password

	_, err = cli.Authenticate(
		userTreeDN,
		drv.ldapConfig.UserObjectclass,
		drv.ldapConfig.UserIdAttribute,
		username,
		password,
		drv.ldapConfig.UserFilter,
		nil,
		drv.userQueryScope(),
	)
	if err != nil {
		log.Errorf("LDAP AUTH error: %s", err)
		if errors.Cause(err) == ldaputils.ErrUserNotFound {
			return nil, httperrors.ErrUserNotFound
		}
		if errors.Cause(err) == ldaputils.ErrUserBadCredential {
			return nil, httperrors.ErrWrongPassword
		}
		return nil, errors.Wrap(err, "Authenticate error")
	}

	usrExt.AuditIds = []string{username}

	return usrExt, nil
}
