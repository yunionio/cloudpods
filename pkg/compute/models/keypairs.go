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
	"strings"

	"golang.org/x/crypto/ssh"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SKeypairManager struct {
	db.SUserResourceBaseManager
	db.SSharableBaseResourceManager
}

var KeypairManager *SKeypairManager

func init() {
	KeypairManager = &SKeypairManager{
		SUserResourceBaseManager: db.NewUserResourceBaseManager(
			SKeypair{},
			"keypairs_tbl",
			"keypair",
			"keypairs",
		),
	}
	KeypairManager.SetVirtualObject(KeypairManager)
}

type SKeypair struct {
	db.SUserResourceBase
	db.SSharableBaseResource

	// 加密类型
	// example: RSA
	Scheme string `width:"12" charset:"ascii" nullable:"true" list:"user" create:"required"`
	// 指纹信息
	// example: 1d:3a:83:4a:a1:f3:75:97:ec:d1:ef:f8:3f:a7:5d:9e
	Fingerprint string `width:"48" charset:"ascii" nullable:"false" list:"user" create:"required"`
	// 私钥
	PrivateKey string `width:"2048" charset:"ascii" nullable:"true" create:"optional"`
	// 公钥
	PublicKey string `width:"1024" charset:"ascii" nullable:"false" list:"user" create:"required"`
}

func (manager *SKeypairManager) GetISharableVirtualModelManager() db.ISharableVirtualModelManager {
	return manager.GetVirtualObject().(db.ISharableVirtualModelManager)
}

func (manager *SKeypairManager) GetIVirtualModelManager() db.IVirtualModelManager {
	return manager.GetVirtualObject().(db.IVirtualModelManager)
}

// 列出ssh密钥对
func (manager *SKeypairManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.KeypairListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, err
	}

	if ((query.Admin != nil && *query.Admin) || query.Scope == string(rbacscope.ScopeSystem)) && db.IsAdminAllowList(userCred, manager).Result.IsAllow() {
		user := query.UserId
		if len(user) > 0 {
			uc, _ := db.UserCacheManager.FetchUserByIdOrName(ctx, user)
			if uc == nil {
				return nil, httperrors.NewUserNotFoundError("user %s not found", user)
			}
			q = q.Equals("owner_id", uc.Id)
		}
	} else {
		q, err = manager.SSharableBaseResourceManager.ListItemFilter(ctx, q, userCred, query.SharableResourceBaseListInput)
		if err != nil {
			return nil, errors.Wrap(err, "SSharableBaseResourceManager.ListItemFilter")
		}
	}

	if len(query.Scheme) > 0 {
		q = q.In("scheme", query.Scheme)
	}
	if len(query.Fingerprint) > 0 {
		q = q.In("fingerprint", query.Fingerprint)
	}

	return q, nil
}

func (manager *SKeypairManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.KeypairListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SUserResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.UserResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SUserResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SKeypairManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SUserResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (km *SKeypairManager) query(manager db.IModelManager, field string, keyIds []string, filter func(*sqlchemy.SQuery) *sqlchemy.SQuery) *sqlchemy.SSubQuery {
	q := manager.Query()

	if filter != nil {
		q = filter(q)
	}

	sq := q.SubQuery()

	return sq.Query(
		sq.Field("keypair_id"),
		sqlchemy.COUNT(field),
	).In("keypair_id", keyIds).GroupBy(sq.Field("keypair_id")).SubQuery()
}

type SKeypairUsageCount struct {
	Id               string
	LinkedGuestCount int
}

func (km *SKeypairManager) TotalResourceCount(keyIds []string) (map[string]SKeypairUsageCount, error) {
	ret := map[string]SKeypairUsageCount{}

	guestSQ := km.query(GuestManager, "guest_cnt", keyIds, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.IsNotEmpty("keypair_id")
	})

	keypairs := km.Query().SubQuery()
	keypairsQ := keypairs.Query(
		sqlchemy.SUM("linked_guest_count", guestSQ.Field("guest_cnt")),
	)

	keypairsQ.AppendField(keypairsQ.Field("id"))

	keypairsQ = keypairsQ.LeftJoin(guestSQ, sqlchemy.Equals(keypairsQ.Field("id"), guestSQ.Field("keypair_id")))

	keypairsQ = keypairsQ.Filter(sqlchemy.In(keypairsQ.Field("id"), keyIds)).GroupBy(keypairsQ.Field("id"))

	counts := []SKeypairUsageCount{}
	err := keypairsQ.All(&counts)
	if err != nil {
		return nil, errors.Wrapf(err, "keyparisQ.All")
	}
	for i := range counts {
		ret[counts[i].Id] = counts[i]
	}

	return ret, nil
}

func (manager *SKeypairManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.KeypairDetails {
	rows := make([]api.KeypairDetails, len(objs))
	userRows := manager.SUserResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	shareRows := manager.SSharableBaseResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	keyIds := make([]string, len(objs))
	for i := range rows {
		keypair := objs[i].(*SKeypair)
		rows[i] = api.KeypairDetails{
			UserResourceDetails:      userRows[i],
			SharableResourceBaseInfo: shareRows[i],
			PrivateKeyLen:            len(keypair.PrivateKey),
		}
		keyIds[i] = keypair.Id
	}

	usages, err := manager.TotalResourceCount(keyIds)
	if err != nil {
		return rows
	}
	for i := range rows {
		if cnt, ok := usages[keyIds[i]]; ok {
			rows[i].LinkedGuestCount = cnt.LinkedGuestCount
		}
	}

	return rows
}

func (self *SKeypair) GetLinkedGuestsCount() (int, error) {
	return GuestManager.Query().Equals("keypair_id", self.Id).CountWithError()
}

func (manager *SKeypairManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.KeypairCreateInput) (api.KeypairCreateInput, error) {
	input.PublicKey = strings.TrimSpace(input.PublicKey)
	if len(input.PublicKey) == 0 {
		if len(input.Scheme) == 0 {
			input.Scheme = api.KEYPAIRE_SCHEME_RSA
		}
		if !utils.IsInStringArray(input.Scheme, api.KEYPAIR_SCHEMAS) {
			return input, httperrors.NewInputParameterError("Unsupported scheme %s", input.Scheme)
		}

		var err error
		if input.Scheme == api.KEYPAIRE_SCHEME_RSA {
			input.PrivateKey, input.PublicKey, err = seclib2.GenerateRSASSHKeypair()
		} else {
			input.PrivateKey, input.PublicKey, err = seclib2.GenerateDSASSHKeypair()
		}
		if err != nil {
			return input, httperrors.NewGeneralError(errors.Wrapf(err, "Generate%sSSHKeypair", input.Scheme))
		}
	}
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(input.PublicKey))
	if err != nil {
		return input, httperrors.NewInputParameterError("invalid public error: %v", err)
	}

	// 只允许上传RSA格式密钥。PS: AWS只支持RSA格式。
	input.Scheme = seclib2.GetPublicKeyScheme(pubKey)
	if input.Scheme != api.KEYPAIRE_SCHEME_RSA {
		return input, httperrors.NewInputParameterError("Unsupported scheme %s", input.Scheme)
	}

	input.Fingerprint = ssh.FingerprintLegacyMD5(pubKey)
	input.UserResourceCreateInput, err = manager.SUserResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.UserResourceCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (self *SKeypair) ValidateDeleteCondition(ctx context.Context, info *api.KeypairDetails) error {
	if gotypes.IsNil(info) {
		info = &api.KeypairDetails{}
		var err error
		info.LinkedGuestCount, err = self.GetLinkedGuestsCount()
		if err != nil {
			return httperrors.NewInternalServerError("GetLinkedGuestsCount failed %s", err)
		}
	}
	if info.LinkedGuestCount > 0 {
		return httperrors.NewNotEmptyError("Cannot delete keypair used by servers")
	}
	return self.SStandaloneResourceBase.ValidateDeleteCondition(ctx, nil)
}

func totalKeypairCount(userId string) (int, error) {
	q := KeypairManager.Query().Equals("owner_id", userId)
	return q.CountWithError()
}

func (manager *SKeypairManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	return db.SharableManagerFilterByOwner(ctx, manager.GetISharableVirtualModelManager(), q, userCred, owner, scope)
}

func (keypair *SKeypair) GetDetailsPrivatekey(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	retval := jsonutils.NewDict()
	if len(keypair.PrivateKey) > 0 {
		retval.Add(jsonutils.NewString(keypair.PrivateKey), "private_key")
		retval.Add(jsonutils.NewString(keypair.Name), "name")
		retval.Add(jsonutils.NewString(keypair.Scheme), "scheme")
		_, err := db.Update(keypair, func() error {
			keypair.PrivateKey = ""
			return nil
		})
		if err != nil {
			return nil, err
		}

		db.OpsLog.LogEvent(keypair, db.ACT_FETCH, nil, userCred)
		logclient.AddActionLogWithContext(ctx, keypair, logclient.ACT_FETCH, nil, userCred, true)
	}
	return retval, nil
}

func (self *SKeypair) GetOwnerId() mcclient.IIdentityProvider {
	owner := &db.SOwnerId{UserId: self.OwnerId}
	obj, err := db.UserCacheManager.FetchById(self.OwnerId)
	if err != nil {
		return owner
	}
	user := obj.(*db.SUser)
	owner.DomainId = user.DomainId
	return owner
}

func (self *SKeypair) GetProjectDomainId() string {
	obj, err := db.UserCacheManager.FetchById(self.OwnerId)
	if err != nil {
		return ""
	}
	user := obj.(*db.SUser)
	return user.DomainId
}

func (self *SKeypair) GetRequiredSharedDomainIds() []string {
	obj, err := db.UserCacheManager.FetchById(self.OwnerId)
	if err != nil {
		return []string{}
	}
	user := obj.(*db.SUser)
	return []string{user.DomainId}
}

func (self *SKeypair) GetSharableTargetDomainIds() []string {
	return []string{}
}

func (self *SKeypair) GetChangeOwnerRequiredDomainIds() []string {
	domainId := self.GetProjectDomainId()
	if len(domainId) > 0 {
		return []string{domainId}
	}
	return []string{}
}

func (self *SKeypair) GetChangeOwnerCandidateDomainIds() []string {
	domains := []db.STenant{}
	db.TenantCacheManager.GetDomainQuery().All(&domains)
	ret := []string{}
	for i := range domains {
		ret = append(ret, domains[i].Id)
	}
	return ret
}

func (self *SKeypair) GetSharedDomains() []string {
	return db.SharableGetSharedProjects(self, db.SharedTargetDomain)
}

func (keypair *SKeypair) GetISharableModel() db.ISharableBaseModel {
	return keypair.GetVirtualObject().(db.ISharableBaseModel)
}

func (keypair *SKeypair) GetDetailsChangeOwnerCandidateDomains(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (apis.ChangeOwnerCandidateDomainsOutput, error) {
	return db.IOwnerResourceBaseModelGetChangeOwnerCandidateDomains(keypair)
}

func (self *SKeypair) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPublicProjectInput) (jsonutils.JSONObject, error) {
	err := db.SharablePerformPublic(self.GetISharableModel(), ctx, userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "SharablePerformPublic")
	}
	return nil, nil
}

func (self *SKeypair) PerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPrivateInput) (jsonutils.JSONObject, error) {
	err := db.SharablePerformPrivate(self.GetISharableModel(), ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "SharablePerformPrivate")
	}
	return nil, nil
}
