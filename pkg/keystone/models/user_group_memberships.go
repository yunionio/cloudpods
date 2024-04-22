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

package models

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SUsergroupManager struct {
	db.SResourceBaseManager
}

var UsergroupManager *SUsergroupManager

func init() {
	db.InitManager(func() {
		UsergroupManager = &SUsergroupManager{
			SResourceBaseManager: db.NewResourceBaseManager(
				SUsergroupMembership{},
				"user_group_membership",
				"usergroup",
				"usergroups",
			),
		}
		UsergroupManager.SetVirtualObject(UsergroupManager)
	})
}

/*
+----------+-------------+------+-----+---------+-------+
| Field    | Type        | Null | Key | Default | Extra |
+----------+-------------+------+-----+---------+-------+
| user_id  | varchar(64) | NO   | PRI | NULL    |       |
| group_id | varchar(64) | NO   | PRI | NULL    |       |
+----------+-------------+------+-----+---------+-------+
*/

// +onecloud:swagger-gen-ignore
type SUsergroupMembership struct {
	db.SResourceBase

	UserId  string `width:"64" charset:"ascii" nullable:"false" primary:"true"`
	GroupId string `width:"64" charset:"ascii" nullable:"false" primary:"true"`
}

func (membership *SUsergroupMembership) GetId() string {
	return fmt.Sprintf("%s-%s", membership.UserId, membership.GroupId)
}

func (membership *SUsergroupMembership) GetName() string {
	return fmt.Sprintf("%s-%s", membership.UserId, membership.GroupId)
}

func (manager *SUsergroupManager) getUserGroupIds(userId string) []string {
	members := make([]SUsergroupMembership, 0)
	q := manager.Query().Equals("user_id", userId)
	err := db.FetchModelObjects(manager, q, &members)
	if err != nil {
		log.Errorf("GetUserGroupIds fail %s", err)
		return nil
	}
	groupIds := make([]string, len(members))
	for i := range members {
		groupIds[i] = members[i].GroupId
	}
	return groupIds
}

func (manager *SUsergroupManager) getGroupUserIds(groupId string) []string {
	members := make([]SUsergroupMembership, 0)
	q := manager.Query().Equals("group_id", groupId)
	err := db.FetchModelObjects(manager, q, &members)
	if err != nil {
		log.Errorf("getGroupUserIds fail %s", err)
		return nil
	}
	userIds := make([]string, len(members))
	for i := range members {
		userIds[i] = members[i].UserId
	}
	return userIds
}

func (manager *SUsergroupManager) SyncUserGroups(ctx context.Context, userCred mcclient.TokenCredential, userId string, groupIds []string) {
	oldGroupIds := manager.getUserGroupIds(userId)
	sort.Strings(oldGroupIds)
	sort.Strings(groupIds)

	deleted, _, added := stringutils2.Split(stringutils2.SSortedStrings(oldGroupIds), stringutils2.SSortedStrings(groupIds))

	usr, _ := UserManager.fetchUserById(userId)
	for _, gid := range deleted {
		grp := GroupManager.fetchGroupById(gid)
		manager.remove(ctx, userCred, usr, grp)
	}
	for _, gid := range added {
		grp := GroupManager.fetchGroupById(gid)
		manager.add(ctx, userCred, usr, grp)
	}
}

func (manager *SUsergroupManager) SyncGroupUsers(ctx context.Context, userCred mcclient.TokenCredential, groupId string, userIds []string) {
	oldUserIds := manager.getGroupUserIds(groupId)
	sort.Strings(oldUserIds)
	sort.Strings(userIds)

	deleted, _, added := stringutils2.Split(stringutils2.SSortedStrings(oldUserIds), stringutils2.SSortedStrings(userIds))

	grp := GroupManager.fetchGroupById(groupId)
	if grp != nil {
		for _, uid := range deleted {
			usr, _ := UserManager.fetchUserById(uid)
			if usr != nil {
				manager.remove(ctx, userCred, usr, grp)
			}
		}
		for _, uid := range added {
			usr, _ := UserManager.fetchUserById(uid)
			if usr != nil {
				manager.add(ctx, userCred, usr, grp)
			}
		}
	}
}

func (manager *SUsergroupManager) remove(ctx context.Context, userCred mcclient.TokenCredential, usr *SUser, grp *SGroup) error {
	q := manager.Query().Equals("user_id", usr.Id).Equals("group_id", grp.Id)
	membership := SUsergroupMembership{}
	membership.SetModelManager(manager, &membership)
	err := q.First(&membership)
	if err != nil {
		return errors.Wrap(err, "Query")
	}
	_, err = db.Update(&membership, func() error {
		return membership.MarkDelete()
	})
	if err != nil {
		return errors.Wrap(err, "MarkDelete")
	}
	db.OpsLog.LogEvent(usr, db.ACT_DETACH, grp.GetShortDesc(ctx), userCred)
	return nil
}

func (manager *SUsergroupManager) add(ctx context.Context, userCred mcclient.TokenCredential, user *SUser, group *SGroup) error {
	q := manager.RawQuery().Equals("user_id", user.Id).Equals("group_id", group.Id)
	membership := SUsergroupMembership{}
	membership.SetModelManager(manager, &membership)
	err := q.First(&membership)
	if err != nil && err != sql.ErrNoRows {
		return errors.Wrap(err, "Query")
	}
	if err == nil {
		if membership.Deleted {
			_, err = db.Update(&membership, func() error {
				membership.Deleted = false
				return nil
			})
			if err != nil {
				return errors.Wrap(err, "Update to undelete")
			}
		} else {
			return nil
		}
	} else {
		// create one
		membership.UserId = user.Id
		membership.GroupId = group.Id
		err = manager.TableSpec().Insert(ctx, &membership)
		if err != nil {
			return errors.Wrap(err, "insert")
		}
	}
	db.OpsLog.LogEvent(user, db.ACT_ATTACH, group.GetShortDesc(ctx), userCred)
	return nil
}

func (manager *SUsergroupManager) delete(userId string, groupId string) error {
	q := manager.Query()
	if len(userId) > 0 {
		q = q.Equals("user_id", userId)
	}
	if len(groupId) > 0 {
		q = q.Equals("group_id", groupId)
	}
	memberships := make([]SUsergroupMembership, 0)
	err := db.FetchModelObjects(manager, q, &memberships)
	if err != nil {
		return errors.Wrap(err, "Query")
	}
	for i := range memberships {
		_, err = db.Update(&memberships[i], func() error {
			return memberships[i].MarkDelete()
		})
		if err != nil {
			return errors.Wrap(err, "MarkDelete")
		}
	}
	return nil
}
