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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=loadbalancer_health_check
// +onecloud:swagger-gen-model-plural=loadbalancerhealthchecks
type SLoadbalancerHealthCheckManager struct {
	SLoadbalancerLogSkipper
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
}

var LoadbalancerHealthCheckManager *SLoadbalancerHealthCheckManager

func init() {
	LoadbalancerHealthCheckManager = &SLoadbalancerHealthCheckManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SLoadbalancerHealthCheck{},
			"loadbalancer_health_checks_tbl",
			"loadbalancer_health_check",
			"loadbalancer_health_checks",
		),
	}
	LoadbalancerHealthCheckManager.SetVirtualObject(LoadbalancerHealthCheckManager)
}

type SLoadbalancerHealthCheck struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase
	SLoadbalancerHealthChecker
}

// 健康检查列表
func (man *SLoadbalancerHealthCheckManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerHealthCheckListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = man.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = man.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	q, err = man.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (man *SLoadbalancerHealthCheckManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerHealthCheckListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (man *SLoadbalancerHealthCheckManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (man *SLoadbalancerHealthCheckManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input *api.SLoadbalancerHealthCheckCreateInput,
) (*api.SLoadbalancerHealthCheckCreateInput, error) {
	var err error
	input.VirtualResourceCreateInput, err = man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ValidateCreateData")
	}
	input.Status = apis.STATUS_CREATING

	regionObj, err := validators.ValidateModel(ctx, userCred, CloudregionManager, &input.CloudregionId)
	if err != nil {
		return nil, err
	}
	region := regionObj.(*SCloudregion)
	if len(input.CloudproviderId) > 0 {
		providerObj, err := validators.ValidateModel(ctx, userCred, CloudproviderManager, &input.CloudproviderId)
		if err != nil {
			return nil, err
		}
		input.ManagerId = input.CloudproviderId
		provider := providerObj.(*SCloudprovider)
		if provider.Provider != region.Provider {
			return nil, httperrors.NewConflictError("conflict region %s and cloudprovider %s", region.Name, provider.Name)
		}
	}
	return input, nil
}

func (hc *SLoadbalancerHealthCheck) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.SLoadbalancerHealthCheckUpdateInput,
) (*api.SLoadbalancerHealthCheckUpdateInput, error) {
	var err error
	input.VirtualResourceBaseUpdateInput, err = hc.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.VirtualResourceBaseUpdateInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (hc *SLoadbalancerHealthCheck) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	hc.SVirtualResourceBase.PostUpdate(ctx, userCred, query, data)
	keys, _ := jsonutils.Marshal(SLoadbalancerHealthChecker{}).(*jsonutils.JSONDict).GetMap()
	needUpdate := false
	for key := range keys {
		if data.Contains(key) {
			needUpdate = true
			break
		}
	}
	if needUpdate {
		hc.StartLoadBalancerHealthCheckUpdateTask(ctx, userCred, "")
	}
}

func (hc *SLoadbalancerHealthCheck) StartLoadBalancerHealthCheckUpdateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	hc.SetStatus(ctx, userCred, api.LB_SYNC_CONF, "")
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerHealthCheckUpdateTask", hc, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	return task.ScheduleRun(nil)
}

func (hc *SLoadbalancerHealthCheck) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	hc.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	hc.SetStatus(ctx, userCred, api.LB_CREATING, "")
	err := hc.StartLoadBalancerHealthCheckCreateTask(ctx, userCred, "")
	if err != nil {
		log.Errorf("Failed to create loadbalancer backend error: %v", err)
	}
}

func (manager *SLoadbalancerHealthCheckManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LoadbalancerHealthCheckDetails {
	rows := make([]api.LoadbalancerHealthCheckDetails, len(objs))
	stdRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.LoadbalancerHealthCheckDetails{
			VirtualResourceDetails:  stdRows[i],
			ManagedResourceInfo:     managerRows[i],
			CloudregionResourceInfo: regionRows[i],
		}
	}

	return rows
}

func (hc *SLoadbalancerHealthCheck) StartLoadBalancerHealthCheckCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerHealthCheckCreateTask", hc, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (hc *SLoadbalancerHealthCheck) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (hc *SLoadbalancerHealthCheck) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return hc.SVirtualResourceBase.Delete(ctx, userCred)
}

func (hc *SLoadbalancerHealthCheck) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	region, err := hc.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion")
	}
	provider, err := hc.GetDriver(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDriver")
	}
	return provider.GetIRegionById(region.ExternalId)
}

func (hc *SLoadbalancerHealthCheck) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	hc.SetStatus(ctx, userCred, api.LB_STATUS_DELETING, "")
	return hc.StartLoadBalancerHealthCheckDeleteTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (hc *SLoadbalancerHealthCheck) StartLoadBalancerHealthCheckDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerHealthCheckDeleteTask", hc, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	return task.ScheduleRun(nil)
}

func (region *SCloudregion) GetLoadbalancerHealthChecks(managerId string) ([]SLoadbalancerHealthCheck, error) {
	q := LoadbalancerHealthCheckManager.Query().Equals("manager_id", managerId).Equals("cloudregion_id", region.Id)
	ret := []SLoadbalancerHealthCheck{}
	err := db.FetchModelObjects(LoadbalancerHealthCheckManager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return ret, nil
}

func (region *SCloudregion) SyncLoadbalancerHealthChecks(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, exts []cloudprovider.ICloudLoadbalancerHealthCheck) compare.SyncResult {
	lockman.LockRawObject(ctx, LoadbalancerHealthCheckManager.Keyword(), region.Id)
	defer lockman.ReleaseRawObject(ctx, LoadbalancerHealthCheckManager.Keyword(), region.Id)

	result := compare.SyncResult{}
	dbRes, err := region.GetLoadbalancerHealthChecks(provider.Id)
	if err != nil {
		result.Error(err)
		return result
	}

	removed := []SLoadbalancerHealthCheck{}
	commondb := []SLoadbalancerHealthCheck{}
	commonext := []cloudprovider.ICloudLoadbalancerHealthCheck{}
	added := []cloudprovider.ICloudLoadbalancerHealthCheck{}

	err = compare.CompareSets(dbRes, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].syncRemove(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}
	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudLoadbalancerHealthCheck(ctx, userCred, commonext[i], provider)
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}
	for i := 0; i < len(added); i++ {
		err := region.newFromCloudLoadbalancerHealthCheck(ctx, userCred, added[i], provider)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}

func (hc *SLoadbalancerHealthCheck) SyncWithCloudLoadbalancerHealthCheck(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ext cloudprovider.ICloudLoadbalancerHealthCheck,
	provider *SCloudprovider,
) error {
	diff, err := db.UpdateWithLock(ctx, hc, func() error {
		hc.HealthCheck = ext.GetHealthCheck()
		hc.HealthCheckType = ext.GetHealthCheckType()
		hc.HealthCheckDomain = ext.GetHealthCheckDomain()
		hc.HealthCheckURI = ext.GetHealthCheckURI()
		hc.HealthCheckHttpCode = ext.GetHealthCheckCode()
		hc.HealthCheckMethod = ext.GetHealthCheckMethod()
		hc.HealthCheckPort = ext.GetHealthCheckPort()
		hc.HealthCheckRise = ext.GetHealthCheckRise()
		hc.HealthCheckFall = ext.GetHealthCheckFail()
		hc.HealthCheckTimeout = ext.GetHealthCheckTimeout()
		hc.HealthCheckInterval = ext.GetHealthCheckInterval()
		hc.HealthCheckReq = ext.GetHealthCheckReq()
		hc.HealthCheckExp = ext.GetHealthCheckExp()
		return nil
	})
	if err != nil {
		return err
	}

	syncVirtualResourceMetadata(ctx, userCred, hc, ext, false)
	SyncCloudProject(ctx, userCred, hc, provider.GetOwnerId(), ext, provider)

	if len(diff) > 0 {
		db.OpsLog.LogSyncUpdate(hc, diff, userCred)
	}

	return nil
}

func (region *SCloudregion) newFromCloudLoadbalancerHealthCheck(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ext cloudprovider.ICloudLoadbalancerHealthCheck,
	provider *SCloudprovider,
) error {
	hc := &SLoadbalancerHealthCheck{}
	hc.SetModelManager(LoadbalancerHealthCheckManager, hc)
	hc.Name = ext.GetName()
	hc.Status = ext.GetStatus()
	hc.ManagerId = provider.Id
	hc.CloudregionId = region.Id
	hc.ExternalId = ext.GetGlobalId()
	hc.HealthCheck = ext.GetHealthCheck()
	hc.HealthCheckType = ext.GetHealthCheckType()
	hc.HealthCheckDomain = ext.GetHealthCheckDomain()
	hc.HealthCheckURI = ext.GetHealthCheckURI()
	hc.HealthCheckHttpCode = ext.GetHealthCheckCode()
	hc.HealthCheckMethod = ext.GetHealthCheckMethod()
	hc.HealthCheckPort = ext.GetHealthCheckPort()
	hc.HealthCheckRise = ext.GetHealthCheckRise()
	hc.HealthCheckFall = ext.GetHealthCheckFail()
	hc.HealthCheckTimeout = ext.GetHealthCheckTimeout()
	hc.HealthCheckInterval = ext.GetHealthCheckInterval()
	hc.HealthCheckReq = ext.GetHealthCheckReq()
	hc.HealthCheckExp = ext.GetHealthCheckExp()
	hc.DomainId = provider.DomainId
	hc.ProjectId = provider.ProjectId
	err := LoadbalancerHealthCheckManager.TableSpec().Insert(ctx, hc)
	if err != nil {
		return errors.Wrap(err, "Insert")
	}

	syncVirtualResourceMetadata(ctx, userCred, hc, ext, false)
	SyncCloudProject(ctx, userCred, hc, provider.GetOwnerId(), ext, provider)

	return nil
}

func (hc *SLoadbalancerHealthCheck) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, hc)
	defer lockman.ReleaseObject(ctx, hc)

	return hc.RealDelete(ctx, userCred)
}

func (hc *SLoadbalancerHealthCheck) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, StartResourceSyncStatusTask(ctx, userCred, hc, "LoadbalancerHealthCheckSyncstatusTask", "")
}

func (manager *SLoadbalancerHealthCheckManager) ListItemExportKeys(ctx context.Context,
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
