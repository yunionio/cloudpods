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

	"yunion.io/x/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SIdentityProviderManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
}

var (
	IdentityProviderManager *SIdentityProviderManager
)

func init() {
	IdentityProviderManager = &SIdentityProviderManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SIdentityProvider{},
			"identity_provider",
			"identity_provider",
			"identity_providers",
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

	Driver   string `width:"32" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"`
	Template string `width:"32" charset:"ascii" nullable:"true" list:"admin" create:"admin_optional"`

	ErrorCount int `list:"admin"`

	SyncStatus    string    `width:"10" charset:"ascii" default:"idle" list:"admin"`
	LastSync      time.Time `list:"admin"` // = Column(DateTime, nullable=True)
	LastSyncEndAt time.Time `list:"admin"`

	SyncIntervalSeconds int `create:"admin_optional" update:"admin"`
}

func (manager *SIdentityProviderManager) InitializeData() error {
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
	sqldrv.Enabled = true
	sqldrv.Status = api.IdentityDriverStatusConnected
	sqldrv.Driver = api.IdentityDriverSQL
	sqldrv.Description = "Default sql identity provider"
	err = manager.TableSpec().Insert(&sqldrv)
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
		drv.Enabled = domains[i].Enabled.Bool()
		drv.Status = api.IdentityDriverStatusDisconnected
		drv.Driver = driver
		drv.Description = domains[i].Description
		err = manager.TableSpec().Insert(&drv)
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
	_, err := db.UpdateWithLock(ctx, ident, func() error {
		ident.ErrorCount = 0
		return nil
	})
	if err != nil {
		return err
	}
	return ident.SetStatus(userCred, api.IdentityDriverStatusConnected, "")
}

func (ident *SIdentityProvider) MarkDisconnected(ctx context.Context, userCred mcclient.TokenCredential) error {
	_, err := db.UpdateWithLock(ctx, ident, func() error {
		ident.ErrorCount = ident.ErrorCount + 1
		return nil
	})
	if err != nil {
		return err
	}
	return ident.SetStatus(userCred, api.IdentityDriverStatusDisconnected, "")
}

func (self *SIdentityProvider) AllowGetDetailsConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGetSpec(userCred, self, "config")
}

func (self *SIdentityProvider) GetDetailsConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	conf, err := self.GetConfig(false)
	if err != nil {
		return nil, err
	}
	result := jsonutils.NewDict()
	result.Add(jsonutils.Marshal(conf), "config")
	return result, nil
}

func (self *SIdentityProvider) GetConfig(all bool) (api.TIdentityProviderConfigs, error) {
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

func (ident *SIdentityProvider) AllowPerformConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) bool {
	return db.IsAdminAllowUpdateSpec(userCred, ident, "config")
}

func (ident *SIdentityProvider) PerformConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	if ident.Status == api.IdentityDriverStatusConnected && ident.Enabled {
		return nil, httperrors.NewInvalidStatusError("cannot update config when enabled and connected")
	}
	if ident.SyncStatus != api.IdentitySyncStatusIdle {
		return nil, httperrors.NewInvalidStatusError("cannot update config when not idle")
	}
	opts := api.TIdentityProviderConfigs{}
	err := data.Unmarshal(&opts, "config")
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid input data")
	}

	err = ident.saveConfig(ctx, userCred, opts)
	if err != nil {
		return nil, httperrors.NewInternalServerError("saveConfig fail %s", err)
	}
	ident.MarkDisconnected(ctx, userCred)
	submitIdpSyncTask(ctx, userCred, ident)
	return ident.GetDetailsConfig(ctx, userCred, query)
}

func (ident *SIdentityProvider) saveConfig(ctx context.Context, userCred mcclient.TokenCredential, opts api.TIdentityProviderConfigs) error {
	whiteListedOpts, sensitiveOpts := getConfigOptions(opts, ident.Id, api.SensitiveDomainConfigMap)
	err := WhitelistedConfigManager.syncConfig(ctx, userCred, ident.Id, whiteListedOpts)
	if err != nil {
		return errors.Wrap(err, "WhitelistedConfigManager.syncConfig")
	}
	err = SensitiveConfigManager.syncConfig(ctx, userCred, ident.Id, sensitiveOpts)
	if err != nil {
		return errors.Wrap(err, "SensitiveConfigManager.syncConfig")
	}
	return nil
}

func (manager *SIdentityProviderManager) getDriveInstanceCount(drvName string) (int, error) {
	return manager.Query().Equals("driver", drvName).CountWithError()
}

func (manager *SIdentityProviderManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	var drvName string

	template, _ := data.GetString("template")
	if len(template) > 0 {
		if _, ok := api.IdpTemplateDriver[template]; !ok {
			return nil, httperrors.NewInputParameterError("invalid template")
		}
		drvName = api.IdpTemplateDriver[template]
		data.Set("driver", jsonutils.NewString(drvName))
	} else {
		drvName, _ = data.GetString("driver")
		if len(drvName) == 0 {
			return nil, httperrors.NewInputParameterError("missing driver")
		}
	}

	drvCls := driver.GetDriverClass(drvName)
	if drvCls == nil {
		return nil, httperrors.NewInputParameterError("driver %s not supported", drvName)
	}

	if drvCls.SingletonInstance() {
		cnt, err := manager.getDriveInstanceCount(drvName)
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
		if cnt >= 1 {
			return nil, httperrors.NewConflictError("driver %s already exists", drvName)
		}
	}

	if data.Contains("sync_interval_seconds") {
		secs, _ := data.Int("sync_interval_seconds")
		if secs < api.MinimalSyncIntervalSeconds {
			data.Set("sync_interval_seconds", jsonutils.NewInt(int64(api.MinimalSyncIntervalSeconds)))
		}
	}

	opts := api.TIdentityProviderConfigs{}
	err := data.Unmarshal(&opts, "config")
	if err != nil {
		return nil, httperrors.NewInputParameterError("parse config error: %s", err)
	}
	return manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
}

func (ident *SIdentityProvider) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	ident.Enabled = true
	return ident.SEnabledStatusStandaloneResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (ident *SIdentityProvider) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	ident.SEnabledStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	opts := api.TIdentityProviderConfigs{}
	err := data.Unmarshal(&opts, "config")
	if err != nil {
		log.Errorf("parse config error %s", err)
		return
	}
	err = ident.saveConfig(ctx, userCred, opts)
	if err != nil {
		log.Errorf("saveConfig fail %s", err)
		return
	}

	submitIdpSyncTask(ctx, userCred, ident)
	return
}

func (manager *SIdentityProviderManager) fetchEnabledProviders() ([]SIdentityProvider, error) {
	q := manager.Query().IsTrue("enabled")
	providers := make([]SIdentityProvider, 0)
	err := db.FetchModelObjects(manager, q, &providers)
	if err != nil {
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
		return options.Options.DefaultSyncIntervalSeoncds
	}
	return self.SyncIntervalSeconds
}

func (self *SIdentityProvider) NeedSync() bool {
	if self.Driver != api.IdentityProviderSyncFull {
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

func (self *SIdentityProvider) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.Enabled {
		return nil, httperrors.NewInvalidStatusError("Account disabled")
	}
	if self.CanSync() {
		submitIdpSyncTask(ctx, userCred, self)
	}
	return nil, nil
}

func (self *SIdentityProvider) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SEnabledStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(extra)
}

func (self *SIdentityProvider) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SEnabledStatusStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return self.getMoreDetails(extra), nil
}

func (self *SIdentityProvider) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	extra = db.FetchModelExtraCountProperties(self, extra)
	extra.Set("sync_interval_seconds", jsonutils.NewInt(int64(self.getSyncIntervalSeconds())))
	return extra
}

func (self *SIdentityProvider) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if data.Contains("sync_interval_seconds") {
		secs, _ := data.Int("sync_interval_seconds")
		if secs < api.MinimalSyncIntervalSeconds {
			data.Set("sync_interval_seconds", jsonutils.NewInt(int64(api.MinimalSyncIntervalSeconds)))
		}
	}
	return self.SEnabledStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (self *SIdentityProvider) GetUserCount() (int, error) {
	if self.Driver == api.IdentityDriverSQL {
		return self.getLocalUserCount()
	} else {
		return self.getNonlocalUserCount()
	}
}

func (self *SIdentityProvider) getNonlocalUserCount() (int, error) {
	users := UserManager.Query().SubQuery()
	idmaps := IdmappingManager.Query().SubQuery()

	q := users.Query()
	q = q.LeftJoin(idmaps, sqlchemy.AND(
		sqlchemy.Equals(users.Field("id"), idmaps.Field("public_id")),
		sqlchemy.Equals(idmaps.Field("entity_type"), api.IdMappingEntityUser),
	))
	q = q.Filter(sqlchemy.Equals(idmaps.Field("domain_id"), self.Id))

	return q.CountWithError()
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
	groups := GroupManager.Query().SubQuery()
	idmaps := IdmappingManager.Query().SubQuery()

	q := groups.Query()
	q = q.LeftJoin(idmaps, sqlchemy.AND(
		sqlchemy.Equals(groups.Field("id"), idmaps.Field("public_id")),
		sqlchemy.Equals(idmaps.Field("entity_type"), api.IdMappingEntityGroup),
	))
	q = q.Filter(sqlchemy.Equals(idmaps.Field("domain_id"), self.Id))

	return q.CountWithError()
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

func (ident *SIdentityProvider) deleteConfig(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := WhitelistedConfigManager.deleteConfig(ctx, userCred, ident.Id)
	if err != nil {
		return errors.Wrap(err, "WhitelistedConfigManager.deleteConfig")
	}
	err = SensitiveConfigManager.deleteConfig(ctx, userCred, ident.Id)
	if err != nil {
		return errors.Wrap(err, "SensitiveConfigManager.deleteConfig")
	}
	return nil
}

func (self *SIdentityProvider) Purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	domains, err := self.getDomains()
	if err != nil {
		return errors.Wrap(err, "getDomains")
	}
	for i := range domains {
		err = domains[i].ValidatePurgeCondition(ctx)
		if err != nil {
			return errors.Wrap(err, "domain.ValidateDeleteCondition")
		}
		err = domains[i].purge(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "purge domain")
		}
	}
	err = self.deleteConfig(ctx, userCred)
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
