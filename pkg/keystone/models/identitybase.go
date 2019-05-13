package models

import (
	"context"
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/log"
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SIdentityBaseResourceManager struct {
	db.SStandaloneResourceBaseManager
}

func NewIdentityBaseResourceManager(dt interface{}, tableName string, keyword string, keywordPlural string) SIdentityBaseResourceManager {
	return SIdentityBaseResourceManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(dt, tableName, keyword, keywordPlural),
	}
}

type SIdentityBaseResource struct {
	db.SStandaloneResourceBase

	Extra    *jsonutils.JSONDict `nullable:"true"`
	DomainId string              `width:"64" charset:"ascii" nullable:"false" index:"true" list:"admin" create:"admin_required"`
}

type SEnabledIdentityBaseResourceManager struct {
	SIdentityBaseResourceManager
}

func NewEnabledIdentityBaseResourceManager(dt interface{}, tableName string, keyword string, keywordPlural string) SEnabledIdentityBaseResourceManager {
	return SEnabledIdentityBaseResourceManager{
		SIdentityBaseResourceManager: NewIdentityBaseResourceManager(dt, tableName, keyword, keywordPlural),
	}
}

type SEnabledIdentityBaseResource struct {
	SIdentityBaseResource

	Enabled tristate.TriState `nullable:"false" default:"true" list:"admin" update:"admin" create:"admin_optional"`
}

func (model *SIdentityBaseResource) IsOwner(userCred mcclient.TokenCredential) bool {
	return userCred.GetProjectDomainId() == model.DomainId
}

func (model *SIdentityBaseResource) GetOwnerProjectId() string {
	return model.DomainId
}

func (model *SIdentityBaseResource) GetDomain() *SDomain {
	if len(model.DomainId) > 0 && model.DomainId != api.KeystoneDomainRoot {
		domain, err := DomainManager.FetchDomainById(model.DomainId)
		if err != nil {
			log.Errorf("GetDomain fail %s", err)
		}
		return domain
	}
	return nil
}

func (manager *SIdentityBaseResourceManager) FilterByOwner(q *sqlchemy.SQuery, owner string) *sqlchemy.SQuery {
	q = q.Equals("domain_id", owner)
	return q
}

func (manager *SIdentityBaseResourceManager) FetchByName(userCred mcclient.IIdentityProvider, idStr string) (db.IModel, error) {
	return db.FetchByName(manager, userCred, idStr)
}

func (manager *SIdentityBaseResourceManager) FetchByIdOrName(userCred mcclient.IIdentityProvider, idStr string) (db.IModel, error) {
	return db.FetchByIdOrName(manager, userCred, idStr)
}

func (manager *SIdentityBaseResourceManager) GetOwnerId(userCred mcclient.IIdentityProvider) string {
	return userCred.GetProjectDomainId()
}

func (manager *SIdentityBaseResourceManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	if jsonutils.QueryBoolean(query, "admin", false) { // admin
		domainStr := jsonutils.GetAnyString(query, []string{"domain", "domain_id", "project_domain", "project_domain_id"})
		if len(domainStr) > 0 {
			domain, err := DomainManager.FetchDomainByIdOrName(domainStr)
			if err != nil {
				if err == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError2("domain", domainStr)
				} else {
					return nil, httperrors.NewGeneralError(err)
				}
			}
			q = q.Equals("domain_id", domain.Id)
		}
	}
	return q, nil
}

func (manager *SIdentityBaseResourceManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	domainStr := jsonutils.GetAnyString(data, []string{"domain_id", "domain"})
	if len(domainStr) > 0 && domainStr != api.KeystoneDomainRoot {
		domain, err := DomainManager.FetchDomainByIdOrName(domainStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(DomainManager.Keyword(), domainStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		if domain.IsReadOnly() {
			return nil, httperrors.NewForbiddenError("domain is readonly")
		}
		data.Add(jsonutils.NewString(domain.Id), "domain_id")
	} else if len(domainStr) == 0 {
		data.Add(jsonutils.NewString(api.DEFAULT_DOMAIN_ID), "domain_id")
	}
	return manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (self *SIdentityBaseResource) ValidateDeleteCondition(ctx context.Context) error {
	domain := self.GetDomain()
	if domain != nil && domain.IsReadOnly() {
		return httperrors.NewForbiddenError("readonly domain")
	}
	return self.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SIdentityBaseResource) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if data.Contains("name") {
		domain := self.GetDomain()
		if domain != nil && domain.IsReadOnly() {
			return nil, httperrors.NewForbiddenError("cannot update name in readonly domain")
		}
	}
	return self.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (self *SEnabledIdentityBaseResource) ValidateDeleteCondition(ctx context.Context) error {
	if self.Enabled.IsTrue() {
		return httperrors.NewResourceBusyError("resource is enabled")
	}
	return self.SIdentityBaseResource.ValidateDeleteCondition(ctx)
}
