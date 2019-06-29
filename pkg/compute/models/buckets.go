package models

import (
	"context"
	"database/sql"
	"regexp"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
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

	CloudregionId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"admin_required"`

	StorageClass string `width:"36" charset:"ascii" nullable:"false" list:"user"`
	Location     string `width:"36" charset:"ascii" nullable:"false" list:"user"`
	Acl          string `width:"36" charset:"ascii" nullable:"false" list:"user"`
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
		err = commondb[i].syncWithCloudBucket(ctx, userCred, commonext[i], provider)
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
	// bucket.Acl = extBucket.GetAcl()

	bucket.IsEmulated = false

	err = manager.TableSpec().Insert(&bucket)
	if err != nil {
		return nil, errors.Wrap(err, "Insert")
	}

	SyncCloudProject(userCred, &bucket, provider.GetOwnerId(), extBucket, provider.Id)

	db.OpsLog.LogEvent(&bucket, db.ACT_CREATE, bucket.GetShortDesc(ctx), userCred)

	return &bucket, nil
}

func (bucket *SBucket) syncWithCloudBucket(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	extBucket cloudprovider.ICloudBucket,
	provider *SCloudprovider,
) error {
	diff, err := db.UpdateWithLock(ctx, bucket, func() error {
		// bucket.Acl = extBucket.GetAcl()
		bucket.Location = extBucket.GetLocation()
		bucket.StorageClass = extBucket.GetStorageClass()

		bucket.Status = api.BUCKET_STATUS_READY
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "db.UpdateWithLock")
	}

	db.OpsLog.LogSyncUpdate(bucket, diff, userCred)

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
	iregion, err := bucket.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "bucket.GetIRegion")
	}
	err = iregion.DeleteIBucket(bucket.ExternalId)
	if err != nil {
		return errors.Wrap(err, "iregion.DeleteIBucket")
	}
	err = bucket.RealDelete(ctx, userCred)
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
	region, err := bucket.GetRegion()
	if err != nil {
		return nil, errors.Wrap(err, "bucket.GetRegion")
	}
	provider, err := bucket.GetDriver()
	if err != nil {
		return nil, err
	}
	return provider.GetIRegionById(region.GetExternalId())
}

var BUCKET_NAME_REG = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)

func isValidBucketName(name string) bool {
	return BUCKET_NAME_REG.MatchString(name)
}

func (manager *SBucketManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data *jsonutils.JSONDict,
) (*jsonutils.JSONDict, error) {
	for _, v := range []validators.IValidator{
		validators.NewModelIdOrNameValidator("cloudregion", CloudregionManager.Keyword(), ownerId),
		validators.NewModelIdOrNameValidator("manager", CloudproviderManager.Keyword(), ownerId),
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
	if !isValidBucketName(nameStr) {
		return nil, httperrors.NewInputParameterError("invalid name, only alphabets, digits and hyphen(-) allowed")
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
		if !isValidBucketName(nameStr) {
			return nil, httperrors.NewInputParameterError("invalid name, only alphabets, digits and hyphen(-) allowed")
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
	err = db.SetExternalId(bucket, userCred, bucket.Name)
	if err != nil {
		return errors.Wrap(err, "db.SetExternalId")
	}
	extBucket, err := iregion.GetIBucketByName(bucket.Name)
	if err != nil {
		return errors.Wrap(err, "iregion.GetIBucketByName")
	}
	err = bucket.syncWithCloudBucket(ctx, userCred, extBucket, nil)
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

func (bucket *SBucket) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	info := bucket.getCloudProviderInfo()
	extra.Update(jsonutils.Marshal(&info))

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
