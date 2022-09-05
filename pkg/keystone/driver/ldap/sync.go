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

	"gopkg.in/ldap.v3"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/util/ldaputils"
)

func (self *SLDAPDriver) Probe(ctx context.Context) error {
	cli, err := self.getClient()
	if err != nil {
		return errors.Wrap(err, "getClient")
	}
	defer cli.Close()
	return nil
}

func (self *SLDAPDriver) Sync(ctx context.Context) error {
	cli, err := self.getClient()
	if err != nil {
		return errors.Wrap(err, "getClient")
	}
	defer cli.Close()

	if len(self.ldapConfig.DomainTreeDN) > 0 {
		return self.syncDomains(ctx, cli)
	} else {
		return self.syncSingleDomain(ctx, cli)
	}
}

func (self *SLDAPDriver) syncSingleDomain(ctx context.Context, cli *ldaputils.SLDAPClient) error {
	var domain *models.SDomain
	if len(self.TargetDomainId) > 0 {
		targetDomain, err := models.DomainManager.FetchDomainById(self.TargetDomainId)
		if err != nil && err != sql.ErrNoRows {
			return errors.Wrap(err, "models.DomainManager.FetchDomainById")
		}
		if targetDomain == nil {
			log.Warningln("target domain not exist!")
		} else {
			domain = targetDomain
		}
	}
	if domain == nil {
		domainInfo := SDomainInfo{DN: self.ldapConfig.Suffix, Id: api.DefaultRemoteDomainId, Name: self.IdpName}
		newDomain, err := self.syncDomainInfo(ctx, domainInfo)
		if err != nil {
			return errors.Wrap(err, "syncDomainInfo")
		}
		domain = newDomain
	}
	userIdMap, err := self.syncUsers(ctx, cli, domain.Id, self.getUserTreeDN())
	if err != nil {
		return errors.Wrap(err, "syncUsers")
	}
	err = self.syncGroups(ctx, cli, domain.Id, self.getGroupTreeDN(), userIdMap)
	if err != nil {
		return errors.Wrap(err, "syncGroups")
	}
	return nil
}

func (self *SLDAPDriver) searchDomainEntries(cli *ldaputils.SLDAPClient, domainid string, entryFunc func(*ldap.Entry) error) error {
	attrMap := make(map[string]string)
	if len(domainid) > 0 {
		attrMap[self.ldapConfig.DomainIdAttribute] = domainid
	}
	_, err := cli.Search(self.getDomainTreeDN(),
		self.ldapConfig.DomainObjectclass,
		attrMap,
		self.ldapConfig.DomainFilter,
		self.domainAttributeList(),
		self.domainQueryScope(),
		options.Options.LdapSearchPageSize, 0,
		func(offset uint32, entry *ldap.Entry) error {
			return entryFunc(entry)
		},
	)
	if err != nil {
		return errors.Wrap(err, "Search")
	}
	return nil
}

func (self *SLDAPDriver) syncDomains(ctx context.Context, cli *ldaputils.SLDAPClient) error {
	domainIds := make([]string, 0)
	err := self.searchDomainEntries(cli, "", func(entry *ldap.Entry) error {
		domainInfo := self.entry2Domain(entry)
		err := domainInfo.isValid()
		if err != nil {
			log.Errorf("invalid domainInfo: %s, skip", err)
			return nil
		}
		domain, err := self.syncDomainInfo(ctx, domainInfo)
		if err != nil {
			return errors.Wrap(err, "syncDomainInfo")
		}
		domainIds = append(domainIds, domain.Id)
		userIdMap, err := self.syncUsers(ctx, cli, domain.Id, domainInfo.DN)
		if err != nil {
			return errors.Wrap(err, "syncUsers")
		}
		err = self.syncGroups(ctx, cli, domain.Id, domainInfo.DN, userIdMap)
		if err != nil {
			return errors.Wrap(err, "syncGroups")
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "searchDomainEntries")
	}

	// remove any obsolete domains
	obsoleteDomainIds, err := models.IdmappingManager.FetchPublicIdsExcludes(self.IdpId, api.IdMappingEntityDomain, domainIds)
	if err != nil {
		return errors.Wrap(err, "models.IdmappingManager.FetchPublicIdsExcludes")
	}
	for _, obsoleteDomainId := range obsoleteDomainIds {
		obsoleteDomain, err := models.DomainManager.FetchDomainById(obsoleteDomainId)
		if err != nil {
			log.Errorf("models.DomainManager.FetchDomainById error %s", err)
			continue
		}
		obsoleteDomain.AppendDescription(models.GetDefaultAdminCred(), "domain source removed")
		// unlink with Idp
		err = obsoleteDomain.UnlinkIdp(self.IdpId)
		if err != nil {
			log.Errorf("obsoleteDomain.UnlinkIdp error %s", err)
			continue
		}
		// remove any user and groups
		err = obsoleteDomain.DeleteUserGroups(ctx, models.GetDefaultAdminCred())
		if err != nil {
			log.Errorf("domain.DeleteUserGroups error %s", err)
			continue
		}
		err = obsoleteDomain.ValidateDeleteCondition(ctx, nil)
		if err != nil {
			log.Errorf("obsoleteDomain.ValidateDeleteCondition error %s", err)
			continue
		}
		err = obsoleteDomain.Delete(ctx, models.GetDefaultAdminCred())
		if err != nil {
			log.Errorf("obsoleteDomain.Delete error %s", err)
			continue
		}
	}
	return nil
}

func (self *SLDAPDriver) syncDomainInfo(ctx context.Context, info SDomainInfo) (*models.SDomain, error) {
	idp, err := models.IdentityProviderManager.FetchIdentityProviderById(self.IdpId)
	if err != nil {
		return nil, errors.Wrap(err, "self.GetIdentityProvider")
	}
	return idp.SyncOrCreateDomain(ctx, info.Id, info.Name, info.DN, true)
}

func (self *SLDAPDriver) syncUsers(ctx context.Context, cli *ldaputils.SLDAPClient, domainId string, baseDN string) (map[string]string, error) {
	userIds := make([]string, 0)
	userIdMap := make(map[string]string)
	_, err := cli.Search(baseDN,
		self.ldapConfig.UserObjectclass,
		nil,
		self.ldapConfig.UserFilter,
		self.userAttributeList(),
		self.userQueryScope(),
		options.Options.LdapSearchPageSize, 0,
		func(offset uint32, entry *ldap.Entry) error {
			userInfo := self.entry2User(entry)
			err := userInfo.isValid()
			if err != nil {
				log.Debugf("userInfo is invalid: %s, skip", err)
				return nil
			}
			userId, err := self.syncUserDB(ctx, userInfo, domainId)
			if err != nil {
				return errors.Wrap(err, "syncUserDB")
			}
			userIds = append(userIds, userId)
			if self.ldapConfig.GroupMembersAreIds {
				userIdMap[userInfo.Id] = userId
			} else {
				userIdMap[userInfo.DN] = userId
			}
			return nil
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "searchLDAP")
	}

	deleteUsers, err := models.UserManager.FetchUsersInDomain(domainId, userIds)
	if err != nil {
		return nil, errors.Wrap(err, "models.UserManager.FetchUserIdsInDomain")
	}
	for i := range deleteUsers {
		if !deleteUsers[i].LinkedWithIdp(self.IdpId) {
			continue
		}
		err := deleteUsers[i].UnlinkIdp(self.IdpId)
		if err != nil {
			log.Errorf("deleteUser.UnlinkIdp error %s", err)
			continue
		}
		err = deleteUsers[i].ValidateDeleteCondition(ctx, nil)
		if err != nil {
			log.Errorf("deleteUser.ValidateDeleteCondition error %s", err)
			continue
		}
		err = deleteUsers[i].Delete(ctx, models.GetDefaultAdminCred())
		if err != nil {
			log.Errorf("deleteUser.Delete error %s", err)
			continue
		}
	}
	return userIdMap, nil
}

func (self *SLDAPDriver) syncUserDB(ctx context.Context, ui SUserInfo, domainId string) (string, error) {
	idp, err := models.IdentityProviderManager.FetchIdentityProviderById(self.IdpId)
	if err != nil {
		return "", errors.Wrap(err, "models.IdentityProviderManager.FetchIdentityProviderById")
	}
	usr, err := idp.SyncOrCreateUser(ctx, ui.Id, ui.Name, domainId, !self.ldapConfig.DisableUserOnImport, func(user *models.SUser) {
		// LDAP user is always enabled
		// if ui.Enabled {
		// 	user.Enabled = tristate.True
		// } else {
		//	user.Enabled = tristate.False
		// }
		if val, ok := ui.Extra["email"]; ok && len(val) > 0 && len(user.Email) == 0 {
			user.Email = val
		}
		if val, ok := ui.Extra["displayname"]; ok && len(val) > 0 && len(user.Displayname) == 0 {
			user.Displayname = val
		}
		if val, ok := ui.Extra["mobile"]; ok && len(val) > 0 && len(user.Mobile) == 0 {
			user.Mobile = val
		}
	})
	if err != nil {
		return "", errors.Wrap(err, "idp.SyncOrCreateUser")
	}

	return usr.Id, nil
}

func (self *SLDAPDriver) syncGroups(ctx context.Context, cli *ldaputils.SLDAPClient, domainId string, baseDN string, userIdMap map[string]string) error {
	groupIds := make([]string, 0)
	_, err := cli.Search(baseDN,
		self.ldapConfig.GroupObjectclass,
		nil,
		self.ldapConfig.GroupFilter,
		self.groupAttributeList(),
		self.groupQueryScope(),
		options.Options.LdapSearchPageSize, 0,
		func(offset uint32, entry *ldap.Entry) error {
			groupInfo := self.entry2Group(entry)
			err := groupInfo.isValid()
			if err != nil {
				log.Errorf("invalid group info: %s, skip", err)
				return nil
			}
			groupId, err := self.syncGroupDB(ctx, groupInfo, domainId, userIdMap)
			if err != nil {
				return errors.Wrap(err, "syncGroupDB")
			}
			groupIds = append(groupIds, groupId)
			return nil
		},
	)
	if err != nil {
		return errors.Wrap(err, "searchLDAP")
	}

	deleteGroups, err := models.GroupManager.FetchGroupsInDomain(domainId, groupIds)
	if err != nil {
		return errors.Wrap(err, "models.GroupManager.FetchGroupsInDomain")
	}
	for i := range deleteGroups {
		if !deleteGroups[i].LinkedWithIdp(self.IdpId) {
			continue
		}
		err := deleteGroups[i].UnlinkIdp(self.IdpId)
		if err != nil {
			log.Errorf("deleteGroup.UnlinkIdp error %s", err)
			continue
		}
		err = deleteGroups[i].ValidateDeleteCondition(ctx, nil)
		if err != nil {
			log.Errorf("deleteGroup.ValidateDeleteCondition error %s", err)
			continue
		}
		err = deleteGroups[i].Delete(ctx, models.GetDefaultAdminCred())
		if err != nil {
			log.Errorf("deleteGroup.Delete error %s", err)
			continue
		}
	}
	return nil
}

func (self *SLDAPDriver) syncGroupDB(ctx context.Context, groupInfo SGroupInfo, domainId string, userIdMap map[string]string) (string, error) {
	grp, err := models.GroupManager.RegisterExternalGroup(ctx, self.IdpId, domainId, groupInfo.Id, groupInfo.Name)
	if err != nil {
		return "", errors.Wrap(err, "GroupManager.RegisterExternalGroup")
	}
	userIds := make([]string, 0)
	for _, userExtId := range groupInfo.Members {
		if uid, ok := userIdMap[userExtId]; ok {
			userIds = append(userIds, uid)
		}
	}
	models.UsergroupManager.SyncGroupUsers(ctx, models.GetDefaultAdminCred(), grp.Id, userIds)
	return grp.Id, nil
}
