package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"
)

type SModelartsPoolManager struct {
	// 由于资源是用户资源，因此定义为Virtual资源
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SDeletePreventableResourceBaseManager

	SCloudregionResourceBaseManager
	SManagedResourceBaseManager
}

var ModelartsPoolManager *SModelartsPoolManager

func init() {
	ModelartsPoolManager = &SModelartsPoolManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SModelartsPool{},
			"modelarts_pool_tbl",
			"modelarts_pool",
			"modelarts_pools",
		),
	}
	ModelartsPoolManager.SetVirtualObject(ModelartsPoolManager)
}

type SModelartsPool struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase
	SBillingResourceBase
	db.SStatusInfrasResourceBase

	SCloudregionResourceBase
	SDeletePreventableResourceBase

	// 备注
	// Description string `width:"256" charset:"utf8" nullable:"true" list:"user" update:"user" create:"optional"`

	// 状态
	State        string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`
	RegionName   string `width:"36" charset:"utf8" nullable:"true" list:"user" update:"user" create:"optional"`
	WorkingCount int    `nullable:"false" default:"0" list:"user" create:"optional"`
	// InstanceType SPoolInstanceType `json:"instance_type"`

	// Memory         int
	// GraphicsMemory string
	// SpecCode       string
	// SpecName       string
	// GpuType        string
	// GpuMemoryUnit  string
	// GpuNum         int
	// Npu            Npu

	NodeCount int `nullable:"false" default:"0" list:"user" create:"optional"`
	// NodeMetrics NodeMetrics `json:"node_metrics"`
	OrderId  string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`
	PoolId   string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`
	PoolName string `width:"36" charset:"utf8" nullable:"true" list:"user" update:"user" create:"optional"`
	PoolType string `width:"36" charset:"utf8" nullable:"true" list:"user" update:"user" create:"optional"`
	SpecCode string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`
}

type SPoolInstanceType struct {
	Memory         int
	GraphicsMemory string
	SpecCode       string
	SpecName       string
	GpuType        string
	GpuMemoryUnit  string
	GpuNum         int
	Npu            Npu
}

type Npu struct {
	Info        string
	Memory      int
	ProductName string
	Unit        string
	UnitNum     int
}

type NodeMetrics struct {
	AbnormalCount int
	CreatingCount int
	DeletingCount int
	RunningCount  int
}

func (manager *SModelartsPoolManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudregionManager},
	}
}

// Pool实例列表
func (man *SModelartsPoolManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ElasticSearchListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = man.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = man.SDeletePreventableResourceBaseManager.ListItemFilter(ctx, q, userCred, query.DeletePreventableResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDeletePreventableResourceBaseManager.ListItemFilter")
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

func (man *SModelartsPoolManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ElasticSearchListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (man *SModelartsPoolManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (man *SModelartsPoolManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.ElasticSearchCreateInput) (api.ElasticSearchCreateInput, error) {
	return input, httperrors.NewNotImplementedError("Not Implemented")
}

func (manager *SModelartsPoolManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ElasticSearchDetails {
	rows := make([]api.ElasticSearchDetails, len(objs))
	virtRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.ElasticSearchDetails{
			VirtualResourceDetails:  virtRows[i],
			ManagedResourceInfo:     manRows[i],
			CloudregionResourceInfo: regRows[i],
		}
	}

	return rows
}
func (manager *SModelartsPoolManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
	}

	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}

	if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}
func (self *SCloudregion) GetPools(managerId string) ([]SElasticSearch, error) {
	q := ElasticSearchManager.Query().Equals("cloudregion_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	ret := []SElasticSearch{}
	err := db.FetchModelObjects(ElasticSearchManager, q, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return ret, nil
}

func (self *SCloudregion) SyncPools(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, exts []cloudprovider.ICloudElasticSearch) compare.SyncResult {
	// 加锁防止重入
	lockman.LockRawObject(ctx, ElasticSearchManager.KeywordPlural(), fmt.Sprintf("%s-%s", provider.Id, self.Id))
	defer lockman.ReleaseRawObject(ctx, ElasticSearchManager.KeywordPlural(), fmt.Sprintf("%s-%s", provider.Id, self.Id))

	result := compare.SyncResult{}

	dbEss, err := self.GetElasticSearchs(provider.Id)
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SElasticSearch, 0)
	commondb := make([]SElasticSearch, 0)
	commonext := make([]cloudprovider.ICloudElasticSearch, 0)
	added := make([]cloudprovider.ICloudElasticSearch, 0)
	// 本地和云上资源列表进行比对
	err = compare.CompareSets(dbEss, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	// 删除云上没有的资源
	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemoveCloudElasticSearch(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	// 和云上资源属性进行同步
	for i := 0; i < len(commondb); i++ {
		err := commondb[i].SyncWithCloudElasticSearch(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}

	// 创建本地没有的云上资源
	for i := 0; i < len(added); i++ {
		_, err := self.newFromCloudElasticSearch(ctx, userCred, provider, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}

// 判断资源是否可以删除
func (self *SModelartsPool) ValidateDeleteCondition(ctx context.Context) error {
	if self.DisableDelete.IsTrue() {
		return httperrors.NewInvalidStatusError("ElasticSearch is locked, cannot delete")
	}
	return self.SStatusStandaloneResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SModelartsPool) syncRemoveCloudElasticSearch(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.Delete(ctx, userCred)
}
func (self *SModelartsPool) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SStatusInfrasResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	self.StartCreateTask(ctx, userCred, jsonutils.GetAnyString(data, []string{"network_id"}), "")
}

func (self *SModelartsPool) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential, networkId string, parentTaskId string) error {
	var err = func() error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(networkId), "network_id")
		task, err := taskman.TaskManager.NewTask(ctx, "FileSystemCreateTask", self, userCred, params, parentTaskId, "", nil)
		if err != nil {
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(nil)
	}()
	if err != nil {
		self.SetStatus(userCred, api.NAS_STATUS_CREATE_FAILED, err.Error())
		return err
	}
	self.SetStatus(userCred, api.NAS_STATUS_CREATING, "")
	return nil
}
