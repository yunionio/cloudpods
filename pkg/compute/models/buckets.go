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
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/pkg/s3utils"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SBucketManager struct {
	db.SVirtualResourceBaseManager
}

var BucketManager *SBucketManager

func init() {
	BucketManager = &SBucketManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SBucket{},
			"buckets_tbl",
			"bucket",
			"buckets",
		),
	}
	BucketManager.SetVirtualObject(BucketManager)
}

type SBucket struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase

	CloudregionId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`

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
	return manager.SVirtualResourceBaseManager.SetHandlerProcessTimeout(info, r)
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
	lockman.LockClass(ctx, manager, "")
	defer lockman.ReleaseClass(ctx, manager, "")

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

	newName, err := db.GenerateName(manager, nil, extBucket.GetName())
	if err != nil {
		return nil, errors.Wrap(err, "db.GenerateName")
	}

	bucket.Name = newName

	created := extBucket.GetCreateAt()
	if !created.IsZero() {
		bucket.CreatedAt = created
	}

	bucket.Location = extBucket.GetLocation()
	bucket.StorageClass = extBucket.GetStorageClass()
	bucket.Acl = string(extBucket.GetAcl())

	stats := extBucket.GetStats()
	bucket.SizeBytes = stats.SizeBytes
	bucket.ObjectCnt = stats.ObjectCount

	limit := extBucket.GetLimit()
	bucket.SizeBytesLimit = limit.SizeBytes
	bucket.ObjectCntLimit = limit.ObjectCount

	bucket.AccessUrls = jsonutils.Marshal(extBucket.GetAccessUrls())

	bucket.IsEmulated = false

	err = manager.TableSpec().Insert(&bucket)
	if err != nil {
		return nil, errors.Wrap(err, "Insert")
	}

	SyncCloudProject(userCred, &bucket, provider.GetOwnerId(), extBucket, provider.Id)

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
	desc := bucket.SVirtualResourceBase.GetShortDesc(ctx)

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
		bucket.ObjectCnt = stats.ObjectCount

		if !statsOnly {
			limit := extBucket.GetLimit()
			bucket.SizeBytesLimit = limit.SizeBytes
			bucket.ObjectCntLimit = limit.ObjectCount

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

	db.OpsLog.LogSyncUpdate(bucket, diff, userCred)

	if !oStats.Equals(extBucket.GetStats()) {
		db.OpsLog.LogEvent(bucket, api.BUCKET_OPS_STATS_CHANGE, bucket.GetShortDesc(ctx), userCred)
	}

	if provider != nil {
		SyncCloudProject(userCred, bucket, provider.GetOwnerId(), extBucket, provider.Id)
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
	return bucket.SVirtualResourceBase.Delete(ctx, userCred)
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
	data *jsonutils.JSONDict,
) (*jsonutils.JSONDict, error) {
	cloudRegionV := validators.NewModelIdOrNameValidator("cloudregion", CloudregionManager.Keyword(), ownerId)
	managerV := validators.NewModelIdOrNameValidator("manager", CloudproviderManager.Keyword(), ownerId)
	for _, v := range []validators.IValidator{
		cloudRegionV,
		managerV,
	} {
		err := v.Validate(data)
		if err != nil {
			return nil, err
		}
	}
	nameStr, _ := data.GetString("name")
	if len(nameStr) == 0 {
		return nil, httperrors.NewInputParameterError("missing name")
	}
	err := isValidBucketName(nameStr)
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid bucket name: %s", err)
	}

	cloudprovider := managerV.Model.(*SCloudprovider)
	quotaPlatformId := cloudprovider.GetQuotaPlatformID()
	pendingUsage := SQuota{Bucket: 1}
	if err := QuotaManager.CheckSetPendingQuota(ctx, userCred, rbacutils.ScopeProject, ownerId, quotaPlatformId, &pendingUsage); err != nil {
		return nil, httperrors.NewOutOfQuotaError("%s", err)
	}

	return manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
}

func (bucket *SBucket) PostCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) {
	cloudprovider := bucket.GetCloudprovider()
	quotaPlatformId := cloudprovider.GetQuotaPlatformID()
	pendingUsage := SQuota{Bucket: 1}
	err := QuotaManager.CancelPendingUsage(ctx, userCred, rbacutils.ScopeProject, ownerId, quotaPlatformId, &pendingUsage, &pendingUsage)
	if err != nil {
		log.Errorf("CancelPendingUsage error %s", err)
	}

	bucket.SetStatus(userCred, api.BUCKET_STATUS_START_CREATE, "PostCreate")
	task, err := taskman.TaskManager.NewTask(ctx, "BucketCreateTask", bucket, userCred, nil, "", "", nil)
	if err != nil {
		log.Errorf("BucketCreateTask newTask error %s", err)
	} else {
		task.ScheduleRun(nil)
	}
}

func (bucket *SBucket) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data *jsonutils.JSONDict,
) (*jsonutils.JSONDict, error) {
	nameStr, _ := data.GetString("name")
	if len(nameStr) > 0 {
		err := isValidBucketName(nameStr)
		if err != nil {
			return nil, httperrors.NewInputParameterError("invalid bucket name: %s", err)
		}
	}
	return bucket.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, data)
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
	err = bucket.syncWithCloudBucket(ctx, userCred, extBucket, nil, false)
	if err != nil {
		return errors.Wrap(err, "bucket.syncWithCloudBucket")
	}
	return nil
}

func (bucket *SBucket) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := bucket.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return bucket.getMoreDetails(extra)
}

func (bucket *SBucket) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := bucket.SVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return bucket.getMoreDetails(extra), nil
}

func joinPath(ep, path string) string {
	return strings.TrimRight(ep, "/") + "/" + strings.TrimLeft(path, "/")
}

func (bucket *SBucket) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	info := bucket.getCloudProviderInfo()
	extra.Update(jsonutils.Marshal(&info))

	s3gwUrl, _ := auth.GetServiceURL("s3gateway", options.Options.Region, "", "public")
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
			extra.Set("access_urls", jsonutils.Marshal(accessUrls))
		}
	}

	return extra
}

func (bucket *SBucket) getCloudProviderInfo() SCloudProviderInfo {
	region, _ := bucket.GetRegion()
	provider := bucket.GetCloudprovider()
	return MakeCloudProviderInfo(region, nil, provider)
}

func (manager *SBucketManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	var err error

	q, err = managedResourceFilterByAccount(q, query, "", nil)
	if err != nil {
		return nil, err
	}
	q = managedResourceFilterByCloudType(q, query, "", nil)

	q, err = managedResourceFilterByDomain(q, query, "", nil)
	if err != nil {
		return nil, err
	}

	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	return q, nil
}

func (manager *SBucketManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	switch field {
	case "tenant":
		tenantCacheQuery := db.TenantCacheManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(tenantCacheQuery.Field("name", "tenant"))
		q = q.Join(tenantCacheQuery, sqlchemy.Equals(q.Field("tenant_id"), tenantCacheQuery.Field("id")))
		q.GroupBy(tenantCacheQuery.Field("name"))
	case "account":
		cloudproviders := CloudproviderManager.Query().SubQuery()
		cloudaccounts := CloudaccountManager.Query("name", "id").Distinct().SubQuery()
		q = q.Join(cloudproviders, sqlchemy.Equals(q.Field("manager_id"), cloudproviders.Field("id")))
		q = q.Join(cloudaccounts, sqlchemy.Equals(cloudproviders.Field("cloudaccount_id"), cloudaccounts.Field("id")))
		q.GroupBy(cloudaccounts.Field("name"))
		q.AppendField(cloudaccounts.Field("name", "account"))
	default:
		return nil, httperrors.NewBadRequestError("unsupport field %s", field)
	}
	return q, nil
}

func (bucket *SBucket) AllowGetDetailsObjects(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) bool {
	return bucket.IsOwner(userCred)
}

func (bucket *SBucket) GetDetailsObjects(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return nil, httperrors.NewInternalServerError("fail to find external bucket: %s", err)
	}
	prefix, _ := query.GetString("prefix")
	isRecursive := jsonutils.QueryBoolean(query, "recursive", false)
	objects, err := iBucket.GetIObjects(prefix, isRecursive)
	if err != nil {
		return nil, httperrors.NewInternalServerError("fail to get objects: %s", err)
	}
	retArray := jsonutils.NewArray()
	for i := range objects {
		retArray.Add(cloudprovider.ICloudObject2JSONObject(objects[i]))
	}
	ret := jsonutils.NewDict()
	ret.Add(retArray, "objects")
	return ret, nil
}

func (bucket *SBucket) AllowPerformTempUrl(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) bool {
	return bucket.IsOwner(userCred)
}

func (bucket *SBucket) PerformTempUrl(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	method, _ := data.GetString("method")
	key, _ := data.GetString("key")
	expire, _ := data.Int("expire_seconds")

	if len(method) == 0 {
		method = "GET"
	}
	if len(key) == 0 {
		return nil, httperrors.NewInputParameterError("missing key")
	}
	if expire == 0 {
		expire = 60 // default 60 seconds
	}

	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return nil, httperrors.NewInternalServerError("fail to find external bucket: %s", err)
	}
	tmpUrl, err := iBucket.GetTempUrl(method, key, time.Duration(expire)*time.Second)
	if err != nil {
		return nil, httperrors.NewInternalServerError("fail to generate temp url: %s", err)
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString(tmpUrl), "url")
	return ret, nil
}

func (bucket *SBucket) AllowPerformMakedir(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) bool {
	return bucket.IsOwner(userCred)
}

func (bucket *SBucket) PerformMakedir(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	key, _ := data.GetString("key")
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
		return nil, httperrors.NewInternalServerError("fail to find external bucket: %s", err)
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
	manager := bucket.GetCloudprovider()
	quotaPlatformId := manager.GetQuotaPlatformID()
	pendingUsage := SQuota{ObjectGB: 0, ObjectCnt: 1}
	if !pendingUsage.IsEmpty() {
		if err := QuotaManager.CheckSetPendingQuota(ctx, userCred, rbacutils.ScopeProject, bucket.GetOwnerId(), quotaPlatformId, &pendingUsage); err != nil {
			return nil, httperrors.NewOutOfQuotaError("%s", err)
		}
	}

	err = cloudprovider.Makedir(ctx, iBucket, key+"/")
	if err != nil {
		return nil, httperrors.NewInternalServerError("fail to mkdir: %s", err)
	}

	db.OpsLog.LogEvent(bucket, db.ACT_MKDIR, key, userCred)
	logclient.AddActionLogWithContext(ctx, bucket, logclient.ACT_MKDIR, key, userCred, true)

	bucket.syncWithCloudBucket(ctx, userCred, iBucket, nil, true)

	if !pendingUsage.IsEmpty() {
		QuotaManager.CancelPendingUsage(ctx, userCred, rbacutils.ScopeProject, bucket.GetOwnerId(), quotaPlatformId, &pendingUsage, &pendingUsage)
	}

	return nil, nil
}

func (bucket *SBucket) AllowPerformDelete(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) bool {
	return bucket.IsOwner(userCred)
}

func (bucket *SBucket) PerformDelete(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	keys, _ := data.Get("keys")
	if keys == nil {
		return nil, httperrors.NewInputParameterError("missing keys")
	}
	keyStrs := keys.(*jsonutils.JSONArray).GetStringArray()
	if len(keyStrs) == 0 {
		return nil, httperrors.NewInputParameterError("empty keys")
	}

	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return nil, httperrors.NewInternalServerError("fail to find external bucket: %s", err)
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

func (bucket *SBucket) PerformUpload(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
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
		return nil, httperrors.NewInternalServerError("fail to find external bucket: %s", err)
	}

	contType := appParams.Request.Header.Get("Content-Type")
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
	aclStr := appParams.Request.Header.Get(api.BUCKET_UPLOAD_OBJECT_ACL_HEADER)
	if len(aclStr) > 0 {
		switch cloudprovider.TBucketACLType(aclStr) {
		case cloudprovider.ACLPrivate, cloudprovider.ACLAuthRead, cloudprovider.ACLPublicRead, cloudprovider.ACLPublicReadWrite:
			// do nothing
		default:
			return nil, httperrors.NewInputParameterError("invalid acl: %s", aclStr)
		}
	}

	inc := cloudprovider.SBucketStats{}
	obj, err := cloudprovider.GetIObject(iBucket, key)
	if err == nil {
		// replace
		inc.SizeBytes = sizeBytes - obj.GetSizeBytes()
		if inc.SizeBytes < 0 {
			inc.SizeBytes = 0
		}
	} else if err == cloudprovider.ErrNotFound {
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

	manager := bucket.GetCloudprovider()
	quotaPlatformId := manager.GetQuotaPlatformID()
	pendingUsage := SQuota{ObjectGB: int(inc.SizeBytes / 1000 / 1000 / 1000), ObjectCnt: inc.ObjectCount}
	if !pendingUsage.IsEmpty() {
		if err := QuotaManager.CheckSetPendingQuota(ctx, userCred, rbacutils.ScopeProject, bucket.GetOwnerId(), quotaPlatformId, &pendingUsage); err != nil {
			return nil, httperrors.NewOutOfQuotaError("%s", err)
		}
	}

	err = cloudprovider.UploadObject(ctx, iBucket, key, 0, appParams.Request.Body, sizeBytes, contType, cloudprovider.TBucketACLType(aclStr), storageClass, false)
	if err != nil {
		return nil, httperrors.NewInternalServerError("put object error %s", err)
	}

	db.OpsLog.LogEvent(bucket, db.ACT_UPLOAD_OBJECT, key, userCred)
	logclient.AddActionLogWithContext(ctx, bucket, logclient.ACT_UPLOAD_OBJECT, key, userCred, true)

	bucket.syncWithCloudBucket(ctx, userCred, iBucket, nil, true)

	if !pendingUsage.IsEmpty() {
		QuotaManager.CancelPendingUsage(ctx, userCred, rbacutils.ScopeProject, bucket.GetOwnerId(), quotaPlatformId, &pendingUsage, &pendingUsage)
	}

	return nil, nil
}

func (bucket *SBucket) AllowPerformAcl(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) bool {
	return bucket.IsOwner(userCred)
}

func (bucket *SBucket) PerformAcl(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	aclStr, _ := data.GetString("acl")
	switch cloudprovider.TBucketACLType(aclStr) {
	case cloudprovider.ACLPrivate, cloudprovider.ACLAuthRead, cloudprovider.ACLPublicRead, cloudprovider.ACLPublicReadWrite:
		// do nothing
	default:
		return nil, httperrors.NewInputParameterError("invalid acl: %s", aclStr)
	}

	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return nil, httperrors.NewInternalServerError("fail to find external bucket: %s", err)
	}

	objKey, _ := data.Get("key")
	if objKey == nil {
		err = iBucket.SetAcl(cloudprovider.TBucketACLType(aclStr))
		if err != nil {
			return nil, httperrors.NewInternalServerError("setAcl error %s", err)
		}

		err = bucket.syncWithCloudBucket(ctx, userCred, iBucket, nil, false)
		if err != nil {
			return nil, httperrors.NewInternalServerError("syncWithCloudBucket error %s", err)
		}
		return nil, nil
	}
	var keys []string
	switch jsonObj := objKey.(type) {
	case *jsonutils.JSONString:
		key, _ := jsonObj.GetString()
		keys = []string{key}
	case *jsonutils.JSONArray:
		keys = jsonObj.GetStringArray()
	}
	var objects []cloudprovider.ICloudObject
	for _, key := range keys {
		if strings.HasSuffix(key, "/") {
			objs, err := cloudprovider.GetIObjects(iBucket, key, true)
			if err != nil {
				return nil, httperrors.NewInternalServerError("iBucket.GetIObjects error %s", err)
			}
			objects = append(objects, objs...)
		} else {
			object, err := cloudprovider.GetIObject(iBucket, key)
			if err != nil {
				if err == cloudprovider.ErrNotFound {
					return nil, httperrors.NewResourceNotFoundError("object %s not found", objKey)
				} else {
					return nil, httperrors.NewInternalServerError("iBucket.GetIObject error %s", err)
				}
			}
			objects = append(objects, object)
		}
	}
	for _, object := range objects {
		err := object.SetAcl(cloudprovider.TBucketACLType(aclStr))
		if err != nil {
			return nil, httperrors.NewInternalServerError("setAcl error %s", err)
		}
	}

	return nil, nil
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
	statsOnly := jsonutils.QueryBoolean(data, "stats_only", false)

	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return nil, httperrors.NewInternalServerError("fail to find external bucket: %s", err)
	}

	err = bucket.syncWithCloudBucket(ctx, userCred, iBucket, nil, statsOnly)
	if err != nil {
		return nil, httperrors.NewInternalServerError("syncWithCloudBucket error %s", err)
	}

	return nil, nil
}

func (bucket *SBucket) ValidatePurgeCondition(ctx context.Context) error {
	return bucket.SVirtualResourceBase.ValidateDeleteCondition(ctx)
}

func (bucket *SBucket) ValidateDeleteCondition(ctx context.Context) error {
	if bucket.ObjectCnt > 0 {
		return httperrors.NewNotEmptyError("not an empty bucket")
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

func (bucket *SBucket) GetDetailsAcl(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return nil, httperrors.NewInternalServerError("fail to find external bucket: %s", err)
	}
	objKey, _ := query.GetString("key")
	var acl cloudprovider.TBucketACLType
	if len(objKey) == 0 {
		acl = iBucket.GetAcl()
	} else {
		object, err := cloudprovider.GetIObject(iBucket, objKey)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				return nil, httperrors.NewNotFoundError("object %s not found", objKey)
			} else {
				return nil, httperrors.NewInternalServerError("iBucket.GetIObjects error %s", err)
			}
		}
		acl = object.GetAcl()
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString(string(acl)), "acl")
	return ret, nil
}

func (manager *SBucketManager) usageQByCloudEnv(q *sqlchemy.SQuery, providers []string, brands []string, cloudEnv string) *sqlchemy.SQuery {
	return CloudProviderFilter(q, q.Field("manager_id"), providers, brands, cloudEnv)
}

func (manager *SBucketManager) usageQByRange(q *sqlchemy.SQuery, rangeObj db.IStandaloneModel) *sqlchemy.SQuery {
	if rangeObj == nil {
		return q
	}

	kw := rangeObj.Keyword()
	switch kw {
	case "zone":
		zone := rangeObj.(*SZone)
		q = q.Filter(sqlchemy.Equals(q.Field("cloudregion_id"), zone.CloudregionId))
	case "wire":
		wire := rangeObj.(*SWire)
		zone := wire.GetZone()
		q = q.Filter(sqlchemy.Equals(q.Field("cloudregion_id"), zone.CloudregionId))
	case "host":
		host := rangeObj.(*SHost)
		zone := host.GetZone()
		q = q.Filter(sqlchemy.Equals(q.Field("cloudregion_id"), zone.CloudregionId))
	case "cloudprovider":
		q = q.Filter(sqlchemy.Equals(q.Field("manager_id"), rangeObj.GetId()))
	case "cloudaccount":
		cloudproviders := CloudproviderManager.Query().SubQuery()
		subq := cloudproviders.Query(cloudproviders.Field("id")).Equals("cloudaccount_id", rangeObj.GetId()).SubQuery()
		q = q.Filter(sqlchemy.In(q.Field("manager_id"), subq))
	case "cloudregion":
		q = q.Filter(sqlchemy.Equals(q.Field("cloudregion_id"), rangeObj.GetId()))
	}

	return q
}

func (manager *SBucketManager) usageQ(q *sqlchemy.SQuery, rangeObj db.IStandaloneModel, providers []string, brands []string, cloudEnv string) *sqlchemy.SQuery {
	q = manager.usageQByRange(q, rangeObj)
	q = manager.usageQByCloudEnv(q, providers, brands, cloudEnv)
	return q
}

type SBucketUsages struct {
	Buckets int
	Objects int
	Bytes   int64
}

func (manager *SBucketManager) TotalCount(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObj db.IStandaloneModel, providers []string, brands []string, cloudEnv string) SBucketUsages {
	usage := SBucketUsages{}
	buckets := manager.Query().SubQuery()
	q := buckets.Query(
		sqlchemy.COUNT("buckets"),
		sqlchemy.SUM("objects", buckets.Field("object_cnt")),
		sqlchemy.SUM("bytes", buckets.Field("size_bytes")),
	)
	q = manager.usageQ(q, rangeObj, providers, brands, cloudEnv)
	switch scope {
	case rbacutils.ScopeSystem:
		// do nothing
	case rbacutils.ScopeDomain:
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	case rbacutils.ScopeProject:
		q = q.Equals("tenant_id", ownerId.GetProjectId())
	}
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
	limit := cloudprovider.SBucketStats{}
	err := data.Unmarshal(&limit, "limit")
	if err != nil {
		return nil, httperrors.NewInputParameterError("unmarshal limit error %s", err)
	}

	iBucket, err := bucket.GetIBucket()
	if err != nil {
		return nil, httperrors.NewInternalServerError("fail to find external bucket: %s", err)
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
