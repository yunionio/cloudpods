package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
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

	SManagedResourceBaseManager
}

var ModelartsPoolManager *SModelartsPoolManager

func init() {
	ModelartsPoolManager = &SModelartsPoolManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SModelartsPool{},
			"modelarts_pools_tbl",
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

	SDeletePreventableResourceBase

	NodeCount    int    `nullable:"false" default:"0" list:"user" create:"optional"`
	InstanceType string `width:"72" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"`
	IsTrain      bool   `default:"false" create:"optional" list:"user"`
	IsNotebook   bool   `default:"false" create:"optional" list:"user"`
	IsInfer      bool   `default:"false" create:"optional" list:"user"`
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
	q, err = man.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (man *SModelartsPoolManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.ModelartsPoolCreateInput) (
	api.ModelartsPoolCreateInput, error) {
	return input, nil
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

	for i := range rows {
		rows[i] = api.ElasticSearchDetails{
			VirtualResourceDetails: virtRows[i],
			ManagedResourceInfo:    manRows[i],
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

func (self *SModelartsPool) SyncModelartsPools(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, exts []cloudprovider.ICloudModelartsPool) compare.SyncResult {
	// 加锁防止重入
	lockman.LockRawObject(ctx, ModelartsPoolManager.KeywordPlural(), fmt.Sprintf("%s-%s", provider.Id, self.Id))
	defer lockman.ReleaseRawObject(ctx, ModelartsPoolManager.KeywordPlural(), fmt.Sprintf("%s-%s", provider.Id, self.Id))

	result := compare.SyncResult{}

	dbEss, err := self.GetIModelartsPoolById(ctx)
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SModelartsPool, 0)
	commondb := make([]SModelartsPool, 0)
	commonext := make([]cloudprovider.ICloudModelartsPool, 0)
	added := make([]cloudprovider.ICloudModelartsPool, 0)
	// 本地和云上资源列表进行比对
	err = compare.CompareSets(dbEss, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	// 删除云上没有的资源
	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemoveCloudModelartsPool(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	// 和云上资源属性进行同步
	for i := 0; i < len(commondb); i++ {
		err := commondb[i].SyncWithCloudModelartsPool(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}

	// 创建本地没有的云上资源
	for i := 0; i < len(added); i++ {
		_, err := self.newFromCloudModelartsPool(ctx, userCred, provider, added[i])
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

func (self *SModelartsPool) syncRemoveCloudModelartsPool(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.Delete(ctx, userCred)
}

func (self *SModelartsPool) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	self.StartCreateTask(ctx, userCred, "")
}

func (self *SModelartsPool) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	var err = func() error {
		params := jsonutils.NewDict()
		task, err := taskman.TaskManager.
			NewTask(ctx, "ModelartsPoolCreateTask", self, userCred, params, parentTaskId, "", nil)
		if err != nil {
			log.Errorln("this is new Task error", err)
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(nil)
	}()
	if err != nil {
		self.SetStatus(userCred, api.NAS_STATUS_CREATE_FAILED, err.Error())
		log.Errorln("this is run Task error", err)
		return err
	}
	self.SetStatus(userCred, api.NAS_STATUS_CREATING, "")
	return nil
}

func (self *SModelartsPool) StartSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return StartResourceSyncStatusTask(ctx, userCred, self, "ModelartsPoolSyncstatusTask", parentTaskId)
}

func (self *SModelartsPool) GetCloudproviderId() string {
	return self.ManagerId
}

func (self *SModelartsPool) GetIModelartsPoolById(ctx context.Context) (cloudprovider.ICloudModelartsPool, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty externalId")
	}
	iProvider, err := self.GetDriver(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "self.GetIRegion")
	}
	return iProvider.GetIModelartsPoolById(self.ExternalId)
}

func (self *SModelartsPool) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SModelartsPool) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SVirtualResourceBase.Delete(ctx, userCred)
}

// 进入删除任务
func (self *SModelartsPool) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteTask(ctx, userCred, "")
}

func (self *SModelartsPool) GetICloudModelartsPool(ctx context.Context) (cloudprovider.ICloudModelartsPool, error) {
	iProvider, err := self.GetDriver(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "self.GetDriver")
	}
	return iProvider.GetIModelartsPoolById(self.ExternalId)
}

func (self *SModelartsPool) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ModelartsPoolDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	self.SetStatus(userCred, api.Modelarts_Pool_STATUS_DELETING, "")
	task.ScheduleRun(nil)
	return nil
}

// 获取云上对应的资源
// func (self *SModelartsPool) GetIElasticSearch() (cloudprovider.ICloudElasticSearch, error) {
// 	if len(self.ExternalId) == 0 {
// 		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty externalId")
// 	}
// 	iRegion, err := self.GetIRegion()
// 	if err != nil {
// 		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "GetIRegion")
// 	}
// 	return iRegion.GetIElasticSearchById(self.ExternalId)
// }

// 同步资源属性
func (self *SModelartsPool) SyncWithCloudModelartsPool(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudModelartsPool) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.ExternalId = ext.GetGlobalId()
		self.Status = ext.GetStatus()

		self.BillingType = ext.GetBillingType()
		if self.BillingType == billing_api.BILLING_TYPE_PREPAID {
			if expiredAt := ext.GetExpiredAt(); !expiredAt.IsZero() {
				self.ExpiredAt = expiredAt
			}
			self.AutoRenew = ext.IsAutoRenew()
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}

	syncVirtualResourceMetadata(ctx, userCred, self, ext)
	if provider := self.GetCloudprovider(); provider != nil {
		SyncCloudProject(userCred, self, provider.GetOwnerId(), ext, provider.Id)
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (self *SModelartsPool) newFromCloudModelartsPool(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudModelartsPool) (*SModelartsPool, error) {
	es := SModelartsPool{}
	es.SetModelManager(ModelartsPoolManager, &es)

	es.ExternalId = ext.GetGlobalId()
	es.ManagerId = provider.Id
	es.IsEmulated = ext.IsEmulated()
	es.Status = ext.GetStatus()

	if createdAt := ext.GetCreatedAt(); !createdAt.IsZero() {
		es.CreatedAt = createdAt
	}

	es.BillingType = ext.GetBillingType()
	if es.BillingType == billing_api.BILLING_TYPE_PREPAID {
		if expired := ext.GetExpiredAt(); !expired.IsZero() {
			es.ExpiredAt = expired
		}
		es.AutoRenew = ext.IsAutoRenew()
	}

	var err error
	err = func() error {
		// 这里加锁是为了防止名称重复
		lockman.LockRawObject(ctx, ModelartsPoolManager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, ModelartsPoolManager.Keyword(), "name")

		es.Name, err = db.GenerateName(ctx, ModelartsPoolManager, provider.GetOwnerId(), ext.GetName())
		if err != nil {
			return errors.Wrapf(err, "db.GenerateName")
		}
		return ModelartsPoolManager.TableSpec().Insert(ctx, &es)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "newFromCloudElasticSearch.Insert")
	}

	// 同步标签
	syncVirtualResourceMetadata(ctx, userCred, &es, ext)
	// 同步项目归属
	SyncCloudProject(userCred, &es, provider.GetOwnerId(), ext, provider.Id)

	db.OpsLog.LogEvent(&es, db.ACT_CREATE, es.GetShortDesc(ctx), userCred)

	return &es, nil
}
