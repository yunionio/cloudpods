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
	"fmt"

	"gopkg.in/fatih/set.v0"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

// 绑定多个安全组
func (self *SGuest) PerformAddSecgroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.GuestAddSecgroupInput,
) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING, api.VM_SUSPEND}) {
		return nil, httperrors.NewInputParameterError("Cannot add security groups in status %s", self.Status)
	}

	maxCount := self.GetDriver().GetMaxSecurityGroupCount()
	if maxCount == 0 {
		return nil, httperrors.NewUnsupportOperationError("Cannot add security groups for hypervisor %s", self.Hypervisor)
	}

	if len(input.SecgroupIds) == 0 {
		return nil, httperrors.NewMissingParameterError("secgroup_ids")
	}

	secgroups, err := self.GetSecgroups()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "GetSecgroups"))
	}
	if len(secgroups)+len(input.SecgroupIds) > maxCount {
		return nil, httperrors.NewUnsupportOperationError("guest %s band to up to %d security groups", self.Name, maxCount)
	}

	secgroupIds := []string{}
	for _, secgroup := range secgroups {
		secgroupIds = append(secgroupIds, secgroup.Id)
	}

	vpc, err := self.GetVpc()
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpc")
	}

	secgroupNames := []string{}
	for i := range input.SecgroupIds {
		secObj, err := validators.ValidateModel(userCred, SecurityGroupManager, &input.SecgroupIds[i])
		if err != nil {
			return nil, err
		}
		secgroup := secObj.(*SSecurityGroup)

		if utils.IsInStringArray(secObj.GetId(), secgroupIds) {
			return nil, httperrors.NewInputParameterError("security group %s has already been assigned to guest %s", secObj.GetName(), self.Name)
		}

		err = vpc.CheckSecurityGroupConsistent(secgroup)
		if err != nil {
			return nil, err
		}

		secgroupIds = append(secgroupIds, secgroup.GetId())
		secgroupNames = append(secgroupNames, secgroup.Name)
	}

	err = self.SaveSecgroups(ctx, userCred, secgroupIds)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "saveSecgroups"))
	}

	notes := map[string][]string{"secgroups": secgroupNames}
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_VM_ASSIGNSECGROUP, notes, userCred, true)
	return nil, self.StartSyncTask(ctx, userCred, true, "")
}

func (self *SGuest) saveDefaultSecgroupId(userCred mcclient.TokenCredential, secGrpId string, isAdmin bool) error {
	if (!isAdmin && secGrpId != self.SecgrpId) || (isAdmin && secGrpId != self.AdminSecgrpId) {
		diff, err := db.Update(self, func() error {
			if isAdmin {
				self.AdminSecgrpId = secGrpId
			} else {
				self.SecgrpId = secGrpId
			}
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "db.Update")
		}
		db.OpsLog.LogEvent(self, db.ACT_UPDATE, diff, userCred)
	}
	return nil
}

func (self *SGuest) PerformRevokeSecgroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.GuestRevokeSecgroupInput,
) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING, api.VM_SUSPEND}) {
		return nil, httperrors.NewInputParameterError("Cannot revoke security groups in status %s", self.Status)
	}

	if len(input.SecgroupIds) == 0 {
		return nil, nil
	}

	secgroups, err := self.GetSecgroups()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "GetSecgroups"))
	}
	secgroupMaps := map[string]string{}
	for _, secgroup := range secgroups {
		secgroupMaps[secgroup.Id] = secgroup.Name
	}

	secgroupNames := []string{}
	for i := range input.SecgroupIds {
		secObj, err := validators.ValidateModel(userCred, SecurityGroupManager, &input.SecgroupIds[i])
		if err != nil {
			return nil, err
		}
		secgrp := secObj.(*SSecurityGroup)
		_, ok := secgroupMaps[secgrp.GetId()]
		if !ok {
			return nil, httperrors.NewInputParameterError("security group %s not assigned to guest %s", secgrp.GetName(), self.Name)
		}
		delete(secgroupMaps, secgrp.GetId())
		secgroupNames = append(secgroupNames, secgrp.GetName())
	}

	secgrpIds := []string{}
	for secgroupId := range secgroupMaps {
		secgrpIds = append(secgrpIds, secgroupId)
	}

	err = self.SaveSecgroups(ctx, userCred, secgrpIds)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "saveSecgroups"))
	}

	notes := map[string][]string{"secgroups": secgroupNames}
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_VM_REVOKESECGROUP, notes, userCred, true)
	return nil, self.StartSyncTask(ctx, userCred, true, "")
}

func (self *SGuest) PerformRevokeAdminSecgroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.GuestRevokeSecgroupInput,
) (jsonutils.JSONObject, error) {
	if !db.IsAdminAllowPerform(ctx, userCred, self, "revoke-admin-secgroup") {
		return nil, httperrors.NewForbiddenError("not allow to revoke admin secgroup")
	}

	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING, api.VM_SUSPEND}) {
		return nil, httperrors.NewInputParameterError("Cannot assign security rules in status %s", self.Status)
	}

	var notes string
	adminSecgrpId := ""
	if len(options.Options.DefaultAdminSecurityGroupId) > 0 {
		adminSecgrp, _ := SecurityGroupManager.FetchSecgroupById(options.Options.DefaultAdminSecurityGroupId)
		if adminSecgrp != nil {
			adminSecgrpId = adminSecgrp.Id
			notes = fmt.Sprintf("reset admin secgroup to %s(%s)", adminSecgrp.Name, adminSecgrp.Id)
		}
	}
	if adminSecgrpId == "" {
		notes = "clean admin secgroup"
	}

	err := self.saveDefaultSecgroupId(userCred, adminSecgrpId, true)
	if err != nil {
		return nil, errors.Wrap(err, "saveDefaultSecgroupId")
	}

	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_VM_REVOKESECGROUP, notes, userCred, true)
	return nil, self.StartSyncTask(ctx, userCred, true, "")
}

// +onecloud:swagger-gen-ignore
func (self *SGuest) PerformAssignSecgroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.GuestAssignSecgroupInput,
) (jsonutils.JSONObject, error) {
	return self.performAssignSecgroup(ctx, userCred, query, input, false)
}

func (self *SGuest) PerformAssignAdminSecgroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.GuestAssignSecgroupInput,
) (jsonutils.JSONObject, error) {
	if !db.IsAdminAllowPerform(ctx, userCred, self, "assign-admin-secgroup") {
		return nil, httperrors.NewForbiddenError("not allow to assign admin secgroup")
	}

	return self.performAssignSecgroup(ctx, userCred, query, input, true)
}

func (self *SGuest) performAssignSecgroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.GuestAssignSecgroupInput,
	isAdmin bool,
) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING, api.VM_SUSPEND}) {
		return nil, httperrors.NewInputParameterError("Cannot assign security rules in status %s", self.Status)
	}

	if len(input.SecgroupId) == 0 {
		return nil, httperrors.NewMissingParameterError("secgroup_id")
	}

	vpc, err := self.GetVpc()
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpc")
	}

	secObj, err := validators.ValidateModel(userCred, SecurityGroupManager, &input.SecgroupId)
	if err != nil {
		return nil, err
	}
	secgroup := secObj.(*SSecurityGroup)

	err = vpc.CheckSecurityGroupConsistent(secgroup)
	if err != nil {
		return nil, err
	}

	err = self.saveDefaultSecgroupId(userCred, input.SecgroupId, isAdmin)
	if err != nil {
		return nil, err
	}

	notes := map[string]string{"name": secObj.GetName(), "id": secObj.GetId(), "is_admin": fmt.Sprintf("%v", isAdmin)}
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_VM_ASSIGNSECGROUP, notes, userCred, true)
	return nil, self.StartSyncTask(ctx, userCred, true, "")
}

// 全量覆盖安全组
func (self *SGuest) PerformSetSecgroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.GuestSetSecgroupInput,
) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING, api.VM_SUSPEND}) {
		return nil, httperrors.NewInputParameterError("Cannot set security rules in status %s", self.Status)
	}
	if len(input.SecgroupIds) == 0 {
		return nil, httperrors.NewMissingParameterError("secgroup_ids")
	}

	maxCount := self.GetDriver().GetMaxSecurityGroupCount()
	if maxCount == 0 {
		return nil, httperrors.NewUnsupportOperationError("Cannot set security group for this guest %s", self.Name)
	}

	if len(input.SecgroupIds) > maxCount {
		return nil, httperrors.NewUnsupportOperationError("guest %s band to up to %d security groups", self.Name, maxCount)
	}

	vpc, err := self.GetVpc()
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpc")
	}

	secgroupIds := []string{}
	secgroupNames := []string{}
	for i := range input.SecgroupIds {
		secObj, err := validators.ValidateModel(userCred, SecurityGroupManager, &input.SecgroupIds[i])
		if err != nil {
			return nil, err
		}
		secgrp := secObj.(*SSecurityGroup)

		err = vpc.CheckSecurityGroupConsistent(secgrp)
		if err != nil {
			return nil, err
		}

		if !utils.IsInStringArray(secgrp.GetId(), secgroupIds) {
			secgroupIds = append(secgroupIds, secgrp.GetId())
			secgroupNames = append(secgroupNames, secgrp.GetName())
		}
	}

	err = self.SaveSecgroups(ctx, userCred, secgroupIds)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "saveSecgroups"))
	}

	notes := map[string][]string{"secgroups": secgroupNames}
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_VM_SETSECGROUP, notes, userCred, true)
	return nil, self.StartSyncTask(ctx, userCred, true, "")
}

func (self *SGuest) GetGuestSecgroups() ([]SGuestsecgroup, error) {
	gss := []SGuestsecgroup{}
	q := GuestsecgroupManager.Query().Equals("guest_id", self.Id)
	err := db.FetchModelObjects(GuestsecgroupManager, q, &gss)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return gss, nil
}

func (self *SGuest) SaveSecgroups(ctx context.Context, userCred mcclient.TokenCredential, secgroupIds []string) error {
	if len(secgroupIds) == 0 {
		return self.RevokeAllSecgroups(ctx, userCred)
	}
	oldIds := set.New(set.ThreadSafe)
	newIds := set.New(set.ThreadSafe)
	gss, err := self.GetGuestSecgroups()
	if err != nil {
		return errors.Wrapf(err, "GetGuestSecgroups")
	}
	secgroupMaps := map[string]SGuestsecgroup{}
	for i := range gss {
		oldIds.Add(gss[i].SecgroupId)
		secgroupMaps[gss[i].SecgroupId] = gss[i]
	}
	for i := 1; i < len(secgroupIds); i++ {
		newIds.Add(secgroupIds[i])
	}
	for _, removed := range set.Difference(oldIds, newIds).List() {
		id := removed.(string)
		gs, ok := secgroupMaps[id]
		if ok {
			err = gs.Delete(ctx, userCred)
			if err != nil {
				return errors.Wrapf(err, "Delete guest secgroup for guest %s secgroup %s", self.Name, id)
			}
		}
	}
	for _, added := range set.Difference(newIds, oldIds).List() {
		id := added.(string)
		err = self.newGuestSecgroup(ctx, id)
		if err != nil {
			return errors.Wrapf(err, "New guest secgroup for guest %s with secgroup %s", self.Name, id)
		}
	}
	return self.saveDefaultSecgroupId(userCred, secgroupIds[0], false)
}

func (self *SGuest) newGuestSecgroup(ctx context.Context, secgroupId string) error {
	gs := &SGuestsecgroup{}
	gs.SetModelManager(GuestsecgroupManager, gs)
	gs.GuestId = self.Id
	gs.SecgroupId = secgroupId
	return GuestsecgroupManager.TableSpec().Insert(ctx, gs)
}

func (self *SGuest) RevokeAllSecgroups(ctx context.Context, userCred mcclient.TokenCredential) error {
	gss, err := self.GetGuestSecgroups()
	if err != nil {
		return errors.Wrapf(err, "GetGuestSecgroups")
	}
	for i := range gss {
		err = gss[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "Delete")
		}
	}
	return self.saveDefaultSecgroupId(userCred, options.Options.DefaultSecurityGroupId, false)
}
