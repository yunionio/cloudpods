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
	"regexp"
	"time"

	"golang.org/x/text/language"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/informer"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/notify/oldmodels"
	"yunion.io/x/onecloud/pkg/notify/options"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	PersonalConfigContactTypes = []string{
		api.EMAIL,
		api.MOBILE,
		api.DINGTALK,
		api.FEISHU,
		api.WORKWX,
	}
	RobotContactTypes = []string{
		api.FEISHU_ROBOT,
		api.DINGTALK_ROBOT,
		api.WORKWX_ROBOT,
	}
	SystemConfigContactTypes = append(
		RobotContactTypes,
		api.WEBCONSOLE,
		api.WEBHOOK,
	)
)

type SReceiverManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SDomainizedResourceBaseManager
	db.SEnabledResourceBaseManager
}

var ReceiverManager *SReceiverManager

func init() {
	ReceiverManager = &SReceiverManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SReceiver{},
			"receivers_tbl",
			"receiver",
			"receivers",
		),
	}
	ReceiverManager.SetVirtualObject(ReceiverManager)
}

type SReceiver struct {
	db.SStatusStandaloneResourceBase
	db.SDomainizedResourceBase
	db.SEnabledResourceBase

	Email string `width:"64" nullable:"false" create:"optional" update:"user" get:"user" list:"user"`
	// swagger:ignore
	Mobile string `width:"32" nullable:"false" create:"optional"`
	Lang   string `width:"8" charset:"ascii" nullable:"false" list:"user" update:"user"`

	// swagger:ignore
	EnabledEmail tristate.TriState `nullable:"false" default:"false" update:"user"`
	// swagger:ignore
	VerifiedEmail tristate.TriState `nullable:"false" default:"false" update:"user"`

	// swagger:ignore
	EnabledMobile tristate.TriState `nullable:"false" default:"false" update:"user"`
	// swagger:ignore
	VerifiedMobile tristate.TriState `nullable:"false" default:"false" update:"user"`

	// swagger:ignore
	subContactCache map[string]*SSubContact `json:"-"`
}

func (rm *SReceiverManager) InitializeData() error {
	ctx := context.Background()
	userCred := auth.AdminCredential()
	log.Infof("Init Receiver...")
	// Fetch all old SContact
	q := oldmodels.ContactManager.Query()
	contacts := make([]oldmodels.SContact, 0, 50)
	err := db.FetchModelObjects(oldmodels.ContactManager, q, &contacts)
	if err != nil {
		return errors.Wrap(err, "db.FetchModelObjects")
	}
	if len(contacts) == 0 {
		return nil
	}

	// build uid map
	uids := make([]string, 0, 10)
	contactMap := make(map[string][]*oldmodels.SContact, 10)
	for i := range contacts {
		uid := contacts[i].UID
		if _, ok := contactMap[uid]; !ok {
			contactMap[uid] = make([]*oldmodels.SContact, 0, 4)
			uids = append(uids, uid)
		}
		contactMap[uid] = append(contactMap[uid], &contacts[i])
	}

	// build uid->uname map
	userMap, err := oldmodels.UserCacheManager.FetchUsersByIDs(context.Background(), uids)
	if err != nil {
		return errors.Wrap(err, "oldmodels.UserCacheManager.FetchUsersByIDs")
	}

	// build Receivers
	for uid, contacts := range contactMap {
		var receiver SReceiver
		receiver.subContactCache = make(map[string]*SSubContact)
		receiver.Enabled = tristate.True
		receiver.Status = api.RECEIVER_STATUS_READY
		receiver.Id = uid
		user, ok := userMap[uid]
		if !ok {
			log.Errorf("no user %q in usercache", uid)
		} else {
			receiver.Name = user.Name
			receiver.DomainId = user.DomainId
		}
		for _, contact := range contacts {
			switch contact.ContactType {
			case api.EMAIL:
				receiver.Email = contact.Contact
				if contact.Enabled == "1" {
					receiver.EnabledEmail = tristate.True
				} else {
					receiver.EnabledEmail = tristate.False
				}
				if contact.Status == oldmodels.CONTACT_VERIFIED {
					receiver.VerifiedEmail = tristate.True
				} else {
					receiver.VerifiedEmail = tristate.False
				}
			case api.MOBILE:
				receiver.Mobile = contact.Contact
				if contact.Enabled == "1" {
					receiver.EnabledMobile = tristate.True
				} else {
					receiver.EnabledMobile = tristate.False
				}
				if contact.Status == oldmodels.CONTACT_VERIFIED {
					receiver.VerifiedMobile = tristate.True
				} else {
					receiver.VerifiedMobile = tristate.False
				}
			case api.WEBCONSOLE:
			default:
				var subContact SSubContact
				subContact.Type = contact.ContactType
				subContact.ParentContactType = api.MOBILE
				subContact.Contact = contact.Contact
				subContact.ReceiverID = uid
				subContact.ParentContactType = api.MOBILE
				if contact.Enabled == "1" {
					subContact.Enabled = tristate.True
				} else {
					subContact.Enabled = tristate.False
				}
				if contact.Status == oldmodels.CONTACT_VERIFIED && len(contact.Contact) > 0 {
					subContact.Verified = tristate.True
				} else {
					subContact.Verified = tristate.False
				}
				receiver.subContactCache[contact.ContactType] = &subContact
			}
		}
		err := rm.TableSpec().InsertOrUpdate(ctx, &receiver)
		if err != nil {
			return errors.Wrap(err, "InsertOrUpdate")
		}
		err = receiver.PushCache(ctx)
		if err != nil {
			return errors.Wrap(err, "PushCache")
		}
		//delete old one
		for _, contact := range contacts {
			err := contact.Delete(ctx, userCred)
			if err != nil {
				return errors.Wrap(err, "Delete")
			}
		}
	}
	return nil
}

func (rm *SReceiverManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.ReceiverCreateInput) (api.ReceiverCreateInput, error) {
	var err error
	input.StatusStandaloneResourceCreateInput, err = rm.SStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusStandaloneResourceCreateInput)
	if err != nil {
		return input, err
	}
	// check uid
	session := auth.GetAdminSession(ctx, "", "")
	if len(input.UID) > 0 {
		userObj, err := modules.UsersV3.GetById(session, input.UID, nil)
		if err != nil {
			if jErr, ok := err.(*httputils.JSONClientError); ok {
				if jErr.Code == 404 {
					return input, httperrors.NewInputParameterError("no such user")
				}
			}
			return input, err
		}
		uname, _ := userObj.GetString("name")
		uid, _ := userObj.GetString("id")
		input.UID = uid
		input.UName = uname
		domainId, _ := userObj.GetString("domain_id")
		input.ProjectDomainId = domainId
	} else {
		if len(input.UName) == 0 {
			return input, httperrors.NewMissingParameterError("uid or uname")
		} else {
			userObj, err := modules.UsersV3.GetByName(session, input.UName, nil)
			if err != nil {
				if jErr, ok := err.(*httputils.JSONClientError); ok {
					if jErr.Code == 404 {
						return input, httperrors.NewInputParameterError("no such user")
					}
				}
				return input, err
			}
			uid, _ := userObj.GetString("id")
			uname, _ := userObj.GetString("name")
			input.UID = uid
			input.UName = uname
			domainId, _ := userObj.GetString("domain_id")
			input.ProjectDomainId = domainId
		}
	}
	// hack
	input.Name = input.UName
	// validate email
	if ok := regutils.MatchEmail(input.Email); len(input.Email) > 0 && !ok {
		return input, httperrors.NewInputParameterError("invalid email")
	}
	// validate mobile
	if ok := LaxMobileRegexp.MatchString(input.InternationalMobile.Mobile); len(input.InternationalMobile.Mobile) > 0 && !ok {
		return input, httperrors.NewInputParameterError("invalid mobile")
	}
	return input, nil
}

var LaxMobileRegexp = regexp.MustCompile(`[0-9]{6,14}`)

func (r *SReceiver) IsEnabledContactType(ct string) (bool, error) {
	if utils.IsInStringArray(ct, SystemConfigContactTypes) {
		return true, nil
	}
	cts, err := r.GetEnabledContactTypes()
	if err != nil {
		return false, errors.Wrap(err, "GetEnabledContactTypes")
	}
	return utils.IsInStringArray(ct, cts), nil
}

func (r *SReceiver) IsVerifiedContactType(ct string) (bool, error) {
	if utils.IsInStringArray(ct, SystemConfigContactTypes) {
		return true, nil
	}
	cts, err := r.GetVerifiedContactTypes()
	if err != nil {
		return false, errors.Wrap(err, "GetVerifiedContactTypes")
	}
	return utils.IsInStringArray(ct, cts), nil
}

func (r *SReceiver) GetEnabledContactTypes() ([]string, error) {
	if err := r.PullCache(false); err != nil {
		return nil, err
	}
	ret := make([]string, 0, 1)
	// for email and mobile
	if r.EnabledEmail.IsTrue() {
		ret = append(ret, api.EMAIL)
	}
	if r.EnabledMobile.IsTrue() {
		ret = append(ret, api.MOBILE)
	}
	for subct, subc := range r.subContactCache {
		if subc.Enabled.IsTrue() {
			ret = append(ret, subct)
		}
	}
	ret = append(ret, api.WEBCONSOLE)
	return ret, nil
}

func (r *SReceiver) setEnabledContactType(contactType string, enabled bool) {
	switch contactType {
	case api.EMAIL:
		r.EnabledEmail = tristate.NewFromBool(enabled)
	case api.MOBILE:
		r.EnabledMobile = tristate.NewFromBool(enabled)
	default:
		if sc, ok := r.subContactCache[contactType]; ok {
			sc.Enabled = tristate.NewFromBool(enabled)
		} else {
			subContact := &SSubContact{
				Type:       contactType,
				ReceiverID: r.Id,
				Enabled:    tristate.NewFromBool(enabled),
			}
			subContact.ParentContactType = api.MOBILE
			r.subContactCache[contactType] = subContact
		}
	}
}

func (r *SReceiver) SetEnabledContactTypes(contactTypes []string) error {
	if err := r.PullCache(false); err != nil {
		return err
	}
	ctSet := sets.NewString(contactTypes...)
	for _, ct := range PersonalConfigContactTypes {
		if ctSet.Has(ct) {
			r.setEnabledContactType(ct, true)
		} else {
			r.setEnabledContactType(ct, false)
		}
	}
	return nil
}

func (r *SReceiver) MarkContactTypeVerified(contactType string) error {
	if err := r.PullCache(false); err != nil {
		return err
	}
	if sc, ok := r.subContactCache[contactType]; ok {
		sc.Verified = tristate.True
	} else {
		subContact := &SSubContact{
			Type:       contactType,
			ReceiverID: r.Id,
			Verified:   tristate.True,
		}
		subContact.ParentContactType = api.MOBILE
		subContact.VerifiedNote = ""
		r.subContactCache[contactType] = subContact
	}
	return nil
}

func (r *SReceiver) MarkContactTypeUnVerified(contactType string, note string) error {
	if err := r.PullCache(false); err != nil {
		return err
	}
	if sc, ok := r.subContactCache[contactType]; ok {
		sc.Verified = tristate.False
		sc.VerifiedNote = note
	} else {
		subContact := &SSubContact{
			Type:         contactType,
			ReceiverID:   r.Id,
			VerifiedNote: note,
			Verified:     tristate.False,
		}
		subContact.ParentContactType = api.MOBILE
		r.subContactCache[contactType] = subContact
	}
	return nil
}

func (r *SReceiver) setVerifiedContactType(contactType string, enabled bool) {
	switch contactType {
	case api.EMAIL:
		r.VerifiedEmail = tristate.NewFromBool(enabled)
	case api.MOBILE:
		r.VerifiedMobile = tristate.NewFromBool(enabled)
	default:
		if sc, ok := r.subContactCache[contactType]; ok {
			sc.Verified = tristate.NewFromBool(enabled)
		} else {
			subContact := &SSubContact{
				Type:       contactType,
				ReceiverID: r.Id,
				Verified:   tristate.NewFromBool(enabled),
			}
			subContact.ParentContactType = api.MOBILE
			r.subContactCache[contactType] = subContact
		}
	}
}

func (r *SReceiver) getVerifiedInfos() ([]api.VerifiedInfo, error) {
	if err := r.PullCache(false); err != nil {
		return nil, err
	}
	infos := []api.VerifiedInfo{
		{
			ContactType: api.EMAIL,
			Verified:    r.VerifiedEmail.Bool(),
		},
		{
			ContactType: api.MOBILE,
			Verified:    r.VerifiedMobile.Bool(),
		},
		{
			ContactType: api.WEBCONSOLE,
			Verified:    true,
		},
	}
	for subct, subc := range r.subContactCache {
		infos = append(infos, api.VerifiedInfo{
			ContactType: subct,
			Verified:    subc.Verified.Bool(),
			Note:        subc.VerifiedNote,
		})
	}
	return infos, nil
}

func (r *SReceiver) GetVerifiedContactTypes() ([]string, error) {
	if err := r.PullCache(false); err != nil {
		return nil, err
	}
	ret := make([]string, 0, 1)
	// for email and mobile
	if r.VerifiedEmail.IsTrue() {
		ret = append(ret, api.EMAIL)
	}
	if r.VerifiedMobile.IsTrue() {
		ret = append(ret, api.MOBILE)
	}
	for subct, subc := range r.subContactCache {
		if subc.Verified.IsTrue() {
			ret = append(ret, subct)
		}
	}
	return ret, nil
}

func (r *SReceiver) SetVerifiedContactTypes(contactTypes []string) error {
	if err := r.PullCache(false); err != nil {
		return err
	}
	ctSet := sets.NewString(contactTypes...)
	for _, ct := range PersonalConfigContactTypes {
		if ctSet.Has(ct) {
			r.setVerifiedContactType(ct, true)
		} else {
			r.setVerifiedContactType(ct, false)
		}
	}
	return nil
}

func (r *SReceiver) PullCache(force bool) error {
	if !force && r.subContactCache != nil {
		return nil
	}
	cache, err := SubContactManager.fetchMapByReceiverID(r.Id)
	if err != nil {
		return err
	}
	r.subContactCache = cache
	return nil
}

func (r *SReceiver) PushCache(ctx context.Context) error {
	for subct, subc := range r.subContactCache {
		err := SubContactManager.TableSpec().InsertOrUpdate(ctx, subc)
		if err != nil {
			return errors.Wrapf(err, "fail to save subcontact %q to db", subct)
		}
	}
	return nil
}

func (rm *SReceiverManager) EnabledContactFilter(contactType string, q *sqlchemy.SQuery) *sqlchemy.SQuery {
	switch contactType {
	case api.MOBILE:
		q = q.IsTrue("enabled_mobile")
	case api.EMAIL:
		q = q.IsTrue("enabled_email")
	default:
		subQuery := SubContactManager.Query("receiver_id").Equals("type", contactType).IsTrue("enabled").SubQuery()
		q = q.Join(subQuery, sqlchemy.Equals(subQuery.Field("receiver_id"), q.Field("id")))
	}
	return q
}

func (rm *SReceiverManager) VerifiedContactFilter(contactType string, q *sqlchemy.SQuery) *sqlchemy.SQuery {
	switch contactType {
	case api.MOBILE:
		q = q.IsTrue("verified_mobile")
	case api.EMAIL:
		q = q.IsTrue("verified_email")
	default:
		subQuery := SubContactManager.Query("receiver_id").Equals("type", contactType).IsTrue("verified").SubQuery()
		q = q.Join(subQuery, sqlchemy.Equals(subQuery.Field("receiver_id"), q.Field("id")))

	}
	return q
}

func (rm *SReceiverManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeUser
}

func (rm *SReceiverManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	userStr, _ := jsonutils.GetAnyString2(data, []string{"uid"})
	if len(userStr) > 0 {
		u, err := db.DefaultUserFetcher(ctx, userStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("user", userStr)
			}
			return nil, errors.Wrap(err, "UserCacheManager.FetchUserByIdOrName")
		}
		ownerId := db.SOwnerId{
			DomainId:     u.DomainId,
			Domain:       u.Domain,
			UserDomain:   u.Domain,
			UserDomainId: u.DomainId,
			UserId:       u.Id,
			User:         u.Name,
		}
		return &ownerId, nil
	}
	return db.FetchDomainInfo(ctx, data)
}

func (rm *SReceiverManager) filterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if owner == nil {
		return q
	}
	switch scope {
	case rbacutils.ScopeDomain:
		q = q.Equals("domain_id", owner.GetProjectDomainId())
	case rbacutils.ScopeProject, rbacutils.ScopeUser:
		q = q.Equals("id", owner.GetUserId())
	}
	return q
}

func (rm *SReceiverManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	return q
}

func (rm *SReceiverManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.ReceiverListInput) (*sqlchemy.SQuery, error) {
	q, err := rm.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	q, err = rm.SDomainizedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.DomainizedResourceListInput)
	if err != nil {
		return nil, err
	}
	q, err = rm.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, input.EnabledResourceBaseListInput)
	if err != nil {
		return nil, err
	}
	if len(input.UID) > 0 {
		q = q.Equals("id", input.UID)
	}
	if len(input.UName) > 0 {
		q = q.Equals("name", input.UName)
	}
	if len(input.EnabledContactType) > 0 {
		q = rm.EnabledContactFilter(input.EnabledContactType, q)
	}
	if len(input.VerifiedContactType) > 0 {
		q = rm.VerifiedContactFilter(input.VerifiedContactType, q)
	}
	if input.ProjectDomainFilter && userCred.GetProjectDomainId() != "" {
		userIds, err := rm.findUserIdsWithProjectDomain(ctx, userCred, userCred.GetProjectDomainId())
		if err != nil {
			return nil, errors.Wrap(err, "unable to findUserIdsWithProjectDomain")
		}
		switch len(userIds) {
		case 0:
			q = q.Equals("id", "")
		case 1:
			q = q.Equals("id", userIds[0])
		default:
			q = q.In("id", userIds)
		}
	} else {
		ownerId, queryScope, err := db.FetchCheckQueryOwnerScope(ctx, userCred, jsonutils.Marshal(input), rm, policy.PolicyActionList, true)
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
		q = rm.filterByOwner(q, ownerId, queryScope)
	}
	return q, nil
}

func (rm *SReceiverManager) findUserIdsWithProjectDomain(ctx context.Context, userCred mcclient.TokenCredential, projectDomainId string) ([]string, error) {
	session := auth.GetSession(ctx, userCred, "", "")
	query := jsonutils.NewDict()
	query.Set("effective", jsonutils.JSONTrue)
	query.Set("project_domain_id", jsonutils.NewString(projectDomainId))
	listRet, err := modules.RoleAssignments.List(session, query)
	if err != nil {
		return nil, errors.Wrap(err, "unable to list RoleAssignments")
	}
	log.Debugf("return value for role-assignments: %s", jsonutils.Marshal(listRet))
	userIds := sets.NewString()
	for i := range listRet.Data {
		ras := listRet.Data[i]
		user, err := ras.Get("user")
		if err == nil {
			id, err := user.GetString("id")
			if err != nil {
				return nil, errors.Wrap(err, "unable to get user.id from result of RoleAssignments.List")
			}
			userIds.Insert(id)
		}
	}
	return userIds.UnsortedList(), nil
}

func (r *SReceiverManager) AllowPerformGetTypes(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (cm *SReceiverManager) PerformGetTypes(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ConfigManagerGetTypesInput) (api.ConfigManagerGetTypesOutput, error) {
	output := api.ConfigManagerGetTypesOutput{}
	allContactType, err := ConfigManager.allContactType()
	if err != nil {
		return output, err
	}
	output.Types = sortContactType(ConfigManager.filterContactType(allContactType, input.Robot))
	return output, nil
}

func (rm *SReceiverManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []api.ReceiverDetails {
	sRows := rm.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	dRows := rm.SDomainizedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	rows := make([]api.ReceiverDetails, len(objs))
	var err error
	for i := range rows {
		rows[i].StatusStandaloneResourceDetails = sRows[i]
		rows[i].DomainizedResourceInfo = dRows[i]
		user := objs[i].(*SReceiver)
		rows[i].InternationalMobile = api.ParseInternationalMobile(user.Mobile)
		if enabledCTs, err := user.GetEnabledContactTypes(); err != nil {
			log.Errorf("GetEnabledContactTypes: %v", err)
		} else {
			rows[i].EnabledContactTypes = sortContactType(enabledCTs)
		}
		if rows[i].VerifiedInfos, err = user.getVerifiedInfos(); err != nil {
			log.Errorf("GetVerifiedContactTypes: %v", err)
		}
	}
	return rows
}

func (rm *SReceiverManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := rm.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err != nil {
		return nil, err
	}
	q, err = rm.SDomainizedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (rm *SReceiverManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.ReceiverListInput) (*sqlchemy.SQuery, error) {
	q, err := rm.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	q, err = rm.SDomainizedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.DomainizedResourceListInput)
	return q, nil
}

func (r *SReceiver) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	r.SStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	// set status
	r.SetStatus(userCred, api.RECEIVER_STATUS_PULLING, "")
	logclient.AddActionLogWithContext(ctx, r, logclient.ACT_CREATE, nil, userCred, true)
	err := r.StartSubcontactPullTask(ctx, userCred, nil, "")
	if err != nil {
		log.Errorf("unable to StartSubcontactPullTask: %v", err)
	}
}

func (r *SReceiver) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	err := r.SStatusStandaloneResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
	if err != nil {
		return nil
	}
	var input api.ReceiverCreateInput
	err = data.Unmarshal(&input)
	if err != nil {
		return err
	}
	// set id and name
	r.Id = input.UID
	r.Name = input.UName
	r.DomainId = input.ProjectDomainId
	if input.Enabled == nil {
		r.Enabled = tristate.True
	}
	r.Mobile = input.InternationalMobile.String()
	err = r.SetEnabledContactTypes(input.EnabledContactTypes)
	if err != nil {
		return errors.Wrap(err, "SetEnabledContactTypes")
	}
	err = r.PushCache(ctx)
	if err != nil {
		return errors.Wrap(err, "PushCache")
	}
	// 需求：管理后台新建的联系人，手机号和邮箱无需进行校验
	// 方案：检查请求者对于创建联系人 是否具有system scope
	allowScope := policy.PolicyManager.AllowScope(userCred, api.SERVICE_TYPE, ReceiverManager.KeywordPlural(), policy.PolicyActionCreate)
	if allowScope == rbacutils.ScopeSystem {
		if r.EnabledEmail.Bool() {
			r.VerifiedEmail = tristate.True
		}
		if r.EnabledMobile.Bool() {
			r.VerifiedMobile = tristate.True
		}
	}
	return nil
}

func (r *SReceiver) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ReceiverUpdateInput) (api.ReceiverUpdateInput, error) {
	var err error
	input.StatusStandaloneResourceBaseUpdateInput, err = r.SStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.StatusStandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, err
	}
	// validate email
	if ok := len(input.Email) == 0 || regutils.MatchEmail(input.Email); !ok {
		return input, httperrors.NewInputParameterError("invalid email")
	}
	// validate mobile
	if ok := len(input.InternationalMobile.Mobile) == 0 || LaxMobileRegexp.MatchString(input.InternationalMobile.Mobile); !ok {
		return input, httperrors.NewInputParameterError("invalid mobile")
	}
	return input, nil
}

func (r *SReceiver) PreUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	r.SStatusStandaloneResourceBase.PreUpdate(ctx, userCred, query, data)
	originEmailEnable, originMobileEnable := r.EnabledEmail, r.EnabledMobile
	var input api.ReceiverUpdateInput
	err := data.Unmarshal(&input)
	if err != nil {
		log.Errorf("fail to unmarshal to ContactUpdateInput: %v", err)
	}
	err = r.PullCache(false)
	if err != nil {
		log.Errorf("PullCache: %v", err)
	}
	err = r.SetEnabledContactTypes(input.EnabledContactTypes)
	if err != nil {
		log.Errorf("unable to SetEnabledContactTypes")
	}
	if len(input.Email) != 0 && input.Email != r.Email {
		r.VerifiedEmail = tristate.False
		for _, c := range r.subContactCache {
			if c.ParentContactType == api.EMAIL {
				c.Verified = tristate.False
				c.VerifiedNote = "email changed, re-verify"
			}
		}
	}
	mobile := input.InternationalMobile.String()
	if len(mobile) != 0 && mobile != r.Mobile {
		r.VerifiedMobile = tristate.False
		r.Mobile = mobile
		for _, c := range r.subContactCache {
			if c.ParentContactType == api.MOBILE {
				c.Verified = tristate.False
				c.VerifiedNote = "mobile changed, re-verify"
			}
		}
	}
	err = r.PushCache(ctx)
	if err != nil {
		log.Errorf("PushCache: %v", err)
	}
	// 管理后台修改联系人，如果修改或者启用手机号和邮箱，无需进行校验
	allowScope := policy.PolicyManager.AllowScope(userCred, api.SERVICE_TYPE, ReceiverManager.KeywordPlural(), policy.PolicyActionCreate)
	if allowScope == rbacutils.ScopeSystem {
		// 修改并启用
		if len(input.Email) != 0 && input.Email != r.Email && r.EnabledEmail.Bool() {
			r.VerifiedEmail = tristate.True
		}
		if len(mobile) != 0 && mobile != r.Mobile && r.EnabledMobile.Bool() {
			r.VerifiedMobile = tristate.True
		}
		// 从禁用变启用
		if !originEmailEnable.Bool() && r.EnabledEmail.Bool() {
			r.VerifiedEmail = tristate.True
		}
		if !originMobileEnable.Bool() && r.EnabledMobile.Bool() {
			r.VerifiedMobile = tristate.True
		}
	}
	err = ReceiverManager.TableSpec().InsertOrUpdate(ctx, r)
	if err != nil {
		log.Errorf("InsertOrUpdate: %v", err)
	}
}

func (r *SReceiver) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	r.SStatusStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)
	// set status
	r.SetStatus(userCred, api.RECEIVER_STATUS_PULLING, "")
	logclient.AddActionLogWithContext(ctx, r, logclient.ACT_UPDATE, nil, userCred, true)
	err := r.StartSubcontactPullTask(ctx, userCred, nil, "")
	if err != nil {
		log.Errorf("unable to StartSubcontactPullTask: %v", err)
	}

}

func (r *SReceiver) StartSubcontactPullTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "SubcontactPullTask", r, userCred, params, parentTaskId, "")
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (r *SReceiver) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := r.PullCache(false)
	if err != nil {
		return err
	}
	for _, sc := range r.subContactCache {
		err := sc.Delete(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return r.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (r *SReceiver) IsOwner(userCred mcclient.TokenCredential) bool {
	return r.Id == userCred.GetUserId()
}

func (rm *SReceiverManager) AllowPerformIntellijGet(_ context.Context, userCred mcclient.TokenCredential, _ jsonutils.JSONObject) bool {
	return true
}

func (rm *SReceiverManager) PerformIntellijGet(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ReceiverIntellijGetInput) (jsonutils.JSONObject, error) {
	getParam := jsonutils.NewDict()
	getParam.Set("scope", jsonutils.NewString(input.Scope))
	// try to get itself
	s := auth.GetSession(ctx, userCred, "", "")
	// modules.NotifyReceiver
	ret, err := modules.NotifyReceiver.Get(s, input.UserId, getParam)
	if err == nil {
		return ret, nil
	}
	jerr, ok := err.(*httputils.JSONClientError)
	if !ok {
		return nil, err
	}
	if jerr.Code != 404 {
		return nil, errors.Wrapf(err, "unable to get NotifyReceiver via id %q", input.UserId)
	}
	if input.CreateIfNo == nil || !*input.CreateIfNo {
		return jsonutils.NewDict(), nil
	}
	// create one
	adminSession := auth.GetAdminSession(ctx, "", "")
	ret, err = modules.UsersV3.GetById(adminSession, input.UserId, getParam)
	if err != nil {
		return nil, errors.Wrap(err, "unable get user from keystone")
	}
	id, _ := ret.GetString("id")
	name, _ := ret.GetString("name")
	r := &SReceiver{}
	r.Id = id
	r.Name = name
	r.SetModelManager(rm, r)
	err = rm.TableSpec().InsertOrUpdate(ctx, r)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create receiver")
	}
	rets, err := db.FetchCustomizeColumns(rm, ctx, userCred, jsonutils.NewDict(), []interface{}{r}, stringutils2.SSortedStrings{}, false)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get details of receiver %q", id)
	}
	if len(rets) == 0 {
		return nil, errors.Wrapf(err, "details of receiver %q is empty", id)
	}
	return rets[0], nil
}

func (r *SReceiver) AllowPerformTriggerVerify(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return r.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, r, "trigger_verify")
}

func (r *SReceiver) PerformTriggerVerify(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ReceiverTriggerVerifyInput) (jsonutils.JSONObject, error) {
	if len(input.ContactType) == 0 {
		return nil, httperrors.NewMissingParameterError("contact_type")
	}
	if !utils.IsInStringArray(input.ContactType, []string{api.EMAIL, api.MOBILE, api.DINGTALK, api.FEISHU, api.WORKWX}) {
		return nil, httperrors.NewInputParameterError("not support such contact type %q", input.ContactType)
	}
	if utils.IsInStringArray(input.ContactType, []string{api.DINGTALK, api.FEISHU, api.WORKWX}) {
		r.SetStatus(userCred, api.RECEIVER_STATUS_PULLING, "")
		params := jsonutils.NewDict()
		params.Set("contact_types", jsonutils.NewArray(jsonutils.NewString(input.ContactType)))
		return nil, r.StartSubcontactPullTask(ctx, userCred, params, "")
	}
	_, err := VerificationManager.Create(ctx, r.Id, input.ContactType)
	if err == ErrVerifyFrequently {
		return nil, httperrors.NewForbiddenError("Send verify message too frequently, please try again later")
	}
	if err != nil {
		return nil, err
	}

	params := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewTask(ctx, "VerificationSendTask", r, userCred, params, "", "")
	if err != nil {
		log.Errorf("ContactPullTask newTask error %v", err)
	} else {
		task.ScheduleRun(nil)
	}
	return nil, nil
}

func (r *SReceiver) AllowPerformVerify(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return r.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, r, "verify")
}

func (r *SReceiver) PerformVerify(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ReceiverVerifyInput) (jsonutils.JSONObject, error) {
	if len(input.ContactType) == 0 {
		return nil, httperrors.NewMissingParameterError("contact_type")
	}
	if !utils.IsInStringArray(input.ContactType, []string{api.EMAIL, api.MOBILE}) {
		return nil, httperrors.NewInputParameterError("not support such contact type %q", input.ContactType)
	}
	verification, err := VerificationManager.Get(r.Id, input.ContactType)
	if err != nil {
		return nil, err
	}
	if verification.CreatedAt.Add(time.Duration(options.Options.VerifyValidInterval) * time.Minute).Before(time.Now()) {
		return nil, httperrors.NewForbiddenError("The validation expires, please retrieve the verification code again")
	}
	if verification.Token != input.Token {
		return nil, httperrors.NewInputParameterError("wrong token")
	}
	_, err = db.Update(r, func() error {
		switch input.ContactType {
		case api.EMAIL:
			r.VerifiedEmail = tristate.True
		case api.MOBILE:
			r.VerifiedMobile = tristate.True
		default:
			// no way
		}
		return nil
	})
	return nil, err
}

func (r *SReceiver) AllowPerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) bool {
	return r.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, r, "enable")
}

func (r *SReceiver) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(r, ctx, userCred, true)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (r *SReceiver) AllowPerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) bool {
	return r.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, r, "disable")
}

func (r *SReceiver) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(r, ctx, userCred, false)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (r *SReceiver) Sync(ctx context.Context) error {
	session := auth.GetAdminSessionWithInternal(ctx, "", "")
	params := jsonutils.NewDict()
	params.Set("scope", jsonutils.NewString("system"))
	params.Set("system", jsonutils.JSONTrue)
	data, err := modules.UsersV3.GetById(session, r.Id, params)
	if err != nil {
		jerr := err.(*httputils.JSONClientError)
		if jerr.Code == 404 {
			err := r.Delete(ctx, session.GetToken())
			if err != nil {
				return errors.Wrapf(err, "unable to delete receiver %s", r.Id)
			}
			return errors.Wrapf(errors.ErrNotFound, "no such receiver %s", r.Id)
		}
		return err
	}
	uname, _ := data.GetString("name")
	domainId, _ := data.GetString("domain_id")
	lang, _ := data.GetString("lang")
	_, err = db.Update(r, func() error {
		r.Name = uname
		r.DomainId = domainId
		r.Lang = lang
		return nil
	})
	return errors.Wrap(err, "unable to update")
}

func (r *SReceiver) GetTemplateLang(ctx context.Context) (string, error) {
	if len(r.Lang) == 0 {
		err := r.Sync(ctx)
		if err != nil {
			return "", err
		}
	}
	log.Infof("lang: %s", r.Lang)
	lang, err := language.Parse(r.Lang)
	if err != nil {
		return "", errors.Wrapf(err, "unable to prase language %q", r.Lang)
	}
	tLang := notifyclientI18nTable.LookupByLang(lang, tempalteLang)
	return tLang, nil
}

// Implemente interface EventHandler
func (rm *SReceiverManager) OnAdd(obj *jsonutils.JSONDict) {
	// do nothing
	return
}

func (rm *SReceiverManager) OnUpdate(oldObj, newObj *jsonutils.JSONDict) {
	userId, _ := newObj.GetString("id")
	receivers, err := rm.FetchByIDs(context.Background(), userId)
	if err != nil {
		log.Errorf("fail to FetchByIDs: %v", err)
		return
	}
	if len(receivers) == 0 {
		return
	}
	receiver := &receivers[0]
	uname, _ := newObj.GetString("name")
	domainId, _ := newObj.GetString("domain_id")
	lang, _ := newObj.GetString("lang")
	if receiver.Name == uname && receiver.DomainId == domainId && receiver.Lang == lang {
		return
	}
	_, err = db.Update(receiver, func() error {
		receiver.Name = uname
		receiver.DomainId = domainId
		receiver.Lang = lang
		return nil
	})
	if err != nil {
		log.Errorf("fail to update uname of contact %q: %v", receiver.Id, err)
	}
}

func (rm *SReceiverManager) OnDelete(obj *jsonutils.JSONDict) {
	userId, _ := obj.GetString("id")
	log.Infof("receiver delete event for user %q", userId)
	receivers, err := rm.FetchByIDs(context.Background(), userId)
	if err != nil {
		log.Errorf("fail to FetchByIDs: %v", err)
		return
	}
	if len(receivers) == 0 {
		return
	}
	receiver := &receivers[0]
	err = receiver.Delete(context.Background(), auth.GetAdminSession(context.Background(), "", "").GetToken())
	if err != nil {
		log.Errorf("fail to delete contact %q: %v", receiver.Id, err)
	}
}

func (rm *SReceiverManager) StartWatchUserInKeystone() error {
	adminSession := auth.GetAdminSession(context.Background(), "", "")
	watchMan, err := informer.NewWatchManagerBySession(adminSession)
	if err != nil {
		return err
	}
	resMan := &modules.UsersV3
	return watchMan.For(resMan).AddEventHandler(context.Background(), rm)
}

func (rm *SReceiverManager) FetchByIDs(ctx context.Context, ids ...string) ([]SReceiver, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var err error
	q := rm.Query()
	if len(ids) == 1 {
		q = q.Equals("id", ids[0])
	} else {
		q = q.In("id", ids)
	}
	contacts := make([]SReceiver, 0, len(ids))
	err = db.FetchModelObjects(rm, q, &contacts)
	if err != nil {
		return nil, err
	}
	return contacts, nil
}

func (rm *SReceiverManager) FetchByIdOrNames(ctx context.Context, idOrNames ...string) ([]SReceiver, error) {
	if len(idOrNames) == 0 {
		return nil, nil
	}
	var err error
	q := rm.Query()
	var conds []sqlchemy.ICondition
	for _, idOrName := range idOrNames {
		conds = append(conds, sqlchemy.Equals(q.Field("name"), idOrName))
		if !stringutils2.IsUtf8(idOrName) {
			conds = append(conds, sqlchemy.Equals(q.Field("id"), idOrName))
		}
	}
	if len(conds) == 1 {
		q = q.Filter(conds[0])
	} else if len(conds) > 1 {
		q = q.Filter(sqlchemy.OR(conds...))
	}
	receivers := make([]SReceiver, 0, len(idOrNames))
	err = db.FetchModelObjects(rm, q, &receivers)
	if err != nil {
		return nil, err
	}
	return receivers, nil
}

func (r *SReceiver) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.ReceiverDetails, error) {
	return api.ReceiverDetails{}, nil
}

func (r *SReceiver) SetContact(cType string, contact string) error {
	if err := r.PullCache(false); err != nil {
		return err
	}
	switch cType {
	case api.EMAIL:
		r.Email = contact
	case api.MOBILE:
		r.Mobile = contact
	default:
		if sc, ok := r.subContactCache[cType]; ok {
			sc.Contact = contact
		}
	}
	return nil
}

func (r *SReceiver) GetContact(cType string) (string, error) {
	if err := r.PullCache(false); err != nil {
		return "", err
	}
	switch {
	case cType == api.EMAIL:
		return r.Email, nil
	case cType == api.MOBILE:
		return r.Mobile, nil
	case cType == api.WEBCONSOLE:
		return r.Id, nil
	case utils.IsInStringArray(cType, RobotContactTypes):
		return r.Mobile, nil
	default:
		if sc, ok := r.subContactCache[cType]; ok {
			return sc.Contact, nil
		}
	}
	return "", nil
}
