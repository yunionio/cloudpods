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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SSnapshotPolicyCacheManager struct {
	db.SResourceBaseManager
}

type SSnapshotPolicyCache struct {
	db.SResourceBase
	SManagedResourceBase

	Id               string `width:"128" charset:"ascii" primary:"true" list:"user"`
	SnapshotpolicyId string `width:"128" charset:"ascii" create:"required"`
	CloudregionId    string `width:"128" charset:"ascii" create:"required"`
	ExternalId       string `width:"256" charset:"utf8" index:"true" list:"admin" create:"admin_optional"`
}

var SnapshotPolicyCacheManager *SSnapshotPolicyCacheManager

func init() {
	SnapshotPolicyCacheManager = &SSnapshotPolicyCacheManager{
		db.NewResourceBaseManager(
			SSnapshotPolicyCache{},
			"snapshotpolicycache_tbl",
			"snapshotpolicycache",
			"snapshotpolicycaches",
		),
	}
	SnapshotPolicyCacheManager.SetVirtualObject(SnapshotPolicyCacheManager)
}

func (spc *SSnapshotPolicyCache) BeforeInsert() {
	if len(spc.Id) == 0 {
		spc.Id = stringutils.UUID4()
	}
}

func (spcm *SSnapshotPolicyCacheManager) FilterById(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	return q.Equals("id", idStr)
}

func (spcm *SSnapshotPolicyCacheManager) GetIStandaloneModelManager() db.IStandaloneModelManager {
	return spcm.GetVirtualObject().(db.IStandaloneModelManager)
}

func (spc *SSnapshotPolicyCache) GetIStandaloneModel() db.IStandaloneModel {
	return spc.GetVirtualObject().(db.IStandaloneModel)
}

func (spc *SSnapshotPolicyCache) ClearSchedDescCache() error {
	return nil
}

func (spcm *SSnapshotPolicyCacheManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := spcm.SResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	if defsecgroup, _ := query.GetString("snapshotpolicy"); len(defsecgroup) > 0 {
		secgroup, err := SecurityGroupManager.FetchByIdOrName(userCred, defsecgroup)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(SecurityGroupManager.Keyword(), defsecgroup)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		q = q.Equals("snapshotpolicy_id", secgroup.GetId())
	}
	return q, nil
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

func (spc *SSnapshotPolicyCache) SetExternalId(userCred mcclient.TokenCredential, externalId string) error {
	diff, err := db.Update(spc, func() error {
		spc.ExternalId = externalId
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogEvent(spc, db.ACT_UPDATE, diff, userCred)
	return nil
}

func (spc SSnapshotPolicyCache) GetExternalId() string {
	return spc.ExternalId
}

// =============================================== detach and delete ===================================================

func (spc *SSnapshotPolicyCache) RealDetele(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, spc)
}

// ================================================= new and regist ====================================================

func (spcm *SSnapshotPolicyCacheManager) NewCache(ctx context.Context, userCred mcclient.TokenCredential,
	snapshotPolicyId, regionId, providerId string) (*SSnapshotPolicyCache, error) {

	snapshotPolicyCache := SSnapshotPolicyCache{
		SnapshotpolicyId: snapshotPolicyId,
		CloudregionId:    regionId,
	}
	snapshotPolicyCache.ManagerId = providerId

	err := snapshotPolicyCache.CreateCloudSnapshotPolicy()
	if err != nil {
		return nil, err
	}

	// should have lock
	if err := spcm.TableSpec().Insert(&snapshotPolicyCache); err != nil {
		return nil, errors.Wrapf(err, "insert snapshotpolicycache failed")
	}
	return &snapshotPolicyCache, nil
}

func (spcm *SSnapshotPolicyCacheManager) NewCacheWithExternalId(ctx context.Context, userCred mcclient.TokenCredential,
	snapshotPolicyId, externalId, regionId, providerId string) (*SSnapshotPolicyCache, error) {

	snapshotPolicyCache := SSnapshotPolicyCache{
		SnapshotpolicyId: snapshotPolicyId,
		CloudregionId:    regionId,
		ExternalId:       externalId,
	}
	snapshotPolicyCache.ManagerId = providerId

	// should have lock
	if err := spcm.TableSpec().Insert(&snapshotPolicyCache); err != nil {
		return nil, errors.Wrapf(err, "insert snapshotpolicycache failed")
	}
	return &snapshotPolicyCache, nil
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

func (spcm *SSnapshotPolicyCacheManager) RegisterWithExternalID(ctx context.Context, userCred mcclient.TokenCredential,
	snapshotPolicyId, externalId, regionId, providerId string) (*SSnapshotPolicyCache, error) {

	snapshotPolicyCache, err := spcm.FetchSnapshotPolicyCache(snapshotPolicyId, regionId, providerId)
	// error
	if err != nil {
		return nil, err
	}

	// no cache
	if snapshotPolicyCache != nil {
		return snapshotPolicyCache, nil
	}

	return spcm.NewCacheWithExternalId(ctx, userCred, snapshotPolicyId, externalId, regionId, providerId)
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
	retChan := make(chan sOperaResult)

	for i := range spCaches {
		spc := spCaches[i]
		wm.Run(func() {
			err := spc.DeleteCloudSnapshotPolicy()
			if err != nil {
				retChan <- sOperaResult{err, spc.GetId()}
				return
			}
			err = spc.RealDetele(ctx, userCred)
			if err != nil {
				retChan <- sOperaResult{errors.Wrap(err, "delete cache in database failed"), spc.GetId()}
				return
			}
			retChan <- sOperaResult{nil, spc.GetId()}
		}, nil, func(e error) {
			retChan <- sOperaResult{e, spc.GetId()}
		})
	}

	failedRecord := make([]string, 0)
	for i := 0; i < len(spCaches); i++ {
		ret := <-retChan
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
		if err == cloudprovider.ErrNotFound {
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
