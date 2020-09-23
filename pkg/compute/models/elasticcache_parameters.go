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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// SElasticcache.Parameter
type SElasticcacheParameterManager struct {
	db.SStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SElasticcacheResourceBaseManager
}

var ElasticcacheParameterManager *SElasticcacheParameterManager

func init() {
	ElasticcacheParameterManager = &SElasticcacheParameterManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SElasticcacheParameter{},
			"elasticcacheparameters_tbl",
			"elasticcacheparameter",
			"elasticcacheparameters",
		),
	}
	ElasticcacheParameterManager.SetVirtualObject(ElasticcacheParameterManager)
}

type SElasticcacheParameter struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SElasticcacheResourceBase `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`

	// ElasticcacheId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"` // elastic cache instance id

	// Parameter KEY
	Key string `width:"64" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required"`

	// Parameter Value
	Value string `width:"256" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required"`

	// 校验代码，参数的可选范围。
	ValueRange string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	// True（可修改）   False（不可修改）
	Modifiable bool `nullable:"true" list:"user" create:"optional"`

	// True（重启生效） False（无需重启，提交后即生效）
	ForceRestart bool `nullable:"true" list:"user" create:"optional"`
}

func (manager *SElasticcacheParameterManager) SyncElasticcacheParameters(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, cloudElasticcacheParameters []cloudprovider.ICloudElasticcacheParameter) compare.SyncResult {
	lockman.LockRawObject(ctx, "elastic-cache-parameters", elasticcache.Id)
	defer lockman.ReleaseRawObject(ctx, "elastic-cache-parameters", elasticcache.Id)

	syncResult := compare.SyncResult{}

	dbParameters, err := elasticcache.GetElasticcacheParameters()
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := make([]SElasticcacheParameter, 0)
	commondb := make([]SElasticcacheParameter, 0)
	commonext := make([]cloudprovider.ICloudElasticcacheParameter, 0)
	added := make([]cloudprovider.ICloudElasticcacheParameter, 0)
	if err := compare.CompareSets(dbParameters, cloudElasticcacheParameters, &removed, &commondb, &commonext, &added); err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemoveCloudElasticcacheParameter(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i++ {
		err := commondb[i].SyncWithCloudElasticcacheParameter(ctx, userCred, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
			continue
		}

		syncResult.Update()
	}

	for i := 0; i < len(added); i++ {
		_, err := manager.newFromCloudElasticcacheParameter(ctx, userCred, elasticcache, added[i])
		if err != nil {
			syncResult.AddError(err)
			continue
		}

		syncResult.Add()
	}
	return syncResult
}

func (self *SElasticcacheParameter) syncRemoveCloudElasticcacheParameter(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return errors.Wrapf(err, "newFromCloudElasticcacheParameter.Remove")
	}
	return self.Delete(ctx, userCred)
}

func (self *SElasticcacheParameter) SyncWithCloudElasticcacheParameter(ctx context.Context, userCred mcclient.TokenCredential, extParameter cloudprovider.ICloudElasticcacheParameter) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = extParameter.GetStatus()
		self.Key = extParameter.GetParameterKey()
		self.Value = extParameter.GetParameterValue()
		self.Modifiable = extParameter.GetModifiable()
		self.ForceRestart = extParameter.GetForceRestart()
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "SyncWithCloudElasticcacheParameter.UpdateWithLock")
	}

	return nil
}

func (manager *SElasticcacheParameterManager) newFromCloudElasticcacheParameter(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, extParameter cloudprovider.ICloudElasticcacheParameter) (*SElasticcacheParameter, error) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	parameter := SElasticcacheParameter{}
	parameter.SetModelManager(manager, &parameter)

	parameter.ElasticcacheId = elasticcache.Id
	parameter.Status = extParameter.GetStatus()
	parameter.Name = extParameter.GetName()
	parameter.ExternalId = extParameter.GetGlobalId()
	parameter.Key = extParameter.GetParameterKey()
	parameter.Value = extParameter.GetParameterValue()
	parameter.ValueRange = extParameter.GetParameterValueRange()
	parameter.Modifiable = extParameter.GetModifiable()
	parameter.ForceRestart = extParameter.GetForceRestart()
	parameter.Description = extParameter.GetDescription()

	err := manager.TableSpec().Insert(ctx, &parameter)
	if err != nil {
		return nil, errors.Wrapf(err, "newFromCloudElasticcacheParameter.Insert")
	}

	return &parameter, nil
}

func (manager *SElasticcacheParameterManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeProject
}

func (manager *SElasticcacheParameterManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return elasticcacheSubResourceFetchOwnerId(ctx, data)
}

func (manager *SElasticcacheParameterManager) FilterByOwner(q *sqlchemy.SQuery, userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	return elasticcacheSubResourceFetchOwner(q, userCred, scope)
}

func (self *SElasticcacheParameter) GetOwnerId() mcclient.IIdentityProvider {
	return ElasticcacheManager.GetOwnerIdByElasticcacheId(self.ElasticcacheId)
}

func (self *SElasticcacheParameter) GetRegion() *SCloudregion {
	ieb, err := db.FetchById(ElasticcacheManager, self.ElasticcacheId)
	if err != nil {
		return nil
	}

	return ieb.(*SElasticcache).GetRegion()
}

func (self *SElasticcacheParameter) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if !self.Modifiable {
		return nil, httperrors.NewConflictError("%s is not modifiable", self.Name)
	}

	_, err := data.GetString("value")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("value")
	}

	return data, nil
}

func (self *SElasticcacheParameter) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	v, _ := data.Get("value")
	params := jsonutils.NewDict()
	paramsObj := jsonutils.NewDict()
	paramsObj.Add(v, self.Name)
	params.Add(paramsObj, "parameters")

	self.SetStatus(userCred, api.ELASTIC_CACHE_PARAMETER_STATUS_UPDATING, "")
	if err := self.StartUpdateElasticcacheParameterTask(ctx, userCred, params, ""); err != nil {
		log.Errorf("ElasticcacheParameter %s", err.Error())
	}

	return
}

func (self *SElasticcacheParameter) StartUpdateElasticcacheParameterTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ElasticcacheParameterUpdateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SElasticcacheParameter) ValidatePurgeCondition(ctx context.Context) error {
	return nil
}

// 列出弹性缓存参数
func (manager *SElasticcacheParameterManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.ElasticcacheParameterListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SElasticcacheResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ElasticcacheFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SElasticcacheResourceBaseManager.ListItemFilter")
	}

	if len(input.Key) > 0 {
		q = q.In("key", input.Key)
	}
	if len(input.Value) > 0 {
		q = q.In("value", input.Value)
	}

	return q, nil
}

func (manager *SElasticcacheParameterManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.ElasticcacheParameterListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}

	q, err = manager.SElasticcacheResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.ElasticcacheFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SElasticcacheResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SElasticcacheParameterManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SElasticcacheResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (self *SElasticcacheParameter) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.ElasticcacheParameterDetails, error) {
	return api.ElasticcacheParameterDetails{}, nil
}

func (manager *SElasticcacheParameterManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ElasticcacheParameterDetails {
	rows := make([]api.ElasticcacheParameterDetails, len(objs))

	stdRows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	cacheRows := manager.SElasticcacheResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.ElasticcacheParameterDetails{
			StandaloneResourceDetails: stdRows[i],
			ElasticcacheResourceInfo:  cacheRows[i],
		}
	}

	return rows
}

func (manager *SElasticcacheParameterManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SElasticcacheResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SElasticcacheResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SElasticcacheResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}
