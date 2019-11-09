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
	"fmt"
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
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
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

	Driver   string `width:"32" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"`
	Template string `width:"32" charset:"ascii" nullable:"true" list:"admin" create:"admin_optional"`

	TargetDomainId string `width:"64" charset:"ascii" nullable:"true" list:"admin" create:"admin_optional" update:"admin"`

	AutoCreateProject tristate.TriState `default:"true" nullable:"true" list:"admin" create:"admin_optional" update:"admin"`

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
	conf, err := GetConfigs(self, false)
	if err != nil {
		return nil, err
	}
	result := jsonutils.NewDict()
	result.Add(jsonutils.Marshal(conf), "config")
	return result, nil
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
	opts := api.TConfigs{}
	err := data.Unmarshal(&opts, "config")
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid input data")
	}
	action, _ := data.GetString("action")
	err = saveConfigs(action, ident, opts, nil, api.SensitiveDomainConfigMap)
	if err != nil {
		return nil, httperrors.NewInternalServerError("saveConfig fail %s", err)
	}
	ident.MarkDisconnected(ctx, userCred)
	submitIdpSyncTask(ctx, userCred, ident)
	return ident.GetDetailsConfig(ctx, userCred, query)
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

	targetDomainStr, _ := data.GetString("target_domain")
	if len(targetDomainStr) > 0 {
		domain, err := DomainManager.FetchDomainById(targetDomainStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(DomainManager.Keyword(), targetDomainStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		data.Set("target_domain_id", jsonutils.NewString(domain.Id))
	}

	opts := api.TConfigs{}
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

	logclient.AddActionLogWithContext(ctx, ident, logclient.ACT_CREATE, data, userCred, true)

	opts := api.TConfigs{}
	err := data.Unmarshal(&opts, "config")
	if err != nil {
		log.Errorf("parse config error %s", err)
		return
	}
	err = saveConfigs("", ident, opts, nil, api.SensitiveDomainConfigMap)
	if err != nil {
		log.Errorf("saveConfig fail %s", err)
		return
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
	if len(self.TargetDomainId) > 0 {
		domain, _ := DomainManager.FetchDomainById(self.TargetDomainId)
		if domain != nil {
			extra.Set("target_domain", jsonutils.NewString(domain.Name))
		}
	}
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

func (self *SIdentityProvider) GetSingleDomain(ctx context.Context, extId string, extName string, extDesc string) (*SDomain, error) {
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
	return self.SyncOrCreateDomain(ctx, extId, extName, extDesc)
}

func (self *SIdentityProvider) SyncOrCreateDomain(ctx context.Context, extId string, extName string, extDesc string) (*SDomain, error) {
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
	err = DomainManager.TableSpec().Insert(domain)
	if err != nil {
		return nil, errors.Wrap(err, "insert")
	}

	if self.AutoCreateProject.IsTrue() && consts.GetNonDefaultDomainProjects() {
		project := &SProject{}
		project.SetModelManager(ProjectManager, project)
		projectName := NormalizeProjectName(fmt.Sprintf("%s_default_project", extName))
		newName, err := db.GenerateName(ProjectManager, nil, projectName)
		if err != nil {
			// ignore the error
			log.Errorf("db.GenerateName error %s for default domain project %s", err, projectName)
			newName = projectName
		}
		project.Name = newName
		project.DomainId = domain.Id
		project.Description = fmt.Sprintf("Default project for domain %s", extName)
		project.IsDomain = tristate.False
		project.ParentId = domain.Id
		err = ProjectManager.TableSpec().Insert(project)
		if err != nil {
			log.Errorf("ProjectManager.Insert fail %s", err)
		}
	}

	return domain, nil
}

func (self *SIdentityProvider) SyncOrCreateUser(ctx context.Context, extId string, extName string, domainId string, syncUserInfo func(*SUser)) (*SUser, error) {
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
		_, err := db.Update(user, func() error {
			if syncUserInfo != nil {
				syncUserInfo(user)
			}
			user.Name = extName
			user.DomainId = domainId
			user.MarkUnDelete()
			return nil
		})
		if err != nil {
			return nil, errors.Wrap(err, "Update")
		}
	} else {
		if syncUserInfo != nil {
			syncUserInfo(user)
		}
		user.Id = userId
		user.Name = extName
		user.DomainId = domainId
		err = UserManager.TableSpec().Insert(user)
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
