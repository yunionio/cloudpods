package models

import (
	"database/sql"

	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	"context"
	"github.com/pkg/errors"
	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SDomainManager struct {
	SIdentityBaseResourceManager
}

var (
	DomainManager *SDomainManager
)

func init() {
	DomainManager = &SDomainManager{
		SIdentityBaseResourceManager: NewIdentityBaseResourceManager(
			SDomain{},
			"project",
			"domain",
			"domains",
		),
	}
}

type SDomain struct {
	SBaseProject
}

func (manager *SDomainManager) InitializeData() error {
	root, err := manager.FetchDomainById(api.KeystoneDomainRoot)
	if err == sql.ErrNoRows {
		root = &SDomain{}
		root.Id = api.KeystoneDomainRoot
		root.Name = api.KeystoneDomainRoot
		root.IsDomain = tristate.True
		root.ParentId = api.KeystoneDomainRoot
		root.DomainId = api.KeystoneDomainRoot
		root.Enabled = tristate.False
		root.Description = "The hidden root domain"
		err := manager.TableSpec().Insert(root)
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
		defDomain.ParentId = api.KeystoneDomainRoot
		defDomain.DomainId = api.KeystoneDomainRoot
		defDomain.Enabled = tristate.True
		defDomain.Description = "The default domain"
		err := manager.TableSpec().Insert(defDomain)
		if err != nil {
			log.Errorf("fail to insert default domain ... %s", err)
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

func (manager *SDomainManager) GetOwnerId(userCred mcclient.IIdentityProvider) string {
	return api.KeystoneDomainRoot
}

func (manager *SDomainManager) FetchDomainByName(domainName string) (*SDomain, error) {
	obj, err := db.NewModelObject(manager)
	if err != nil {
		return nil, err
	}
	q := manager.Query().Equals("name", domainName).IsTrue("is_domain").NotEquals("id", api.KeystoneDomainRoot)
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
	q := manager.Query().Equals("id", domainId).IsTrue("is_domain") // .NotEquals("id", api.KeystoneDomainRoot)
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
	obj, err := db.NewModelObject(manager)
	if err != nil {
		return nil, err
	}
	q := manager.Query().IsTrue("is_domain").NotEquals("id", api.KeystoneDomainRoot)
	q = q.Filter(sqlchemy.OR(
		sqlchemy.Equals(q.Field("id"), domain),
		sqlchemy.Equals(q.Field("name"), domain),
	))
	err = q.First(obj)
	if err != nil {
		return nil, err
	}
	return obj.(*SDomain), err
}

func (manager *SDomainManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SIdentityBaseResourceManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	q = q.NotEquals("id", api.KeystoneDomainRoot).IsTrue("is_domain")
	return q, nil
}

func (self *SDomain) AllowGetDetailsConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGetSpec(userCred, self, "config")
}

func (self *SDomain) GetDetailsConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	appParams := appsrv.AppContextGetParams(ctx)
	appParams.OverrideResponseBodyWrapper = true

	conf, err := self.GetConfig(false)
	if err != nil {
		return nil, err
	}
	result := jsonutils.NewDict()
	result.Add(jsonutils.Marshal(conf), "config")
	return result, nil
}

func (self *SDomain) GetConfig(all bool) (TDomainConfigs, error) {
	opts, err := WhitelistedConfigManager.fetchConfigs(self.Id, nil, nil)
	if err != nil {
		return nil, err
	}
	if all {
		opts2, err := SensitiveConfigManager.fetchConfigs(self.Id, nil, nil)
		if err != nil {
			return nil, err
		}
		opts = append(opts, opts2...)
	}
	return config2map(opts), nil
}

func (manager *SDomainManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data.Set("domain_id", jsonutils.NewString(api.KeystoneDomainRoot))
	data.Set("is_domain", jsonutils.JSONTrue)
	return manager.SIdentityBaseResourceManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (domain *SDomain) GetProjectCount() (int, error) {
	q := ProjectManager.Query().IsFalse("is_domain").Equals("domain_id", domain.Id)
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

func (domain *SDomain) ValidateDeleteCondition(ctx context.Context) error {
	projCnt, _ := domain.GetProjectCount()
	if projCnt > 0 {
		return httperrors.NewNotEmptyError("domain is in use")
	}
	usrCnt, _ := domain.GetUserCount()
	if usrCnt > 0 {
		return httperrors.NewNotEmptyError("domain is in use")
	}
	grpCnt, _ := domain.GetGroupCount()
	if grpCnt > 0 {
		return httperrors.NewNotEmptyError("domain is in use")
	}
	return domain.SEnabledIdentityBaseResource.ValidateDeleteCondition(ctx)
}

func (domain *SDomain) GetDriver() string {
	drv, _ := domain.getDriver()
	return drv
}

func (domain *SDomain) getDriver() (string, error) {
	opts, err := WhitelistedConfigManager.fetchConfigs(domain.Id, []string{"identity"}, []string{"driver"})
	if err != nil {
		return "", errors.WithMessage(err, "WhitelistedConfigManager.fetchConfigs")
	}
	if len(opts) == 1 {
		return opts[0].Value.GetString()
	}
	return api.IdentityDriverSQL, nil
}

func (domain *SDomain) IsReadOnly() bool {
	if domain.GetDriver() == api.IdentityDriverSQL {
		return false
	}
	return true
}

func (domain *SDomain) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := domain.SEnabledIdentityBaseResource.GetCustomizeColumns(ctx, userCred, query)
	return domainExtra(domain, extra)
}

func (domain *SDomain) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := domain.SEnabledIdentityBaseResource.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return domainExtra(domain, extra), nil
}

func domainExtra(domain *SDomain, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	if domain.IsReadOnly() {
		extra.Add(jsonutils.JSONTrue, "readonly")
	}
	extra.Add(jsonutils.NewString(domain.GetDriver()), "driver")

	usrCnt, _ := domain.GetUserCount()
	extra.Add(jsonutils.NewInt(int64(usrCnt)), "user_count")
	grpCnt, _ := domain.GetGroupCount()
	extra.Add(jsonutils.NewInt(int64(grpCnt)), "group_count")
	prjCnt, _ := domain.GetProjectCount()
	extra.Add(jsonutils.NewInt(int64(prjCnt)), "project_count")
	return extra
}

func (domain *SDomain) AllowUpdateConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) bool {
	return db.IsAdminAllowUpdateSpec(userCred, domain, "config")
}

func (domain *SDomain) UpdateConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	opts := TDomainConfigs{}
	err := data.Unmarshal(&opts, "config")
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid input data")
	}
	whiteListedOpts, sensitiveOpts := opts.getConfigOptions(domain.Id, api.SensitiveDomainConfigMap)
	err = WhitelistedConfigManager.syncConfig(ctx, userCred, domain.Id, whiteListedOpts)
	if err != nil {
		return nil, httperrors.NewInternalServerError("WhitelistedConfigManager.syncConfig fail %s", err)
	}
	err = SensitiveConfigManager.syncConfig(ctx, userCred, domain.Id, sensitiveOpts)
	if err != nil {
		return nil, httperrors.NewInternalServerError("SensitiveConfigManager.syncConfig fail %s", err)
	}
	return domain.GetDetailsConfig(ctx, userCred, query)
}

func (domain *SDomain) AllowDeleteConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) bool {
	return db.IsAdminAllowDeleteSpec(userCred, domain, "config")
}

func (domain *SDomain) DeleteConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	err := WhitelistedConfigManager.deleteConfig(ctx, userCred, domain.Id)
	if err != nil {
		return nil, httperrors.NewInternalServerError("WhitelistedConfigManager.syncConfig fail %s", err)
	}
	err = SensitiveConfigManager.deleteConfig(ctx, userCred, domain.Id)
	if err != nil {
		return nil, httperrors.NewInternalServerError("SensitiveConfigManager.syncConfig fail %s", err)
	}
	return domain.GetDetailsConfig(ctx, userCred, query)
}
