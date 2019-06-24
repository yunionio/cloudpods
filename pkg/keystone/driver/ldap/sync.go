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
	"fmt"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/keystone/models"
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

	if self.ldapConfig.ImportDomain {
		return self.syncDomains(ctx, cli)
	} else {
		return self.syncSingleDomain(ctx, cli)
	}
}

func (self *SLDAPDriver) syncSingleDomain(ctx context.Context, cli *ldaputils.SLDAPClient) error {
	domainInfo := SDomainInfo{DN: self.ldapConfig.Suffix, Id: api.DefaultRemoteDomainId, Name: self.IdpName}
	domain, err := self.syncDomainInfo(ctx, domainInfo)
	if err != nil {
		return errors.Wrap(err, "syncDomainInfo")
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

func (self *SLDAPDriver) syncDomains(ctx context.Context, cli *ldaputils.SLDAPClient) error {
	entries, err := cli.Search(self.getDomainTreeDN(),
		self.ldapConfig.DomainObjectclass,
		nil,
		self.ldapConfig.DomainFilter,
		self.domainAttributeList(),
		self.domainQueryScope(),
	)
	if err != nil {
		return errors.Wrap(err, "searchLDAP")
	}
	domainLocalIds := make([]string, len(entries))
	for i := range entries {
		domainInfo := self.entry2Domain(entries[i])
		domainLocalIds[i] = domainInfo.Id
		domain, err := self.syncDomainInfo(ctx, domainInfo)
		if err != nil {
			return errors.Wrap(err, "syncDomainInfo")
		}
		userIdMap, err := self.syncUsers(ctx, cli, domain.Id, domainInfo.DN)
		if err != nil {
			return errors.Wrap(err, "syncUsers")
		}
		err = self.syncGroups(ctx, cli, domain.Id, domainInfo.DN, userIdMap)
		if err != nil {
			return errors.Wrap(err, "syncGroups")
		}
	}
	// remove any obsolete domain Id_mappings
	err = models.IdmappingManager.DeleteAny(self.IdpId, api.IdMappingEntityDomain, domainLocalIds)
	if err != nil {
		log.Errorf("delete remvoed remote domain fail %s", err)
	}
	return nil
}

func (self *SLDAPDriver) syncDomainInfo(ctx context.Context, info SDomainInfo) (*models.SDomain, error) {
	domainId, err := models.IdmappingManager.RegisterIdMap(ctx, self.IdpId, info.Id, api.IdMappingEntityDomain)
	if err != nil {
		return nil, errors.Wrap(err, "IdmappingManager.RegisterIdMap")
	}

	domain, err := models.DomainManager.FetchDomainById(domainId)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "DomainManager.FetchDomainById")
	}
	if err == nil {
		if domain.Name != info.Name {
			// sync domain name
			newName, err := db.GenerateName(models.DomainManager, nil, info.Name)
			if err != nil {
				log.Errorf("sync existing domain name (%s=%s) generate fail %s", domain.Name, info.Name, err)
			} else {
				_, err = db.Update(domain, func() error {
					domain.Name = newName
					return nil
				})
				if err != nil {
					log.Errorf("sync existing domain name (%s=%s) update fail %s", domain.Name, info.Name, err)
				}
			}
		}
		return domain, nil
	}

	lockman.LockClass(ctx, models.DomainManager, "")
	lockman.ReleaseClass(ctx, models.DomainManager, "")

	domain = &models.SDomain{}
	domain.SetModelManager(models.DomainManager, domain)
	domain.Id = domainId
	newName, err := db.GenerateName(models.DomainManager, nil, info.Name)
	if err != nil {
		return nil, errors.Wrap(err, "GenerateName")
	}
	domain.Name = newName
	domain.Enabled = tristate.True
	domain.IsDomain = tristate.True
	domain.DomainId = api.KeystoneDomainRoot
	domain.Description = fmt.Sprintf("domain for %s", info.DN)
	err = models.DomainManager.TableSpec().Insert(domain)
	if err != nil {
		return nil, errors.Wrap(err, "insert")
	}

	return domain, nil
}

func (self *SLDAPDriver) syncUsers(ctx context.Context, cli *ldaputils.SLDAPClient, domainId string, baseDN string) (map[string]string, error) {
	entries, err := cli.Search(baseDN,
		self.ldapConfig.UserObjectclass,
		nil,
		self.ldapConfig.UserFilter,
		self.userAttributeList(),
		self.userQueryScope(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "searchLDAP")
	}
	userLocalIds := make([]string, len(entries))
	userIdMap := make(map[string]string)
	for i := range entries {
		userInfo := self.entry2User(entries[i])
		userLocalIds[i] = userInfo.Id
		userId, err := self.syncUserDB(ctx, userInfo, domainId)
		if err != nil {
			return nil, errors.Wrap(err, "syncUserDB")
		}
		if self.ldapConfig.GroupMembersAreIds {
			userIdMap[userInfo.Id] = userId
		} else {
			userIdMap[userInfo.DN] = userId
		}
	}
	err = models.IdmappingManager.DeleteAny(self.IdpId, api.IdMappingEntityUser, userLocalIds)
	if err != nil {
		log.Errorf("delete removed remote user fail %s", err)
	}
	return userIdMap, nil
}

func copyUserInfo(ui SUserInfo, userId string, domainId string, user *models.SUser) {
	user.Id = userId
	user.Name = ui.Name
	if ui.Enabled {
		user.Enabled = tristate.True
	} else {
		user.Enabled = tristate.False
	}
	user.DomainId = domainId
	if val, ok := ui.Extra["email"]; ok && len(val) > 0 {
		user.Email = val
	}
	if val, ok := ui.Extra["displayname"]; ok && len(val) > 0 {
		user.Displayname = val
	}
	if val, ok := ui.Extra["mobile"]; ok && len(val) > 0 {
		user.Mobile = val
	}
}

func registerNonlocalUser(ctx context.Context, ui SUserInfo, userId string, domainId string) error {
	lockman.LockRawObject(ctx, models.UserManager.Keyword(), userId)
	defer lockman.ReleaseRawObject(ctx, models.UserManager.Keyword(), userId)

	userObj, err := db.NewModelObject(models.UserManager)
	if err != nil {
		return errors.Wrap(err, "db.NewModelObject")
	}
	user := userObj.(*models.SUser)
	q := models.UserManager.Query().Equals("id", userId)
	err = q.First(user)
	if err != nil && err != sql.ErrNoRows {
		return errors.Wrap(err, "Query user")
	}
	if err == nil {
		// update
		_, err := db.Update(user, func() error {
			copyUserInfo(ui, userId, domainId, user)
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "Update")
		}
	} else {
		// new user
		copyUserInfo(ui, userId, domainId, user)
		err = models.UserManager.TableSpec().Insert(user)
		if err != nil {
			return errors.Wrap(err, "Insert")
		}
	}
	return nil
}

func (self *SLDAPDriver) syncUserDB(ctx context.Context, ui SUserInfo, domainId string) (string, error) {
	userId, err := models.IdmappingManager.RegisterIdMap(ctx, self.IdpId, ui.Id, api.IdMappingEntityUser)
	if err != nil {
		return "", errors.Wrap(err, "models.IdmappingManager.RegisterIdMap")
	}

	// insert nonlocal user
	err = registerNonlocalUser(ctx, ui, userId, domainId)
	if err != nil {
		return "", errors.Wrap(err, "registerNonlocalUser")
	}

	return userId, nil
}

func (self *SLDAPDriver) syncGroups(ctx context.Context, cli *ldaputils.SLDAPClient, domainId string, baseDN string, userIdMap map[string]string) error {
	entries, err := cli.Search(baseDN,
		self.ldapConfig.GroupObjectclass,
		nil,
		self.ldapConfig.GroupFilter,
		self.groupAttributeList(),
		self.groupQueryScope(),
	)
	if err != nil {
		return errors.Wrap(err, "searchLDAP")
	}
	groupLocalIds := make([]string, len(entries))
	for i := range entries {
		groupInfo := self.entry2Group(entries[i])
		groupLocalIds[i] = groupInfo.Id
		err := self.syncGroupDB(ctx, groupInfo, domainId, userIdMap)
		if err != nil {
			return errors.Wrap(err, "syncGroupDB")
		}
	}
	err = models.IdmappingManager.DeleteAny(self.IdpId, api.IdMappingEntityGroup, groupLocalIds)
	if err != nil {
		log.Errorf("delete removed remote group fail %s", err)
	}
	return nil
}

func (self *SLDAPDriver) syncGroupDB(ctx context.Context, groupInfo SGroupInfo, domainId string, userIdMap map[string]string) error {
	grp, err := models.GroupManager.RegisterExternalGroup(ctx, self.IdpId, domainId, groupInfo.Id, groupInfo.Name)
	if err != nil {
		return errors.Wrap(err, "GroupManager.RegisterExternalGroup")
	}
	userIds := make([]string, 0)
	for _, userExtId := range groupInfo.Members {
		if uid, ok := userIdMap[userExtId]; ok {
			userIds = append(userIds, uid)
		}
	}
	models.UsergroupManager.SyncGroupUsers(ctx, models.GetDefaultAdminCred(), grp.Id, userIds)
	return nil
}
