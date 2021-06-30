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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SDomainManager struct {
	db.SStandaloneResourceBaseManager
}

var (
	DomainManager *SDomainManager
)

func init() {
	DomainManager = &SDomainManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SDomain{},
			"project",
			"domain",
			"domains",
		),
	}
	DomainManager.SetVirtualObject(DomainManager)
}

type SDomain struct {
	db.SStandaloneResourceBase

	Extra *jsonutils.JSONDict `nullable:"true"`

	Enabled  tristate.TriState `nullable:"false" default:"true" list:"admin" update:"admin" create:"admin_optional"`
	IsDomain tristate.TriState `default:"false" nullable:"false"`

	// IdpId string `token:"parent_id" width:"64" charset:"ascii" index:"true" list:"admin"`

	DomainId string `width:"64" charset:"ascii" default:"default" nullable:"false" index:"true"`
	ParentId string `width:"64" charset:"ascii"`
}

func (manager *SDomainManager) InitializeData() error {
	root, err := manager.FetchDomainById(api.KeystoneDomainRoot)
	if err == sql.ErrNoRows {
		root = &SDomain{}
		root.Id = api.KeystoneDomainRoot
		root.Name = api.KeystoneDomainRoot
		root.IsDomain = tristate.True
		// root.ParentId = api.KeystoneDomainRoot
		root.DomainId = api.KeystoneDomainRoot
		root.Enabled = tristate.False
		root.Description = "The hidden root domain"
		err := manager.TableSpec().Insert(context.TODO(), root)
		if err != nil {
			log.Errorf("fail to insert root domain ... %s", err)
			return err
		}
	} else if err != nil {
		return err
	}
	defDomain, err := manager.FetchDomainById(api.DEFAULT_DOMAIN_ID)
	if err == sql.ErrNoRows {
		defDomain = &SDomain{}
		defDomain.Id = api.DEFAULT_DOMAIN_ID
		defDomain.Name = api.DEFAULT_DOMAIN_NAME
		defDomain.IsDomain = tristate.True
		// defDomain.ParentId = api.KeystoneDomainRoot
		defDomain.DomainId = api.KeystoneDomainRoot
		defDomain.Enabled = tristate.True
		defDomain.Description = "The default domain"
		err := manager.TableSpec().Insert(context.TODO(), defDomain)
		if err != nil {
			log.Errorf("fail to insert default domain ... %s", err)
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

func (manager *SDomainManager) Query(fields ...string) *sqlchemy.SQuery {
	return manager.SStandaloneResourceBaseManager.Query(fields...).IsTrue("is_domain")
}

func (manager *SDomainManager) FetchDomainByName(domainName string) (*SDomain, error) {
	obj, err := db.NewModelObject(manager)
	if err != nil {
		return nil, err
	}
	q := manager.Query().Equals("name", domainName).NotEquals("id", api.KeystoneDomainRoot)
	err = q.First(obj)
	if err != nil {
		return nil, err
	}
	return obj.(*SDomain), err
}

func (manager *SDomainManager) FetchDomainById(domainId string) (*SDomain, error) {
	obj, err := db.NewModelObject(manager)
	if err != nil {
		return nil, err
	}
	q := manager.Query().Equals("id", domainId)
	err = q.First(obj)
	if err != nil {
		return nil, err
	}
	return obj.(*SDomain), err
}

func (manager *SDomainManager) FetchDomain(domainId string, domainName string) (*SDomain, error) {
	if len(domainId) == 0 && len(domainName) == 0 {
		domainId = api.DEFAULT_DOMAIN_ID
	}
	if len(domainId) > 0 {
		return manager.FetchDomainById(domainId)
	} else {
		return manager.FetchDomainByName(domainName)
	}
}

func (manager *SDomainManager) FetchDomainByIdOrName(domain string) (*SDomain, error) {
	if stringutils2.IsUtf8(domain) {
		return manager.FetchDomainByName(domain)
	}
	obj, err := db.NewModelObject(manager)
	if err != nil {
		return nil, err
	}
	q := manager.Query().NotEquals("id", api.KeystoneDomainRoot)
	if stringutils2.IsUtf8(domain) {
		q = q.Equals("name", domain)
	} else {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.Equals(q.Field("id"), domain),
			sqlchemy.Equals(q.Field("name"), domain),
		))
	}
	err = q.First(obj)
	if err != nil {
		return nil, err
	}
	return obj.(*SDomain), err
}

// 域列表
func (manager *SDomainManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DomainListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	q = q.NotEquals("id", api.KeystoneDomainRoot)

	if query.Enabled != nil {
		if *query.Enabled {
			q = q.IsTrue("enabled")
		} else {
			q = q.IsFalse("enabled")
		}
	}

	if len(query.IdpId) > 0 {
		idpObj, err := IdentityProviderManager.FetchByIdOrName(userCred, query.IdpId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", IdentityProviderManager.Keyword(), query.IdpId)
			} else {
				return nil, errors.Wrap(err, "IdentityProviderManager.FetchByIdOrName")
			}
		}
		subq := IdmappingManager.FetchPublicIdsExcludesQuery(idpObj.GetId(), api.IdMappingEntityDomain, nil)
		q = q.In("id", subq.SubQuery())
	}

	if len(query.IdpEntityId) > 0 {
		subq := IdmappingManager.Query("public_id").Equals("local_id", query.IdpEntityId).Equals("entity_type", api.IdMappingEntityDomain)
		q = q.Equals("id", subq.SubQuery())
	}

	return q, nil
}

func (manager *SDomainManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DomainListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SDomainManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (domain *SDomain) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	// domain.ParentId = api.KeystoneDomainRoot
	domain.DomainId = api.KeystoneDomainRoot
	domain.IsDomain = tristate.True
	return domain.SStandaloneResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (domain *SDomain) GetProjectCount() (int, error) {
	q := ProjectManager.Query().Equals("domain_id", domain.Id)
	return q.CountWithError()
}

func (domain *SDomain) GetRoleCount() (int, error) {
	q := RoleManager.Query().Equals("domain_id", domain.Id)
	return q.CountWithError()
}

func (domain *SDomain) GetPolicyCount() (int, error) {
	q := PolicyManager.Query().Equals("domain_id", domain.Id)
	return q.CountWithError()
}

func (domain *SDomain) GetUserCount() (int, error) {
	q := UserManager.Query().Equals("domain_id", domain.Id)
	return q.CountWithError()
}

func (domain *SDomain) GetGroupCount() (int, error) {
	q := GroupManager.Query().Equals("domain_id", domain.Id)
	return q.CountWithError()
}

func (domain *SDomain) GetIdpCount() (int, error) {
	q := IdentityProviderManager.Query().Equals("target_domain_id", domain.Id)
	return q.CountWithError()
}

func (domain *SDomain) ValidatePurgeCondition(ctx context.Context) error {
	if domain.Id == api.DEFAULT_DOMAIN_ID {
		return httperrors.NewForbiddenError("cannot delete default domain")
	}
	if domain.Enabled.IsTrue() {
		return httperrors.NewInvalidStatusError("domain is enabled")
	}
	usrCnt, _ := domain.GetUserCount()
	if usrCnt > 0 {
		return httperrors.NewInvalidStatusError("domain is in use by user")
	}
	groupCnt, _ := domain.GetGroupCount()
	if groupCnt > 0 {
		return httperrors.NewInvalidStatusError("domain is in use by group")
	}
	projCnt, _ := domain.GetProjectCount()
	if projCnt > 0 {
		return httperrors.NewNotEmptyError("domain is in use by project")
	}
	roleCnt, _ := domain.GetRoleCount()
	if roleCnt > 0 {
		return httperrors.NewNotEmptyError("domain is in use by role")
	}
	policyCnt, _ := domain.GetPolicyCount()
	if policyCnt > 0 {
		return httperrors.NewNotEmptyError("domain is in use by policy")
	}
	external, _, _ := domain.getExternalResources()
	if len(external) > 0 {
		return httperrors.NewNotEmptyError("domain contains external resources")
	}
	return nil
}

func (domain *SDomain) ValidateDeleteCondition(ctx context.Context) error {
	if domain.IsReadOnly() {
		return httperrors.NewForbiddenError("readonly")
	}
	err := domain.ValidatePurgeCondition(ctx)
	if err != nil {
		return err
	}
	return domain.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (domain *SDomain) ValidateUpdateCondition(ctx context.Context) error {
	if domain.Id == api.DEFAULT_DOMAIN_ID {
		return httperrors.NewForbiddenError("default domain is protected")
	}
	return domain.SStandaloneResourceBase.ValidateUpdateCondition(ctx)
}

func (domain *SDomain) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DomainUpdateInput) (api.DomainUpdateInput, error) {
	data := jsonutils.Marshal(input)
	if domain.IsReadOnly() {
		for _, k := range []string{
			"name",
		} {
			if data.Contains(k) {
				return input, httperrors.NewForbiddenError("field %s is readonly", k)
			}
		}
	}
	var err error
	input.StandaloneResourceBaseUpdateInput, err = domain.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.StandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStandaloneResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (manager *SDomainManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.DomainDetails {
	rows := make([]api.DomainDetails, len(objs))

	stdRows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	idList := make([]string, len(rows))
	for i := range rows {
		rows[i] = api.DomainDetails{
			StandaloneResourceDetails: stdRows[i],
		}
		domain := objs[i].(*SDomain)
		idList[i] = domain.Id
		rows[i].UserCout, _ = domain.GetUserCount()
		rows[i].GroupCount, _ = domain.GetGroupCount()
		rows[i].ProjectCout, _ = domain.GetProjectCount()
		rows[i].RoleCount, _ = domain.GetRoleCount()
		rows[i].PolicyCount, _ = domain.GetPolicyCount()
		rows[i].IdpCount, _ = domain.GetIdpCount()

		external, update, _ := domain.getExternalResources()
		if len(external) > 0 {
			rows[i].ExtResource = jsonutils.Marshal(external)
			rows[i].ExtResourcesLastUpdate = update
			if update.IsZero() {
				update = time.Now()
			}
			nextUpdate := update.Add(time.Duration(options.Options.FetchScopeResourceCountIntervalSeconds) * time.Second)
			rows[i].ExtResourcesNextUpdate = nextUpdate
		}
	}

	idpRows := expandIdpAttributes(api.IdMappingEntityDomain, idList, fields)

	for i := range rows {
		rows[i].IdpResourceInfo = idpRows[i]
	}

	return rows
}

func (domain *SDomain) getUsers() ([]SUser, error) {
	q := UserManager.Query().Equals("domain_id", domain.Id)
	usrs := make([]SUser, 0)
	err := db.FetchModelObjects(UserManager, q, &usrs)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return usrs, nil
}

func (domain *SDomain) getGroups() ([]SGroup, error) {
	q := GroupManager.Query().Equals("domain_id", domain.Id)
	grps := make([]SGroup, 0)
	err := db.FetchModelObjects(GroupManager, q, &grps)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return grps, nil
}

func (domain *SDomain) DeleteUserGroups(ctx context.Context, userCred mcclient.TokenCredential) error {
	usrs, err := domain.getUsers()
	if err != nil {
		return errors.Wrap(err, "domain.getUsers")
	}
	for i := range usrs {
		err = usrs[i].ValidateDeleteCondition(ctx)
		if err != nil {
			return errors.Wrap(err, "usr.ValidateDeleteCondition")
		}
		err = usrs[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "usr.Delete")
		}
	}
	grps, err := domain.getGroups()
	if err != nil {
		return errors.Wrap(err, "domain.getGroups")
	}
	for i := range grps {
		err = grps[i].ValidateDeleteCondition(ctx)
		if err != nil {
			return errors.Wrap(err, "grp.ValidateDeleteCondition")
		}
		err = grps[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "grp.Delete")
		}
	}
	return nil
}

func (domain *SDomain) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := domain.DeleteUserGroups(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "domain.DeleteUserGroups")
	}
	return domain.SStandaloneResourceBase.Delete(ctx, userCred)
}

func (domain *SDomain) getIdmapping() (*SIdmapping, error) {
	return IdmappingManager.FetchFirstEntity(domain.Id, api.IdMappingEntityDomain)
}

func (domain *SDomain) IsReadOnly() bool {
	idmap, _ := domain.getIdmapping()
	if idmap != nil {
		return true
	}
	return false
}

func (domain *SDomain) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	domain.SStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	logclient.AddActionLogWithContext(ctx, domain, logclient.ACT_CREATE, data, userCred, true)
}

func (domain *SDomain) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	domain.SStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)
	logclient.AddActionLogWithContext(ctx, domain, logclient.ACT_UPDATE, data, userCred, true)
}

func (domain *SDomain) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	domain.SStandaloneResourceBase.PostDelete(ctx, userCred)
	logclient.AddActionLogWithContext(ctx, domain, logclient.ACT_DELETE, nil, userCred, true)
}

func (domain *SDomain) UnlinkIdp(idpId string) error {
	usrs, err := domain.getUsers()
	if err != nil {
		return errors.Wrap(err, "domain.getUsers")
	}
	for i := range usrs {
		err = usrs[i].UnlinkIdp(idpId)
		if err != nil {
			return errors.Wrap(err, "usr.UnlinkIdp")
		}
	}
	grps, err := domain.getGroups()
	if err != nil {
		return errors.Wrap(err, "domain.getGroups")
	}
	for i := range grps {
		err = grps[i].UnlinkIdp(idpId)
		if err != nil {
			return errors.Wrap(err, "grp.UnlinkIdp")
		}
	}
	return IdmappingManager.deleteAny(idpId, api.IdMappingEntityDomain, domain.Id)
}

func (domain *SDomain) getExternalResources() (map[string]int, time.Time, error) {
	return ScopeResourceManager.getScopeResource(domain.Id, "", "")
}

func (manager *SDomainManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.DomainCreateInput,
) (api.DomainCreateInput, error) {
	var err error

	input.StandaloneResourceCreateInput, err = manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StandaloneResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStandaloneResourceBaseManager.ValidateCreateData")
	}

	return input, nil
}

func (domain *SDomain) AllowPerformUnlinkIdp(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.UserUnlinkIdpInput,
) bool {
	return db.IsAdminAllowPerform(userCred, domain, "unlink-idp")
}

// domain和IDP的指定entityId解除关联
func (domain *SDomain) PerformUnlinkIdp(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.UserUnlinkIdpInput,
) (jsonutils.JSONObject, error) {
	mapping, err := domain.getIdmapping()
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, errors.Wrap(err, "domain.getIdmapping")
		}
	}
	err = domain.UnlinkIdp(mapping.IdpId)
	if err != nil {
		return nil, errors.Wrap(err, "domain.UnlinkIdp")
	}
	return nil, nil
}
