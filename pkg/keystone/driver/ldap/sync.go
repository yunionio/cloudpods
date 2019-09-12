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

	"gopkg.in/ldap.v3"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
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

func (self *SLDAPDriver) searchDomainEntries(cli *ldaputils.SLDAPClient, domainid string) ([]*ldap.Entry, error) {
	attrMap := make(map[string]string)
	if len(domainid) > 0 {
		attrMap[self.ldapConfig.DomainIdAttribute] = domainid
	}
	return cli.Search(self.getDomainTreeDN(),
		self.ldapConfig.DomainObjectclass,
		attrMap,
		self.ldapConfig.DomainFilter,
		self.domainAttributeList(),
		self.domainQueryScope(),
	)
}

func (self *SLDAPDriver) syncDomains(ctx context.Context, cli *ldaputils.SLDAPClient) error {
	entries, err := self.searchDomainEntries(cli, "")
	if err != nil {
		return errors.Wrap(err, "searchLDAP")
	}
	domainIds := make([]string, 0)
	for i := range entries {
		domainInfo := self.entry2Domain(entries[i])
		if !domainInfo.isValid() {
			continue
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
		err = obsoleteDomain.ValidateDeleteCondition(ctx)
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
			newName, err := db.GenerateName2(models.DomainManager, nil, info.Name, domain, 1)
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

	if self.AutoCreateProject && consts.GetNonDefaultDomainProjects() {
		project := &models.SProject{}
		project.SetModelManager(models.ProjectManager, project)
		projectName := models.NormalizeProjectName(fmt.Sprintf("%s_default_project", info.Name))
		newName, err := db.GenerateName(models.ProjectManager, nil, projectName)
		if err != nil {
			// ignore the error
			log.Errorf("db.GenerateName error %s for default domain project %s", err, projectName)
			newName = projectName
		}
		project.Name = newName
		project.DomainId = domain.Id
		project.Description = fmt.Sprintf("Default project for domain %s", info.Name)
		project.IsDomain = tristate.False
		project.ParentId = domain.Id
		err = models.ProjectManager.TableSpec().Insert(project)
		if err != nil {
			log.Errorf("models.ProjectManager.Insert fail %s", err)
		}
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
	userIds := make([]string, 0)
	userIdMap := make(map[string]string)
	for i := range entries {
		userInfo := self.entry2User(entries[i])
		if !userInfo.isValid() {
			continue
		}
		userId, err := self.syncUserDB(ctx, userInfo, domainId)
		if err != nil {
			return nil, errors.Wrap(err, "syncUserDB")
		}
		userIds = append(userIds, userId)
		if self.ldapConfig.GroupMembersAreIds {
			userIdMap[userInfo.Id] = userId
		} else {
			userIdMap[userInfo.DN] = userId
		}
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
		err = deleteUsers[i].ValidateDeleteCondition(ctx)
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
	q := models.UserManager.RawQuery().Equals("id", userId)
	err = q.First(user)
	if err != nil && err != sql.ErrNoRows {
		return errors.Wrap(err, "Query user")
	}
	if err == nil {
		// update
		_, err := db.Update(user, func() error {
			copyUserInfo(ui, userId, domainId, user)
			user.MarkUnDelete()
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

	log.Debugf("syncUserDB: %s", userId)

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
	groupIds := make([]string, 0)
	for i := range entries {
		groupInfo := self.entry2Group(entries[i])
		if !groupInfo.isValid() {
			continue
		}
		groupId, err := self.syncGroupDB(ctx, groupInfo, domainId, userIdMap)
		if err != nil {
			return errors.Wrap(err, "syncGroupDB")
		}
		groupIds = append(groupIds, groupId)
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
		err = deleteGroups[i].ValidateDeleteCondition(ctx)
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
