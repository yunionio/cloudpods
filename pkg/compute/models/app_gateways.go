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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SAppGatewayManager struct {
	db.SEnabledStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
}

var AppGatewayManager *SAppGatewayManager

func init() {
	AppGatewayManager = &SAppGatewayManager{
		SEnabledStatusInfrasResourceBaseManager: db.NewEnabledStatusInfrasResourceBaseManager(
			SAppGateway{},
			"app_gateways_tbl",
			"app_gateway",
			"app_gateways",
		),
	}
	AppGatewayManager.SetVirtualObject(AppGatewayManager)
}

type SAppGateway struct {
	db.SEnabledStatusInfrasResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase `width:"36" charset:"ascii" nullable:"false" list:"domain" create:"domain_required" default:"default"`

	// 类型
	InstanceType string `width:"64" charset:"utf8" nullable:"true" list:"user" create:"optional"`
}

func (manager *SAppGatewayManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudregionManager},
	}
}

func (self *SAppGateway) ValidateDeleteCondition(ctx context.Context) error {
	return self.SEnabledStatusInfrasResourceBase.ValidateDeleteCondition(ctx)
}

func (manager *SAppGatewayManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.AppGatewayDetails {
	rows := make([]api.AppGatewayDetails, len(objs))
	stdRows := manager.SEnabledStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.AppGatewayDetails{
			EnabledStatusInfrasResourceBaseDetails: stdRows[i],
			ManagedResourceInfo:                    managerRows[i],
			CloudregionResourceInfo:                regionRows[i],
		}
	}
	return rows
}

func (self *SAppGateway) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SAppGateway) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SEnabledStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SAppGateway) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return errors.Wrapf(err, "ValidateDeleteCondition")
	}
	return self.RealDelete(ctx, userCred)
}

// 列出应用程序网关
func (manager *SAppGatewayManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AppGatewayListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SAppGatewayManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SEnabledStatusInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	q, err = manager.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SAppGatewayManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AppGatewayListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SAppGatewayManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusInfrasResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}

//同步应用程序网关状态
func (self *SAppGateway) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "syncstatus")
}

func (self *SAppGateway) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, StartResourceSyncStatusTask(ctx, userCred, self, "AppGatewaySyncStatusTask", "")
}

func (self *SAppGateway) GetRegion() (*SCloudregion, error) {
	region, err := CloudregionManager.FetchById(self.CloudregionId)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchById(%s)", self.CloudregionId)
	}
	return region.(*SCloudregion), nil
}

func (self *SAppGateway) GetIRegion() (cloudprovider.ICloudRegion, error) {
	provider, err := self.GetDriver()
	if err != nil {
		return nil, errors.Wrapf(err, "GetDriver")
	}
	region, err := self.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion")
	}
	return provider.GetIRegionById(region.ExternalId)
}

func (self *SAppGateway) GetICloudAppGateway() (cloudprovider.ICloudApplicationGateway, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	iRegion, err := self.GetIRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "GetIRegion")
	}
	return iRegion.GetICloudApplicationGatewayById(self.ExternalId)
}

func (self *SCloudregion) GetAppGateways() ([]SAppGateway, error) {
	q := AppGatewayManager.Query().Equals("cloudregion_id", self.Id)
	ret := []SAppGateway{}
	err := db.FetchModelObjects(AppGatewayManager, q, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return ret, nil
}

func (self *SCloudregion) SyncAppGateways(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, exts []cloudprovider.ICloudApplicationGateway) compare.SyncResult {
	lockman.LockRawObject(ctx, self.Id, AppGatewayManager.Keyword())
	defer lockman.ReleaseRawObject(ctx, self.Id, AppGatewayManager.Keyword())

	result := compare.SyncResult{}

	dbApps, err := self.GetAppGateways()
	if err != nil {
		result.Error(errors.Wrapf(err, "self.GetAppGateways"))
		return result
	}

	removed := make([]SAppGateway, 0)
	commondb := make([]SAppGateway, 0)
	commonext := make([]cloudprovider.ICloudApplicationGateway, 0)
	added := make([]cloudprovider.ICloudApplicationGateway, 0)
	err = compare.CompareSets(dbApps, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrapf(err, "compare.CompareSets"))
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemove(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].SyncWithCloudAppGateway(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}
	for i := 0; i < len(added); i += 1 {
		_, err := self.newFromCloudAppGateway(ctx, userCred, provider, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (self *SAppGateway) SyncWithCloudAppGateway(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudApplicationGateway) error {
	_, err := db.Update(self, func() error {
		self.Status = ext.GetStatus()
		self.InstanceType = ext.GetInstanceType()
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}

	syncMetadata(ctx, userCred, self, ext)
	provider := self.GetCloudprovider()
	if provider != nil {
		SyncCloudDomain(userCred, self, provider.GetOwnerId())
	}

	return nil
}

func (self *SCloudregion) newFromCloudAppGateway(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudApplicationGateway) (*SAppGateway, error) {
	app := &SAppGateway{}
	app.SetModelManager(AppGatewayManager, app)
	app.Status = ext.GetStatus()
	app.Enabled = tristate.True
	app.CloudregionId = self.Id
	app.ManagerId = provider.Id
	app.ExternalId = ext.GetGlobalId()
	app.InstanceType = ext.GetInstanceType()

	err := func() error {
		lockman.LockRawObject(ctx, AppGatewayManager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, AppGatewayManager.Keyword(), "name")

		var err error
		app.Name, err = db.GenerateName(ctx, AppGatewayManager, userCred, ext.GetName())
		if err != nil {
			return errors.Wrapf(err, "db.GenerateName")
		}

		return AppGatewayManager.TableSpec().Insert(ctx, app)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	syncMetadata(ctx, userCred, app, ext)
	SyncCloudDomain(userCred, app, provider.GetOwnerId())

	return app, nil
}

func (self *SAppGateway) AllowGetDetailsBackends(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowGetSpec(userCred, self, "backends")
}

func (self *SAppGateway) GetDetailsBackends(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (cloudprovider.SAppGatewayBackends, error) {
	ret := cloudprovider.SAppGatewayBackends{}
	iApp, err := self.GetICloudAppGateway()
	if err != nil {
		return ret, errors.Wrapf(err, "GetICloudAppGateway")
	}
	ret.Data, err = iApp.GetBackends()
	if err != nil {
		return ret, errors.Wrapf(err, "GetBackends")
	}
	ret.Total = len(ret.Data)
	return ret, nil
}

func (self *SAppGateway) AllowGetDetailsFrontends(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowGetSpec(userCred, self, "frontends")
}

func (self *SAppGateway) GetDetailsFrontends(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (cloudprovider.SAppGatewayFrontends, error) {
	ret := cloudprovider.SAppGatewayFrontends{}
	iApp, err := self.GetICloudAppGateway()
	if err != nil {
		return ret, errors.Wrapf(err, "GetICloudAppGateway")
	}
	ret.Data, err = iApp.GetFrontends()
	if err != nil {
		return ret, errors.Wrapf(err, "GetFrontends")
	}
	ret.Total = len(ret.Data)
	return ret, nil
}
