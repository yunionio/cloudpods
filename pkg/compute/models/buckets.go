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
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/pkg/s3utils"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	identity_apis "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SBucketManager struct {
	db.SSharableVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SCloudregionResourceBaseManager
	SManagedResourceBaseManager
}

var BucketManager *SBucketManager

func init() {
	BucketManager = &SBucketManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SBucket{},
			"buckets_tbl",
			"bucket",
			"buckets",
		),
	}
	BucketManager.SetVirtualObject(BucketManager)
}

type SBucket struct {
	db.SSharableVirtualResourceBase
	db.SExternalizedResourceBase
	SCloudregionResourceBase `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
	SManagedResourceBase

	// CloudregionId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`

	StorageClass string `width:"36" charset:"ascii" nullable:"false" list:"user"`
	Location     string `width:"36" charset:"ascii" nullable:"false" list:"user"`
	Acl          string `width:"36" charset:"ascii" nullable:"false" list:"user"`

	SizeBytes int64 `nullable:"false" default:"0" list:"user"`
	ObjectCnt int   `nullable:"false" default:"0" list:"user"`

	SizeBytesLimit int64 `nullable:"false" default:"0" list:"user"`
	ObjectCntLimit int   `nullable:"false" default:"0" list:"user"`

	AccessUrls jsonutils.JSONObject `nullable:"true" list:"user"`
}

func (manager *SBucketManager) SetHandlerProcessTimeout(info *appsrv.SHandlerInfo, r *http.Request) time.Duration {
	if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/upload") && r.Header.Get(api.BUCKET_UPLOAD_OBJECT_KEY_HEADER) != "" {
		log.Debugf("upload object, set process timeout to 2 hour!!!")
		return 2 * time.Hour
	}
	return manager.SSharableVirtualResourceBaseManager.SetHandlerProcessTimeout(info, r)
}

func (manager *SBucketManager) fetchBuckets(provider *SCloudprovider, region *SCloudregion) ([]SBucket, error) {
	q := manager.Query()
	if provider != nil {
		q = q.Equals("manager_id", provider.GetId())
	}
	if region != nil {
		q = q.Equals("cloudregion_id", region.GetId())
	}
	buckets := make([]SBucket, 0)
	err := db.FetchModelObjects(manager, q, &buckets)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return buckets, nil
}

func (manager *SBucketManager) syncBuckets(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, buckets []cloudprovider.ICloudBucket) compare.SyncResult {
	lockman.LockRawObject(ctx, "buckets", fmt.Sprintf("%s-%s", provider.Id, region.Id))
	defer lockman.ReleaseRawObject(ctx, "buckets", fmt.Sprintf("%s-%s", provider.Id, region.Id))

	syncResult := compare.SyncResult{}

	dbBuckets, err := manager.fetchBuckets(provider, region)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := make([]SBucket, 0)
	commondb := make([]SBucket, 0)
	commonext := make([]cloudprovider.ICloudBucket, 0)
	added := make([]cloudprovider.ICloudBucket, 0)

	err = compare.CompareSets(dbBuckets, buckets, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemoveCloudBucket(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].syncWithCloudBucket(ctx, userCred, commonext[i], provider, false)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		_, err := manager.newFromCloudBucket(ctx, userCred, added[i], provider, region)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncResult.Add()
		}
	}

	return syncResult
}

func (manager *SBucketManager) newFromCloudBucket(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	extBucket cloudprovider.ICloudBucket,
	provider *SCloudprovider,
	region *SCloudregion,
) (*SBucket, error) {

	bucket := SBucket{}
	bucket.SetModelManager(manager, &bucket)

	bucket.ExternalId = extBucket.GetGlobalId()
	bucket.ManagerId = provider.Id
	bucket.CloudregionId = region.Id
	bucket.Status = api.BUCKET_STATUS_READY

	created := extBucket.GetCreateAt()
	if !created.IsZero() {
		bucket.CreatedAt = created
	}

	bucket.Location = extBucket.GetLocation()
	bucket.StorageClass = extBucket.GetStorageClass()
	bucket.Acl = string(extBucket.GetAcl())

	stats := extBucket.GetStats()
	bucket.SizeBytes = stats.SizeBytes
	if bucket.SizeBytes < 0 {
		bucket.SizeBytes = 0
	}
	bucket.ObjectCnt = stats.ObjectCount
	if bucket.ObjectCnt < 0 {
		bucket.ObjectCnt = 0
	}

	limit := extBucket.GetLimit()
	limitSupport := extBucket.LimitSupport()
	if limitSupport.SizeBytes > 0 {
		bucket.SizeBytesLimit = limit.SizeBytes
	}
	if limitSupport.ObjectCount > 0 {
		bucket.ObjectCntLimit = limit.ObjectCount
	}

	bucket.AccessUrls = jsonutils.Marshal(extBucket.GetAccessUrls())

	bucket.IsEmulated = false

	var err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		var err error
		bucket.Name, err = db.GenerateName(ctx, manager, nil, extBucket.GetName())
		if err != nil {
			return errors.Wrap(err, "db.GenerateName")
		}

		return manager.TableSpec().Insert(ctx, &bucket)
	}()
	if err != nil {
		return nil, err
	}

	SyncCloudProject(userCred, &bucket, provider.GetOwnerId(), extBucket, provider.Id)
	bucket.SyncShareState(ctx, userCred, provider.getAccountShareInfo())

	syncVirtualResourceMetadata(ctx, userCred, &bucket, extBucket)
	db.OpsLog.LogEvent(&bucket, db.ACT_CREATE, bucket.GetShortDesc(ctx), userCred)

	return &bucket, nil
}

func (bucket *SBucket) getStats() cloudprovider.SBucketStats {
	return cloudprovider.SBucketStats{
		SizeBytes:   bucket.SizeBytes,
		ObjectCount: bucket.ObjectCnt,
	}
}

func (bucket *SBucket) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := bucket.SSharableVirtualResourceBase.GetShortDesc(ctx)

	desc.Add(jsonutils.NewInt(bucket.SizeBytes), "size_bytes")
	desc.Add(jsonutils.NewInt(int64(bucket.ObjectCnt)), "object_cnt")
	desc.Add(jsonutils.NewString(bucket.Acl), "acl")
	desc.Add(jsonutils.NewString(bucket.StorageClass), "storage_class")

	info := bucket.getCloudProviderInfo()
	desc.Update(jsonutils.Marshal(&info))

	return desc
}

func (bucket *SBucket) syncWithCloudBucket(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	extBucket cloudprovider.ICloudBucket,
	provider *SCloudprovider,
	statsOnly bool,
) error {
	oStats := bucket.getStats()
	diff, err := db.UpdateWithLock(ctx, bucket, func() error {
		stats := extBucket.GetStats()
		bucket.SizeBytes = stats.SizeBytes
		if bucket.SizeBytes < 0 {
			bucket.SizeBytes = 0
		}
		bucket.ObjectCnt = stats.ObjectCount
		if bucket.ObjectCnt < 0 {
			bucket.ObjectCnt = 0
		}

		if !statsOnly {
			limit := extBucket.GetLimit()
			limitSupport := extBucket.LimitSupport()
			if limitSupport.SizeBytes > 0 {
				bucket.SizeBytesLimit = limit.SizeBytes
			}
			if limitSupport.ObjectCount > 0 {
				bucket.ObjectCntLimit = limit.ObjectCount
			}

			bucket.Acl = string(extBucket.GetAcl())
			bucket.Location = extBucket.GetLocation()
			bucket.StorageClass = extBucket.GetStorageClass()

			bucket.AccessUrls = jsonutils.Marshal(extBucket.GetAccessUrls())

			bucket.Status = api.BUCKET_STATUS_READY
		}

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "db.UpdateWithLock")
	}

	syncVirtualResourceMetadata(ctx, userCred, bucket, extBucket)

	db.OpsLog.LogSyncUpdate(bucket, diff, userCred)

	if !oStats.Equals(extBucket.GetStats()) {
		db.OpsLog.LogEvent(bucket, api.BUCKET_OPS_STATS_CHANGE, bucket.GetShortDesc(ctx), userCred)
	}

	if provider != nil {
		SyncCloudProject(userCred, bucket, provider.GetOwnerId(), extBucket, provider.Id)
		bucket.SyncShareState(ctx, userCred, provider.getAccountShareInfo())
	}

	return nil
}

func (bucket *SBucket) syncRemoveCloudBucket(
	ctx context.Context,
	userCred mcclient.TokenCredential,
) error {
	lockman.LockObject(ctx, bucket)
	defer lockman.ReleaseObject(ctx, bucket)

	err := bucket.RealDelete(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "RealDelete")
	}
	return nil
}

func (bucket *SBucket) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	// override
	log.Infof("bucket delete do nothing")
	return nil
}

func (bucket *SBucket) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return bucket.SSharableVirtualResourceBase.Delete(ctx, userCred)
}

func (bucket *SBucket) RemoteDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	if len(bucket.ExternalId) > 0 {
		iregion, err := bucket.GetIRegion()
		if err != nil {
			return errors.Wrap(err, "bucket.GetIRegion")
		}
		err = iregion.DeleteIBucket(bucket.ExternalId)
		if err != nil {
			return errors.Wrap(err, "iregion.DeleteIBucket")
		}
	}
	err := bucket.RealDelete(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "bucket.RealDelete")
	}
	return nil
}

func (bucket *SBucket) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return bucket.StartBucketDeleteTask(ctx, userCred, "")
}

func (bucket *SBucket) StartBucketDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "BucketDeleteTask", bucket, userCred, params, parentTaskId, "", nil)
	if err != nil {
		log.Errorf("%s", err)
		return err
	}
	bucket.SetStatus(userCred, api.CLOUD_PROVIDER_START_DELETE, "StartBucketDeleteTask")
	task.ScheduleRun(nil)
	return nil
}

func (bucket *SBucket) GetRegion() (*SCloudregion, error) {
	region, err := CloudregionManager.FetchById(bucket.CloudregionId)
	if err != nil {
		return nil, errors.Wrap(err, "CloudregionManager.FetchById")
	}
	return region.(*SCloudregion), nil
}

func (bucket *SBucket) GetIRegion() (cloudprovider.ICloudRegion, error) {
	provider, err := bucket.GetDriver()
	if err != nil {
		return nil, err
	}
	if provider.GetFactory().IsOnPremise() {
		return provider.GetOnPremiseIRegion()
	} else {
		region, err := bucket.GetRegion()
		if err != nil {
			return nil, errors.Wrap(err, "bucket.GetRegion")
		}
		return provider.GetIRegionById(region.GetExternalId())
	}
}

func (bucket *SBucket) GetIBucket() (cloudprovider.ICloudBucket, error) {
	iregion, err := bucket.GetIRegion()
	if err != nil {
		return nil, errors.Wrap(err, "bucket.GetIRegion")
	}
	return iregion.GetIBucketById(bucket.ExternalId)
}

func isValidBucketName(name string) error {
	return s3utils.CheckValidBucketNameStrict(name)
}

func (manager *SBucketManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.BucketCreateInput,
) (api.BucketCreateInput, error) {
	var err error
	var cloudRegionV *SCloudregion
	cloudRegionV, input.CloudregionResourceInput, err = ValidateCloudregionResourceInput(userCred, input.CloudregionResourceInput)
	if err != nil {
		return input, errors.Wrap(err, "ValidateCloudregionResourceInput")
	}
	var managerV *SCloudprovider
	managerV, input.CloudproviderResourceInput, err = ValidateCloudproviderResourceInput(userCred, input.CloudproviderResourceInput)
	if err != nil {
		return input, errors.Wrap(err, "ValidateCloudproviderResourceInput")
	}

	if len(input.Name) == 0 {
		return input, httperrors.NewInputParameterError("missing name")
	}
	err = isValidBucketName(input.Name)
	if err != nil {
		return input, httperrors.NewInputParameterError("invalid bucket name %s: %s", input.Name, err)
	}

	if len(input.StorageClass) > 0 {
		driver, err := managerV.GetProvider()
		if err != nil {
			return input, errors.Wrap(err, "GetProvider")
		}
		if !utils.IsInStringArray(input.StorageClass, driver.GetStorageClasses(cloudRegionV.Id)) {
			return input, errors.Wrapf(httperrors.ErrInputParameter, "invalid storage class %s", input.StorageClass)
		}
	}

	quotaKeys := fetchRegionalQuotaKeys(rbacutils.ScopeProject, ownerId, cloudRegionV, managerV)
	pendingUsage := SRegionQuota{Bucket: 1}
	pendingUsage.SetKeys(quotaKeys)
	if err := quotas.CheckSetPendingQuota(ctx, userCred, &pendingUsage); err != nil {
		return input, httperrors.NewOutOfQuotaError("%s", err)
	}

	input.SharableVirtualResourceCreateInput, err = manager.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ValidateCreateData")
	}
	return input, nil
}

func (bucket *SBucket) GetQuotaKeys() (quotas.IQuotaKeys, error) {
	region, _ := bucket.GetRegion()
	if region == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no valid region")
	}
	return fetchRegionalQuotaKeys(
		rbacutils.ScopeProject,
		bucket.GetOwnerId(),
		region,
		bucket.GetCloudprovider(),
	), nil
}

func (bucket *SBucket) PostCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) {
	bucket.SSharableVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	pendingUsage := SRegionQuota{Bucket: 1}
	keys, err := bucket.GetQuotaKeys()
	if err != nil {
		log.Errorf("bucket.GetQuotaKeys fail %s", err)
	} else {
		pendingUsage.SetKeys(keys)
		err = quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage, true)
		if err != nil {
			log.Errorf("CancelPendingUsage error %s", err)
		}
	}

	bucket.SetStatus(userCred, api.BUCKET_STATUS_START_CREATE, "PostCreate")
	task, err := taskman.TaskManager.NewTask(ctx, "BucketCreateTask", bucket, userCred, nil, "", "", nil)
	if err != nil {
		bucket.SetStatus(userCred, api.BUCKET_STATUS_CREATE_FAIL, errors.Wrapf(err, "NewTask").Error())
		return
	}
	task.ScheduleRun(nil)
}

func (bucket *SBucket) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketUpdateInput,
) (api.BucketUpdateInput, error) {
	var err error
	if len(input.Name) > 0 {
		err := isValidBucketName(input.Name)
		if err != nil {
			return input, httperrors.NewInputParameterError("invalid bucket name(%s): %s", input.Name, err)
		}
	}
	input.SharableVirtualResourceBaseUpdateInput, err = bucket.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.SharableVirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SSharableVirtualResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (bucket *SBucket) RemoteCreate(ctx context.Context, userCred mcclient.TokenCredential) error {
	iregion, err := bucket.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "bucket.GetIRegion")
	}
	err = iregion.CreateIBucket(bucket.Name, bucket.StorageClass, bucket.Acl)
	if err != nil {
		return errors.Wrap(err, "iregion.CreateIBucket")
	}
	extBucket, err := iregion.GetIBucketByName(bucket.Name)
	if err != nil {
		return errors.Wrap(err, "iregion.GetIBucketByName")
	}
	err = db.SetExternalId(bucket, userCred, extBucket.GetGlobalId())
	if err != nil {
		return errors.Wrap(err, "db.SetExternalId")
	}
	tags, _ := bucket.GetAllUserMetadata()
	if len(tags) > 0 {
		_, err = cloudprovider.SetBucketTags(ctx, extBucket, bucket.ManagerId, tags)
		if err != nil {
			logclient.AddSimpleActionLog(bucket, logclient.ACT_UPDATE_TAGS, err, userCred, false)
		}
	}
	err = bucket.syncWithCloudBucket(ctx, userCred, extBucket, nil, false)
	if err != nil {
		return errors.Wrap(err, "bucket.syncWithCloudBucket")
	}
	return nil
}

func (bucket *SBucket) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.BucketDetails, error) {
	return api.BucketDetails{}, nil
}

func (manager *SBucketManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.BucketDetails {
	rows := make([]api.BucketDetails, len(objs))
	virtRows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.BucketDetails{
			SharableVirtualResourceDetails: virtRows[i],
			ManagedResourceInfo:            managerRows[i],
			CloudregionResourceInfo:        regionRows[i],
		}
		rows[i] = objs[i].(*SBucket).getMoreDetails(rows[i])
	}
	return rows
}

func joinPath(ep, path string) string {
	return strings.TrimRight(ep, "/") + "/" + strings.TrimLeft(path, "/")
}

func (bucket *SBucket) getMoreDetails(out api.BucketDetails) api.BucketDetails {
	s3gwUrl, _ := auth.GetServiceURL("s3gateway", options.Options.Region, "", identity_apis.EndpointInterfacePublic)

	if len(s3gwUrl) > 0 {
		accessUrls := make([]cloudprovider.SBucketAccessUrl, 0)
		if bucket.AccessUrls != nil {
			err := bucket.AccessUrls.Unmarshal(&accessUrls)
			if err != nil {
				log.Errorf("bucket.AccessUrls.Unmarshal fail %s", err)
			}
		}
		find := false
		for i := range accessUrls {
			if strings.HasPrefix(accessUrls[i].Url, s3gwUrl) {
				find = true
				break
			}
		}
		if !find {
			accessUrls = append(accessUrls, cloudprovider.SBucketAccessUrl{
				Url:         joinPath(s3gwUrl, bucket.Name),
				Description: "s3gateway",
			})
			out.AccessUrls = accessUrls
		}
	}

	return out
}

func (bucket *SBucket) getCloudProviderInfo() SCloudProviderInfo {
	region, _ := bucket.GetRegion()
	provider := bucket.GetCloudprovider()
	return MakeCloudProviderInfo(region, nil, provider)
}

// 对象存储的存储桶列表
func (manager *SBucketManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.BucketListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemFilter")
	}

	if len(query.StorageClass) > 0 {
		q = q.In("storage_class", query.StorageClass)
	}
	if len(query.Location) > 0 {
		q = q.In("location", query.Location)
	}
	if len(query.Acl) > 0 {
		q = q.In("acl", query.Acl)
	}

	return q, nil
}

func (manager *SBucketManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SSharableVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
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

func (manager *SBucketManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.BucketListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}

	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}

	q, err = manager.SSharableVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (bucket *SBucket) AllowGetDetailsObjects(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input api.BucketGetObjectsInput,
) bool {
	return bucket.IsOwner(userCred)
}

// 获取bucket的对象列表
//
// 获取bucket的对象列表
func (bucket *SBucket) GetDetailsObjects(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input api.BucketGetObjectsInput,
) (api.BucketGetObjectsOutput, error) {
	output := api.BucketGetObjectsOutput{}
	if len(bucket.ExternalId) == 0 {
		return output, httperrors.NewInvalidStatusError("no external bucket")
	}
	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return output, errors.Wrap(err, "GetIBucket")
	}
	prefix := input.Prefix
	isRecursive := false
	if input.Recursive != nil {
		isRecursive = *input.Recursive
	}
	marker := input.PagingMarker
	limit := 0
	if input.Limit != nil {
		limit = *input.Limit
	}
	if limit <= 0 {
		limit = 50
	} else if limit > 1000 {
		limit = 1000
	}
	objects, nextMarker, err := cloudprovider.GetPagedObjects(iBucket, prefix, isRecursive, marker, int(limit))
	if err != nil {
		return output, httperrors.NewInternalServerError("fail to get objects: %s", err)
	}
	for i := range objects {
		output.Data = append(output.Data, cloudprovider.ICloudObject2Struct(objects[i]))
	}
	output.MarkerField = "key"
	output.MarkerOrder = "DESC"
	if len(nextMarker) > 0 {
		output.NextMarker = nextMarker
	}
	return output, nil
}

func (bucket *SBucket) AllowPerformTempUrl(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketPerformTempUrlInput,
) bool {
	return bucket.IsOwner(userCred)
}

// 获取访问对象的临时URL
//
// 获取访问对象的临时URL
func (bucket *SBucket) PerformTempUrl(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketPerformTempUrlInput,
) (api.BucketPerformTempUrlOutput, error) {
	output := api.BucketPerformTempUrlOutput{}

	if len(bucket.ExternalId) == 0 {
		return output, httperrors.NewInvalidStatusError("no external bucket")
	}

	method := input.Method
	key := input.Key
	expire := 0
	if input.ExpireSeconds != nil {
		expire = *input.ExpireSeconds
	}

	if len(method) == 0 {
		method = "GET"
	}
	if len(key) == 0 {
		return output, httperrors.NewInputParameterError("missing key")
	}
	if expire == 0 {
		expire = 60 // default 60 seconds
	}

	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return output, errors.Wrap(err, "GetIBucket")
	}
	tmpUrl, err := iBucket.GetTempUrl(method, key, time.Duration(expire)*time.Second)
	if err != nil {
		return output, httperrors.NewInternalServerError("fail to generate temp url: %s", err)
	}
	output.Url = tmpUrl
	return output, nil
}

func (bucket *SBucket) AllowPerformMakedir(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) bool {
	return bucket.IsOwner(userCred)
}

// 新建对象目录
//
// 新建对象目录
func (bucket *SBucket) PerformMakedir(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketPerformMakedirInput,
) (jsonutils.JSONObject, error) {
	if len(bucket.ExternalId) == 0 {
		return nil, httperrors.NewInvalidStatusError("no external bucket")
	}

	key := input.Key
	key = strings.Trim(key, " /")
	if len(key) == 0 {
		return nil, httperrors.NewInputParameterError("empty directory name")
	}

	err := s3utils.CheckValidObjectName(key)
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid key %s: %s", key, err)
	}

	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return nil, errors.Wrap(err, "GetIBucket")
	}

	_, err = cloudprovider.GetIObject(iBucket, key+"/")
	if err == nil {
		// replace
		return nil, nil
	} else if err != cloudprovider.ErrNotFound {
		return nil, httperrors.NewInternalServerError("GetIObject fail %s", err)
	}

	if bucket.ObjectCntLimit > 0 && bucket.ObjectCntLimit < bucket.ObjectCnt+1 {
		return nil, httperrors.NewOutOfQuotaError("object count limit exceeds")
	}
	pendingUsage := SRegionQuota{ObjectGB: 0, ObjectCnt: 1}
	keys, err := bucket.GetQuotaKeys()
	if err != nil {
		return nil, httperrors.NewInternalServerError("bucket.GetQuotaKeys %s", err)
	}
	pendingUsage.SetKeys(keys)
	if err := quotas.CheckSetPendingQuota(ctx, userCred, &pendingUsage); err != nil {
		return nil, httperrors.NewOutOfQuotaError("%s", err)
	}

	err = cloudprovider.Makedir(ctx, iBucket, key+"/")
	if err != nil {
		return nil, httperrors.NewInternalServerError("fail to mkdir: %s", err)
	}

	db.OpsLog.LogEvent(bucket, db.ACT_MKDIR, key, userCred)
	logclient.AddActionLogWithContext(ctx, bucket, logclient.ACT_MKDIR, key, userCred, true)

	bucket.syncWithCloudBucket(ctx, userCred, iBucket, nil, true)

	quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage, true)

	return nil, nil
}

func (bucket *SBucket) AllowPerformDelete(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) bool {
	return bucket.IsOwner(userCred)
}

// 删除对象
//
// 删除对象
func (bucket *SBucket) PerformDelete(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketPerformDeleteInput,
) (jsonutils.JSONObject, error) {
	if len(bucket.ExternalId) == 0 {
		return nil, httperrors.NewInvalidStatusError("no external bucket")
	}

	keyStrs := input.Keys
	if len(keyStrs) == 0 {
		return nil, httperrors.NewInputParameterError("empty keys")
	}

	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return nil, errors.Wrap(err, "GetIBucket")
	}
	ok := jsonutils.NewDict()
	results := modulebase.BatchDo(keyStrs, func(key string) (jsonutils.JSONObject, error) {
		if strings.HasSuffix(key, "/") {
			err = cloudprovider.DeletePrefix(ctx, iBucket, key)
		} else {
			err = iBucket.DeleteObject(ctx, key)
		}
		if err != nil {
			return nil, errors.Wrap(err, "DeletePrefix")
		} else {
			return ok, nil
		}
	})

	db.OpsLog.LogEvent(bucket, db.ACT_DELETE_OBJECT, keyStrs, userCred)
	logclient.AddActionLogWithContext(ctx, bucket, logclient.ACT_DELETE_OBJECT, keyStrs, userCred, true)

	bucket.syncWithCloudBucket(ctx, userCred, iBucket, nil, true)

	return modulebase.SubmitResults2JSON(results), nil
}

func (bucket *SBucket) AllowPerformUpload(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) bool {
	return bucket.IsOwner(userCred)
}

// 上传对象
//
// 上传对象
func (bucket *SBucket) PerformUpload(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	if len(bucket.ExternalId) == 0 {
		return nil, httperrors.NewInvalidStatusError("no external bucket")
	}

	appParams := appsrv.AppContextGetParams(ctx)

	key := appParams.Request.Header.Get(api.BUCKET_UPLOAD_OBJECT_KEY_HEADER)

	if strings.HasSuffix(key, "/") {
		return nil, httperrors.NewInputParameterError("object key should not ends with /")
	}

	err := s3utils.CheckValidObjectName(key)
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid object key: %s", err)
	}

	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return nil, errors.Wrap(err, "GetIBucket")
	}

	meta := cloudprovider.FetchMetaFromHttpHeader(cloudprovider.META_HEADER_PREFIX, appParams.Request.Header)

	sizeStr := appParams.Request.Header.Get("Content-Length")
	if len(sizeStr) == 0 {
		return nil, httperrors.NewInputParameterError("missing Content-Length")
	}
	sizeBytes, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Illegal Content-Length %s", sizeStr)
	}
	if sizeBytes < 0 {
		return nil, httperrors.NewInputParameterError("Content-Length negative %d", sizeBytes)
	}
	storageClass := appParams.Request.Header.Get(api.BUCKET_UPLOAD_OBJECT_STORAGECLASS_HEADER)
	driver, err := bucket.GetDriver()
	if err != nil {
		return nil, errors.Wrap(err, "GetDriver")
	}
	if len(storageClass) > 0 && !utils.IsInStringArray(storageClass, driver.GetStorageClasses(bucket.CloudregionId)) {
		return nil, errors.Wrapf(httperrors.ErrInputParameter, "invalid storage class %s", storageClass)
	}

	aclStr := appParams.Request.Header.Get(api.BUCKET_UPLOAD_OBJECT_ACL_HEADER)
	if len(aclStr) > 0 && !utils.IsInStringArray(aclStr, driver.GetObjectCannedAcls(bucket.CloudregionId)) {
		return nil, errors.Wrapf(httperrors.ErrInputParameter, "invalid acl %s", aclStr)
	}

	inc := cloudprovider.SBucketStats{}
	obj, err := cloudprovider.GetIObject(iBucket, key)
	if err == nil {
		// replace
		inc.SizeBytes = sizeBytes - obj.GetSizeBytes()
		if inc.SizeBytes < 0 {
			inc.SizeBytes = 0
		}
	} else if errors.Cause(err) == cloudprovider.ErrNotFound {
		// new upload
		inc.SizeBytes = sizeBytes
		inc.ObjectCount = 1
	} else {
		return nil, httperrors.NewInternalServerError("GetIObject error %s", err)
	}

	if bucket.SizeBytesLimit > 0 && inc.SizeBytes > 0 && bucket.SizeBytesLimit < bucket.SizeBytes+inc.SizeBytes {
		return nil, httperrors.NewOutOfQuotaError("object size limit exceeds")
	}
	if bucket.ObjectCntLimit > 0 && inc.ObjectCount > 0 && bucket.ObjectCntLimit < bucket.ObjectCnt+inc.ObjectCount {
		return nil, httperrors.NewOutOfQuotaError("object count limit exceeds")
	}

	pendingUsage := SRegionQuota{ObjectGB: int(inc.SizeBytes / 1000 / 1000 / 1000), ObjectCnt: inc.ObjectCount}
	keys, err := bucket.GetQuotaKeys()
	if err != nil {
		return nil, httperrors.NewInternalServerError("bucket.GetQuotaKeys fail %s", err)
	}
	pendingUsage.SetKeys(keys)
	if !pendingUsage.IsEmpty() {
		if err := quotas.CheckSetPendingQuota(ctx, userCred, &pendingUsage); err != nil {
			return nil, httperrors.NewOutOfQuotaError("%s", err)
		}

	}

	err = cloudprovider.UploadObject(ctx, iBucket, key, 0, appParams.Request.Body, sizeBytes, cloudprovider.TBucketACLType(aclStr), storageClass, meta, false)
	if err != nil {
		if !pendingUsage.IsEmpty() {
			quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage, false)
		}
		return nil, httperrors.NewInternalServerError("put object error %s", err)
	}

	db.OpsLog.LogEvent(bucket, db.ACT_UPLOAD_OBJECT, key, userCred)
	logclient.AddActionLogWithContext(ctx, bucket, logclient.ACT_UPLOAD_OBJECT, key, userCred, true)

	bucket.syncWithCloudBucket(ctx, userCred, iBucket, nil, true)

	if !pendingUsage.IsEmpty() {
		quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage, true)
	}

	return nil, nil
}

func (bucket *SBucket) AllowPerformAcl(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketAclInput,
) bool {
	return bucket.IsOwner(userCred)
}

// 设置对象和bucket的ACL
//
// 设置对象和bucket的ACL
func (bucket *SBucket) PerformAcl(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketAclInput,
) (jsonutils.JSONObject, error) {
	err := input.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "ValidateInput")
	}

	provider, err := bucket.GetDriver()
	if err != nil {
		return nil, errors.Wrap(err, "GetDriver")
	}

	iBucket, objects, err := bucket.processObjectsActionInput(input.BucketObjectsActionInput)
	if err != nil {
		return nil, errors.Wrap(err, "processObjectsActionInput")
	}

	if len(objects) == 0 {
		if !utils.IsInStringArray(string(input.Acl), provider.GetBucketCannedAcls(bucket.CloudregionId)) {
			return nil, errors.Wrapf(httperrors.ErrInputParameter, "unsupported bucket canned acl %s", input.Acl)
		}
		err = iBucket.SetAcl(input.Acl)
		if err != nil {
			return nil, httperrors.NewInternalServerError("setAcl error %s", err)
		}

		err = bucket.syncWithCloudBucket(ctx, userCred, iBucket, nil, false)
		if err != nil {
			return nil, httperrors.NewInternalServerError("syncWithCloudBucket error %s", err)
		}
		return nil, nil
	}

	if !utils.IsInStringArray(string(input.Acl), provider.GetObjectCannedAcls(bucket.CloudregionId)) {
		return nil, errors.Wrapf(httperrors.ErrInputParameter, "unsupported object canned acl %s", input.Acl)
	}

	errs := make([]error, 0)
	for _, object := range objects {
		err := object.SetAcl(input.Acl)
		if err != nil {
			errs = append(errs, errors.Wrap(err, object.GetKey()))
		}
	}

	if len(errs) > 0 {
		return nil, errors.NewAggregate(errs)
	} else {
		return nil, nil
	}
}

func (bucket *SBucket) AllowPerformSyncstatus(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) bool {
	return bucket.IsOwner(userCred)
}

// 同步存储桶状态
//
// 同步存储桶状态
func (bucket *SBucket) PerformSyncstatus(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketSyncstatusInput,
) (jsonutils.JSONObject, error) {
	var openTask = true
	count, err := taskman.TaskManager.QueryTasksOfObject(bucket, time.Now().Add(-3*time.Minute), &openTask).CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, httperrors.NewBadRequestError("Bucket has %d task active, can't sync status", count)
	}

	return nil, StartResourceSyncStatusTask(ctx, userCred, bucket, "BucketSyncstatusTask", "")
}

func (bucket *SBucket) AllowPerformSync(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) bool {
	return bucket.IsOwner(userCred)
}

func (bucket *SBucket) PerformSync(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	if len(bucket.ExternalId) == 0 {
		return nil, httperrors.NewInvalidStatusError("no external bucket")
	}

	statsOnly := jsonutils.QueryBoolean(data, "stats_only", false)

	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return nil, errors.Wrap(err, "GetIBucket")
	}

	err = bucket.syncWithCloudBucket(ctx, userCred, iBucket, nil, statsOnly)
	if err != nil {
		return nil, httperrors.NewInternalServerError("syncWithCloudBucket error %s", err)
	}

	return nil, nil
}

func (bucket *SBucket) ValidatePurgeCondition(ctx context.Context) error {
	return bucket.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx)
}

func (bucket *SBucket) ValidateDeleteCondition(ctx context.Context) error {
	if bucket.Status == api.BUCKET_STATUS_UNKNOWN {
		return bucket.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx)
	}
	if bucket.ObjectCnt > 0 {
		return httperrors.NewNotEmptyError("Buckets that are not empty do not support this operation")
	}
	return bucket.ValidatePurgeCondition(ctx)
}

func (bucket *SBucket) AllowGetDetailsAcl(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) bool {
	return bucket.IsOwner(userCred)
}

// 获取对象或bucket的ACL
//
// 获取对象或bucket的ACL
func (bucket *SBucket) GetDetailsAcl(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input api.BucketGetAclInput,
) (api.BucketGetAclOutput, error) {
	output := api.BucketGetAclOutput{}
	if len(bucket.ExternalId) == 0 {
		return output, httperrors.NewInvalidStatusError("no external bucket")
	}
	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return output, errors.Wrap(err, "GetIBucket")
	}
	objKey := input.Key
	var acl cloudprovider.TBucketACLType
	if len(objKey) == 0 {
		acl = iBucket.GetAcl()
	} else {
		object, err := cloudprovider.GetIObject(iBucket, objKey)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return output, httperrors.NewNotFoundError("object %s not found", objKey)
			} else {
				return output, httperrors.NewInternalServerError("iBucket.GetIObjects error %s", err)
			}
		}
		acl = object.GetAcl()
	}
	output.Acl = string(acl)
	return output, nil
}

func (bucket *SBucket) AllowPerformSetWebsite(
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketWebsiteConf,
) bool {
	return bucket.IsOwner(userCred)
}

func (bucket *SBucket) PerformSetWebsite(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketWebsiteConf,
) (jsonutils.JSONObject, error) {
	err := input.Validate()
	if err != nil {
		return nil, err
	}
	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return nil, errors.Wrap(err, "GetIBucket")
	}
	bucketWebsiteConf := cloudprovider.SBucketWebsiteConf{
		Index:         input.Index,
		ErrorDocument: input.ErrorDocument,
		Protocol:      input.Protocol,
	}
	for i := range input.Rules {
		bucketWebsiteConf.Rules = append(bucketWebsiteConf.Rules, cloudprovider.SBucketWebsiteRoutingRule{
			ConditionErrorCode: input.Rules[i].ConditionErrorCode,
			ConditionPrefix:    input.Rules[i].ConditionPrefix,

			RedirectProtocol:         input.Rules[i].RedirectProtocol,
			RedirectReplaceKey:       input.Rules[i].RedirectReplaceKey,
			RedirectReplaceKeyPrefix: input.Rules[i].RedirectReplaceKeyPrefix,
		})
	}
	err = iBucket.SetWebsite(bucketWebsiteConf)
	if err != nil {
		return nil, httperrors.NewInternalServerError("iBucket.SetWebsite error %s", err)
	}
	db.OpsLog.LogEvent(bucket, db.ACT_SET_WEBSITE, bucketWebsiteConf, userCred)
	logclient.AddActionLogWithContext(ctx, bucket, logclient.ACT_SET_WEBSITE, bucketWebsiteConf, userCred, true)
	return nil, nil
}

func (bucket *SBucket) AllowPerformDeleteWebsite(
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketWebsiteConf,
) bool {
	return bucket.IsOwner(userCred)
}

func (bucket *SBucket) PerformDeleteWebsite(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return nil, errors.Wrap(err, "GetIBucket")
	}
	err = iBucket.DeleteWebSiteConf()
	if err != nil {
		return nil, httperrors.NewInternalServerError("iBucket.DeleteWebSiteConf error %s", err)
	}
	db.OpsLog.LogEvent(bucket, db.ACT_DELETE_WEBSITE, "", userCred)
	logclient.AddActionLogWithContext(ctx, bucket, logclient.ACT_DELETE_WEBSITE, "", userCred, true)
	return nil, nil
}

func (bucket *SBucket) AllowGetDetailsWebsite(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) bool {
	return bucket.IsOwner(userCred)
}

func (bucket *SBucket) GetDetailsWebsite(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input jsonutils.JSONObject,
) (api.BucketWebsiteConf, error) {
	websiteConf := api.BucketWebsiteConf{}
	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return websiteConf, errors.Wrap(err, "GetIBucket")
	}
	conf, err := iBucket.GetWebsiteConf()
	if err != nil {
		return websiteConf, httperrors.NewInternalServerError("iBucket.GetWebsiteConf error %s", err)
	}

	websiteConf.Index = conf.Index
	websiteConf.ErrorDocument = conf.ErrorDocument
	websiteConf.Protocol = conf.Protocol
	websiteConf.Url = conf.Url

	for i := range conf.Rules {
		websiteConf.Rules = append(websiteConf.Rules, api.BucketWebsiteRoutingRule{
			ConditionErrorCode: conf.Rules[i].ConditionErrorCode,
			ConditionPrefix:    conf.Rules[i].ConditionPrefix,

			RedirectProtocol:         conf.Rules[i].RedirectProtocol,
			RedirectReplaceKey:       conf.Rules[i].RedirectReplaceKey,
			RedirectReplaceKeyPrefix: conf.Rules[i].RedirectReplaceKeyPrefix,
		})
	}
	return websiteConf, nil
}

func (bucket *SBucket) AllowPerformSetCors(
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketCORSRules,
) bool {
	return bucket.IsOwner(userCred)
}

func (bucket *SBucket) PerformSetCors(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketCORSRules,
) (jsonutils.JSONObject, error) {
	err := input.Validate()
	if err != nil {
		return nil, err
	}
	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return nil, errors.Wrap(err, "GetIBucket")
	}
	rules := []cloudprovider.SBucketCORSRule{}
	for i := range input.Data {
		rules = append(rules, cloudprovider.SBucketCORSRule{
			AllowedOrigins: input.Data[i].AllowedOrigins,
			AllowedMethods: input.Data[i].AllowedMethods,
			AllowedHeaders: input.Data[i].AllowedHeaders,
			MaxAgeSeconds:  input.Data[i].MaxAgeSeconds,
			ExposeHeaders:  input.Data[i].ExposeHeaders,
			Id:             input.Data[i].Id,
		})
	}
	err = cloudprovider.SetBucketCORS(iBucket, rules)
	if err != nil {
		return nil, httperrors.NewInternalServerError("cloudprovider.SetBucketCORS error %s", err)
	}
	db.OpsLog.LogEvent(bucket, db.ACT_SET_CORS, rules, userCred)
	logclient.AddActionLogWithContext(ctx, bucket, logclient.ACT_SET_CORS, rules, userCred, true)
	return nil, nil
}

func (bucket *SBucket) AllowPerformDeleteCors(
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketCORSRuleDeleteInput,
) bool {
	return bucket.IsOwner(userCred)
}

func (bucket *SBucket) PerformDeleteCors(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketCORSRuleDeleteInput,
) (jsonutils.JSONObject, error) {
	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return nil, errors.Wrap(err, "GetIBucket")
	}
	result, err := cloudprovider.DeleteBucketCORS(iBucket, input.Id)
	if err != nil {
		return nil, httperrors.NewInternalServerError("iBucket.DeleteCORS error %s", err)
	}
	db.OpsLog.LogEvent(bucket, db.ACT_DELETE_CORS, result, userCred)
	logclient.AddActionLogWithContext(ctx, bucket, logclient.ACT_DELETE_CORS, result, userCred, true)
	return nil, nil
}

func (bucket *SBucket) AllowGetDetailsCors(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) bool {
	return bucket.IsOwner(userCred)
}

func (bucket *SBucket) GetDetailsCors(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input jsonutils.JSONObject,
) (api.BucketCORSRules, error) {
	rules := api.BucketCORSRules{}
	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return rules, errors.Wrap(err, "GetIBucket")
	}
	corsRules, err := iBucket.GetCORSRules()
	if err != nil {
		return rules, httperrors.NewInternalServerError("iBucket.GetCORSRules error %s", err)
	}

	for i := range corsRules {
		rules.Data = append(rules.Data, api.BucketCORSRule{
			AllowedOrigins: corsRules[i].AllowedOrigins,
			AllowedMethods: corsRules[i].AllowedMethods,
			AllowedHeaders: corsRules[i].AllowedHeaders,
			MaxAgeSeconds:  corsRules[i].MaxAgeSeconds,
			ExposeHeaders:  corsRules[i].ExposeHeaders,
			Id:             corsRules[i].Id,
		})
	}

	return rules, nil
}

func (bucket *SBucket) AllowGetDetailsCdnDomain(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) bool {
	return bucket.IsOwner(userCred)
}

func (bucket *SBucket) GetDetailsCdnDomain(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input jsonutils.JSONObject,
) (api.CdnDomains, error) {
	domains := api.CdnDomains{}
	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return domains, errors.Wrap(err, "GetIBucket")
	}
	cdnDomains, err := iBucket.GetCdnDomains()
	if err != nil {
		return domains, httperrors.NewInternalServerError("iBucket.GetCdnDomains error %s", err)
	}
	for i := range cdnDomains {
		domains.Data = append(domains.Data, api.CdnDomain{
			Domain:     cdnDomains[i].Domain,
			Status:     cdnDomains[i].Status,
			Area:       cdnDomains[i].Area,
			Cname:      cdnDomains[i].Cname,
			Origin:     cdnDomains[i].Origin,
			OriginType: cdnDomains[i].OriginType,
		})
	}
	return domains, nil
}

func (bucket *SBucket) AllowPerformSetReferer(
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketRefererConf,
) bool {
	return bucket.IsOwner(userCred)
}

func (bucket *SBucket) PerformSetReferer(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketRefererConf,
) (jsonutils.JSONObject, error) {
	err := input.Validate()
	if err != nil {
		return nil, err
	}
	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return nil, errors.Wrap(err, "GetIBucket")
	}

	conf := cloudprovider.SBucketRefererConf{
		Enabled:         input.Enabled,
		DomainList:      input.DomainList,
		RefererType:     input.RefererType,
		AllowEmptyRefer: input.AllowEmptyRefer,
	}

	err = iBucket.SetReferer(conf)
	if err != nil {
		return nil, httperrors.NewInternalServerError("iBucket.SetRefer error %s", err)
	}
	db.OpsLog.LogEvent(bucket, db.ACT_SET_REFERER, conf, userCred)
	logclient.AddActionLogWithContext(ctx, bucket, logclient.ACT_SET_REFERER, conf, userCred, true)
	return nil, nil
}

func (bucket *SBucket) AllowGetDetailsReferer(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) bool {
	return bucket.IsOwner(userCred)
}

func (bucket *SBucket) GetDetailsReferer(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input jsonutils.JSONObject,
) (api.BucketRefererConf, error) {
	conf := api.BucketRefererConf{}
	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return conf, errors.Wrap(err, "GetIBucket")
	}
	referConf, err := iBucket.GetReferer()
	if err != nil {
		return conf, httperrors.NewInternalServerError("iBucket.GetRefer error %s", err)
	}
	conf.Enabled = referConf.Enabled
	if conf.Enabled {
		conf.DomainList = referConf.DomainList
		conf.RefererType = referConf.RefererType
		conf.AllowEmptyRefer = referConf.AllowEmptyRefer
	}

	return conf, nil
}

func (bucket *SBucket) AllowGetDetailsPolicy(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) bool {
	return bucket.IsOwner(userCred)
}

func (bucket *SBucket) GetDetailsPolicy(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input jsonutils.JSONObject,
) (api.BucketPolicy, error) {
	policy := api.BucketPolicy{}
	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return policy, errors.Wrap(err, "GetIBucket")
	}
	policyStatements, err := iBucket.GetPolicy()
	if err != nil {
		return policy, httperrors.NewInternalServerError("iBucket.GetPolicy error %s", err)
	}
	for i := range policyStatements {
		policy.Data = append(policy.Data, api.BucketPolicyStatement{
			Principal:      policyStatements[i].Principal,
			Action:         policyStatements[i].Action,
			Effect:         policyStatements[i].Effect,
			Resource:       policyStatements[i].Resource,
			Condition:      policyStatements[i].Condition,
			PrincipalId:    policyStatements[i].PrincipalId,
			PrincipalNames: policyStatements[i].PrincipalNames,
			CannedAction:   policyStatements[i].CannedAction,
			ResourcePath:   policyStatements[i].ResourcePath,
			Id:             policyStatements[i].Id,
		})
	}
	return policy, nil
}

func (bucket *SBucket) AllowPerformSetPolicy(
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketPolicyStatementInput,
) bool {
	return bucket.IsOwner(userCred)
}

func (bucket *SBucket) PerformSetPolicy(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketPolicyStatementInput,
) (jsonutils.JSONObject, error) {
	err := input.Validate()
	if err != nil {
		return nil, err
	}
	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return nil, errors.Wrap(err, "GetIBucket")
	}
	opts := cloudprovider.SBucketPolicyStatementInput{
		PrincipalId:  input.PrincipalId,
		CannedAction: input.CannedAction,
		Effect:       input.Effect,
		ResourcePath: input.ResourcePath,
		IpEquals:     input.IpEquals,
		IpNotEquals:  input.IpNotEquals,
	}

	err = iBucket.SetPolicy(opts)
	if err != nil {
		return nil, httperrors.NewInternalServerError("iBucket.SetPolicy error %s", err)
	}
	db.OpsLog.LogEvent(bucket, db.ACT_SET_POLICY, opts, userCred)
	logclient.AddActionLogWithContext(ctx, bucket, logclient.ACT_SET_POLICY, opts, userCred, true)
	return nil, nil
}

func (bucket *SBucket) AllowPerformDeletePolicy(
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketPolicyDeleteInput,
) bool {
	return bucket.IsOwner(userCred)
}

func (bucket *SBucket) PerformDeletePolicy(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketPolicyDeleteInput,
) (jsonutils.JSONObject, error) {
	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return nil, errors.Wrap(err, "GetIBucket")
	}
	result, err := iBucket.DeletePolicy(input.Id)
	if err != nil {
		return nil, httperrors.NewInternalServerError("iBucket.DeletePolicy error %s", err)
	}
	db.OpsLog.LogEvent(bucket, db.ACT_DELETE_POLICY, result, userCred)
	logclient.AddActionLogWithContext(ctx, bucket, logclient.ACT_DELETE_POLICY, result, userCred, true)
	return nil, nil
}

func (manager *SBucketManager) usageQByCloudEnv(q *sqlchemy.SQuery, providers []string, brands []string, cloudEnv string) *sqlchemy.SQuery {
	return CloudProviderFilter(q, q.Field("manager_id"), providers, brands, cloudEnv)
}

func (manager *SBucketManager) usageQByRanges(q *sqlchemy.SQuery, rangeObjs []db.IStandaloneModel) *sqlchemy.SQuery {
	return RangeObjectsFilter(q, rangeObjs, q.Field("cloudregion_id"), nil, q.Field("manager_id"), nil, nil)
}

func (manager *SBucketManager) usageQ(q *sqlchemy.SQuery, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string) *sqlchemy.SQuery {
	q = manager.usageQByRanges(q, rangeObjs)
	q = manager.usageQByCloudEnv(q, providers, brands, cloudEnv)
	return q
}

type SBucketUsages struct {
	Buckets int
	Objects int
	Bytes   int64
}

func (manager *SBucketManager) TotalCount(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string) SBucketUsages {
	usage := SBucketUsages{}
	buckets := manager.Query().SubQuery()
	bucketsQ := buckets.Query(
		sqlchemy.NewFunction(
			sqlchemy.NewCase().When(
				sqlchemy.GE(buckets.Field("object_cnt"), 0),
				buckets.Field("object_cnt"),
			).Else(sqlchemy.NewConstField(0)),
			"object_cnt1",
		),
		sqlchemy.NewFunction(
			sqlchemy.NewCase().When(
				sqlchemy.GE(buckets.Field("size_bytes"), 0),
				buckets.Field("size_bytes"),
			).Else(sqlchemy.NewConstField(0)),
			"size_bytes1",
		),
	)
	bucketsQ = manager.usageQ(bucketsQ, rangeObjs, providers, brands, cloudEnv)
	bucketsQ = scopeOwnerIdFilter(bucketsQ, scope, ownerId)
	buckets = bucketsQ.SubQuery()
	q := buckets.Query(
		sqlchemy.COUNT("buckets"),
		sqlchemy.SUM("objects", buckets.Field("object_cnt1")),
		sqlchemy.SUM("bytes", buckets.Field("size_bytes1")),
	)
	err := q.First(&usage)
	if err != nil {
		log.Errorf("Query bucket usage error %s", err)
	}
	return usage
}

func (bucket *SBucket) AllowPerformLimit(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) bool {
	return bucket.IsOwner(userCred)
}

func (bucket *SBucket) PerformLimit(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	if len(bucket.ExternalId) == 0 {
		return nil, httperrors.NewInvalidStatusError("no external bucket")
	}

	limit := cloudprovider.SBucketStats{}
	err := data.Unmarshal(&limit, "limit")
	if err != nil {
		return nil, httperrors.NewInputParameterError("unmarshal limit error %s", err)
	}

	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return nil, errors.Wrap(err, "GetIBucket")
	}

	err = iBucket.SetLimit(limit)
	if err != nil && err != cloudprovider.ErrNotSupported {
		return nil, httperrors.NewInternalServerError("SetLimit error %s", err)
	}

	diff, err := db.Update(bucket, func() error {
		bucket.SizeBytesLimit = limit.SizeBytes
		bucket.ObjectCntLimit = limit.ObjectCount
		return nil
	})

	if err != nil {
		return nil, httperrors.NewInternalServerError("Update error %s", err)
	}

	if len(diff) > 0 {
		db.OpsLog.LogEvent(bucket, db.ACT_UPDATE, diff, userCred)
		logclient.AddActionLogWithContext(ctx, bucket, logclient.ACT_UPDATE, diff, userCred, true)
	}

	return nil, nil
}

func (bucket *SBucket) AllowGetDetailsAccessInfo(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) bool {
	return bucket.IsOwner(userCred)
}

func (bucket *SBucket) GetDetailsAccessInfo(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	if len(bucket.ExternalId) == 0 {
		return nil, httperrors.NewInvalidStatusError("no external bucket")
	}
	manager := bucket.GetCloudprovider()
	if manager == nil {
		return nil, httperrors.NewInternalServerError("missing manager?")
	}
	info, err := manager.GetDetailsClirc(ctx, userCred, nil)
	if err != nil {
		return nil, err
	}
	account := manager.GetCloudaccount()
	info.(*jsonutils.JSONDict).Add(jsonutils.NewString(account.Brand), "PROVIDER")
	return info, err
}

func (bucket *SBucket) GetUsages() []db.IUsage {
	if bucket.PendingDeleted || bucket.Deleted {
		return nil
	}
	usage := SRegionQuota{Bucket: 1}
	keys, err := bucket.GetQuotaKeys()
	if err != nil {
		log.Errorf("bucket.GetQuotaKeys fail %s", err)
		return nil
	}
	usage.SetKeys(keys)
	return []db.IUsage{
		&usage,
	}
}

func (bucket *SBucket) AllowPerformMetadata(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketMetadataInput,
) bool {
	return bucket.IsOwner(userCred)
}

func (bucket *SBucket) PerformMetadata(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.BucketMetadataInput,
) (jsonutils.JSONObject, error) {
	err := input.Validate()
	if err != nil {
		return nil, err
	}
	_, objects, err := bucket.processObjectsActionInput(input.BucketObjectsActionInput)
	if err != nil {
		return nil, err
	}
	errs := make([]error, 0)
	for _, obj := range objects {
		err := obj.SetMeta(ctx, input.Metadata)
		if err != nil {
			errs = append(errs, errors.Wrap(err, obj.GetKey()))
		}
	}
	if len(errs) > 0 {
		return nil, errors.NewAggregate(errs)
	} else {
		return nil, nil
	}
}

func (bucket *SBucket) processObjectsActionInput(input api.BucketObjectsActionInput) (cloudprovider.ICloudBucket, []cloudprovider.ICloudObject, error) {
	if len(bucket.ExternalId) == 0 {
		return nil, nil, httperrors.NewInvalidStatusError("no external bucket")
	}
	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return nil, nil, errors.Wrap(err, "GetIBucket")
	}
	objects := make([]cloudprovider.ICloudObject, 0)
	for _, key := range input.Key {
		if strings.HasSuffix(key, "/") {
			objs, err := cloudprovider.GetAllObjects(iBucket, key, true)
			if err != nil {
				return nil, nil, httperrors.NewInternalServerError("iBucket.GetIObjects error %s", err)
			}
			objects = append(objects, objs...)
		} else {
			object, err := cloudprovider.GetIObject(iBucket, key)
			if err != nil {
				if errors.Cause(err) == cloudprovider.ErrNotFound {
					return nil, nil, httperrors.NewResourceNotFoundError("object %s not found", key)
				} else {
					return nil, nil, httperrors.NewInternalServerError("iBucket.GetIObject error %s", err)
				}
			}
			objects = append(objects, object)
		}
	}
	return iBucket, objects, nil
}

func (bucket *SBucket) OnMetadataUpdated(ctx context.Context, userCred mcclient.TokenCredential) {
	if len(bucket.ExternalId) == 0 {
		return
	}
	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return
	}
	tags, err := bucket.GetAllUserMetadata()
	if err != nil {
		return
	}
	diff, err := cloudprovider.SetBucketTags(ctx, iBucket, bucket.ManagerId, tags)
	if err != nil {
		logclient.AddSimpleActionLog(bucket, logclient.ACT_UPDATE_TAGS, err, userCred, false)
		return
	}
	if diff.IsChanged() {
		logclient.AddSimpleActionLog(bucket, logclient.ACT_UPDATE_TAGS, diff, userCred, true)
	}
	syncVirtualResourceMetadata(ctx, userCred, bucket, iBucket)
}

func (manager *SBucketManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SSharableVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemExportKeys")
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
