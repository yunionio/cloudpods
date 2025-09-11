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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SAiGatewayManager struct {
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
}

var AiGatewayManager *SAiGatewayManager

func init() {
	AiGatewayManager = &SAiGatewayManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SAiGateway{},
			"ai_gateways_tbl",
			"ai_gateway",
			"ai_gateways",
		),
	}
	AiGatewayManager.SetVirtualObject(AiGatewayManager)
}

type SAiGateway struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase

	Authentication          bool   `default:"false" list:"user" create:"optional"`
	CacheInvalidateOnUpdate bool   `default:"false" list:"user" create:"optional"`
	CacheTTL                int    `default:"0" list:"user" create:"optional"`
	CollectLogs             bool   `default:"false" list:"user" create:"optional"`
	RateLimitingInterval    int    `default:"0" list:"user" create:"optional"`
	RateLimitingLimit       int    `default:"0" list:"user" create:"optional"`
	RateLimitingTechnique   string `width:"32" charset:"ascii" default:"" list:"user" create:"optional"`
}

// AI网关列表
func (manager *SAiGatewayManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AiGatewayListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, err
	}

	return q, nil
}

func (manager *SAiGatewayManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AiGatewayListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SAiGatewayManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SAiGatewayManager) QueryDistinctExtraFields(q *sqlchemy.SQuery, resource string, fields []string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraFields(q, resource, fields)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (self *SAiGateway) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.AiGatewayUpdateInput) (*api.AiGatewayUpdateInput, error) {
	var err error

	input.VirtualResourceBaseUpdateInput, err = self.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.VirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
	}

	return input, nil
}

func (manager *SAiGatewayManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input *api.AiGatewayCreateInput,
) (*api.AiGatewayCreateInput, error) {
	var err error
	input.VirtualResourceCreateInput, err = manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SVirtualResourceBaseManager.ValidateCreateData")
	}
	if len(input.CloudproviderId) > 0 {
		obj, err := validators.ValidateModel(ctx, userCred, CloudproviderManager, &input.CloudproviderId)
		if err != nil {
			return nil, err
		}
		input.ManagerId = obj.GetId()
	}
	input.Status = apis.STATUS_CREATING
	return input, nil
}

func (self *SAiGateway) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	self.StartAiGatewayCreateTask(ctx, userCred, "")
}

func (self *SAiGateway) StartAiGatewayCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	kwargs := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "AiGatewayCreateTask", self, userCred, kwargs, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	return task.ScheduleRun(nil)
}

func (self *SAiGateway) GetProvider(ctx context.Context) (cloudprovider.ICloudProvider, error) {
	if len(self.ManagerId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty manager id")
	}
	provider := self.GetCloudprovider()
	if provider == nil {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "failed to found provider")
	}
	return provider.GetProvider(ctx)
}

func (self *SAiGateway) GetIAiGateway(ctx context.Context) (cloudprovider.IAiGateway, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	provider, err := self.GetProvider(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetProvider")
	}
	ret, err := provider.GetIAiGatewayById(self.ExternalId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetIAiGatewayById(%s)", self.ExternalId)
	}
	return ret, nil
}

func (provider *SCloudprovider) GetAiGateways() ([]SAiGateway, error) {
	q := AiGatewayManager.Query().Equals("manager_id", provider.Id)
	gateways := make([]SAiGateway, 0)
	err := db.FetchModelObjects(AiGatewayManager, q, &gateways)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchModelObjects")
	}
	return gateways, nil
}

func (provider *SCloudprovider) SyncAiGateways(ctx context.Context, userCred mcclient.TokenCredential, gateways []cloudprovider.IAiGateway, xor bool) compare.SyncResult {
	lockman.LockRawObject(ctx, AiGatewayManager.Keyword(), provider.Id)
	defer lockman.ReleaseRawObject(ctx, AiGatewayManager.Keyword(), provider.Id)

	result := compare.SyncResult{}

	dbGateways, err := provider.GetAiGateways()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SAiGateway, 0)
	commondb := make([]SAiGateway, 0)
	commonext := make([]cloudprovider.IAiGateway, 0)
	added := make([]cloudprovider.IAiGateway, 0)

	err = compare.CompareSets(dbGateways, gateways, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemove(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}

	if !xor {
		for i := 0; i < len(commondb); i += 1 {
			err = commondb[i].SyncWithCloudAiGateway(ctx, userCred, commonext[i])
			if err != nil {
				result.UpdateError(err)
				continue
			}
			result.Update()
		}
	}

	for i := 0; i < len(added); i += 1 {
		err := provider.newFromCloudAiGateway(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}

func (self *SAiGateway) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.RealDelete(ctx, userCred)
	if err != nil {
		return err
	}
	return nil
}

func (self *SAiGateway) SyncWithCloudAiGateway(ctx context.Context, userCred mcclient.TokenCredential, extAiGateway cloudprovider.IAiGateway) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		if options.Options.EnableSyncName {
			newName, _ := db.GenerateAlterName(self, extAiGateway.GetName())
			if len(newName) > 0 {
				self.Name = newName
			}
		}

		self.ExternalId = extAiGateway.GetGlobalId()
		self.Authentication = extAiGateway.IsAuthentication()
		self.CacheInvalidateOnUpdate = extAiGateway.IsCacheInvalidateOnUpdate()
		self.CacheTTL = extAiGateway.GetCacheTTL()
		self.CollectLogs = extAiGateway.IsCollectLogs()
		self.RateLimitingInterval = extAiGateway.GetRateLimitingInterval()
		self.RateLimitingLimit = extAiGateway.GetRateLimitingLimit()
		self.RateLimitingTechnique = extAiGateway.GetRateLimitingTechnique()

		self.Status = extAiGateway.GetStatus()
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.UpdateWithLock")
	}
	if len(diff) > 0 {
		db.OpsLog.LogSyncUpdate(self, diff, userCred)
	}

	return nil
}

func (provider *SCloudprovider) newFromCloudAiGateway(ctx context.Context, userCred mcclient.TokenCredential, extAiGateway cloudprovider.IAiGateway) error {
	gateway := SAiGateway{}
	gateway.SetModelManager(AiGatewayManager, &gateway)

	gateway.Status = extAiGateway.GetStatus()
	gateway.ExternalId = extAiGateway.GetGlobalId()
	gateway.ManagerId = provider.Id

	gateway.Authentication = extAiGateway.IsAuthentication()
	gateway.CacheInvalidateOnUpdate = extAiGateway.IsCacheInvalidateOnUpdate()
	gateway.CacheTTL = extAiGateway.GetCacheTTL()
	gateway.CollectLogs = extAiGateway.IsCollectLogs()
	gateway.RateLimitingInterval = extAiGateway.GetRateLimitingInterval()
	gateway.RateLimitingLimit = extAiGateway.GetRateLimitingLimit()
	gateway.RateLimitingTechnique = extAiGateway.GetRateLimitingTechnique()

	if createAt := extAiGateway.GetCreatedAt(); !createAt.IsZero() {
		gateway.CreatedAt = createAt
	}

	var err = func() error {
		lockman.LockRawObject(ctx, AiGatewayManager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, AiGatewayManager.Keyword(), "name")

		newName, err := db.GenerateName(ctx, AiGatewayManager, provider.GetOwnerId(), extAiGateway.GetName())
		if err != nil {
			return err
		}
		gateway.Name = newName

		return AiGatewayManager.TableSpec().Insert(ctx, &gateway)
	}()
	if err != nil {
		return errors.Wrapf(err, "newFromCloudAiGateway")
	}

	syncVirtualResourceMetadata(ctx, userCred, &gateway, extAiGateway, false)
	SyncCloudProject(ctx, userCred, &gateway, provider.GetOwnerId(), extAiGateway, provider)

	db.OpsLog.LogEvent(&gateway, db.ACT_CREATE, gateway.GetShortDesc(ctx), userCred)

	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    &gateway,
		Action: notifyclient.ActionSyncCreate,
	})

	return nil
}

func (self *SAiGateway) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SAiGateway) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SVirtualResourceBase.Delete(ctx, userCred)
}

func (self *SAiGateway) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartAiGatewayDeleteTask(ctx, userCred, "")
}

func (manager *SAiGatewayManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.AiGatewayDetails {
	rows := make([]api.AiGatewayDetails, len(objs))
	virtRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managedRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.AiGatewayDetails{
			VirtualResourceDetails: virtRows[i],
			ManagedResourceInfo:    managedRows[i],
		}
	}
	return rows
}

func (self *SAiGateway) StartAiGatewayDeleteTask(
	ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string,
) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "AiGatewayDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	self.SetStatus(ctx, userCred, apis.STATUS_DELETING, "")
	return task.ScheduleRun(nil)
}

func (manager *SAiGatewayManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
	}

	return q, nil
}

// 同步状态
func (self *SAiGateway) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, StartResourceSyncStatusTask(ctx, userCred, self, "AiGatewaySyncstatusTask", "")
}

// 更改配置
func (self *SAiGateway) PerformChangeConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *cloudprovider.AiGatewayChangeConfigOptions) (jsonutils.JSONObject, error) {
	return nil, self.StartAiGatewayChangeConfigTask(ctx, userCred, input)
}

func (self *SAiGateway) StartAiGatewayChangeConfigTask(ctx context.Context, userCred mcclient.TokenCredential, opts *cloudprovider.AiGatewayChangeConfigOptions) error {
	kwargs := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewTask(ctx, "AiGatewayChangeConfigTask", self, userCred, kwargs, "", "", nil)
	if err != nil {
		return err
	}
	self.SetStatus(ctx, userCred, apis.STATUS_CHANGE_CONFIG, "")
	return task.ScheduleRun(nil)
}
