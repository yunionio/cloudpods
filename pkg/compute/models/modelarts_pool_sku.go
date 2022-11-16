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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SModelartsPoolSkuManager struct {
	db.SExternalizedResourceBaseManager
	db.SEnabledStatusStandaloneResourceBaseManager

	SCloudregionResourceBaseManager
	SManagedResourceBaseManager
}

var ModelartsPoolSkuManager *SModelartsPoolSkuManager

func init() {
	ModelartsPoolSkuManager = &SModelartsPoolSkuManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SModelartsPoolSku{},
			"modelarts_pool_skus_tbl",
			"modelarts_pool_sku",
			"modelarts_pool_skus",
		),
	}
	ModelartsPoolSkuManager.NameRequireAscii = false
	ModelartsPoolSkuManager.SetVirtualObject(ModelartsPoolSkuManager)
}

type SModelartsPoolSku struct {
	SManagedResourceBase
	db.SEnabledStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SCloudregionResourceBase

	Type string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"admin_optional" update:"admin"` // 资源规格类型
	// CPU 架构 x86|xarm
	CpuArch string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"admin_optional" update:"admin"`
	//CPU核心数量
	CpuCount int `list:"user" create:"admin_optional" update:"admin"`
	// GPU卡类型
	GpuType string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"admin_optional" update:"admin"`
	// GPU卡数量
	GpuSize int `nullable:"true" list:"user" create:"admin_optional" update:"admin"`
	// NPU卡类型
	NpuType string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"admin_optional" update:"admin"`
	// NPU卡数量
	NpuSize int `nullable:"true" list:"user" create:"admin_optional" update:"admin"`
	// 内存
	Memory int `nullable:"true" list:"user" create:"admin_optional" update:"admin"`
}

func (manager *SModelartsPoolSkuManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{}
}

func (man *SModelartsPoolSkuManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ModelartsPoolSkuListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = man.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.ListItemFilter")
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

	if len(query.CpuArch) > 0 {
		q = q.Equals("cpu_arch", query.CpuArch)
	}
	if len(query.ProcessorType) > 0 {
		q = q.Equals("processor_type", query.ProcessorType)
	}

	return q, nil
}

func (man *SModelartsPoolSkuManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ModelartsPoolSkuListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SEnabledStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
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

func (man *SModelartsPoolSkuManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := man.SEnabledStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SModelartsPoolSkuManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ModelartsPoolSkuDetails {
	rows := make([]api.ModelartsPoolSkuDetails, len(objs))
	enabledRows := manager.SEnabledStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.ModelartsPoolSkuDetails{
			EnabledStatusStandaloneResourceDetails: enabledRows[i],
			ManagedResourceInfo:                    manRows[i],
			CloudregionResourceInfo:                regRows[i],
		}
	}

	return rows
}
func (manager *SModelartsPoolSkuManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
	}

	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}
func (self *SCloudregion) GetModelartsPoolSkus(managerId string) ([]SModelartsPoolSku, error) {
	q := ModelartsPoolSkuManager.Query().Equals("cloudregion_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	ret := []SModelartsPoolSku{}
	err := db.FetchModelObjects(ModelartsPoolSkuManager, q, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return ret, nil
}

func (self *SCloudregion) SyncModelartsPoolSkus(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, exts []cloudprovider.ICloudModelartsPoolSku) compare.SyncResult {
	// 加锁防止重入
	lockman.LockRawObject(ctx, self.Provider, "modelarts-pool-sku")
	defer lockman.ReleaseRawObject(ctx, self.Provider, "modelarts-pool-sku")
	result := compare.SyncResult{}
	dbPoolSku, err := self.GetModelartsPoolSkus(provider.Id)
	if err != nil {
		result.Error(err)
		return result
	}
	removed := make([]SModelartsPoolSku, 0)
	commondb := make([]SModelartsPoolSku, 0)
	commonext := make([]cloudprovider.ICloudModelartsPoolSku, 0)
	added := make([]cloudprovider.ICloudModelartsPoolSku, 0)
	// 本地和云上资源列表进行比对
	err = compare.CompareSets(dbPoolSku, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	// 删除云上没有的资源
	for i := 0; i < len(removed); i++ {
		err := removed[i].Delete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	// 和云上资源属性进行同步
	for i := 0; i < len(commondb); i++ {
		log.Errorln("this is commonext", commonext[i])
		err := commondb[i].syncWithCloudSku(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}

	// 创建本地没有的云上资源
	for i := 0; i < len(added); i++ {
		err := self.newFromCloudModelartsPoolSku(ctx, userCred, provider, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}

func (self *SModelartsPoolSku) syncWithCloudSku(ctx context.Context, userCred mcclient.TokenCredential, isku cloudprovider.ICloudModelartsPoolSku) error {
	_, err := db.Update(self, func() error {
		self.Name = isku.GetName()
		self.CpuCount = isku.GetCpuCoreCount()
		self.CpuArch = isku.GetCpuArch()
		self.Status = isku.GetStatus()
		self.Type = isku.GetPoolType()
		self.CreatedAt = isku.GetCreatedAt()
		self.GpuType = isku.GetGpuType()
		self.GpuSize = isku.GetGpuSize()
		self.Memory = isku.GetMemorySizeMB()
		self.NpuType = isku.GetNpuType()
		self.NpuSize = isku.GetNpuSize()
		self.ExternalId = isku.GetGlobalId()
		return nil
	})
	return err
}

func (self *SCloudregion) newFromCloudModelartsPoolSku(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, isku cloudprovider.ICloudModelartsPoolSku) error {
	sku := SModelartsPoolSku{}
	sku.SetModelManager(ModelartsPoolSkuManager, &sku)
	sku.CloudregionId = self.Id
	sku.Name = isku.GetName()
	sku.CpuCount = isku.GetCpuCoreCount()
	sku.CpuArch = isku.GetCpuArch()
	sku.Status = isku.GetStatus()
	sku.Type = isku.GetPoolType()
	sku.CreatedAt = isku.GetCreatedAt()
	sku.GpuType = isku.GetGpuType()
	sku.GpuSize = isku.GetGpuSize()
	sku.Memory = isku.GetMemorySizeMB()
	sku.NpuType = isku.GetNpuType()
	sku.NpuSize = isku.GetNpuSize()
	sku.ExternalId = isku.GetGlobalId()
	sku.ManagerId = provider.Id
	return ModelartsPoolSkuManager.TableSpec().Insert(ctx, &sku)
}
