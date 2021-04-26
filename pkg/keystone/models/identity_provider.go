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
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/keystone/saml"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/samlutils/sp"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SIdentityProviderManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
	db.SDomainizedResourceBaseManager
}

var (
	IdentityProviderManager *SIdentityProviderManager
)

func init() {
	IdentityProviderManager = &SIdentityProviderManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SIdentityProvider{},
			api.IDENTITY_PROVIDER_TABLE,
			api.IDENTITY_PROVIDER_RESOURCE_TYPE,
			api.IDENTITY_PROVIDER_RESOURCE_TYPES,
		),
	}
	IdentityProviderManager.SetVirtualObject(IdentityProviderManager)
}

/*
desc identity_provider;
+-------------+-------------+------+-----+---------+-------+
| Field       | Type        | Null | Key | Default | Extra |
+-------------+-------------+------+-----+---------+-------+
| id          | varchar(64) | NO   | PRI | NULL    |       |
| enabled     | tinyint(1)  | NO   |     | NULL    |       |
| description | text        | YES  |     | NULL    |       |
| domain_id   | varchar(64) | NO   | MUL | NULL    |       |
+-------------+-------------+------+-----+---------+-------+
*/

type SIdentityProvider struct {
	db.SEnabledStatusStandaloneResourceBase

	db.SDomainizedResourceBase `default:""`

	Driver   string `width:"32" charset:"ascii" nullable:"false" list:"domain" create:"domain_required"`
	Template string `width:"32" charset:"ascii" nullable:"true" list:"domain" create:"domain_optional"`

	TargetDomainId string `width:"64" charset:"ascii" nullable:"true" list:"domain" create:"admin_optional"`

	// 是否自动创建项目
	AutoCreateProject tristate.TriState `default:"true" nullable:"true" list:"domain" create:"domain_optional" update:"domain"`
	// 是否自动创建用户
	AutoCreateUser tristate.TriState `nullable:"true" list:"domain" create:"domain_optional" update:"domain"`

	ErrorCount int `list:"domain"`

	SyncStatus    string    `width:"10" charset:"ascii" default:"idle" list:"domain"`
	LastSync      time.Time `list:"domain"` // = Column(DateTime, nullable=True)
	LastSyncEndAt time.Time `list:"domain"`

	SyncIntervalSeconds int `create:"domain_optional" update:"domain"`

	// 认证源图标
	IconUri string `width:"256" charset:"utf8" nullable:"true" list:"user" create:"domain_optional" update:"domain"`
	// 是否是SSO登录方式
	IsSso tristate.TriState `nullable:"true" list:"domain"`
	// 是否是缺省SSO登录方式
	IsDefault tristate.TriState `nullable:"true" list:"domain"`
}

func (manager *SIdentityProviderManager) initializeAutoCreateUser() error {
	q := manager.Query().IsNull("auto_create_user")
	idps := make([]SIdentityProvider, 0)
	err := db.FetchModelObjects(manager, q, &idps)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil
		} else {
			return errors.Wrap(err, "FetchModelObjects")
		}
	}
	for i := range idps {
		drvCls := idps[i].getDriverClass()
		_, err := db.Update(&idps[i], func() error {
			if drvCls.ForceSyncUser() {
				idps[i].AutoCreateUser = tristate.True
			} else {
				idps[i].AutoCreateUser = tristate.False
			}
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "update auto_create_user")
		}
	}
	return nil
}

func (manager *SIdentityProviderManager) initializeIcon() error {
	q := manager.Query().IsNull("is_sso")
	idps := make([]SIdentityProvider, 0)
	err := db.FetchModelObjects(manager, q, &idps)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil
		} else {
			return errors.Wrap(err, "FetchModelObjects")
		}
	}
	for i := range idps {
		drvCls := idps[i].getDriverClass()
		_, err := db.Update(&idps[i], func() error {
			if drvCls.IsSso() {
				idps[i].IsSso = tristate.True
				idps[i].IconUri = drvCls.GetDefaultIconUri(idps[i].Template)
			} else {
				idps[i].IsSso = tristate.False
				idps[i].IconUri = drvCls.GetDefaultIconUri(idps[i].Template)
			}
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "update is_sso")
		}
	}
	return nil
}

func (manager *SIdentityProviderManager) InitializeData() error {
	err := manager.initializeAutoCreateUser()
	if err != nil {
		return errors.Wrap(err, "initializeAutoCreateUser")
	}
	err = manager.initializeIcon()
	if err != nil {
		return errors.Wrap(err, "initializeIcon")
	}

	cnt, err := manager.Query().CountWithError()
	if err != nil {
		return errors.Wrap(err, "CountWithError")
	}
	if cnt > 0 {
		return nil
	}

	// copy domains
	// first create a sql provider
	sqldrv := SIdentityProvider{}
	sqldrv.SetModelManager(manager, &sqldrv)
	sqldrv.Id = api.DEFAULT_IDP_ID
	sqldrv.Name = api.IdentityDriverSQL
	sqldrv.SetEnabled(true)
	sqldrv.Status = api.IdentityDriverStatusConnected
	sqldrv.Driver = api.IdentityDriverSQL
	sqldrv.Description = "Default sql identity provider"
	sqldrv.AutoCreateUser = tristate.True
	sqldrv.AutoCreateProject = tristate.False
	sqldrv.IsSso = tristate.False
	sqldrv.IconUri = ""
	err = manager.TableSpec().Insert(context.TODO(), &sqldrv)
	if err != nil {
		return errors.Wrap(err, "insert default sql driver")
	}

	// then, insert all none-sql domain drivers
	q := DomainManager.Query().NotEquals("id", api.KeystoneDomainRoot)
	domains := make([]SDomain, 0)
	err = db.FetchModelObjects(DomainManager, q, &domains)
	if err != nil {
		return errors.Wrap(err, "query domains")
	}

	for i := range domains {
		driver, err := WhitelistedConfigManager.getDriver(domains[i].Id)
		if err != nil {
			// get driver fail
			return errors.Wrap(err, "WhitelistedConfigManager.getDriver")
		}
		if driver == api.IdentityDriverSQL {
			// sql driver, skip
			continue
		}

		drv := SIdentityProvider{}
		drv.SetModelManager(manager, &drv)
		drv.Id = domains[i].Id // identical ID with domain, for backward compatibility
		drv.Name = domains[i].Name
		drv.SetEnabled(domains[i].Enabled.Bool())
		drv.Status = api.IdentityDriverStatusDisconnected
		drv.Driver = driver
		drv.Description = domains[i].Description
		err = manager.TableSpec().Insert(context.TODO(), &drv)
		if err != nil {
			return errors.Wrap(err, "insert driver")
		}
		_, err = IdmappingManager.RegisterIdMapWithId(context.Background(), drv.Id, api.DefaultRemoteDomainId, api.IdMappingEntityDomain, domains[i].Id)
		if err != nil {
			return errors.Wrap(err, "RegisterIdMapWithId")
		}
	}
	return nil
}

func (ident *SIdentityProvider) SetSyncStatus(ctx context.Context, userCred mcclient.TokenCredential, status string) error {
	if status != ident.SyncStatus {
		_, err := db.UpdateWithLock(ctx, ident, func() error {
			ident.SyncStatus = status
			switch status {
			case api.IdentitySyncStatusQueued:
				ident.LastSync = time.Now().UTC()
				ident.LastSyncEndAt = time.Time{}
			case api.IdentitySyncStatusSyncing:
				ident.LastSync = time.Now().UTC()
			case api.IdentitySyncStatusIdle:
				ident.LastSyncEndAt = time.Now().UTC()
			}
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "updateWithLock")
		}
	}
	return nil
}

func (ident *SIdentityProvider) MarkConnected(ctx context.Context, userCred mcclient.TokenCredential) error {
	if ident.ErrorCount > 0 {
		_, err := db.UpdateWithLock(ctx, ident, func() error {
			ident.ErrorCount = 0
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "UpdateWithLock")
		}
	}
	if ident.Status != api.IdentityDriverStatusConnected {
		logclient.AddSimpleActionLog(ident, logclient.ACT_ENABLE, nil, userCred, true)
		return ident.SetStatus(userCred, api.IdentityDriverStatusConnected, "")
	}
	return nil
}

func (ident *SIdentityProvider) MarkDisconnected(ctx context.Context, userCred mcclient.TokenCredential, reason error) error {
	_, err := db.UpdateWithLock(ctx, ident, func() error {
		ident.ErrorCount = ident.ErrorCount + 1
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "UpdateWithLock")
	}
	logclient.AddSimpleActionLog(ident, logclient.ACT_DISABLE, reason.Error(), userCred, false)
	if ident.Status != api.IdentityDriverStatusDisconnected {
		return ident.SetStatus(userCred, api.IdentityDriverStatusDisconnected, reason.Error())
	}
	return nil
}

func (self *SIdentityProvider) AllowGetDetailsConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGetSpec(userCred, self, "config")
}

func (self *SIdentityProvider) GetDetailsConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	sensitive := jsonutils.QueryBoolean(query, "sensitive", false)
	if sensitive {
		if !db.IsAdminAllowGetSpec(userCred, self, "config") {
			return nil, httperrors.NewNotSufficientPrivilegeError("get sensitive config requires admin priviliges")
		}
	}
	conf, err := GetConfigs(self, sensitive, nil, nil)
	if err != nil {
		return nil, err
	}
	result := jsonutils.NewDict()
	result.Add(jsonutils.Marshal(conf), "config")
	return result, nil
}

func (ident *SIdentityProvider) getDriverClass() driver.IIdentityBackendClass {
	return driver.GetDriverClass(ident.Driver)
}

// 配置认证源
func (ident *SIdentityProvider) AllowPerformConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.PerformConfigInput) bool {
	return db.IsAdminAllowUpdateSpec(userCred, ident, "config")
}

// 配置认证源
func (ident *SIdentityProvider) PerformConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.PerformConfigInput) (jsonutils.JSONObject, error) {
	if ident.Status == api.IdentityDriverStatusConnected && ident.GetEnabled() {
		return nil, httperrors.NewInvalidStatusError("cannot update config when enabled and connected")
	}
	if ident.SyncStatus != api.IdentitySyncStatusIdle {
		return nil, httperrors.NewInvalidStatusError("cannot update config when not idle")
	}

	var err error
	input.Config, err = ident.getDriverClass().ValidateConfig(ctx, userCred, ident.Template, input.Config, ident.Id, ident.DomainId)
	if err != nil {
		return nil, errors.Wrap(err, "ValidateConfig")
	}

	opts := input.Config
	action := input.Action
	changed, err := saveConfigs(userCred, action, ident, opts, nil, nil, api.SensitiveDomainConfigMap)
	if err != nil {
		return nil, httperrors.NewInternalServerError("saveConfigs fail %s", err)
	}
	if changed {
		ident.MarkDisconnected(ctx, userCred, fmt.Errorf("change config"))
		submitIdpSyncTask(ctx, userCred, ident)
	}
	return ident.GetDetailsConfig(ctx, userCred, query)
}

func (manager *SIdentityProviderManager) getDriveInstanceCount(drvName string) (int, error) {
	return manager.Query().Equals("driver", drvName).CountWithError()
}

func (manager *SIdentityProviderManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.IdentityProviderCreateInput,
) (api.IdentityProviderCreateInput, error) {
	var drvName string

	template := input.Template
	if len(template) > 0 {
		if _, ok := api.IdpTemplateDriver[template]; !ok {
			return input, httperrors.NewInputParameterError("invalid template")
		}
		drvName = api.IdpTemplateDriver[template]
		input.Driver = drvName
	} else {
		drvName = input.Driver
		if len(drvName) == 0 {
			return input, httperrors.NewInputParameterError("missing driver")
		}
	}

	drvCls := driver.GetDriverClass(drvName)
	if drvCls == nil {
		return input, httperrors.NewInputParameterError("driver %s not supported", drvName)
	}

	if drvCls.SingletonInstance() {
		cnt, err := manager.getDriveInstanceCount(drvName)
		if err != nil {
			return input, httperrors.NewGeneralError(err)
		}
		if cnt >= 1 {
			return input, httperrors.NewConflictError("driver %s already exists", drvName)
		}
	}

	if input.SyncIntervalSeconds != nil {
		secs := *input.SyncIntervalSeconds
		if secs < api.MinimalSyncIntervalSeconds {
			secs = api.MinimalSyncIntervalSeconds
			input.SyncIntervalSeconds = &secs
		}
	}

	ownerDomainStr := input.OwnerDomainId
	if len(ownerDomainStr) > 0 {
		domain, err := DomainManager.FetchDomainByIdOrName(ownerDomainStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return input, httperrors.NewResourceNotFoundError2(DomainManager.Keyword(), ownerDomainStr)
			} else {
				return input, httperrors.NewGeneralError(err)
			}
		}
		input.OwnerDomainId = domain.Id
		if domain.Id != ownerId.GetProjectDomainId() && !db.IsAdminAllowCreate(userCred, manager) {
			return input, errors.Wrap(httperrors.ErrNotSufficientPrivilege, "require system priviliges to specify owner_domain_id")
		}
	} else if !db.IsAdminAllowCreate(userCred, manager) {
		input.OwnerDomainId = ownerId.GetProjectDomainId()
	}

	targetDomainStr := input.TargetDomainId
	if len(targetDomainStr) > 0 {
		domain, err := DomainManager.FetchDomainByIdOrName(targetDomainStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return input, httperrors.NewResourceNotFoundError2(DomainManager.Keyword(), targetDomainStr)
			} else {
				return input, httperrors.NewGeneralError(err)
			}
		}
		input.TargetDomainId = domain.Id

		if domain.Id != ownerId.GetProjectDomainId() && !db.IsAdminAllowCreate(userCred, manager) {
			return input, errors.Wrap(httperrors.ErrNotSufficientPrivilege, "require system priviliges to specify target_domain_id")
		}

		if len(input.OwnerDomainId) > 0 && input.OwnerDomainId != input.TargetDomainId {
			return input, errors.Wrap(httperrors.ErrInputParameter, "inconsistent owner_domain_id and target_domain_id")
		}
	} else if !db.IsAdminAllowCreate(userCred, manager) {
		input.TargetDomainId = ownerId.GetProjectDomainId()
	}

	var err error
	input.Config, err = drvCls.ValidateConfig(ctx, userCred, input.Template, input.Config, "", input.OwnerDomainId)
	if err != nil {
		return input, errors.Wrap(err, "ValidateConfig")
	}

	input.EnabledStatusStandaloneResourceCreateInput, err = manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusStandaloneResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData")
	}

	return input, nil
}

func (ident *SIdentityProvider) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	ident.SetEnabled(true)
	if !db.IsAdminAllowCreate(userCred, ident.GetModelManager()) {
		ident.DomainId = ownerId.GetProjectDomainId()
		ident.TargetDomainId = ownerId.GetProjectDomainId()
	} else {
		ownerDomainId, _ := data.GetString("owner_domain_id")
		if len(ownerDomainId) > 0 {
			ident.DomainId = ownerDomainId
			ident.TargetDomainId = ownerDomainId
		}
	}
	drvCls := ident.getDriverClass()
	if drvCls.IsSso() {
		ident.IsSso = tristate.True
	} else {
		ident.IsSso = tristate.False
	}
	if len(ident.IconUri) == 0 {
		ident.IconUri = drvCls.GetDefaultIconUri(ident.Template)
	}
	if drvCls.ForceSyncUser() {
		ident.AutoCreateUser = tristate.True
	}
	return ident.SEnabledStatusStandaloneResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (ident *SIdentityProvider) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	ident.SEnabledStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	logclient.AddActionLogWithContext(ctx, ident, logclient.ACT_CREATE, data, userCred, true)

	opts := api.TConfigs{}
	err := data.Unmarshal(&opts, "config")
	if err != nil {
		log.Errorf("parse config error %s", err)
		return
	}
	_, err = saveConfigs(userCred, "", ident, opts, nil, nil, api.SensitiveDomainConfigMap)
	if err != nil {
		log.Errorf("saveConfig fail %s", err)
		return
	}

	if len(ident.TargetDomainId) == 0 && ident.AutoCreateUser.IsTrue() && ident.IsSso.IsTrue() && !ident.isAutoCreateDomain() {
		// SSO driver need to create the target domain immediately
		domain, err := ident.SyncOrCreateDomain(ctx, api.DefaultRemoteDomainId, ident.Name, fmt.Sprintf("%s provider %s", ident.Driver, ident.Name), false)
		if err != nil {
			log.Errorf("create domain fail %s", err)
		} else {
			// save domain_id into target_domain_id
			_, err := db.Update(ident, func() error {
				ident.TargetDomainId = domain.Id
				return nil
			})
			if err != nil {
				log.Errorf("save target_domain_id fail: %s", err)
			}
		}
	}

	submitIdpSyncTask(ctx, userCred, ident)
	return
}

func (manager *SIdentityProviderManager) FetchEnabledProviders(driver string) ([]SIdentityProvider, error) {
	q := manager.Query().IsTrue("enabled")
	if len(driver) > 0 {
		q = q.Equals("driver", driver)
	}
	providers := make([]SIdentityProvider, 0)
	err := db.FetchModelObjects(manager, q, &providers)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return providers, nil
}

func (self *SIdentityProvider) CanSync() bool {
	if self.SyncStatus == api.IdentitySyncStatusQueued || self.SyncStatus == api.IdentitySyncStatusSyncing {
		if self.LastSync.IsZero() || time.Now().Sub(self.LastSync) > 1800*time.Second {
			return true
		} else {
			return false
		}
	} else {
		return true
	}
}

func (self *SIdentityProvider) getSyncIntervalSeconds() int {
	if self.SyncIntervalSeconds == 0 {
		return options.Options.DefaultSyncIntervalSeconds
	}
	return self.SyncIntervalSeconds
}

func (self *SIdentityProvider) NeedSync() bool {
	drvCls := driver.GetDriverClass(self.Driver)
	if drvCls == nil {
		return false
	}
	if drvCls.SyncMethod() != api.IdentityProviderSyncFull {
		return false
	}
	if !self.LastSync.IsZero() && time.Now().Sub(self.LastSync) < time.Duration(self.getSyncIntervalSeconds())*time.Second {
		return false
	}

	return true
}

func (self *SIdentityProvider) AllowPerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "sync")
}

// 手动同步认证源
func (self *SIdentityProvider) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.GetEnabled() {
		return nil, httperrors.NewInvalidStatusError("Account disabled")
	}
	if self.CanSync() {
		submitIdpSyncTask(ctx, userCred, self)
	}
	return nil, nil
}

func (self *SIdentityProvider) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.IdentityProviderDetails, error) {
	return api.IdentityProviderDetails{}, nil
}

func (manager *SIdentityProviderManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.IdentityProviderDetails {
	rows := make([]api.IdentityProviderDetails, len(objs))

	stdRows := manager.SEnabledStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	domainRows := manager.SDomainizedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i].EnabledStatusStandaloneResourceDetails = stdRows[i]
		rows[i].DomainizedResourceInfo = domainRows[i]
		rows[i] = objs[i].(*SIdentityProvider).getMoreDetails(rows[i])
	}

	return rows
}

func (self *SIdentityProvider) getMoreDetails(out api.IdentityProviderDetails) api.IdentityProviderDetails {
	out.RoleCount, _ = self.GetRoleCount()
	out.UserCount, _ = self.GetUserCount()
	out.PolicyCount, _ = self.GetPolicyCount()
	out.DomainCount, _ = self.GetDomainCount()
	out.ProjectCount, _ = self.GetProjectCount()
	out.GroupCount, _ = self.GetGroupCount()
	out.SyncIntervalSeconds = self.getSyncIntervalSeconds()
	if len(self.TargetDomainId) > 0 {
		domain, _ := DomainManager.FetchDomainById(self.TargetDomainId)
		if domain != nil {
			out.TargetDomain = domain.Name
		}
	}
	return out
}

func (self *SIdentityProvider) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.IdentityProviderUpdateInput) (api.IdentityProviderUpdateInput, error) {
	if input.SyncIntervalSeconds != nil {
		secs := *input.SyncIntervalSeconds
		if secs < api.MinimalSyncIntervalSeconds {
			secs = api.MinimalSyncIntervalSeconds
			input.SyncIntervalSeconds = &secs
		}
	}
	var err error
	input.EnabledStatusStandaloneResourceBaseUpdateInput, err = self.SEnabledStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.EnabledStatusStandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledStatusStandaloneResourceBase.ValidateUpdateData")
	}

	return input, nil
}

func (self *SIdentityProvider) GetUserCount() (int, error) {
	if self.Driver == api.IdentityDriverSQL {
		return getSQLUserCount()
	} else {
		return self.getLinkedUserCount()
	}
}

func (self *SIdentityProvider) getLinkedUserCount() (int, error) {
	return self.getLinkedUserQuery().CountWithError()
}

func (self *SIdentityProvider) getLinkedUserQuery() *sqlchemy.SQuery {
	return self.getLinkedEntityQuery(UserManager, api.IdMappingEntityUser)
}

func (self *SIdentityProvider) getLinkedEntityQuery(manager db.IStandaloneModelManager, typeStr string) *sqlchemy.SQuery {
	users := manager.Query().SubQuery()
	idmaps := IdmappingManager.Query().SubQuery()

	q := users.Query()
	q = q.LeftJoin(idmaps, sqlchemy.AND(
		sqlchemy.Equals(users.Field("id"), idmaps.Field("public_id")),
		sqlchemy.Equals(idmaps.Field("entity_type"), typeStr),
	))
	q = q.Filter(sqlchemy.Equals(idmaps.Field("domain_id"), self.Id))

	return q
}

func (self *SIdentityProvider) getLinkedUsers() ([]SUser, error) {
	q := self.getLinkedUserQuery()
	users := make([]SUser, 0)
	err := db.FetchModelObjects(UserManager, q, &users)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return users, nil
}

func getSQLEntityQuery(manager db.IStandaloneModelManager, typeStr string) *sqlchemy.SQuery {
	subq := IdmappingManager.Query("public_id")
	subq = subq.Equals("entity_type", typeStr)
	return manager.Query().NotIn("id", subq.SubQuery())
}

func getSQLUserQuery() *sqlchemy.SQuery {
	return getSQLEntityQuery(UserManager, api.IdMappingEntityUser)
}

func getSQLUserCount() (int, error) {
	return getSQLUserQuery().CountWithError()
}

func getSQLGroupQuery() *sqlchemy.SQuery {
	return getSQLEntityQuery(GroupManager, api.IdMappingEntityGroup)
}

func getSQLGroupCount() (int, error) {
	return getSQLGroupQuery().CountWithError()
}

func getSQLDomainQuery() *sqlchemy.SQuery {
	return getSQLEntityQuery(DomainManager, api.IdMappingEntityDomain).NotEquals("id", api.KeystoneDomainRoot)
}

func getSQLDomainCount() (int, error) {
	return getSQLDomainQuery().CountWithError()
}

func (self *SIdentityProvider) GetGroupCount() (int, error) {
	if self.Driver == api.IdentityDriverSQL {
		return getSQLGroupCount()
	} else {
		return self.getLinkedGroupCount()
	}
}

func (self *SIdentityProvider) getLinkedGroupCount() (int, error) {
	return self.getLinkedGroupQuery().CountWithError()
}

func (self *SIdentityProvider) getLinkedGroupQuery() *sqlchemy.SQuery {
	return self.getLinkedEntityQuery(GroupManager, api.IdMappingEntityGroup)
}

func (self *SIdentityProvider) getLinkedGroups() ([]SGroup, error) {
	q := self.getLinkedGroupQuery()
	groups := make([]SGroup, 0)
	err := db.FetchModelObjects(GroupManager, q, &groups)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return groups, nil
}

func (self *SIdentityProvider) GetDomainCount() (int, error) {
	if self.Driver == api.IdentityDriverSQL {
		return getSQLDomainCount()
	} else {
		return self.getLinkedDomainCount()
	}
}

func (self *SIdentityProvider) getDomainQuery() *sqlchemy.SQuery {
	if self.Driver == api.IdentityDriverSQL {
		return getSQLDomainQuery()
	} else {
		return self.getLinkedDomainQuery()
	}
}

func (self *SIdentityProvider) getLinkedDomainCount() (int, error) {
	q := self.getLinkedDomainQuery()
	return q.CountWithError()
}

func (self *SIdentityProvider) getLinkedDomainQuery() *sqlchemy.SQuery {
	q := self.getLinkedEntityQuery(DomainManager, api.IdMappingEntityDomain)
	q = q.NotEquals("id", api.KeystoneDomainRoot)
	return q
}

func (self *SIdentityProvider) GetProjectCount() (int, error) {
	subq := self.getDomainQuery().SubQuery()
	q := ProjectManager.Query().In("domain_id", subq.Query(subq.Field("id")).SubQuery())
	return q.CountWithError()
}

func (self *SIdentityProvider) GetRoleCount() (int, error) {
	subq := self.getDomainQuery().SubQuery()
	q := RoleManager.Query().In("domain_id", subq.Query(subq.Field("id")).SubQuery())
	return q.CountWithError()
}

func (self *SIdentityProvider) GetPolicyCount() (int, error) {
	subq := self.getDomainQuery().SubQuery()
	q := PolicyManager.Query().In("domain_id", subq.Query(subq.Field("id")).SubQuery())
	return q.CountWithError()
}

func (self *SIdentityProvider) ValidateDeleteCondition(ctx context.Context) error {
	if self.Driver == api.IdentityDriverSQL {
		return httperrors.NewForbiddenError("cannot delete default SQL identity provider")
	}
	if self.Enabled.IsTrue() {
		return httperrors.NewInvalidStatusError("cannot delete enabled idp")
	}
	if self.Driver == api.IdentityDriverLDAP || (self.IsSso.IsTrue() && self.isAutoCreateDomain()) || self.AutoCreateUser.IsTrue() {
		prjCnt, err := self.GetProjectCount()
		if err != nil {
			return httperrors.NewGeneralError(err)
		}
		if prjCnt > 0 {
			return httperrors.NewConflictError("identity provider with projects")
		}
		domains, err := self.getLinkedDomains()
		if err != nil {
			return httperrors.NewGeneralError(err)
		}
		for i := range domains {
			if domains[i].Enabled.IsTrue() {
				return httperrors.NewInvalidStatusError("enabled domain %s cannot be deleted", domains[i].Name)
			}
		}
	}
	return self.SEnabledStatusStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SIdentityProvider) ValidateUpdateCondition(ctx context.Context) error {
	if self.SyncStatus != api.IdentitySyncStatusIdle {
		return httperrors.NewConflictError("cannot update in sync status")
	}
	return self.SEnabledStatusStandaloneResourceBase.ValidateUpdateCondition(ctx)
}

func (self *SIdentityProvider) getLinkedDomains() ([]SDomain, error) {
	q := self.getLinkedDomainQuery()
	domains := make([]SDomain, 0)
	err := db.FetchModelObjects(DomainManager, q, &domains)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return domains, nil
}

func (ident *SIdentityProvider) deleteConfigs(ctx context.Context, userCred mcclient.TokenCredential) error {
	_, err := WhitelistedConfigManager.deleteConfigs(ident)
	if err != nil {
		return errors.Wrap(err, "WhitelistedConfigManager.deleteConfig")
	}
	_, err = SensitiveConfigManager.deleteConfigs(ident)
	if err != nil {
		return errors.Wrap(err, "SensitiveConfigManager.deleteConfig")
	}
	return nil
}

func (self *SIdentityProvider) isSsoIdp() bool {
	if self.Driver == api.IdentityDriverLDAP || self.Driver == api.IdentityDriverSQL {
		return false
	}
	return true
}

func (self *SIdentityProvider) Purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	// delete users
	users, err := self.getLinkedUsers()
	if err != nil {
		return errors.Wrap(err, "getNonlocalUsers")
	}
	for i := range users {
		err = users[i].UnlinkIdp(self.Id)
		if err != nil {
			return errors.Wrap(err, "users[i].UnlinkIdp")
		}
		if self.isSsoIdp() && self.AutoCreateUser.IsFalse() {
			continue
		}
		err = users[i].ValidatePurgeCondition(ctx)
		if err != nil {
			db.OpsLog.LogEvent(&users[i], db.ACT_DELETE_FAIL, err, userCred)
			log.Errorf("users %s ValidatePurgeCondition fail %s", users[i].Name, err)
			continue
		}
		err = users[i].Delete(ctx, userCred)
		if err != nil {
			db.OpsLog.LogEvent(&users[i], db.ACT_DELETE_FAIL, err, userCred)
			return errors.Wrap(err, "delete users[i]")
		}
	}
	// delete groups
	groups, err := self.getLinkedGroups()
	if err != nil {
		return errors.Wrap(err, "getNonlocalGroups")
	}
	for i := range groups {
		err = groups[i].UnlinkIdp(self.Id)
		if err != nil {
			return errors.Wrap(err, "groups[i].UnlinkIdp")
		}
		if self.isSsoIdp() && self.AutoCreateUser.IsFalse() {
			continue
		}
		err = groups[i].ValidateDeleteCondition(ctx)
		if err != nil {
			db.OpsLog.LogEvent(&groups[i], db.ACT_DELETE_FAIL, err, userCred)
			log.Errorf("group %s ValidateDeleteCondition fail %s", groups[i].Name, err)
			continue
		}
		err = groups[i].Delete(ctx, userCred)
		if err != nil {
			db.OpsLog.LogEvent(&groups[i], db.ACT_DELETE_FAIL, err, userCred)
			return errors.Wrap(err, "delete groups[i]")
		}
	}
	// delete domains
	domains, err := self.getLinkedDomains()
	if err != nil {
		return errors.Wrap(err, "getDomains")
	}
	for i := range domains {
		if self.isSsoIdp() && self.AutoCreateUser.IsFalse() {
			err = domains[i].UnlinkIdp(self.Id)
			if err != nil {
				return errors.Wrap(err, "domains[i].UnlinkIdp")
			}
		} else {
			err = domains[i].ValidatePurgeCondition(ctx)
			if err != nil {
				db.OpsLog.LogEvent(&domains[i], db.ACT_DELETE_FAIL, err, userCred)
				return errors.Wrap(err, "domain.ValidatePurgeCondition")
			}
			err = domains[i].UnlinkIdp(self.Id)
			if err != nil {
				return errors.Wrap(err, "domains[i].UnlinkIdp")
			}
			err = domains[i].Delete(ctx, userCred)
			if err != nil {
				db.OpsLog.LogEvent(&domains[i], db.ACT_DELETE_FAIL, err, userCred)
				return errors.Wrap(err, "delete domain")
			}
		}
	}
	err = self.deleteConfigs(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "self.deleteConfig")
	}
	err = IdmappingManager.deleteByIdpId(self.Id)
	if err != nil {
		return errors.Wrap(err, "self.deleteIdmappings")
	}
	return self.RealDelete(ctx, userCred)
}

func (self *SIdentityProvider) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("SIdentityProvider delete do nothing")
	return nil
}

func (self *SIdentityProvider) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SEnabledStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SIdentityProvider) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.startDeleteIdentityProviderTask(ctx, userCred, "")
}

func (self *SIdentityProvider) startDeleteIdentityProviderTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	self.SetStatus(userCred, api.IdentityDriverStatusDeleting, "")

	task, err := taskman.TaskManager.NewTask(ctx, "IdentityProviderDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SIdentityProvider) GetSingleDomain(ctx context.Context, extId string, extName string, extDesc string, createDefaultProject bool) (*SDomain, error) {
	if len(self.TargetDomainId) > 0 {
		targetDomain, err := DomainManager.FetchDomainById(self.TargetDomainId)
		if err != nil && err != sql.ErrNoRows {
			return nil, errors.Wrap(err, "DomainManager.FetchDomainById")
		}
		if targetDomain == nil {
			log.Warningln("target domain not exist!")
		} else {
			return targetDomain, nil
		}
	}
	return self.SyncOrCreateDomain(ctx, extId, extName, extDesc, createDefaultProject)
}

func (self *SIdentityProvider) SyncOrCreateDomain(ctx context.Context, extId string, extName string, extDesc string, createDefaultProject bool) (*SDomain, error) {
	log.Debugf("SyncOrCreateDomain extId: %s extName: %s", extId, extName)

	domainId, err := IdmappingManager.RegisterIdMap(ctx, self.Id, extId, api.IdMappingEntityDomain)
	if err != nil {
		return nil, errors.Wrap(err, "IdmappingManager.RegisterIdMap")
	}
	domain, err := DomainManager.FetchDomainById(domainId)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "DomainManager.FetchDomainById")
	}
	if err == nil {
		// find the domain
		if domain.Name != extName {
			// sync domain name
			newName, err := db.GenerateName2(ctx, DomainManager, nil, extName, domain, 1)
			if err != nil {
				log.Errorf("sync existing domain name (%s=%s) generate fail %s", domain.Name, extName, err)
			} else {
				_, err = db.Update(domain, func() error {
					domain.Name = newName
					return nil
				})
				if err != nil {
					log.Errorf("sync existing domain name (%s=%s) update fail %s", domain.Name, extName, err)
				}
			}
		}
		return domain, nil
	}

	// otherwise, create the domain
	domain = &SDomain{}
	domain.SetModelManager(DomainManager, domain)
	domain.Id = domainId
	domain.Enabled = tristate.True
	domain.IsDomain = tristate.True
	domain.DomainId = api.KeystoneDomainRoot
	domain.Description = fmt.Sprintf("domain for %s", extDesc)

	err = func() error {
		lockman.LockClass(ctx, DomainManager, "name")
		defer lockman.ReleaseClass(ctx, DomainManager, "name")

		newName, err := db.GenerateName(ctx, DomainManager, nil, extName)
		if err != nil {
			return errors.Wrap(err, "GenerateName")
		}
		domain.Name = newName

		return DomainManager.TableSpec().Insert(ctx, domain)
	}()
	if err != nil {
		return nil, errors.Wrap(err, "insert")
	}

	if self.AutoCreateProject.IsTrue() && consts.GetNonDefaultDomainProjects() && createDefaultProject {
		_, err := ProjectManager.NewProject(ctx,
			fmt.Sprintf("%s_default_project", extName),
			fmt.Sprintf("Default project for domain %s", extName),
			domain.Id,
		)
		if err != nil {
			log.Errorf("ProjectManager.NewProject fail %s", err)
		}
	}

	return domain, nil
}

func (self *SIdentityProvider) SyncOrCreateUser(ctx context.Context, extId string, extName string, domainId string, enableDefault bool, syncUserInfo func(*SUser)) (*SUser, error) {
	log.Debugf("SyncOrCreateUser extId: %s extName: %s", extId, extName)
	userId, err := IdmappingManager.RegisterIdMap(ctx, self.Id, extId, api.IdMappingEntityUser)
	if err != nil {
		return nil, errors.Wrap(err, "IdmappingManager.RegisterIdMap")
	}

	lockman.LockRawObject(ctx, UserManager.Keyword(), userId)
	defer lockman.ReleaseRawObject(ctx, UserManager.Keyword(), userId)

	userObj, err := db.NewModelObject(UserManager)
	if err != nil {
		return nil, errors.Wrap(err, "db.NewModelObject")
	}
	user := userObj.(*SUser)
	q := UserManager.RawQuery().Equals("id", userId)
	err = q.First(user)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "Query user")
	}
	if err == nil {
		// update
		log.Debugf("find user %s", extName)
		newName, err := db.GenerateAlterName(user, extName)
		if err != nil {
			return nil, errors.Wrapf(err, "db.GenerateAlterName %s", extName)
		}
		_, err = db.Update(user, func() error {
			if syncUserInfo != nil {
				syncUserInfo(user)
			}
			user.Name = newName
			user.DomainId = domainId
			if user.Deleted {
				user.MarkUnDelete()
				if enableDefault {
					user.Enabled = tristate.True
				} else {
					user.Enabled = tristate.False
				}
			}
			return nil
		})
		if err != nil {
			return nil, errors.Wrap(err, "Update")
		}
	} else {
		if syncUserInfo != nil {
			syncUserInfo(user)
		}
		if enableDefault {
			user.Enabled = tristate.True
		} else {
			user.Enabled = tristate.False
		}
		domainOwnerId := &db.SOwnerId{DomainId: domainId}

		user.Id = userId
		user.DomainId = domainId
		err = func() error {
			lockman.LockRawObject(ctx, UserManager.Keyword(), "name")
			defer lockman.ReleaseRawObject(ctx, UserManager.Keyword(), "name")

			user.Name, err = db.GenerateName(ctx, UserManager, domainOwnerId, extName)
			if err != nil {
				return errors.Wrapf(err, "db.GenerateName %s", extName)
			}

			return UserManager.TableSpec().Insert(ctx, user)
		}()
		if err != nil {
			return nil, errors.Wrap(err, "Insert")
		}
	}
	return user, nil
}

func (manager *SIdentityProviderManager) FetchIdentityProviderById(idstr string) (*SIdentityProvider, error) {
	obj, err := manager.FetchById(idstr)
	if err != nil {
		return nil, errors.Wrap(err, "manager.FetchById")
	}
	return obj.(*SIdentityProvider), nil
}

func (manager *SIdentityProviderManager) FetchPasswordProtectedIdpIdsQuery() *sqlchemy.SSubQuery {
	q := manager.Query("id").In("driver", api.PASSWORD_PROTECTED_IDPS)
	return q.SubQuery()
}

func (manager *SIdentityProviderManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.IdentityProviderListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SDomainizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.DomainizedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDomainizedResourceBaseManager.ListItemFilter")
	}
	if len(query.Driver) > 0 {
		q = q.In("driver", query.Driver)
	}
	if len(query.Template) > 0 {
		q = q.In("template", query.Template)
	}
	if len(query.SyncStatus) > 0 {
		q = q.In("sync_status", query.SyncStatus)
	}
	if len(query.SsoDomain) > 0 {
		q = q.IsTrue("is_sso")
		if strings.EqualFold(query.SsoDomain, "all") {
			q = q.IsNullOrEmpty("domain_id")
		} else if len(query.SsoDomain) > 0 {
			ssoDomain, err := DomainManager.FetchDomainByIdOrName(query.SsoDomain)
			if err != nil {
				if errors.Cause(err) == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError2(DomainManager.Keyword(), query.SsoDomain)
				} else {
					return nil, errors.Wrap(err, "FetchDomainByIdOrName")
				}
			}
			q = q.Equals("domain_id", ssoDomain.Id)
		}
	}
	if query.AutoCreateProject != nil {
		if *query.AutoCreateProject {
			q = q.IsTrue("auto_create_project")
		} else {
			q = q.IsFalse("auto_create_project")
		}
	}
	if query.AutoCreateUser != nil {
		if *query.AutoCreateUser {
			q = q.IsTrue("auto_create_user")
		} else {
			q = q.IsFalse("auto_create_user")
		}
	}
	return q, nil
}

func (manager *SIdentityProviderManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.IdentityProviderListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SDomainizedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.DomainizedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDomainizedResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SIdentityProviderManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SDomainizedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func fetchAttribute(attrs map[string][]string, key string) string {
	if v, ok := attrs[key]; ok && len(v) > 0 {
		return v[0]
	}
	return ""
}

func fetchAttributes(attrs map[string][]string, key string) []string {
	if v, ok := attrs[key]; ok && len(v) > 0 {
		return v
	}
	return nil
}

func (idp *SIdentityProvider) TryUserJoinProject(attrConf api.SIdpAttributeOptions, ctx context.Context, usr *SUser, domainId string, attrs map[string][]string) {
	if idp.AutoCreateUser.IsFalse() {
		return
	}
	// update user attributes
	_, err := db.Update(usr, func() error {
		if v, ok := attrs[attrConf.UserDisplaynameAttribtue]; ok && len(v) > 0 && len(v[0]) > 0 && usr.Displayname != v[0] {
			usr.Displayname = v[0]
		}
		if v, ok := attrs[attrConf.UserEmailAttribute]; ok && len(v) > 0 && len(v[0]) > 0 && usr.Email != v[0] {
			usr.Email = v[0]
		}
		if v, ok := attrs[attrConf.UserMobileAttribute]; ok && len(v) > 0 && len(v[0]) > 0 && usr.Mobile != v[0] {
			usr.Mobile = v[0]
		}
		return nil
	})
	if err != nil {
		log.Errorf("update user attributes fail %s", err)
	}

	var targetProject *SProject
	log.Debugf("userTryJoinProject resp %s proj %s", attrs, attrConf.ProjectAttribute)
	if !consts.GetNonDefaultDomainProjects() {
		// if non-default-domain-project is disabled, place new project in default domain
		domainId = api.DEFAULT_DOMAIN_ID
	}
	if len(attrConf.ProjectAttribute) > 0 {
		projName := fetchAttribute(attrs, attrConf.ProjectAttribute)
		if len(projName) > 0 {
			projDomainId := ""
			if ProjectManager.NamespaceScope() == rbacutils.ScopeDomain {
				projDomainId = domainId
			}
			targetProject, err = ProjectManager.FetchProject("", projName, projDomainId, "")
			if err != nil {
				log.Errorf("fetch project %s fail %s", projName, err)
				if errors.Cause(err) == sql.ErrNoRows && idp.AutoCreateProject.IsTrue() {
					targetProject, err = ProjectManager.NewProject(ctx, projName, fmt.Sprintf("auto create project for idp %s", idp.Name), domainId)
					if err != nil {
						log.Errorf("auto create project %s fail %s", projName, err)
					}
				}
			}
		}
	}
	if targetProject == nil && len(attrConf.DefaultProjectId) > 0 {
		targetProject, err = ProjectManager.FetchProjectById(attrConf.DefaultProjectId)
		if err != nil {
			log.Errorf("fetch default project %s fail %s", attrConf.DefaultProjectId, err)
		}
	}
	if targetProject != nil {
		// put user in project
		targetRoles := make([]*SRole, 0)
		if len(attrConf.RolesAttribute) > 0 {
			roleNames := fetchAttributes(attrs, attrConf.RolesAttribute)
			for _, roleName := range roleNames {
				if len(roleName) > 0 {
					targetRole, err := RoleManager.FetchRole("", roleName, domainId, "")
					if err != nil {
						log.Errorf("fetch role %s fail %s", roleName, err)
					} else {
						targetRoles = append(targetRoles, targetRole)
					}
				}
			}
		}
		if len(targetRoles) == 0 && len(attrConf.DefaultRoleId) > 0 {
			targetRole, err := RoleManager.FetchRoleById(attrConf.DefaultRoleId)
			if err != nil {
				log.Errorf("fetch default role %s fail %s", attrConf.DefaultRoleId, err)
			} else {
				targetRoles = append(targetRoles, targetRole)
			}
		}
		for _, targetRole := range targetRoles {
			err = AssignmentManager.ProjectAddUser(ctx, GetDefaultAdminCred(), targetProject, usr, targetRole)
			if err != nil {
				log.Errorf("CAS user %s join project %s with role %s fail %s", usr.Name, targetProject.Name, targetRole.Name, err)
			}
		}
	}
}

func (idp *SIdentityProvider) AllowGetDetailsSamlMetadata(ctx context.Context, userCred mcclient.TokenCredential, query api.GetIdpSamlMetadataInput) bool {
	return db.IsDomainAllowGetSpec(userCred, idp, "saml-metadata")
}

func (idp *SIdentityProvider) GetDetailsSamlMetadata(ctx context.Context, userCred mcclient.TokenCredential, query api.GetIdpSamlMetadataInput) (api.GetIdpSamlMetadataOutput, error) {
	output := api.GetIdpSamlMetadataOutput{}
	if !saml.IsSAMLEnabled() {
		return output, errors.Wrap(httperrors.ErrNotSupported, "enable SSL first")
	}
	if idp.Driver != api.IdentityDriverSAML {
		return output, errors.Wrap(httperrors.ErrNotSupported, "not a saml IDP")
	}
	if len(query.RedirectUri) == 0 {
		return output, errors.Wrap(httperrors.ErrInputParameter, "missing redirect_uri")
	}

	spInst := sp.NewSpInstance(saml.SAMLInstance(), idp.Name, nil, nil)
	spInst.SetAssertionConsumerUri(query.RedirectUri)
	ed := spInst.GetMetadata()
	var xmlBytes []byte
	if query.Pretty != nil && *query.Pretty {
		xmlBytes, _ = xml.MarshalIndent(ed, "", "  ")
	} else {
		xmlBytes, _ = xml.Marshal(ed)
	}
	output.Metadata = string(xmlBytes)
	return output, nil
}

func (idp *SIdentityProvider) AllowGetDetailsSsoRedirectUri(ctx context.Context, userCred mcclient.TokenCredential, query api.GetIdpSsoRedirectUriInput) bool {
	return db.IsDomainAllowGetSpec(userCred, idp, "sso-redirect-uri")
}

func (idp *SIdentityProvider) GetDetailsSsoRedirectUri(ctx context.Context, userCred mcclient.TokenCredential, query api.GetIdpSsoRedirectUriInput) (api.GetIdpSsoRedirectUriOutput, error) {
	output := api.GetIdpSsoRedirectUriOutput{}
	conf, err := GetConfigs(idp, true, nil, nil)
	if err != nil {
		return output, errors.Wrap(err, "idp.GetConfig")
	}

	backend, err := driver.GetDriver(idp.Driver, idp.Id, idp.Name, idp.Template, idp.TargetDomainId, conf)
	if err != nil {
		return output, errors.Wrap(err, "driver.GetDriver")
	}

	uri, err := backend.GetSsoRedirectUri(ctx, query.RedirectUri, query.State)
	if err != nil {
		return output, errors.Wrap(err, "backend.GetSsoRedirectUri")
	}

	output.Uri = uri
	output.Driver = idp.Driver

	return output, nil
}

func (idp *SIdentityProvider) SyncOrCreateDomainAndUser(ctx context.Context, extDomainId, extDomainName string, extUsrId, extUsrName string) (*SDomain, *SUser, error) {
	var (
		domain *SDomain
		usr    *SUser
		err    error
	)
	if len(extUsrId) == 0 && len(extUsrName) == 0 {
		return nil, nil, errors.Wrap(httperrors.ErrUnauthenticated, "empty userId or userName")
	}
	if len(extUsrId) == 0 {
		extUsrId = extUsrName
	} else if len(extUsrName) == 0 {
		extUsrName = extUsrId
	}

	var domainDesc string
	if len(extDomainId) == 0 && len(extDomainName) == 0 {
		extDomainId = api.DefaultRemoteDomainId
		extDomainName = idp.Name
		domainDesc = fmt.Sprintf("%s provider %s", idp.Driver, idp.Name)
	} else if len(extDomainId) == 0 {
		extDomainId = extDomainName
		domainDesc = fmt.Sprintf("%s provider %s autocreated for %s", idp.Driver, idp.Name, extDomainName)
	} else if len(extDomainName) == 0 {
		extDomainName = extDomainId
		domainDesc = fmt.Sprintf("%s provider %s autocreated for %s", idp.Driver, idp.Name, extDomainId)
	} else {
		domainDesc = fmt.Sprintf("%s provider %s autocreated for %s(%s)", idp.Driver, idp.Name, extDomainName, extDomainId)
	}

	if idp.AutoCreateUser.IsTrue() {
		domain, err = idp.GetSingleDomain(ctx, extDomainId, extDomainName, domainDesc, false)
		if err != nil {
			return nil, nil, errors.Wrap(err, "idp.GetSingleDomain")
		}
		usr, err = idp.SyncOrCreateUser(ctx, extUsrId, extUsrName, domain.Id, true, nil)
		if err != nil {
			return nil, nil, errors.Wrap(err, "idp.SyncOrCreateUser")
		}
	} else {
		modelUsrId, err := IdmappingManager.FetchByIdpAndEntityId(ctx, idp.Id, extUsrId, api.IdMappingEntityUser)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, nil, errors.Wrap(httperrors.ErrUserNotFound, extUsrId)
			} else {
				return nil, nil, errors.Wrap(err, "IdmappingManager.FetchByIdpAndEntityId")
			}
		}
		usrObj, err := UserManager.FetchById(modelUsrId)
		if err != nil {
			return nil, nil, errors.Wrap(err, "UserManager.FetchById")
		}
		usr = usrObj.(*SUser)
		domain = usr.GetDomain()
	}
	return domain, usr, nil
}

func (manager *SIdentityProviderManager) FetchIdentityProvidersByUserId(uid string, drivers []string) ([]SIdentityProvider, error) {
	idps := make([]SIdentityProvider, 0)
	idmappings := IdmappingManager.Query().SubQuery()
	q := manager.Query()
	q = q.Join(idmappings, sqlchemy.Equals(q.Field("id"), idmappings.Field("domain_id")))
	q = q.Filter(sqlchemy.Equals(idmappings.Field("entity_type"), api.IdMappingEntityUser))
	q = q.Filter(sqlchemy.Equals(idmappings.Field("public_id"), uid))
	if len(drivers) > 0 {
		q = q.Filter(sqlchemy.In(q.Field("driver"), drivers))
	}
	err := db.FetchModelObjects(manager, q, &idps)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return idps, nil
}

func (manager *SIdentityProviderManager) CheckUniqueness(extIdpId string, domainId string, driver string, template string, group string, option string, value jsonutils.JSONObject) (bool, error) {
	configs := WhitelistedConfigManager.Query().SubQuery()
	q := manager.Query()
	if len(group) > 0 {
		q = q.Join(configs, sqlchemy.AND(
			sqlchemy.Equals(configs.Field("res_type"), manager.Keyword()),
			sqlchemy.Equals(configs.Field("domain_id"), q.Field("id")),
		))
	}
	if len(domainId) == 0 {
		q = q.IsNullOrEmpty("domain_id")
	} else {
		q = q.Equals("domain_id", domainId)
	}
	q = q.Equals("driver", driver)
	if len(template) > 0 {
		q = q.Equals("template", template)
	}
	if len(group) > 0 {
		q = q.Filter(sqlchemy.Equals(configs.Field("group"), group))
		if len(option) > 0 {
			q = q.Filter(sqlchemy.Equals(configs.Field("option"), option))
		}
		if value != nil {
			q = q.Filter(sqlchemy.Equals(configs.Field("value"), value.String()))
		}
	}
	if len(extIdpId) > 0 {
		q = q.Filter(sqlchemy.NotEquals(q.Field("id"), extIdpId))
	}
	cnt, err := q.CountWithError()
	if err != nil {
		return false, errors.Wrap(err, "CountWithError")
	}
	return cnt == 0, nil
}

func (idp *SIdentityProvider) PerformDisable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.PerformDisableInput,
) (jsonutils.JSONObject, error) {
	if idp.Driver == api.IdentityDriverSQL {
		return nil, errors.Wrap(httperrors.ErrForbidden, "not allow to disable sql idp")
	}
	if idp.Driver == api.IdentityDriverLDAP || idp.AutoCreateUser.IsTrue() {
		domains, _ := idp.getLinkedDomains()
		for i := range domains {
			db.Update(&domains[i], func() error {
				domains[i].Enabled = tristate.False
				return nil
			})
		}
	}
	return idp.SEnabledStatusStandaloneResourceBase.PerformDisable(ctx, userCred, query, input)
}

func (idp *SIdentityProvider) PerformEnable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.PerformEnableInput,
) (jsonutils.JSONObject, error) {
	if idp.Driver == api.IdentityDriverLDAP || idp.AutoCreateUser.IsTrue() {
		domains, _ := idp.getLinkedDomains()
		for i := range domains {
			db.Update(&domains[i], func() error {
				domains[i].Enabled = tristate.True
				return nil
			})
		}
	}
	return idp.SEnabledStatusStandaloneResourceBase.PerformEnable(ctx, userCred, query, input)
}

func (idp *SIdentityProvider) isAutoCreateDomain() bool {
	configs, err := GetConfigs(idp, false, nil, nil)
	if err != nil {
		log.Errorf("GetConfigs fail %s", err)
		return false
	}
	if vjson, ok := configs[idp.Driver]["domain_id_attribute"]; ok {
		v, _ := vjson.GetString()
		if len(v) > 0 {
			return true
		}
	}
	if vjson, ok := configs[idp.Driver]["domain_id_attribute"]; ok {
		v, _ := vjson.GetString()
		if len(v) > 0 {
			return true
		}
	}
	return false
}

func (idp *SIdentityProvider) PerformDefaultSso(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.PerformDefaultSsoInput,
) (jsonutils.JSONObject, error) {
	if !idp.IsSso.IsTrue() {
		return nil, errors.Wrap(httperrors.ErrNotSupported, "idp is not a sso idp")
	}

	if input.Enable != nil {
		if *input.Enable {
			// enable
			// first disable any other idp in the same domain
			q := IdentityProviderManager.Query().IsTrue("is_sso").IsTrue("is_default").NotEquals("id", idp.Id)
			if len(idp.DomainId) > 0 {
				// a domain specific IDP
				q = q.Equals("domain_id", idp.DomainId)
			} else {
				// a system IDP
				q = q.IsNullOrEmpty("domain_id")
			}
			idps := make([]SIdentityProvider, 0)
			err := db.FetchModelObjects(IdentityProviderManager, q, &idps)
			if err != nil && errors.Cause(err) != sql.ErrNoRows {
				return nil, errors.Wrap(err, "FetchModelObjects")
			}
			for i := range idps {
				err := idps[i].setIsDefault(tristate.False)
				if err != nil {
					return nil, errors.Wrap(err, "disable other idp fail")
				}
			}
			if !idp.IsDefault.IsTrue() {
				err := idp.setIsDefault(tristate.True)
				if err != nil {
					return nil, errors.Wrap(err, "update is_default fail")
				}
			}
		} else {
			// disable
			if idp.IsDefault.IsTrue() {
				err := idp.setIsDefault(tristate.False)
				if err != nil {
					return nil, errors.Wrap(err, "update is_default fail")
				}
			}
		}
	}

	return nil, nil
}

func (idp *SIdentityProvider) setIsDefault(val tristate.TriState) error {
	_, err := db.Update(idp, func() error {
		idp.IsDefault = val
		return nil
	})
	return errors.Wrap(err, "update")
}
