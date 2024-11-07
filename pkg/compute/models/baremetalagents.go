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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=baremetalagent
// +onecloud:swagger-gen-model-plural=baremetalagents
type SBaremetalagentManager struct {
	db.SStandaloneResourceBaseManager
	SZoneResourceBaseManager
}

type SBaremetalagent struct {
	db.SStandaloneResourceBase
	SZoneResourceBase `width:"128" charset:"ascii" nullable:"false" list:"admin" update:"admin" create:"admin_required"`

	Status     string `width:"36" charset:"ascii" nullable:"false" default:"disable" create:"optional"`
	AccessIp   string `width:"16" charset:"ascii" nullable:"false" list:"admin" update:"admin" create:"admin_required"`
	ManagerUri string `width:"256" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_required"`
	// ZoneId     string `width:"128" charset:"ascii" nullable:"false" list:"admin" update:"admin" create:"admin_required"`

	AgentType string `width:"32" charset:"ascii" nullable:"true" default:"baremetal" list:"admin" update:"admin" create:"admin_optional"`

	Version string `width:"128" charset:"ascii" list:"admin" update:"admin" create:"admin_optional"` // Column(VARCHAR(64, charset='ascii'))

	StoragecacheId    string `width:"36" charset:"ascii" nullable:"true" list:"admin" get:"admin" update:"admin" create:"admin_optional"`
	DisableImageCache bool   `default:"false" list:"admin" create:"admin_optional" update:"admin"`
}

var BaremetalagentManager *SBaremetalagentManager

func init() {
	BaremetalagentManager = &SBaremetalagentManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(SBaremetalagent{},
			"baremetalagents_tbl",
			"baremetalagent",
			"baremetalagents",
		)}
	BaremetalagentManager.SetVirtualObject(BaremetalagentManager)
}

func (self *SBaremetalagent) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if self.Status == api.BAREMETAL_AGENT_ENABLED {
		return fmt.Errorf("Cannot delete in status %s", self.Status)
	}
	storageCache, _ := self.getStorageCache()
	if storageCache != nil {
		err := storageCache.ValidateDeleteCondition(ctx, nil)
		if err != nil {
			return fmt.Errorf("storagecache cannot be delete: %s", err)
		}
	}
	return self.SStandaloneResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SBaremetalagent) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.BaremetalagentUpdateInput) (api.BaremetalagentUpdateInput, error) {
	var err error
	mangerUri := input.ManagerUri
	if len(mangerUri) > 0 {
		count, err := BaremetalagentManager.Query().Equals("manager_uri", mangerUri).
			NotEquals("id", self.Id).CountWithError()
		if err != nil {
			return input, httperrors.NewInternalServerError("check agent uniqness fail %s", err)
		}
		if count > 0 {
			return input, httperrors.NewConflictError("Conflict manager_uri %s", mangerUri)
		}
	}
	if len(input.ZoneId) > 0 {
		_, input.ZoneResourceInput, err = ValidateZoneResourceInput(ctx, userCred, input.ZoneResourceInput)
		if err != nil {
			return input, errors.Wrap(err, "ValidateZoneResourceInput")
		}
	}
	input.StandaloneResourceBaseUpdateInput, err = self.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.StandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStandaloneResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (manager *SBaremetalagentManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.BaremetalagentCreateInput) (api.BaremetalagentCreateInput, error) {
	var err error
	mangerUri := input.ManagerUri
	if len(mangerUri) == 0 {
		return input, errors.Wrap(httperrors.ErrMissingParameter, "manager_uri")
	}
	count, err := manager.Query().Equals("manager_uri", mangerUri).CountWithError()
	if err != nil {
		return input, httperrors.NewInternalServerError("check agent uniqness fail %s", err)
	}
	if count > 0 {
		return input, httperrors.NewDuplicateResourceError("Duplicate manager_uri %s", mangerUri)
	}
	if len(input.ZoneId) == 0 {
		return input, errors.Wrap(httperrors.ErrMissingParameter, "zone/zone_id")
	}
	_, input.ZoneResourceInput, err = ValidateZoneResourceInput(ctx, userCred, input.ZoneResourceInput)
	if err != nil {
		return input, errors.Wrap(err, "ValidateZoneResourceInput")
	}
	input.StandaloneResourceCreateInput, err = manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StandaloneResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStandaloneResourceBaseManager.ValidateCreateData")
	}
	return input, nil
}

func (self *SBaremetalagent) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status != api.BAREMETAL_AGENT_ENABLED {
		db.Update(self, func() error {
			self.Status = api.BAREMETAL_AGENT_ENABLED
			return nil
		})
		db.OpsLog.LogEvent(self, db.ACT_ENABLE, "", userCred)
	}
	return nil, nil
}

func (self *SBaremetalagent) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status != api.BAREMETAL_AGENT_DISABLED {
		db.Update(self, func() error {
			self.Status = api.BAREMETAL_AGENT_DISABLED
			return nil
		})
		db.OpsLog.LogEvent(self, db.ACT_DISABLE, "", userCred)
	}
	return nil, nil
}

func (self *SBaremetalagent) PerformEnableImageCache(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.DisableImageCache {
		db.Update(self, func() error {
			self.DisableImageCache = false
			return nil
		})
	}
	return nil, nil
}

func (self *SBaremetalagent) PerformDisableImageCache(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.DisableImageCache {
		db.Update(self, func() error {
			self.DisableImageCache = true
			return nil
		})
	}
	return nil, nil
}

func (self *SBaremetalagent) PerformOnline(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status == api.BAREMETAL_AGENT_OFFLINE {
		db.Update(self, func() error {
			self.Status = api.BAREMETAL_AGENT_ENABLED
			return nil
		})
		db.OpsLog.LogEvent(self, db.ACT_ONLINE, "", userCred)
	}
	return nil, nil
}

func (self *SBaremetalagent) PerformOffline(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status == api.BAREMETAL_AGENT_ENABLED {
		db.Update(self, func() error {
			self.Status = api.BAREMETAL_AGENT_OFFLINE
			return nil
		})
		db.OpsLog.LogEvent(self, db.ACT_OFFLINE, "", userCred)
	}
	return nil, nil
}

func (self *SBaremetalagent) GetZone() *SZone {
	if len(self.ZoneId) > 0 && regutils.MatchUUIDExact(self.ZoneId) {
		return ZoneManager.FetchZoneById(self.ZoneId)
	}
	return nil
}

func (manager *SBaremetalagentManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.BaremetalagentDetails {
	rows := make([]api.BaremetalagentDetails, len(objs))
	stdRows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	zoneRows := manager.SZoneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.BaremetalagentDetails{
			StandaloneResourceDetails: stdRows[i],
			ZoneResourceInfo:          zoneRows[i],
		}
	}
	return rows
}

func (manager *SBaremetalagentManager) GetAgent(agentType api.TAgentType, zoneId string) *SBaremetalagent {
	q := manager.Query().Equals("agent_type", agentType)
	if len(zoneId) > 0 {
		q = q.Equals("zone_id", zoneId)
	}
	q = q.Asc("created_at")
	agents := make([]SBaremetalagent, 0)
	err := db.FetchModelObjects(manager, q, &agents)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("GetAgent query fail %s", err)
		}
		return nil
	}
	if len(agents) == 0 {
		return nil
	}
	for i := range agents {
		if agents[i].Status == api.BAREMETAL_AGENT_ENABLED {
			return &agents[i]
		}
	}
	return &agents[0]
}

func (cache *SBaremetalagent) getStorageCache() (*SStoragecache, error) {
	if len(cache.StoragecacheId) > 0 {
		cacheObj, err := StoragecacheManager.FetchById(cache.StoragecacheId)
		if err != nil {
			return nil, err
		}
		return cacheObj.(*SStoragecache), nil
	}
	return nil, nil
}

func (agent *SBaremetalagent) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	err := agent.SStandaloneResourceBase.CustomizeDelete(ctx, userCred, query, data)
	if err != nil {
		return err
	}
	cache, _ := agent.getStorageCache()
	if cache != nil {
		err = cache.Delete(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (agent *SBaremetalagent) setStoragecacheId(cacheId string) error {
	_, err := db.Update(agent, func() error {
		agent.StoragecacheId = cacheId
		return nil
	})
	return err
}

// 管理代理服务列表
func (manager *SBaremetalagentManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.BaremetalagentListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SZoneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ZonalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemFilter")
	}
	if len(query.Status) > 0 {
		q = q.In("status", query.Status)
	}
	if len(query.AccessIp) > 0 {
		q = q.In("access_ip", query.AccessIp)
	}
	if len(query.AgentType) > 0 {
		q = q.In("agent_type", query.AgentType)
	}
	return q, nil
}

func (manager *SBaremetalagentManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SZoneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SBaremetalagentManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.BaremetalagentListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SZoneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ZonalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SZoneResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SBaremetalagentManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SZoneResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SZoneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}
