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
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	identity_api "yunion.io/x/onecloud/pkg/apis/identity"
	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/informer"
	identity_modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/notify/options"
	"yunion.io/x/onecloud/pkg/util/logclient"
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
	db.SVirtualResourceBaseManager
	db.SEnabledResourceBaseManager
}

var ReceiverManager *SReceiverManager

func init() {
	ReceiverManager = &SReceiverManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SReceiver{},
			"receivers_tbl",
			"receiver",
			"receivers",
		),
	}
	ReceiverManager.SetVirtualObject(ReceiverManager)
}

// 接收人
type SReceiver struct {
	db.SVirtualResourceBase
	db.SEnabledResourceBase

	Email string `width:"128" nullable:"false" create:"optional" update:"user" get:"user" list:"user"`
	// swagger:ignore
	Mobile string `width:"32" nullable:"false" create:"optional"`
	Lang   string `width:"8" charset:"ascii" nullable:"false" list:"user" update:"user"`

	// swagger:ignore
	EnabledEmail tristate.TriState `default:"false" update:"user"`
	// swagger:ignore
	VerifiedEmail tristate.TriState `default:"false" update:"user"`

	// swagger:ignore
	EnabledMobile tristate.TriState `default:"false" update:"user"`
	// swagger:ignore
	VerifiedMobile tristate.TriState `default:"false" update:"user"`

	// swagger:ignore
	// subContactCache map[string]*SSubContact `json:"-"`
}

func (rm *SReceiverManager) CreateByInsertOrUpdate() bool {
	return true
}

func (rm *SReceiverManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.ReceiverCreateInput) (api.ReceiverCreateInput, error) {
	var err error
	input.VirtualResourceCreateInput, err = rm.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return input, err
	}
	if len(input.UID) == 0 && len(input.UName) == 0 {
		return input, httperrors.NewMissingParameterError("uid or uname")
	}
	uid := input.UID
	if len(input.UID) == 0 {
		uid = input.UName
	}
	user, err := db.UserCacheManager.FetchUserByIdOrName(ctx, uid)
	if err != nil {
		return input, err
	}
	input.UID = user.Id
	input.UName = user.Name
	input.ProjectDomainId = user.DomainId
	// hack
	input.Name = input.UName
	// validate email
	if ok := regutils.MatchEmail(input.Email); len(input.Email) > 0 && !ok {
		return input, httperrors.NewInputParameterError("invalid email")
	}
	// validate mobile
	input.InternationalMobile.AcceptExtMobile()
	if ok := LaxMobileRegexp.MatchString(input.InternationalMobile.Mobile); len(input.InternationalMobile.Mobile) > 0 && !ok {
		return input, httperrors.NewInputParameterError("invalid mobile")
	}
	input.Enabled = pTrue
	for _, cType := range input.EnabledContactTypes {
		driver := GetDriver(cType)
		if driver == nil {
			return input, httperrors.NewInputParameterError("invalid enabled contact type %s", cType)
		}
	}
	return input, nil
}

func (r *SReceiver) IsEnabled() bool {
	return r.Enabled.Bool()
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
	ret := make([]string, 0, 1)
	// for email and mobile
	if r.EnabledEmail.IsTrue() {
		ret = append(ret, api.EMAIL)
	}
	if r.EnabledMobile.IsTrue() {
		ret = append(ret, api.MOBILE)
	}
	subs, _ := r.GetSubContacts()
	for _, sub := range subs {
		if sub.Enabled.IsTrue() {
			ret = append(ret, sub.Type)
		}
	}
	ret = append(ret, api.WEBCONSOLE)
	return ret, nil
}

func (r *SReceiver) markContactType(ctx context.Context, contactType string, isVerified bool, note string) error {
	if contactType == api.MOBILE {
		_, err := db.Update(r, func() error {
			r.VerifiedMobile = tristate.NewFromBool(isVerified)
			return nil
		})
		return err
	}
	if contactType == api.EMAIL {
		_, err := db.Update(r, func() error {
			r.VerifiedEmail = tristate.NewFromBool(isVerified)
			return nil
		})
		return err
	}
	subs, err := r.GetSubContacts()
	if err != nil {
		return err
	}
	for i := range subs {
		if subs[i].Type == contactType {
			_, err := db.Update(&subs[i], func() error {
				subs[i].Verified = tristate.NewFromBool(isVerified)
				subs[i].VerifiedNote = note
				return nil
			})
			return err
		}
	}
	sub := &SSubContact{
		Type:              contactType,
		ReceiverID:        r.Id,
		Verified:          tristate.NewFromBool(isVerified),
		VerifiedNote:      note,
		ParentContactType: api.MOBILE,
	}
	sub.SetModelManager(SubContactManager, sub)
	return SubContactManager.TableSpec().Insert(ctx, sub)
}

func (r *SReceiver) MarkContactTypeVerified(ctx context.Context, contactType string) error {
	return r.markContactType(ctx, contactType, true, "")
}

func (r *SReceiver) MarkContactTypeUnVerified(ctx context.Context, contactType string, note string) error {
	return r.markContactType(ctx, contactType, false, note)
}

func (r *SReceiver) GetVerifiedContactTypes() ([]string, error) {
	ret := make([]string, 0, 1)
	// for email and mobile
	if r.VerifiedEmail.IsTrue() {
		ret = append(ret, api.EMAIL)
	}
	if r.VerifiedMobile.IsTrue() {
		ret = append(ret, api.MOBILE)
	}
	subs, _ := r.GetSubContacts()
	for _, sub := range subs {
		if sub.Verified.IsTrue() {
			ret = append(ret, sub.Type)
		}
	}
	return ret, nil
}

func (self *SReceiver) GetSubContacts() ([]SSubContact, error) {
	ret := []SSubContact{}
	q := SubContactManager.Query().Equals("receiver_id", self.Id)
	err := db.FetchModelObjects(SubContactManager, q, &ret)
	return ret, err
}

func (rm *SReceiverManager) FetchSubContacts(ids []string) (map[string][]SSubContact, error) {
	ret := map[string][]SSubContact{}
	q := SubContactManager.Query().In("receiver_id", ids)
	subContacts := []SSubContact{}
	err := db.FetchModelObjects(SubContactManager, q, &subContacts)
	if err != nil {
		return ret, err
	}
	for i := range subContacts {
		_, ok := ret[subContacts[i].ReceiverID]
		if !ok {
			ret[subContacts[i].ReceiverID] = []SSubContact{}
		}
		ret[subContacts[i].ReceiverID] = append(ret[subContacts[i].ReceiverID], subContacts[i])
	}
	return ret, nil
}

func (rm *SReceiverManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeUser
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

func (rm *SReceiverManager) filterByOwnerAndProjectDomain(ctx context.Context, userCred mcclient.TokenCredential, q *sqlchemy.SQuery, scope rbacscope.TRbacScope) (*sqlchemy.SQuery, error) {
	if userCred == nil {
		return q, nil
	}

	userIds, err := rm.findUserIdsWithProjectDomain(ctx, userCred, userCred.GetProjectDomainId())
	if err != nil {
		return nil, errors.Wrap(err, "unable to findUserIdsWithProjectDomain")
	}
	var projectDomainCondition, ownerCondition sqlchemy.ICondition
	switch len(userIds) {
	case 0:
		projectDomainCondition = nil
	case 1:
		projectDomainCondition = sqlchemy.Equals(q.Field("id"), userIds[0])
	default:
		projectDomainCondition = sqlchemy.In(q.Field("id"), userIds)
	}

	switch scope {
	case rbacscope.ScopeDomain:
		ownerCondition = sqlchemy.Equals(q.Field("domain_id"), userCred.GetProjectDomainId())
	case rbacscope.ScopeProject:
		ownerCondition = sqlchemy.Equals(q.Field("id"), userCred.GetUserId())
	}

	if projectDomainCondition != nil && ownerCondition != nil {
		return q.Filter(sqlchemy.OR(projectDomainCondition, ownerCondition)), nil
	}
	if projectDomainCondition != nil {
		return q.Filter(projectDomainCondition), nil
	}
	if ownerCondition != nil {
		return q.Filter(ownerCondition), nil
	}
	return q, nil
}

func (rm *SReceiverManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	log.Debugf("SReceiverManager FilterByOwner is called owner %s scope %s", jsonutils.Marshal(owner), scope)
	return rm.SDomainizedResourceBaseManager.FilterByOwner(ctx, q, man, userCred, owner, scope)
}

func (rm *SReceiverManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.ReceiverListInput) (*sqlchemy.SQuery, error) {
	q, err := rm.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.VirtualResourceListInput)
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
		switch input.EnabledContactType {
		case api.MOBILE:
			q = q.IsTrue("enabled_mobile")
		case api.EMAIL:
			q = q.IsTrue("enabled_email")
		default:
			sq := SubContactManager.Query("receiver_id").Equals("type", input.EnabledContactType).IsTrue("enabled").SubQuery()
			q = q.Join(sq, sqlchemy.Equals(sq.Field("receiver_id"), q.Field("id")))
		}
	}
	if len(input.VerifiedContactType) > 0 {
		switch input.VerifiedContactType {
		case api.MOBILE:
			q = q.IsTrue("verified_mobile")
		case api.EMAIL:
			q = q.IsTrue("verified_email")
		default:
			sq := SubContactManager.Query("receiver_id").Equals("type", input.VerifiedContactType).IsTrue("verified").SubQuery()
			q = q.Join(sq, sqlchemy.Equals(sq.Field("receiver_id"), q.Field("id")))
		}
	}
	return q, nil
}

func (rm *SReceiverManager) findUserIdsWithProjectDomain(ctx context.Context, userCred mcclient.TokenCredential, projectDomainId string) ([]string, error) {
	session := auth.GetSession(ctx, userCred, "")
	query := jsonutils.NewDict()
	query.Set("effective", jsonutils.JSONTrue)
	query.Set("project_domain_id", jsonutils.NewString(projectDomainId))
	listRet, err := identity_modules.RoleAssignments.List(session, query)
	if err != nil {
		return nil, errors.Wrap(err, "unable to list RoleAssignments")
	}
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

func (rm *SReceiverManager) domainIdsFromReceivers(ctx context.Context, receivers []string) ([]string, error) {
	res, err := rm.FetchByIdOrNames(ctx, receivers...)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch receivres by id or names")
	}
	domainIds := sets.NewString()
	for i := range res {
		domainIds.Insert(res[i].DomainId)
	}
	return domainIds.UnsortedList(), nil
}

func (rm *SReceiverManager) PerformGetTypes(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ConfigManagerGetTypesInput) (api.ConfigManagerGetTypesOutput, error) {
	output := api.ConfigManagerGetTypesOutput{}
	var reduce func([]*sqlchemy.SQuery) *sqlchemy.SQuery
	var err error
	switch input.Operation {
	case "", "merge":
		reduce = func(qs []*sqlchemy.SQuery) *sqlchemy.SQuery {
			q := qs[0]
			for i := 1; i < len(qs); i++ {
				q = q.In("type", qs[i])
			}
			return q
		}
	case "union":
		reduce = func(qs []*sqlchemy.SQuery) *sqlchemy.SQuery {
			if len(qs) == 1 {
				return qs[0]
			}
			iqs := make([]sqlchemy.IQuery, 0, len(qs))
			for i := range qs {
				iqs = append(iqs, qs[i])
			}
			union, _ := sqlchemy.UnionWithError(iqs...)
			if err != nil {
			}
			return union.Query()
		}
	default:
		return output, httperrors.NewInputParameterError("unkown operation %q", input.Operation)
	}
	domainIds, err := rm.domainIdsFromReceivers(ctx, input.Receivers)
	if err != nil {
		return output, err
	}
	domainIds = sets.NewString(append(domainIds, input.DomainIds...)...).UnsortedList()
	qs := make([]*sqlchemy.SQuery, 0, len(domainIds))
	if len(domainIds) == 0 {
		qs = append(qs, ConfigManager.contactTypesQuery(""))
	} else {
		for i := range domainIds {
			ctypeQ := ConfigManager.contactTypesQuery(domainIds[i])
			qs = append(qs, ctypeQ)
		}
	}
	q := reduce(qs)
	allTypes := make([]struct {
		Type string
	}, 0, 3)
	err = q.All(&allTypes)
	if err != nil {
		return output, err
	}
	ret := make([]string, len(allTypes))
	for i := range ret {
		ret[i] = allTypes[i].Type
	}
	if !utils.IsInStringArray(api.WEBCONSOLE, ret) {
		ret = append(ret, api.WEBCONSOLE)
	}
	output.Types = ret
	return output, nil
}

func (rm *SReceiverManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []api.ReceiverDetails {
	sRows := rm.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	rows := make([]api.ReceiverDetails, len(objs))
	recvIds := []string{}
	for i := range rows {
		rows[i].VirtualResourceDetails = sRows[i]
		user := objs[i].(*SReceiver)
		recvIds = append(recvIds, user.Id)
		rows[i].InternationalMobile = api.ParseInternationalMobile(user.Mobile)
		rows[i].EnabledContactTypes = []string{}
		if user.EnabledEmail.Bool() {
			rows[i].EnabledContactTypes = append(rows[i].EnabledContactTypes, api.EMAIL)
		}
		if user.EnabledMobile.Bool() {
			rows[i].EnabledContactTypes = append(rows[i].EnabledContactTypes, api.MOBILE)
		}
		rows[i].EnabledContactTypes = append(rows[i].EnabledContactTypes, api.WEBCONSOLE)
		rows[i].VerifiedInfos = []api.VerifiedInfo{
			{
				ContactType: api.EMAIL,
				Verified:    user.VerifiedEmail.Bool(),
			},
			{
				ContactType: api.MOBILE,
				Verified:    user.VerifiedMobile.Bool(),
			},
			{
				ContactType: api.WEBCONSOLE,
				Verified:    true,
			},
		}
	}
	subContacts, err := rm.FetchSubContacts(recvIds)
	if err != nil {
		return rows
	}
	for i := range rows {
		for _, contact := range subContacts[recvIds[i]] {
			if contact.Enabled.Bool() {
				rows[i].EnabledContactTypes = append(rows[i].EnabledContactTypes, contact.Type)
			}
			rows[i].VerifiedInfos = append(rows[i].VerifiedInfos, api.VerifiedInfo{
				ContactType: contact.Type,
				Verified:    contact.Verified.Bool(),
				Note:        contact.VerifiedNote,
			})
		}
		rows[i].EnabledContactTypes = sortContactType(rows[i].EnabledContactTypes)
	}

	return rows
}

func (rm *SReceiverManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := rm.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (rm *SReceiverManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.ReceiverListInput) (*sqlchemy.SQuery, error) {
	q, err := rm.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (r *SReceiver) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	r.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	cTypes := jsonutils.GetQueryStringArray(data, "enabled_contact_types")
	err := r.StartSubcontactPullTask(ctx, userCred, cTypes, "")
	if err != nil {
		logclient.AddActionLogWithContext(ctx, r, logclient.ACT_CREATE, err, userCred, false)
		return
	}
	logclient.AddActionLogWithContext(ctx, r, logclient.ACT_CREATE, err, userCred, true)
}

func (r *SReceiver) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	err := r.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
	if err != nil {
		return nil
	}
	var input api.ReceiverCreateInput
	err = data.Unmarshal(&input)
	if err != nil {
		return err
	}
	r.Id = input.UID
	r.Mobile = input.InternationalMobile.String()

	// 需求：管理后台新建的联系人，手机号和邮箱无需进行校验
	// 方案：检查请求者对于创建联系人 是否具有system scope
	allowScope, _ := policy.PolicyManager.AllowScope(userCred, api.SERVICE_TYPE, ReceiverManager.KeywordPlural(), policy.PolicyActionCreate)
	if allowScope == rbacscope.ScopeSystem {
		if utils.IsInStringArray(api.EMAIL, input.EnabledContactTypes) {
			r.VerifiedEmail = tristate.True
		}
		if utils.IsInStringArray(api.MOBILE, input.EnabledContactTypes) {
			r.VerifiedMobile = tristate.True
		}
	}

	return nil
}

func (r *SReceiver) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ReceiverUpdateInput) (api.ReceiverUpdateInput, error) {
	var err error
	input.VirtualResourceBaseUpdateInput, err = r.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.VirtualResourceBaseUpdateInput)
	if err != nil {
		return input, err
	}
	// validate email
	if ok := len(input.Email) == 0 || regutils.MatchEmail(input.Email); !ok {
		return input, httperrors.NewInputParameterError("invalid email")
	}
	// validate mobile
	input.InternationalMobile.AcceptExtMobile()
	if ok := len(input.InternationalMobile.Mobile) == 0 || LaxMobileRegexp.MatchString(input.InternationalMobile.Mobile); !ok {
		return input, httperrors.NewInputParameterError("invalid mobile")
	}

	for _, cType := range input.EnabledContactTypes {
		driver := GetDriver(cType)
		if driver == nil {
			return input, httperrors.NewInputParameterError("invalid enabled contact type %s", cType)
		}
	}

	return input, nil
}

func (r *SReceiver) PreUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	r.SVirtualResourceBase.PreUpdate(ctx, userCred, query, data)
	originEmailEnable, originMobileEnable := r.EnabledEmail, r.EnabledMobile
	var input api.ReceiverUpdateInput
	data.Unmarshal(&input)
	if len(input.Email) != 0 && input.Email != r.Email {
		db.Update(r, func() error {
			r.VerifiedEmail = tristate.False
			return nil
		})
		r.VerifiedEmail = tristate.False
		subs, _ := r.GetSubContacts()
		for i := range subs {
			if subs[i].ParentContactType == api.EMAIL {
				db.Update(&subs[i], func() error {
					subs[i].Verified = tristate.False
					subs[i].VerifiedNote = "email changed, re-verify"
					return nil
				})
			}
		}
	}
	mobile := input.InternationalMobile.String()
	if len(mobile) != 0 && mobile != r.Mobile {
		db.Update(r, func() error {
			r.VerifiedMobile = tristate.False
			return nil
		})
		subs, _ := r.GetSubContacts()
		for i := range subs {
			if subs[i].ParentContactType == api.MOBILE {
				db.Update(&subs[i], func() error {
					subs[i].Verified = tristate.False
					subs[i].VerifiedNote = "mobile changed, re-verify"
					return nil
				})
			}
		}
	}

	// 管理后台修改联系人，如果修改或者启用手机号和邮箱，无需进行校验
	if input.ForceVerified {
		allowScope, _ := policy.PolicyManager.AllowScope(userCred, api.SERVICE_TYPE, ReceiverManager.KeywordPlural(), policy.PolicyActionCreate)
		if allowScope == rbacscope.ScopeSystem {
			db.Update(r, func() error {
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
				return nil
			})
		}
	}
	r.Mobile = mobile
	err := ReceiverManager.TableSpec().InsertOrUpdate(ctx, r)
	if err != nil {
		log.Errorf("InsertOrUpdate: %v", err)
	}
}

func (r *SReceiver) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	r.SVirtualResourceBase.PostUpdate(ctx, userCred, query, data)
	cTypes := jsonutils.GetQueryStringArray(data, "enabled_contact_types")
	err := r.StartSubcontactPullTask(ctx, userCred, cTypes, "")
	if err != nil {
		logclient.AddActionLogWithContext(ctx, r, logclient.ACT_UPDATE, err, userCred, false)
		return
	}
	logclient.AddActionLogWithContext(ctx, r, logclient.ACT_UPDATE, err, userCred, true)
}

func (r *SReceiver) StartSubcontactPullTask(ctx context.Context, userCred mcclient.TokenCredential, contactTypes []string, parentTaskId string) error {
	if len(r.Mobile) == 0 {
		return nil
	}
	r.SetStatus(ctx, userCred, api.RECEIVER_STATUS_PULLING, "")
	params := jsonutils.NewDict()
	if len(contactTypes) > 0 {
		params.Set("contact_types", jsonutils.NewStringArray(contactTypes))
	}
	task, err := taskman.TaskManager.NewTask(ctx, "SubcontactPullTask", r, userCred, params, parentTaskId, "")
	if err != nil {
		return err
	}
	return task.ScheduleRun(nil)
}

func (r *SReceiver) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	subs, _ := r.GetSubContacts()
	for _, sc := range subs {
		err := sc.Delete(ctx, userCred)
		if err != nil {
			return err
		}
	}
	r.deleteReceiverInSubscriber(ctx)
	return r.SVirtualResourceBase.Delete(ctx, userCred)
}

func (r *SReceiver) deleteReceiverInSubscriber(ctx context.Context) error {
	q := SubscriberReceiverManager.Query().Equals("receiver_id", r.Id)
	srs := make([]SSubscriberReceiver, 0, 2)
	err := db.FetchModelObjects(SubscriberReceiverManager, q, &srs)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range srs {
		srs[i].Delete(ctx, nil)
	}
	return nil
}

func (r *SReceiver) IsOwner(userCred mcclient.TokenCredential) bool {
	return r.Id == userCred.GetUserId()
}

// 获取用户订阅
func (r *SReceiver) PerformGetSubscription(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ReceiverGetSubscriptionOptions) (jsonutils.JSONObject, error) {
	s := auth.GetAdminSession(ctx, options.Options.Region)
	subscribers, err := getSubscriberByReceiverId(r.Id, input.ShowDisabled)
	if err != nil {
		return nil, errors.Wrap(err, "getSubscriberByReceiverId")
	}
	type retStruct struct {
		SSubscriber
		IdentityName string
		TopicName    string
		TopicType    string
	}
	res := []retStruct{}
	for _, subscriber := range subscribers {
		topicModel, err := TopicManager.FetchById(subscriber.TopicId)
		if err != nil {
			if errors.Cause(err) != errors.ErrNotFound {
				continue
			}
			return nil, errors.Wrap(err, "fetch topic by id")
		}
		topic := topicModel.(*STopic)
		if topic.Enabled == tristate.False {
			continue
		}
		identityName := ""
		if subscriber.Type == api.SUBSCRIBER_TYPE_ROLE {
			role := identity_api.RoleDetails{}
			roleDetail, err := identity_modules.RolesV3.GetById(s, subscriber.Identification, nil)
			if err != nil {
				log.Warningf("get %s role details err:%s", subscriber.Identification, err)
			}
			if roleDetail != nil {
				roleDetail.Unmarshal(&role)
				identityName = role.Name
			}
		} else {
			identityName = r.Name
		}
		res = append(res, retStruct{SSubscriber: subscriber, IdentityName: identityName, TopicName: topic.GetName(), TopicType: topic.Type})
	}
	return jsonutils.Marshal(res), nil
}

func (rm *SReceiverManager) PerformIntellijGet(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ReceiverIntellijGetInput) (jsonutils.JSONObject, error) {
	ret := &SReceiver{}
	ret.SetModelManager(rm, ret)
	err := rm.Query().Equals("id", input.UserId).First(ret)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, err
	}
	if len(ret.Id) > 0 {
		return jsonutils.Marshal(ret), nil
	}
	// create one
	adminSession := auth.GetAdminSession(ctx, "")
	resp, err := identity_modules.UsersV3.GetById(adminSession, input.UserId, jsonutils.Marshal(map[string]string{"scope": "system"}))
	if err != nil {
		return nil, errors.Wrap(err, "unable get user from keystone")
	}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	err = rm.TableSpec().InsertOrUpdate(ctx, ret)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create receiver")
	}
	return jsonutils.Marshal(ret), nil
}

func (r *SReceiver) PerformTriggerVerify(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ReceiverTriggerVerifyInput) (jsonutils.JSONObject, error) {
	if len(input.ContactType) == 0 {
		return nil, httperrors.NewMissingParameterError("contact_type")
	}
	if !utils.IsInStringArray(input.ContactType, []string{api.EMAIL, api.MOBILE, api.DINGTALK, api.FEISHU, api.WORKWX}) {
		return nil, httperrors.NewInputParameterError("not support such contact type %q", input.ContactType)
	}
	driver := GetDriver(input.ContactType)
	if driver.IsPullType() {
		return nil, r.StartSubcontactPullTask(ctx, userCred, []string{input.ContactType}, "")
	}
	_, err := VerificationManager.Create(ctx, r.Id, input.ContactType)
	/*if err == ErrVerifyFrequently {
		return nil, httperrors.NewForbiddenError("Send verify message too frequently, please try again later")
	}*/
	if err != nil {
		return nil, errors.Wrap(err, "VerifyManager.Create")
	}

	params := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewTask(ctx, "VerificationSendTask", r, userCred, params, "", "")
	if err != nil {
		return nil, err
	}
	return nil, task.ScheduleRun(nil)
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

func (r *SReceiver) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(r, ctx, userCred, true)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (r *SReceiver) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(r, ctx, userCred, false)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (r *SReceiver) Sync(ctx context.Context) error {
	user, err := db.UserCacheManager.FetchUserById(ctx, r.Id)
	if err != nil {
		return err
	}
	_, err = db.Update(r, func() error {
		r.Name = user.Name
		r.DomainId = user.DomainId
		if len(user.Lang) > 0 {
			r.Lang = user.Lang
		}
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
	err = receiver.Delete(context.Background(), auth.GetAdminSession(context.Background(), "").GetToken())
	if err != nil {
		log.Errorf("fail to delete contact %q: %v", receiver.Id, err)
	}
}

// 监听User变化
func (rm *SReceiverManager) StartWatchUserInKeystone() error {
	adminSession := auth.GetAdminSession(context.Background(), "")
	watchMan, err := informer.NewWatchManagerBySession(adminSession)
	if err != nil {
		return err
	}
	resMan := &identity_modules.UsersV3
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

func idOrNameFilter(q *sqlchemy.SQuery, idOrNames ...string) *sqlchemy.SQuery {
	if len(idOrNames) == 0 {
		return q
	}
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
	return q
}

func (rm *SReceiverManager) FetchByIdOrNames(ctx context.Context, idOrNames ...string) ([]SReceiver, error) {
	if len(idOrNames) == 0 {
		return nil, nil
	}
	var err error
	q := idOrNameFilter(rm.Query(), idOrNames...)
	receivers := make([]SReceiver, 0, len(idOrNames))
	err = db.FetchModelObjects(rm, q, &receivers)
	if err != nil {
		return nil, err
	}
	return receivers, nil
}

func (rm *SReceiverManager) FetchEnableReceiversByIdOrNames(ctx context.Context, idOrNames ...string) ([]SReceiver, error) {
	if len(idOrNames) == 0 {
		return nil, nil
	}
	var err error
	q := idOrNameFilter(rm.Query(), idOrNames...)
	q.Equals("enabled", true)
	receivers := make([]SReceiver, 0, len(idOrNames))
	err = db.FetchModelObjects(rm, q, &receivers)
	if err != nil {
		return nil, err
	}
	return receivers, nil
}

func (r *SReceiver) SetContact(cType string, contact string) error {
	var err error
	switch cType {
	case api.EMAIL:
		_, err = db.Update(r, func() error {
			r.Email = contact
			return nil
		})
	case api.MOBILE:
		_, err = db.Update(r, func() error {
			r.Mobile = contact
			return nil
		})
		r.Mobile = contact
	default:
		subs, _ := r.GetSubContacts()
		for i := range subs {
			if subs[i].Type == cType {
				_, err = db.Update(&subs[i], func() error {
					subs[i].Contact = contact
					return nil
				})
			}
		}
	}
	return err
}

func (r *SReceiver) GetContact(cType string) (string, error) {
	switch cType {
	case api.EMAIL:
		return r.Email, nil
	case api.MOBILE:
		return r.Mobile, nil
	case api.WEBCONSOLE:
		return r.Id, nil
	case api.FEISHU_ROBOT, api.DINGTALK_ROBOT, api.WORKWX_ROBOT:
		return r.Mobile, nil
	default:
		subs, _ := r.GetSubContacts()
		for _, sub := range subs {
			if sub.Type == cType {
				return sub.Contact, nil
			}
		}
	}
	return "", nil
}

func (r *SReceiver) PerformEnableContactType(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ReceiverEnableContactTypeInput) (jsonutils.JSONObject, error) {
	for _, cType := range input.EnabledContactTypes {
		driver := GetDriver(cType)
		if driver == nil {
			return nil, httperrors.NewInputParameterError("invalid enabled contact type %s", cType)
		}
	}
	return nil, r.StartSubcontactPullTask(ctx, userCred, input.EnabledContactTypes, "")
}

func (rm *SReceiverManager) InitializeData() error {
	return nil
}

func (self *SReceiver) GetNotifyReceiver() api.SNotifyReceiver {
	ret := api.SNotifyReceiver{
		DomainId: self.DomainId,
		Enabled:  self.Enabled.Bool(),
	}
	return ret
}

func (manager *SReceiverManager) SyncUserFromKeystone(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	err := manager.syncUserFromKeystone(ctx, userCred, isStart)
	if err != nil {
		log.Errorf("syncUserFromKeystone error %s", err)
	}
}

func InitReceiverProject(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	q := ReceiverManager.Query().IsNullOrEmpty("tenant_id")
	receivers := []SReceiver{}
	err := db.FetchModelObjects(ReceiverManager, q, &receivers)
	if err != nil {
		log.Errorln(errors.Wrap(err, "fetch receiver"))
		return
	}
	for _, receiver := range receivers {
		db.Update(&receiver, func() error {
			receiver.ProjectId = auth.AdminCredential().GetTenantId()
			return nil
		})
	}
}

func (manager *SReceiverManager) syncUserFromKeystone(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(string(rbacscope.ScopeSystem)), "scope")
	// params.Add(jsonutils.JSONTrue, "system")
	params.Add(jsonutils.JSONTrue, "enabled")
	params.Add(jsonutils.NewInt(100), "limit")
	offset := 0
	total := -1
	for total < 0 || offset < total {
		params.Set("offset", jsonutils.NewInt(int64(offset)))
		results, err := identity_modules.UsersV3.List(auth.GetAdminSession(ctx, options.Options.Region), params)
		if err != nil {
			return errors.Wrap(err, "List user")
		}
		for i := range results.Data {
			err := manager.syncUser(ctx, userCred, results.Data[i])
			if err != nil {
				return errors.Wrapf(err, "sync user %s", results.Data[i])
			}
		}
		total = results.Total
		offset += len(results.Data)
	}
	return nil
}

func (manager *SReceiverManager) syncUser(ctx context.Context, userCred mcclient.TokenCredential, usrData jsonutils.JSONObject) error {
	usr := identity_api.UserDetails{}
	err := usrData.Unmarshal(&usr)
	if err != nil {
		return errors.Wrap(err, "usrData.Unmarshal")
	}
	if len(usr.Id) == 0 {
		log.Fatalf("sync user with empty id?")
	}
	if len(usr.Email) == 0 && len(usr.Mobile) == 0 {
		// no need to sync
		return nil
	}
	if usr.IsSystemAccount != nil && *usr.IsSystemAccount {
		// no need to sync
		return nil
	}
	recvObj, err := manager.FetchById(usr.Id)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			return errors.Wrap(err, "FetchById")
		}
		// new receiver
		recver := SReceiver{}
		recver.SetModelManager(manager, &recver)
		recver.Id = usr.Id
		recver.Name = usr.Name
		recver.Status = api.RECEIVER_STATUS_READY
		recver.DomainId = usr.DomainId
		recver.Enabled = tristate.True
		recver.Mobile = usr.Mobile
		recver.Email = usr.Email
		if len(recver.Mobile) > 0 {
			recver.VerifiedMobile = tristate.True
		}
		if len(recver.Email) > 0 {
			recver.VerifiedEmail = tristate.True
		}
		err := manager.TableSpec().InsertOrUpdate(ctx, &recver)
		if err != nil {
			return errors.Wrap(err, "Insert")
		}
		logclient.AddSimpleActionLog(&recver, logclient.ACT_CREATE, &recver, userCred, true)
	} else {
		// update receiver
		recver := recvObj.(*SReceiver)
		if (len(recver.Mobile) == 0 && len(usr.Mobile) > 0) || (len(recver.Email) == 0 && len(usr.Email) > 0) {
			// need update
			diff, err := db.Update(recver, func() error {
				if len(recver.Mobile) == 0 && len(usr.Mobile) > 0 {
					recver.Mobile = usr.Mobile
					recver.VerifiedMobile = tristate.True
				}
				if len(recver.Email) == 0 && len(usr.Email) > 0 {
					recver.Email = usr.Email
					recver.VerifiedEmail = tristate.True
				}
				return nil
			})
			if err != nil {
				return errors.Wrap(err, "Update")
			}
			logclient.AddSimpleActionLog(recver, logclient.ACT_UPDATE, diff, userCred, true)
		}
	}
	return nil
}

func (r *SReceiver) GetDomainId() string {
	return r.DomainId
}

func (r *SReceiver) IsRobot() bool {
	return false
}

func (r *SReceiver) IsReceiver() bool {
	return true
}

func (r *SReceiver) GetName() string {
	return r.Name
}

func (manager *SReceiverManager) GetPropertyRoleContactType(ctx context.Context, userCred mcclient.TokenCredential, input api.SRoleContactInput) (jsonutils.JSONObject, error) {
	out := api.SRoleContactOutput{
		ContactType: []string{},
	}
	if len(input.RoleIds) == 0 {
		return nil, httperrors.NewMissingParameterError("role_ids")
	}
	receiverIds := []string{}
	params := jsonutils.NewDict()
	params.Set("roles", jsonutils.NewStringArray(input.RoleIds))
	params.Set("effective", jsonutils.JSONTrue)
	switch input.Scope {
	case api.SUBSCRIBER_SCOPE_SYSTEM:
	case api.SUBSCRIBER_SCOPE_DOMAIN:
		if len(input.ProjectDomainId) == 0 {
			return nil, httperrors.NewMissingParameterError("project_domain_id")
		}
		params.Set("project_domain_id", jsonutils.NewString(input.ProjectDomainId))
	case api.SUBSCRIBER_SCOPE_PROJECT:
		if len(input.ProjectId) == 0 {
			return nil, httperrors.NewMissingParameterError("project_id")
		}
		params.Add(jsonutils.NewString(input.ProjectId), "scope", "project", "id")
	}
	s := auth.GetAdminSession(ctx, "")
	listRet, err := identity_modules.RoleAssignments.List(s, params)
	if err != nil {
		return nil, errors.Wrap(err, "unable to list RoleAssignments")
	}

	roleAssignmentOut := []identity_api.SRoleAssignment{}
	jsonutils.Update(&roleAssignmentOut, listRet.Data)
	for i := range roleAssignmentOut {
		receiverIds = append(receiverIds, roleAssignmentOut[i].User.Id)
	}

	subContacts, err := manager.FetchSubContacts(receiverIds)
	if err != nil {
		return nil, errors.Wrap(err, "fetch subcontacts")
	}
	contactMap := map[string]struct{}{}
	for i := range receiverIds {
		for _, contact := range subContacts[receiverIds[i]] {
			if contact.Enabled.Bool() {
				contactMap[contact.Type] = struct{}{}
			}
		}
	}
	for contactType := range contactMap {
		out.ContactType = append(out.ContactType, contactType)
	}
	return jsonutils.Marshal(out), nil
}
