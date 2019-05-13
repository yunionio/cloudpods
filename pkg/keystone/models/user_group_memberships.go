package models

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/pkg/errors"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

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

func (manager *SUsergroupManager) SyncUserGroups(ctx context.Context, userCred mcclient.TokenCredential, userId string, groupIds []string) {
	oldGroupIds := manager.getUserGroupIds(userId)
	sort.Strings(oldGroupIds)
	sort.Strings(groupIds)

	deleted, _, added := stringutils2.Split(stringutils2.SSortedStrings(oldGroupIds), stringutils2.SSortedStrings(groupIds))

	usr := UserManager.fetchUserById(userId)
	for _, gid := range deleted {
		grp := GroupManager.fetchGroupById(gid)
		manager.remove(ctx, userCred, usr, grp)
	}
	for _, gid := range added {
		grp := GroupManager.fetchGroupById(gid)
		manager.add(ctx, userCred, usr, grp)
	}
}

func (manager *SUsergroupManager) remove(ctx context.Context, userCred mcclient.TokenCredential, usr *SUser, grp *SGroup) error {
	q := manager.Query().Equals("user_id", usr.Id).Equals("group_id", grp.Id)
	membership := SUsergroupMembership{}
	membership.SetModelManager(manager)
	err := q.First(&membership)
	if err != nil {
		return errors.WithMessage(err, "Query")
	}
	_, err = db.Update(&membership, func() error {
		return membership.MarkDelete()
	})
	if err != nil {
		return errors.WithMessage(err, "MarkDelete")
	}
	db.OpsLog.LogEvent(usr, db.ACT_DETACH, grp.GetShortDesc(ctx), userCred)
	return nil
}

func (manager *SUsergroupManager) add(ctx context.Context, userCred mcclient.TokenCredential, user *SUser, group *SGroup) error {
	q := manager.RawQuery().Equals("user_id", user.Id).Equals("group_id", group.Id)
	membership := SUsergroupMembership{}
	membership.SetModelManager(manager)
	err := q.First(&membership)
	if err != nil && err != sql.ErrNoRows {
		return errors.WithMessage(err, "Query")
	}
	if err == nil {
		if membership.Deleted {
			_, err = db.Update(&membership, func() error {
				membership.Deleted = false
				return nil
			})
			if err != nil {
				return errors.WithMessage(err, "Update to undelete")
			}
		} else {
			return nil
		}
	} else {
		// create one
		membership.UserId = user.Id
		membership.GroupId = group.Id
		err = manager.TableSpec().Insert(&membership)
		if err != nil {
			return errors.WithMessage(err, "insert")
		}
	}
	db.OpsLog.LogEvent(user, db.ACT_ATTACH, group.GetShortDesc(ctx), userCred)
	return nil
}
