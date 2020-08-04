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
}

func (manager *SIdentityProviderManager) initializeAutoCreateUser() error {
	q := manager.Query().IsNull("auto_create_user")
	idps := make([]SIdentityProvider, 0)
	err := db.FetchModelObjects(manager, q, &idps)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil
		} else {
			return errors.Wrap(err, "FetchModelObjeccts")
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
			return errors.Wrap(err, "FetchModelObjeccts")
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
	input.Config, err = ident.getDriverClass().ValidateConfig(ctx, userCred, ident.Template, input.Config)
	if err != nil {
		return nil, errors.Wrap(err, "ValidateConfig")
	}

	opts := input.Config
	action := input.Action
	err = saveConfigs(userCred, action, ident, opts, nil, nil, api.SensitiveDomainConfigMap)
	if err != nil {
		return nil, httperrors.NewInternalServerError("saveConfig fail %s", err)
	}
	ident.MarkDisconnected(ctx, userCred, fmt.Errorf("change config"))
	submitIdpSyncTask(ctx, userCred, ident)
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
	}

	var err error
	input.Config, err = drvCls.ValidateConfig(ctx, userCred, input.Template, input.Config)
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
	err = saveConfigs(userCred, "", ident, opts, nil, nil, api.SensitiveDomainConfigMap)
	if err != nil {
		log.Errorf("saveConfig fail %s", err)
		return
	}

	if len(ident.TargetDomainId) == 0 && ident.AutoCreateUser.IsTrue() && ident.IsSso.IsTrue() {
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
		return self.getLocalUserCount()
	} else {
		return self.getNonlocalUserCount()
	}
}

func (self *SIdentityProvider) getNonlocalUserCount() (int, error) {
	return self.getNonlocalUserQuery().CountWithError()
}

func (self *SIdentityProvider) getNonlocalUserQuery() *sqlchemy.SQuery {
	users := UserManager.Query().SubQuery()
	idmaps := IdmappingManager.Query().SubQuery()

	q := users.Query()
	q = q.LeftJoin(idmaps, sqlchemy.AND(
		sqlchemy.Equals(users.Field("id"), idmaps.Field("public_id")),
		sqlchemy.Equals(idmaps.Field("entity_type"), api.IdMappingEntityUser),
	))
	q = q.Filter(sqlchemy.Equals(idmaps.Field("domain_id"), self.Id))

	return q
}

func (self *SIdentityProvider) getNonlocalUsers() ([]SUser, error) {
	q := self.getNonlocalUserQuery()
	users := make([]SUser, 0)
	err := db.FetchModelObjects(UserManager, q, &users)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return users, nil
}

func (self *SIdentityProvider) getLocalUserCount() (int, error) {
	subq := IdmappingManager.Query("public_id").Equals("entity_type", api.IdMappingEntityUser)
	q := UserManager.Query().NotIn("id", subq.SubQuery())
	return q.CountWithError()
}

func (self *SIdentityProvider) GetGroupCount() (int, error) {
	if self.Driver == api.IdentityDriverSQL {
		return self.getLocalGroupCount()
	} else {
		return self.getNonlocalGroupCount()
	}
}

func (self *SIdentityProvider) getNonlocalGroupCount() (int, error) {
	return self.getNonlocalGroupQuery().CountWithError()
}

func (self *SIdentityProvider) getNonlocalGroupQuery() *sqlchemy.SQuery {
	groups := GroupManager.Query().SubQuery()
	idmaps := IdmappingManager.Query().SubQuery()

	q := groups.Query()
	q = q.LeftJoin(idmaps, sqlchemy.AND(
		sqlchemy.Equals(groups.Field("id"), idmaps.Field("public_id")),
		sqlchemy.Equals(idmaps.Field("entity_type"), api.IdMappingEntityGroup),
	))
	q = q.Filter(sqlchemy.Equals(idmaps.Field("domain_id"), self.Id))

	return q
}

func (self *SIdentityProvider) getNonlocalGroups() ([]SGroup, error) {
	q := self.getNonlocalGroupQuery()
	groups := make([]SGroup, 0)
	err := db.FetchModelObjects(GroupManager, q, &groups)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return groups, nil
}

func (self *SIdentityProvider) getLocalGroupCount() (int, error) {
	subq := IdmappingManager.Query("public_id").Equals("entity_type", api.IdMappingEntityGroup)
	q := GroupManager.Query().NotIn("id", subq.SubQuery())
	return q.CountWithError()
}

func (self *SIdentityProvider) GetDomainCount() (int, error) {
	if self.Driver == api.IdentityDriverSQL {
		return self.getLocalDomainCount()
	} else {
		return self.getNonlocalDomainCount()
	}
}

func (self *SIdentityProvider) getDomainQuery() *sqlchemy.SQuery {
	if self.Driver == api.IdentityDriverSQL {
		return self.getLocalDomainQuery()
	} else {
		return self.getNonlocalDomainQuery()
	}
}

func (self *SIdentityProvider) getNonlocalDomainCount() (int, error) {
	q := self.getNonlocalDomainQuery()
	return q.CountWithError()
}

func (self *SIdentityProvider) getNonlocalDomainQuery() *sqlchemy.SQuery {
	domains := DomainManager.Query().SubQuery()
	idmaps := IdmappingManager.Query().SubQuery()

	q := domains.Query()
	q = q.Join(idmaps, sqlchemy.AND(
		sqlchemy.Equals(domains.Field("id"), idmaps.Field("public_id")),
		sqlchemy.Equals(idmaps.Field("entity_type"), api.IdMappingEntityDomain),
	))
	q = q.Filter(sqlchemy.NotEquals(domains.Field("id"), api.KeystoneDomainRoot))
	q = q.Filter(sqlchemy.Equals(idmaps.Field("domain_id"), self.Id))

	return q
}

func (self *SIdentityProvider) getLocalDomainCount() (int, error) {
	q := self.getLocalDomainQuery()
	return q.CountWithError()
}

func (self *SIdentityProvider) getLocalDomainQuery() *sqlchemy.SQuery {
	subq := IdmappingManager.Query("public_id").Equals("entity_type", api.IdMappingEntityDomain)
	q := DomainManager.Query().NotIn("id", subq.SubQuery()).NotEquals("id", api.KeystoneDomainRoot)
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
	prjCnt, err := self.GetProjectCount()
	if err != nil {
		return httperrors.NewGeneralError(err)
	}
	if prjCnt > 0 {
		return httperrors.NewConflictError("identity provider with projects")
	}
	domains, err := self.getDomains()
	if err != nil {
		return httperrors.NewGeneralError(err)
	}
	for i := range domains {
		if domains[i].Enabled.IsTrue() {
			return httperrors.NewInvalidStatusError("domain %s should be disabled", domains[i].Name)
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

func (self *SIdentityProvider) getDomains() ([]SDomain, error) {
	q := self.getDomainQuery()
	domains := make([]SDomain, 0)
	err := db.FetchModelObjects(DomainManager, q, &domains)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return domains, nil
}

func (ident *SIdentityProvider) deleteConfigs(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := WhitelistedConfigManager.deleteConfigs(ident)
	if err != nil {
		return errors.Wrap(err, "WhitelistedConfigManager.deleteConfig")
	}
	err = SensitiveConfigManager.deleteConfigs(ident)
	if err != nil {
		return errors.Wrap(err, "SensitiveConfigManager.deleteConfig")
	}
	return nil
}

func (self *SIdentityProvider) Purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	// delete users
	users, err := self.getNonlocalUsers()
	if err != nil {
		return errors.Wrap(err, "getNonlocalUsers")
	}
	for i := range users {
		err = users[i].UnlinkIdp(self.Id)
		if err != nil {
			return errors.Wrap(err, "users[i].UnlinkIdp")
		}
		err = users[i].ValidateDeleteCondition(ctx)
		if err != nil {
			return errors.Wrap(err, "users[i].ValidateDeleteCondition")
		}
		err = users[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "delete users[i]")
		}
	}
	// delete groups
	groups, err := self.getNonlocalGroups()
	if err != nil {
		return errors.Wrap(err, "getNonlocalGroups")
	}
	for i := range groups {
		err = groups[i].UnlinkIdp(self.Id)
		if err != nil {
			return errors.Wrap(err, "groups[i].UnlinkIdp")
		}
		err = groups[i].ValidateDeleteCondition(ctx)
		if err != nil {
			return errors.Wrap(err, "groups[i].ValidateDeleteCondition")
		}
		err = groups[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "delete groups[i]")
		}
	}
	// delete domains
	domains, err := self.getDomains()
	if err != nil {
		return errors.Wrap(err, "getDomains")
	}
	for i := range domains {
		err = domains[i].ValidatePurgeCondition(ctx)
		if err != nil {
			return errors.Wrap(err, "domain.ValidatePurgeCondition")
		}
		err = domains[i].UnlinkIdp(self.Id)
		if err != nil {
			return errors.Wrap(err, "domains[i].UnlinkIdp")
		}
		err = domains[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "delete domain")
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
	domainId, err := IdmappingManager.RegisterIdMap(ctx, self.Id, extId, api.IdMappingEntityDomain)
	if err != nil {
		return nil, errors.Wrap(err, "IdmappingManager.RegisterIdMap")
	}
	domain, err := DomainManager.FetchDomainById(domainId)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "DomainManager.FetchDomainById")
	}
	if err == nil {
		if domain.Name != extName {
			// sync domain name
			newName, err := db.GenerateName2(DomainManager, nil, extName, domain, 1)
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

	lockman.LockClass(ctx, DomainManager, "")
	lockman.ReleaseClass(ctx, DomainManager, "")

	domain = &SDomain{}
	domain.SetModelManager(DomainManager, domain)
	domain.Id = domainId
	newName, err := db.GenerateName(DomainManager, nil, extName)
	if err != nil {
		return nil, errors.Wrap(err, "GenerateName")
	}
	domain.Name = newName
	domain.Enabled = tristate.True
	domain.IsDomain = tristate.True
	domain.DomainId = api.KeystoneDomainRoot
	domain.Description = fmt.Sprintf("domain for %s", extDesc)
	err = DomainManager.TableSpec().Insert(ctx, domain)
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
		_, err := db.Update(user, func() error {
			if syncUserInfo != nil {
				syncUserInfo(user)
			}
			user.Name = extName
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
		user.Id = userId
		user.Name = extName
		user.DomainId = domainId
		err = UserManager.TableSpec().Insert(ctx, user)
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
			q = q.Filter(sqlchemy.OR(
				sqlchemy.IsNullOrEmpty(q.Field("domain_id")),
				sqlchemy.Equals(q.Field("domain_id"), query.SsoDomain),
			))
			q = q.Filter(sqlchemy.OR(
				sqlchemy.IsNullOrEmpty(q.Field("target_domain_id")),
				sqlchemy.Equals(q.Field("target_domain_id"), query.SsoDomain),
			))
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
	// update user attributes
	_, err := db.Update(usr, func() error {
		if v, ok := attrs[attrConf.UserDisplaynameAttribtue]; ok && len(v) > 0 {
			usr.Displayname = v[0]
		}
		if v, ok := attrs[attrConf.UserEmailAttribute]; ok && len(v) > 0 {
			usr.Email = v[0]
		}
		if v, ok := attrs[attrConf.UserMobileAttribute]; ok && len(v) > 0 {
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
			targetProject, err = ProjectManager.FetchProject("", projName, domainId, "")
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

func (idp *SIdentityProvider) SyncOrCreateDomainAndUser(ctx context.Context, extUsrId, extUsrName string) (*SDomain, *SUser, error) {
	var (
		domain *SDomain
		usr    *SUser
		err    error
	)
	if idp.AutoCreateUser.IsTrue() {
		domain, err = idp.GetSingleDomain(ctx, api.DefaultRemoteDomainId, idp.Name, fmt.Sprintf("%s provider %s", idp.Driver, idp.Name), false)
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
