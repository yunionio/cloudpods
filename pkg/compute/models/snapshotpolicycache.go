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
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SSnapshotPolicyCacheManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SCloudregionResourceBaseManager
	SManagedResourceBaseManager
	SSnapshotPolicyResourceBaseManager
}

type SSnapshotPolicyCache struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SCloudregionResourceBase
	SManagedResourceBase
	SSnapshotPolicyResourceBase `width:"128" charset:"ascii" create:"required"`
	// SnapshotpolicyId string `width:"128" charset:"ascii" create:"required"`
}

var SnapshotPolicyCacheManager *SSnapshotPolicyCacheManager

func init() {
	SnapshotPolicyCacheManager = &SSnapshotPolicyCacheManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SSnapshotPolicyCache{},
			"snapshotpolicycache_tbl",
			"snapshotpolicycache",
			"snapshotpolicycaches",
		),
	}
	SnapshotPolicyCacheManager.SetVirtualObject(SnapshotPolicyCacheManager)
}

func NewSSnapshotPolicyCache(snapshotpolicyId, cloudregionId, externalId string) *SSnapshotPolicyCache {
	cache := SSnapshotPolicyCache{
		// SnapshotpolicyId:          snapshotpolicyId,
		SCloudregionResourceBase: SCloudregionResourceBase{
			CloudregionId: cloudregionId,
		},
		SExternalizedResourceBase: db.SExternalizedResourceBase{
			ExternalId: externalId,
		},
	}
	cache.SnapshotpolicyId = snapshotpolicyId
	cache.SetModelManager(SnapshotPolicyCacheManager, &cache)
	return &cache
}

// 快照策略缓存列表
func (spcm *SSnapshotPolicyCacheManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.SnapshotPolicyCacheListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = spcm.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemFilter")
	}

	q, err = spcm.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = spcm.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}

	q, err = spcm.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}

	q, err = spcm.SSnapshotPolicyResourceBaseManager.ListItemFilter(ctx, q, userCred, query.SnapshotPolicyFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSnapshotPolicyResourceBaseManager.ListItemFilter")
	}
	/*if snapshotpolicyIden := query.Snapshotpolicy; len(snapshotpolicyIden) > 0 {
		snapshotpolicy, err := SnapshotPolicyManager.FetchByIdOrName(userCred, snapshotpolicyIden)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(SnapshotPolicyManager.Keyword(), snapshotpolicyIden)
			} else {
				return nil, err
			}
		}
		q = q.Equals("snapshotpolicy_id", snapshotpolicy.GetId())
	}*/
	return q, nil
}

func (spcm *SSnapshotPolicyCacheManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.SnapshotPolicyCacheListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = spcm.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = spcm.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = spcm.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}
	q, err = spcm.SSnapshotPolicyResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.SnapshotPolicyFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSnapshotPolicyResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (spcm *SSnapshotPolicyCacheManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = spcm.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = spcm.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = spcm.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = spcm.SSnapshotPolicyResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (spc *SSnapshotPolicyCache) GetIRegion() (cloudprovider.ICloudRegion, error) {
	provider, err := spc.GetDriver()
	if err != nil {
		return nil, err
	}
	if region := CloudregionManager.FetchRegionById(spc.CloudregionId); region != nil {
		return provider.GetIRegionById(region.ExternalId)
	}
	return nil, fmt.Errorf("failed to find iregion for snapshotpolicycache %s: cloudregion %s manager %s", spc.Id,
		spc.CloudregionId, spc.ManagerId)
}

func (spc *SSnapshotPolicyCache) GetSnapshotPolicy() (*SSnapshotPolicy, error) {
	model, err := SnapshotPolicyManager.FetchById(spc.SnapshotpolicyId)
	if err != nil {
		return nil, fmt.Errorf("failed to fetchsnapshotpolicy by %s", spc.SnapshotpolicyId)
	}
	return model.(*SSnapshotPolicy), nil
}

func (spcm *SSnapshotPolicyCacheManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.SnapshotPolicyCacheDetails {
	rows := make([]api.SnapshotPolicyCacheDetails, len(objs))

	stdRows := spcm.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := spcm.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := spcm.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	snapshotPolicyRows := spcm.SSnapshotPolicyResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.SnapshotPolicyCacheDetails{
			StatusStandaloneResourceDetails: stdRows[i],
			ManagedResourceInfo:             manRows[i],
			CloudregionResourceInfo:         regionRows[i],
			SnapshotPolicyResourceInfo:      snapshotPolicyRows[i],
		}
	}

	return rows
}

func (spc *SSnapshotPolicyCache) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.SnapshotPolicyCacheDetails, error) {
	return api.SnapshotPolicyCacheDetails{}, nil
}

// =============================================== detach and delete ===================================================

func (spc *SSnapshotPolicyCache) RealDetele(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, spc)
}

func (spc *SSnapshotPolicyCache) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (spc *SSnapshotPolicyCache) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	spc.SetStatus(userCred, api.SNAPSHOT_POLICY_CACHE_STATUS_DELETING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "SnapshotPolicyCacheDeleteTask", spc, userCred, nil, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

// ================================================= new and regist ====================================================

func (spcm *SSnapshotPolicyCacheManager) NewCache(ctx context.Context, userCred mcclient.TokenCredential,
	snapshotPolicyId, regionId, providerId string) (*SSnapshotPolicyCache, error) {

	snapshotPolicyCache := NewSSnapshotPolicyCache(snapshotPolicyId, regionId, "")
	snapshotPolicyCache.ManagerId = providerId

	err := snapshotPolicyCache.CreateCloudSnapshotPolicy()
	if err != nil {
		return nil, err
	}
	snapshotPolicyCache.Status = api.SNAPSHOT_POLICY_CACHE_STATUS_READY

	// should have lock
	if err := spcm.TableSpec().Insert(ctx, snapshotPolicyCache); err != nil {
		return nil, errors.Wrapf(err, "insert snapshotpolicycache failed")
	}
	return snapshotPolicyCache, nil
}

func (spcm *SSnapshotPolicyCacheManager) NewCacheWithExternalId(ctx context.Context, userCred mcclient.TokenCredential,
	snapshotPolicyId, externalId, regionId, providerId string, name string) (*SSnapshotPolicyCache, error) {

	snapshotPolicyCache := NewSSnapshotPolicyCache(snapshotPolicyId, regionId, externalId)
	snapshotPolicyCache.ManagerId = providerId

	snapshotPolicyCache.Status = api.SNAPSHOT_POLICY_CACHE_STATUS_READY
	snapshotPolicyCache.Name = name
	// should have lock
	if err := spcm.TableSpec().Insert(ctx, snapshotPolicyCache); err != nil {
		return nil, errors.Wrapf(err, "insert snapshotpolicycache failed")
	}
	return snapshotPolicyCache, nil
}

func (spcm *SSnapshotPolicyCacheManager) Register(ctx context.Context, userCred mcclient.TokenCredential, snapshotPolicyId,
	regionId string, providerId string) (*SSnapshotPolicyCache, error) {

	// Many request about same snapshot policy with same region and provider coming will cause many same cache
	// building without lock, so we must Lock.
	lockman.LockRawObject(ctx, snapshotPolicyId, regionId+providerId)
	defer lockman.ReleaseRawObject(ctx, snapshotPolicyId, regionId+providerId)
	snapshotPolicyCache, err := spcm.FetchSnapshotPolicyCache(snapshotPolicyId, regionId, providerId)
	// error
	if err != nil {
		return nil, err
	}

	// no cache
	if snapshotPolicyCache != nil {
		return snapshotPolicyCache, nil
	}

	return spcm.NewCache(ctx, userCred, snapshotPolicyId, regionId, providerId)
}

// ==================================================== fetch =========================================================

func (spcm *SSnapshotPolicyCacheManager) FetchSnapshotpolicyCaheById(cacheId string) (*SSnapshotPolicyCache, error) {
	q := spcm.FilterById(spcm.Query(), cacheId)
	return spcm.fetchByQuery(q)
}

func (spcm *SSnapshotPolicyCacheManager) FetchSnapshotPolicyCache(snapshotPolicyId, regionId, providerId string) (*SSnapshotPolicyCache, error) {

	q := spcm.Query()
	q.Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("snapshotpolicy_id"), snapshotPolicyId),
		sqlchemy.Equals(q.Field("cloudregion_id"), regionId),
		sqlchemy.Equals(q.Field("manager_id"), providerId)))

	return spcm.fetchByQuery(q)
}

func (spcm *SSnapshotPolicyCacheManager) FetchSnapshotPolicyCacheByExtId(externalId, regionId,
	providerId string) (*SSnapshotPolicyCache, error) {

	q := spcm.Query()
	q.Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("external_id"), externalId),
		sqlchemy.Equals(q.Field("cloudregion_id"), regionId),
		sqlchemy.Equals(q.Field("manager_id"), providerId)))

	return spcm.fetchByQuery(q)
}

func (spcm *SSnapshotPolicyCacheManager) FetchAllByExtIds(externalIds []string, regionId,
	providerId string) ([]SSnapshotPolicyCache, error) {

	q := spcm.Query().In("external_id", externalIds).Equals("cloudregion_id", regionId).Equals("manager_id", providerId)
	return spcm.fetchAllByQuery(q)
}

func (spcm *SSnapshotPolicyCacheManager) FetchAllBySnpId(snapshotPolicyId string) ([]SSnapshotPolicyCache, error) {

	q := spcm.Query().Equals("snapshotpolicy_id", snapshotPolicyId)
	return spcm.fetchAllByQuery(q)
}

func (spcm *SSnapshotPolicyCacheManager) FetchAllByRegionProvider(cloudregionId,
	managerId string) ([]SSnapshotPolicyCache, error) {

	q := spcm.Query().Equals("cloudregion_id", cloudregionId).Equals("manager_id", managerId)
	return spcm.fetchAllByQuery(q)

}

func (spcm *SSnapshotPolicyCacheManager) fetchAllByQuery(q *sqlchemy.SQuery) ([]SSnapshotPolicyCache, error) {
	caches := make([]SSnapshotPolicyCache, 0, 1)
	if err := db.FetchModelObjects(spcm, q, &caches); err != nil {
		return nil, err
	}
	return caches, nil
}

func (spcm *SSnapshotPolicyCacheManager) fetchByQuery(q *sqlchemy.SQuery) (*SSnapshotPolicyCache, error) {
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, nil
	}
	snapshotPolicyCache := SSnapshotPolicyCache{}
	// if exist, only one
	q.First(&snapshotPolicyCache)
	snapshotPolicyCache.SetModelManager(spcm, &snapshotPolicyCache)
	return &snapshotPolicyCache, nil
}

// ============================================== cloud operation ======================================================
// This function should call in the task.

type sOperaResult struct {
	err              error
	snapshotPolicyId string
}

func (spcm *SSnapshotPolicyCacheManager) UpdateCloudSnapshotPolicy(snapshotPolicyId string,
	input *cloudprovider.SnapshotPolicyInput) error {
	//todo maybe

	return fmt.Errorf("Not implement")
}

type snapshotPolicyTask struct {
	ctx      context.Context
	userCred mcclient.TokenCredential
	spc      SSnapshotPolicyCache
	retChan  chan sOperaResult
}

func (t *snapshotPolicyTask) Run() {
	err := t.spc.DeleteCloudSnapshotPolicy()
	if err != nil {
		t.retChan <- sOperaResult{err, t.spc.GetId()}
		return
	}
	err = t.spc.RealDetele(t.ctx, t.userCred)
	if err != nil {
		t.retChan <- sOperaResult{errors.Wrap(err, "delete cache in database failed"), t.spc.GetId()}
		return
	}
	t.retChan <- sOperaResult{nil, t.spc.GetId()}

}

func (t *snapshotPolicyTask) Dump() string {
	return ""
}

func (spcm *SSnapshotPolicyCacheManager) DeleteCloudSnapshotPolices(ctx context.Context,
	userCred mcclient.TokenCredential, snapshotPolicyId string) error {

	spCaches, err := spcm.FetchAllBySnpId(snapshotPolicyId)
	if err != nil {
		return errors.Wrapf(err, "fetch all snapshotPolicyCaches ofsnapshotpolicy %s failed", snapshotPolicyId)
	}

	if len(spCaches) == 0 {
		return nil
	}

	wm := appsrv.NewWorkerManager("delete-cloud-snapshotpolices", len(spCaches), 1, false)

	task := &snapshotPolicyTask{
		ctx:      ctx,
		userCred: userCred,
		retChan:  make(chan sOperaResult),
	}

	for i := range spCaches {
		task.spc = spCaches[i]
		wm.Run(task, nil, func(e error) {
			task.retChan <- sOperaResult{e, task.spc.GetId()}
		})
	}

	failedRecord := make([]string, 0)
	for i := 0; i < len(spCaches); i++ {
		ret := <-task.retChan
		if ret.err != nil {
			failedRecord = append(failedRecord, fmt.Sprintf("%s failed because that %s", ret.snapshotPolicyId,
				ret.err.Error()))
		}
	}

	if len(failedRecord) != 0 {
		return fmt.Errorf("delete: " + strings.Join(failedRecord, "; "))
	}
	return nil
}

func (spc *SSnapshotPolicyCache) CreateCloudSnapshotPolicy() error {
	// create correspondingsnapshotpolicy in cloud
	iregion, err := spc.GetIRegion()
	if err != nil {
		return err
	}
	snapshotPolicy, err := spc.GetSnapshotPolicy()
	if err != nil {
		return err
	}

	externalId, err := iregion.CreateSnapshotPolicy(snapshotPolicy.GenerateCreateSpParams())
	if err != nil {
		return errors.Wrap(err, "createsnapshotpolicy failed")
	}
	spc.ExternalId = externalId
	spc.Name = snapshotPolicy.Name

	iPolicy, err := iregion.GetISnapshotPolicyById(externalId)
	if err != nil {
		return err
	}
	err = cloudprovider.WaitStatus(iPolicy, api.SNAPSHOT_POLICY_READY, 10*time.Second, 300*time.Second)
	if err != nil {
		return err
	}
	return nil
}

func (spc *SSnapshotPolicyCache) DeleteCloudSnapshotPolicy() error {
	if len(spc.ExternalId) > 0 {
		iregion, err := spc.GetIRegion()
		if err != nil {
			return err
		}
		cloudSp, err := iregion.GetISnapshotPolicyById(spc.ExternalId)
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return nil
		}
		if err != nil {
			return errors.Wrap(err, "fetch snapshotpolicy from cloud before deleting failed")
		}
		if cloudSp == nil {
			return nil
		}
		return iregion.DeleteSnapshotPolicy(spc.ExternalId)
	}
	return nil
}

func (spc *SSnapshotPolicyCache) UpdateCloudSnapshotPolicy(input *cloudprovider.SnapshotPolicyInput) error {
	iregion, err := spc.GetIRegion()
	if err != nil {
		return err
	}

	err = iregion.UpdateSnapshotPolicy(input, spc.ExternalId)
	if err != nil {
		return errors.Wrap(err, "createsnapshotpolicy failed")
	}

	return nil
}

func (manager *SSnapshotPolicyCacheManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
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

func (manager *SSnapshotPolicyCacheManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeProject
}

func (spc *SSnapshotPolicyCache) GetOwnerId() mcclient.IIdentityProvider {
	p, err := spc.GetSnapshotPolicy()
	if err != nil {
		log.Errorf("unable to get snapshotpolicy of snapshotpolicycache %s: %v", spc.GetId(), err)
		return nil
	}
	return p.GetOwnerId()
}
