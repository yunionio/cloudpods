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

	"github.com/serialx/hashring"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/imagetools"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SStoragecacheManager struct {
	db.SStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
}

var StoragecacheManager *SStoragecacheManager

func init() {
	StoragecacheManager = &SStoragecacheManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SStoragecache{},
			"storagecaches_tbl",
			"storagecache",
			"storagecaches",
		),
	}
	StoragecacheManager.SetVirtualObject(StoragecacheManager)
}

type SStoragecache struct {
	db.SStandaloneResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase

	// 镜像存储地址
	Path string `width:"256" charset:"utf8" nullable:"true" list:"user" update:"admin" create:"admin_optional"` // = Column(VARCHAR(256, charset='utf8'), nullable=True)
	// master host id
	MasterHost string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user" json:"master_host"`
}

func (sc *SStoragecache) getStorages() []SStorage {
	storages := make([]SStorage, 0)
	q := StorageManager.Query().Equals("storagecache_id", sc.Id)
	err := db.FetchModelObjects(StorageManager, q, &storages)
	if err != nil {
		return nil
	}
	return storages
}

func (sc *SStoragecache) getValidStorages() []SStorage {
	storages := []SStorage{}
	q := StorageManager.Query()
	zones := ZoneManager.Query().Equals("status", api.ZONE_ENABLE).SubQuery()
	q = q.Equals("storagecache_id", sc.Id).
		Filter(sqlchemy.In(q.Field("status"), []string{api.STORAGE_ENABLED, api.STORAGE_ONLINE})).
		Filter(sqlchemy.IsTrue(q.Field("enabled"))).
		Filter(sqlchemy.IsFalse(q.Field("deleted")))
	q = q.Join(zones, sqlchemy.Equals(q.Field("zone_id"), zones.Field("id")))
	err := db.FetchModelObjects(StorageManager, q, &storages)
	if err != nil {
		return nil
	}
	return storages
}

func (sc *SStoragecache) getStorageNames() []string {
	storages := sc.getStorages()
	if storages == nil {
		return nil
	}
	names := make([]string, len(storages))
	for i := 0; i < len(storages); i += 1 {
		names[i] = storages[i].Name
	}
	return names
}

func (sc *SStoragecache) GetEsxiAgentHostDesc() (*jsonutils.JSONDict, error) {
	if !strings.Contains(sc.Name, "esxiagent") {
		return nil, nil
	}
	obj, err := BaremetalagentManager.FetchById(sc.ExternalId)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to fetch baremetalagent %s", obj.GetId())
	}
	agent := obj.(*SBaremetalagent)
	host := &SHost{}
	host.Id = agent.Id
	host.Name = agent.Name
	host.ZoneId = agent.ZoneId
	host.SetModelManager(HostManager, host)
	ret := host.GetShortDesc(context.Background())
	ret.Set("provider", jsonutils.NewString(api.CLOUD_PROVIDER_VMWARE))
	ret.Set("brand", jsonutils.NewString(api.CLOUD_PROVIDER_VMWARE))
	return ret, nil
}

func (sc *SStoragecache) GetMasterHost() (*SHost, error) {
	if sc.MasterHost != "" {
		host, err := HostManager.FetchById(sc.MasterHost)
		if err != nil {
			return nil, errors.Wrap(err, "HostManager.FetchById")
		}
		return host.(*SHost), nil
	}

	hostId, err := sc.getHostId()
	if err != nil {
		return nil, errors.Wrap(err, "sc.getHostId")
	}
	if len(hostId) == 0 {
		return nil, errors.Errorf("failed to get any available host for storagecache %s", sc.Name)
	}

	host, err := HostManager.FetchById(hostId)
	if err != nil {
		return nil, errors.Wrap(err, "HostManager.FetchById")
	}
	return host.(*SHost), nil
}

func (sc *SStoragecache) GetRegion() (*SCloudregion, error) {
	host, err := sc.GetMasterHost()
	if err != nil {
		return nil, errors.Wrapf(err, "GetHost")
	}
	region, err := host.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion")
	}
	return region, nil
}

func (sc *SStoragecache) GetHosts() ([]SHost, error) {
	hoststorages := HoststorageManager.Query().SubQuery()
	storages := StorageManager.Query().SubQuery()

	hosts := make([]SHost, 0)
	host := HostManager.Query().SubQuery()
	q := host.Query(host.Field("id"))
	err := q.Join(hoststorages, sqlchemy.AND(
		sqlchemy.Equals(hoststorages.Field("host_id"), host.Field("id")),
		sqlchemy.OR(
			sqlchemy.Equals(host.Field("host_status"), api.HOST_ONLINE),
			sqlchemy.Equals(host.Field("host_type"), api.HOST_TYPE_BAREMETAL),
		),
		sqlchemy.IsTrue(host.Field("enabled")),
	)).
		Join(storages, sqlchemy.AND(sqlchemy.Equals(storages.Field("storagecache_id"), sc.Id),
			sqlchemy.In(storages.Field("status"), []string{api.STORAGE_ENABLED, api.STORAGE_ONLINE}),
			sqlchemy.IsTrue(storages.Field("enabled")))).
		Filter(sqlchemy.Equals(hoststorages.Field("storage_id"), storages.Field("id"))).All(&hosts)
	if err != nil {
		return nil, err
	}
	return hosts, nil
}

func (sc *SStoragecache) getHostId() (string, error) {
	hosts, err := sc.GetHosts()
	if err != nil {
		return "", errors.Wrap(err, "GetHosts")
	}

	hostIds := make([]string, 0)
	for _, h := range hosts {
		hostIds = append(hostIds, h.Id)
	}

	if len(hostIds) == 0 {
		return "", nil
	}
	ring := hashring.New(hostIds)
	ret, _ := ring.GetNode(sc.Id)
	return ret, nil
}

func (manager *SStoragecacheManager) SyncWithCloudStoragecache(ctx context.Context, userCred mcclient.TokenCredential, cloudCache cloudprovider.ICloudStoragecache, provider *SCloudprovider, xor bool) (*SStoragecache, bool, error) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	localCacheObj, err := db.FetchByExternalIdAndManagerId(manager, cloudCache.GetGlobalId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("manager_id", provider.Id)
	})
	if err != nil {
		if err == sql.ErrNoRows {
			localCache, err := manager.newFromCloudStoragecache(ctx, userCred, cloudCache, provider)
			if err != nil {
				return nil, false, err
			} else {
				return localCache, true, nil
			}
		} else {
			return nil, false, errors.Wrapf(err, "db.FetchByExternalIdAndManagerId(%s)", cloudCache.GetGlobalId())
		}
	} else {
		localCache := localCacheObj.(*SStoragecache)
		if !xor {
			localCache.syncWithCloudStoragecache(ctx, userCred, cloudCache, provider)
		}
		return localCache, false, nil
	}
}

func (manager *SStoragecacheManager) newFromCloudStoragecache(ctx context.Context, userCred mcclient.TokenCredential, cloudCache cloudprovider.ICloudStoragecache, provider *SCloudprovider) (*SStoragecache, error) {
	local := SStoragecache{}
	local.SetModelManager(manager, &local)

	local.ExternalId = cloudCache.GetGlobalId()

	local.IsEmulated = cloudCache.IsEmulated()
	local.ManagerId = provider.Id

	local.Path = cloudCache.GetPath()

	var err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		newName, err := db.GenerateName(ctx, manager, userCred, cloudCache.GetName())
		if err != nil {
			return err
		}
		local.Name = newName

		return manager.TableSpec().Insert(ctx, &local)
	}()
	if err != nil {
		return nil, err
	}

	db.OpsLog.LogEvent(&local, db.ACT_CREATE, local.GetShortDesc(ctx), userCred)

	return &local, nil
}

func (sc *SStoragecache) syncWithCloudStoragecache(ctx context.Context, userCred mcclient.TokenCredential, cloudCache cloudprovider.ICloudStoragecache, provider *SCloudprovider) error {
	diff, err := db.UpdateWithLock(ctx, sc, func() error {
		sc.Name = cloudCache.GetName()

		sc.Path = cloudCache.GetPath()

		sc.IsEmulated = cloudCache.IsEmulated()
		sc.ManagerId = provider.Id

		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(sc, diff, userCred)
	return nil
}

func (manager *SStoragecacheManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.StoragecacheDetails {
	rows := make([]api.StoragecacheDetails, len(objs))

	stdRows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.StoragecacheDetails{
			StandaloneResourceDetails: stdRows[i],
			ManagedResourceInfo:       manRows[i],
		}
		rows[i] = objs[i].(*SStoragecache).getMoreDetails(ctx, rows[i])
	}

	return rows
}

func (sc *SStoragecache) getMoreDetails(ctx context.Context, out api.StoragecacheDetails) api.StoragecacheDetails {
	out.Storages = sc.getStorageNames()
	out.Size = sc.getCachedImageSize()
	out.Count = sc.getCachedImageCount()

	host, _ := sc.GetMasterHost()
	if host != nil {
		out.Host = host.GetShortDesc(ctx)
	}
	return out
}

func (sc *SStoragecache) getCachedImageList(excludeIds []string, imageType string, status []string) []SCachedimage {
	images := make([]SCachedimage, 0)

	cachedImages := CachedimageManager.Query().SubQuery()
	storagecachedImages := StoragecachedimageManager.Query().SubQuery()

	q := cachedImages.Query()
	q = q.Join(storagecachedImages, sqlchemy.Equals(cachedImages.Field("id"), storagecachedImages.Field("cachedimage_id")))
	q = q.Filter(sqlchemy.Equals(storagecachedImages.Field("storagecache_id"), sc.Id))

	if len(excludeIds) > 0 {
		q = q.Filter(sqlchemy.NotIn(cachedImages.Field("id"), excludeIds))
	}
	if len(imageType) > 0 {
		q = q.Filter(sqlchemy.Equals(cachedImages.Field("image_type"), imageType))
	}
	if len(status) > 0 {
		q = q.Filter(sqlchemy.In(storagecachedImages.Field("status"), status))
	}

	err := db.FetchModelObjects(CachedimageManager, q, &images)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("%s", err)
		}
		return nil
	}
	return images
}

func (sc *SStoragecache) getCachedImages() ([]SStoragecachedimage, error) {
	images := make([]SStoragecachedimage, 0)
	q := StoragecachedimageManager.Query().Equals("storagecache_id", sc.Id)
	err := db.FetchModelObjects(StoragecachedimageManager, q, &images)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return images, nil
}

func (sc *SStoragecache) getCustomdCachedImages() ([]SStoragecachedimage, error) {
	images := make([]SStoragecachedimage, 0)
	sq := CachedimageManager.Query("id").Equals("image_type", "customized").SubQuery()
	q := StoragecachedimageManager.Query().Equals("storagecache_id", sc.Id).In("cachedimage_id", sq)
	err := db.FetchModelObjects(StoragecachedimageManager, q, &images)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return images, nil
}

func (sc *SStoragecache) getCachedImageCount() int {
	images, _ := sc.getCachedImages()
	return len(images)
}

func (sc *SStoragecache) getCachedImageSize() int64 {
	images, _ := sc.getCachedImages()
	if images == nil {
		return 0
	}
	var size int64 = 0
	for _, img := range images {
		imginfo := img.GetCachedimage()
		if imginfo != nil {
			size += imginfo.Size
		}
	}
	return size
}

func (manager *SStoragecacheManager) StartImageCacheTask(ctx context.Context, userCred mcclient.TokenCredential, scs []SStoragecache, input api.CacheImageInput) error {
	objs := make([]db.IStandaloneModel, len(scs))
	inputs := make([]api.CacheImageInput, len(scs))
	for i := range scs {
		objs[i] = &scs[i]
		inputs[i] = input
	}
	params := jsonutils.NewDict()
	params.Add(jsonutils.Marshal(inputs), "params")
	task, err := taskman.TaskManager.NewParallelTask(ctx, "StorageBatchCacheImageTask", objs, userCred, params, input.ParentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewParallelTask")
	}
	return task.ScheduleRun(nil)
}

func (sc *SStoragecache) StartImageCacheTask(ctx context.Context, userCred mcclient.TokenCredential, input api.CacheImageInput) error {
	StoragecachedimageManager.Register(ctx, userCred, sc.Id, input.ImageId, "")

	image, _ := CachedimageManager.GetImageById(ctx, userCred, input.ImageId, false)
	if image != nil {
		imgInfo := imagetools.NormalizeImageInfo(image.Name, image.Properties["os_arch"], image.Properties["os_type"],
			image.Properties["os_distribution"], image.Properties["os_version"])
		input.OsType = imgInfo.OsType
		input.OsArch = imgInfo.OsArch
		input.OsDistribution = imgInfo.OsDistro
		input.OsVersion = imgInfo.OsVersion
		input.OsFullVersion = imgInfo.OsFullVersion
		input.ImageName = image.Name
	}
	data := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewTask(ctx, "StorageCacheImageTask", sc, userCred, data, input.ParentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (sc *SStoragecache) StartImageUncacheTask(ctx context.Context, userCred mcclient.TokenCredential, imageId string, isPurge bool, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(imageId), "image_id")
	if isPurge {
		data.Add(jsonutils.JSONTrue, "is_purge")
	}
	task, err := taskman.TaskManager.NewTask(ctx, "StorageUncacheImageTask", sc, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (sc *SStoragecache) GetIStorageCache(ctx context.Context) (cloudprovider.ICloudStoragecache, error) {
	storages := sc.getValidStorages()
	if len(storages) == 0 {
		msg := fmt.Sprintf("no storages for this storagecache %s(%s)???", sc.Name, sc.Id)
		log.Errorf("%v", msg)
		return nil, fmt.Errorf("%v", msg)
	}
	istorage, err := storages[0].GetIStorage(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetIStorages")
	}
	return istorage.GetIStoragecache(), nil
}

// 镜像缓存存储列表
func (manager *SStoragecacheManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.StoragecacheListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}

	if len(query.Path) > 0 {
		q = q.In("path", query.Path)
	}

	return q, nil
}

func (manager *SStoragecacheManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.StoragecacheListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}

	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SStoragecacheManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SStoragecacheManager) QueryDistinctExtraFields(q *sqlchemy.SQuery, resource string, fields []string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraFields(q, resource, fields)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SStoragecacheManager) FetchStoragecacheById(storageCacheId string) *SStoragecache {
	iStorageCache, _ := manager.FetchById(storageCacheId)
	if iStorageCache == nil {
		return nil
	}
	return iStorageCache.(*SStoragecache)
}

func (manager *SStoragecacheManager) GetCachePathById(storageCacheId string) string {
	iStorageCache, _ := manager.FetchById(storageCacheId)
	if iStorageCache == nil {
		return ""
	}
	sc := iStorageCache.(*SStoragecache)
	return sc.Path
}

func (sc *SStoragecache) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if sc.getCachedImageCount() > 0 {
		return httperrors.NewNotEmptyError("storage cache not empty")
	}
	storages := sc.getStorages()
	if len(storages) > 0 {
		return httperrors.NewNotEmptyError("referered by storages")
	}
	return sc.SStandaloneResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (sc *SStoragecache) PerformUncacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	imageStr, _ := data.GetString("image")
	if len(imageStr) == 0 {
		return nil, httperrors.NewMissingParameterError("image")
	}

	isForce := jsonutils.QueryBoolean(data, "is_force", false)

	var imageId string

	imgObj, err := CachedimageManager.FetchByIdOrName(ctx, nil, imageStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError2(CachedimageManager.Keyword(), imageStr)
		} else {
			return nil, httperrors.NewGeneralError(err)
		}
	} else {
		cachedImage := imgObj.(*SCachedimage)
		if cloudprovider.TImageType(cachedImage.ImageType) != cloudprovider.ImageTypeCustomized && !isForce {
			return nil, httperrors.NewForbiddenError("cannot uncache non-customized images")
		}
		imageId = imgObj.GetId()
		_, err := CachedimageManager.getImageInfo(ctx, userCred, imageStr, isForce)
		if err != nil && errors.Cause(err) != httperrors.ErrNotFound {
			log.Infof("get image %s info error %s", imageStr, err)
			if !isForce {
				return nil, errors.Wrapf(err, "get image %s info", imageStr)
			}
		}
	}

	scimg := StoragecachedimageManager.GetStoragecachedimage(sc.Id, imageId)
	if scimg == nil {
		return nil, httperrors.NewResourceNotFoundError("storage not cache image")
	}

	if scimg.Status == api.CACHED_IMAGE_STATUS_INIT || isForce {
		err = scimg.Detach(ctx, userCred)
		return nil, err
	}

	err = scimg.markDeleting(ctx, userCred, isForce)
	if err != nil {
		return nil, httperrors.NewInvalidStatusError("Fail to mark cache status: %s", err)
	}

	err = sc.StartImageUncacheTask(ctx, userCred, imageId, isForce, "")

	return nil, err
}

func (sc *SStoragecache) PerformCacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CacheImageInput) (jsonutils.JSONObject, error) {
	if len(input.ImageId) == 0 {
		return nil, httperrors.NewMissingParameterError("image_id")
	}

	image, err := CachedimageManager.getImageInfo(ctx, userCred, input.ImageId, input.IsForce)
	if err != nil {
		return nil, httperrors.NewImageNotFoundError(input.ImageId)
	}

	if len(image.Checksum) == 0 {
		return nil, httperrors.NewInvalidStatusError("Cannot cache image with no checksum")
	}

	input.ImageId = image.Id
	return nil, sc.StartImageCacheTask(ctx, userCred, input)
}

func (sc *SStoragecache) SyncCloudImages(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	iStoragecache cloudprovider.ICloudStoragecache,
	region *SCloudregion,
	xor bool,
) compare.SyncResult {
	lockman.LockObject(ctx, sc)
	defer lockman.ReleaseObject(ctx, sc)

	lockman.LockRawObject(ctx, CachedimageManager.Keyword(), sc.Id)
	defer lockman.ReleaseRawObject(ctx, CachedimageManager.Keyword(), sc.Id)

	result := compare.SyncResult{}

	driver, err := sc.GetProviderFactory()
	if err != nil {
		result.Error(errors.Wrapf(err, "GetProviderFactory(%s)", region.Provider))
		return result
	}
	if driver.IsPublicCloud() {
		err = func() error {
			err := region.SyncCloudImages(ctx, userCred, false, xor)
			if err != nil {
				return errors.Wrapf(err, "SyncCloudImages")
			}
			err = sc.CheckCloudimages(ctx, userCred, region.Name, region.Id)
			if err != nil {
				return errors.Wrapf(err, "CheckCloudimages")
			}
			return nil
		}()
		if err != nil {
			log.Errorf("sync public image error: %v", err)
		}

		log.Debugln("localCachedImages started")
		localCachedImages, err := sc.getCustomdCachedImages()
		if err != nil {
			result.Error(errors.Wrapf(err, "getCustomdCachedImages"))
			return result
		}
		log.Debugf("localCachedImages %d", len(localCachedImages))
		remoteImages, err := iStoragecache.GetICustomizedCloudImages()
		if err != nil {
			result.Error(errors.Wrapf(err, "GetICustomizedCloudImages"))
			return result
		}
		result = sc.syncCloudImages(ctx, userCred, localCachedImages, remoteImages, xor)
	} else {
		log.Debugln("localCachedImages started")
		localCachedImages, err := sc.getCachedImages()
		if err != nil {
			result.Error(errors.Wrapf(err, "getCachedImages"))
			return result
		}
		log.Debugf("localCachedImages %d", len(localCachedImages))
		remoteImages, err := iStoragecache.GetICloudImages()
		if err != nil {
			result.Error(errors.Wrapf(err, "GetICloudImages"))
			return result
		}
		result = sc.syncCloudImages(ctx, userCred, localCachedImages, remoteImages, xor)
	}

	return result
}

func (cache *SStoragecache) syncCloudImages(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	localCachedImages []SStoragecachedimage,
	remoteImages []cloudprovider.ICloudImage,
	xor bool,
) compare.SyncResult {
	syncResult := compare.SyncResult{}

	var syncOwnerId mcclient.IIdentityProvider

	provider := cache.GetCloudprovider()
	if provider != nil {
		syncOwnerId = provider.GetOwnerId()
	}

	removed := make([]SStoragecachedimage, 0)
	commondb := make([]SStoragecachedimage, 0)
	commonext := make([]cloudprovider.ICloudImage, 0)
	added := make([]cloudprovider.ICloudImage, 0)
	err := compare.CompareSets(localCachedImages, remoteImages, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(errors.Wrapf(err, "compare.CompareSets"))
		return syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err := removed[i].syncRemoveCloudImage(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	if !xor {
		for i := 0; i < len(commondb); i += 1 {
			err = commondb[i].syncWithCloudImage(ctx, userCred, syncOwnerId, commonext[i], cache.ManagerId)
			if err != nil {
				syncResult.UpdateError(err)
			} else {
				syncResult.Update()
			}
		}
	}
	for i := 0; i < len(added); i += 1 {
		err = StoragecachedimageManager.newFromCloudImage(ctx, userCred, syncOwnerId, added[i], cache)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncResult.Add()
		}
	}

	return syncResult
}

func (sc *SStoragecache) IsReachCapacityLimit(imageId string) bool {
	imgObj, _ := CachedimageManager.FetchById(imageId)
	if imgObj == nil {
		log.Debugf("no such cached image %s", imageId)
		return false
	}
	cachedImage := imgObj.(*SCachedimage)
	if cloudprovider.TImageType(cachedImage.ImageType) != cloudprovider.ImageTypeCustomized {
		// no need to cache
		log.Debugf("image %s is not a customized image, no need to cache", imageId)
		return false
	}
	cachedImages := sc.getCachedImageList(nil, string(cloudprovider.ImageTypeCustomized), []string{api.CACHED_IMAGE_STATUS_ACTIVE})
	for i := range cachedImages {
		if cachedImages[i].Id == imageId {
			// already cached
			log.Debugf("image %s has been cached in storage cache %s(%s)", imageId, sc.Id, sc.Name)
			return false
		}
	}
	host, _ := sc.GetMasterHost()
	if host == nil {
		return false
	}
	driver, _ := host.GetHostDriver()
	if driver == nil {
		return false
	}
	return driver.IsReachStoragecacheCapacityLimit(host, cachedImages)
}

func (sc *SStoragecache) GetStoragecachedimages() ([]SStoragecachedimage, error) {
	q := StoragecachedimageManager.Query().Equals("storagecache_id", sc.Id)
	ret := []SStoragecachedimage{}
	return ret, db.FetchModelObjects(StoragecachedimageManager, q, &ret)
}

func (sc *SStoragecache) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	scis, err := sc.GetStoragecachedimages()
	if err != nil {
		return errors.Wrapf(err, "GetStoragecachedimages")
	}
	for i := range scis {
		err := scis[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "delete storagecached images %d", scis[i].RowId)
		}
	}
	if len(sc.ManagerId) > 0 {
		return db.RealDeleteModel(ctx, userCred, sc)
	}
	return sc.SStandaloneResourceBase.Delete(ctx, userCred)
}

func (sc *SStoragecache) StartRelinquishLeastUsedCachedImageTask(ctx context.Context, userCred mcclient.TokenCredential, imageId string, parentTaskId string) error {
	cachedImages := sc.getCachedImageList([]string{imageId}, string(cloudprovider.ImageTypeCustomized), []string{api.CACHED_IMAGE_STATUS_ACTIVE})
	leastUsedIdx := -1
	leastRefCount := -1
	for i := range cachedImages {
		if leastRefCount < 0 || leastRefCount > cachedImages[i].RefCount {
			leastRefCount = cachedImages[i].RefCount
			leastUsedIdx = i
		}
	}
	return sc.StartImageUncacheTask(ctx, userCred, cachedImages[leastUsedIdx].GetId(), false, parentTaskId)
}

func (cache *SStoragecache) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	err := cache.SStandaloneResourceBase.CustomizeDelete(ctx, userCred, query, data)
	if err != nil {
		return err
	}
	if len(cache.ExternalId) > 0 {
		agentObj, err := BaremetalagentManager.FetchById(cache.ExternalId)
		if err == nil {
			agentObj.(*SBaremetalagent).setStoragecacheId("")
		} else if err != sql.ErrNoRows {
			return err
		}
	}
	return nil
}

func (manager *SStoragecacheManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}

func (sc *SStoragecache) linkCloudimages(ctx context.Context, regionName, regionId string) (int, error) {
	cloudimages := CloudimageManager.Query("external_id").Equals("cloudregion_id", regionId).SubQuery()
	sq := StoragecachedimageManager.Query("cachedimage_id").Equals("storagecache_id", sc.Id).SubQuery()
	q := CachedimageManager.Query().Equals("image_type", cloudprovider.ImageTypeSystem).In("external_id", cloudimages).NotIn("id", sq)
	images := []SCachedimage{}
	err := db.FetchModelObjects(CachedimageManager, q, &images)
	if err != nil {
		return 0, errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range images {
		sci := &SStoragecachedimage{}
		sci.SetModelManager(StoragecachedimageManager, sci)
		sci.StoragecacheId = sc.Id
		sci.CachedimageId = images[i].Id
		sci.Status = api.CACHED_IMAGE_STATUS_ACTIVE
		err = StoragecachedimageManager.TableSpec().Insert(ctx, sci)
		if err != nil {
			return 0, errors.Wrapf(err, "Insert")
		}
	}
	return len(images), nil
}

func (sc *SStoragecache) unlinkCloudimages(ctx context.Context, userCred mcclient.TokenCredential, regionName, regionId string) (int, error) {
	cloudimages := CloudimageManager.Query("external_id").Equals("cloudregion_id", regionId).SubQuery()
	sq := CachedimageManager.Query("id").Equals("image_type", cloudprovider.ImageTypeSystem).NotIn("external_id", cloudimages).SubQuery()
	q := StoragecachedimageManager.Query().Equals("storagecache_id", sc.Id).In("cachedimage_id", sq)
	scis := []SStoragecachedimage{}
	err := db.FetchModelObjects(StoragecachedimageManager, q, &scis)
	if err != nil {
		return 0, errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range scis {
		err = scis[i].Delete(ctx, userCred)
		if err != nil {
			log.Warningf("detach image %v error: %v", scis[i].GetCachedimage(), err)
		}
	}
	return len(scis), nil
}

func (sc *SStoragecache) updateSystemImageStatus() (int, error) {
	sq := CachedimageManager.Query("id").Equals("image_type", cloudprovider.ImageTypeSystem)
	q := StoragecachedimageManager.Query().
		Equals("storagecache_id", sc.Id).In("cachedimage_id", sq.SubQuery()).
		NotEquals("status", api.CACHED_IMAGE_STATUS_ACTIVE)
	scis := []SStoragecachedimage{}
	err := db.FetchModelObjects(StoragecachedimageManager, q, &scis)
	if err != nil {
		return 0, errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range scis {
		db.Update(&scis[i], func() error {
			scis[i].Status = api.CACHED_IMAGE_STATUS_ACTIVE
			return nil
		})
	}
	return len(scis), nil
}

func (sc *SStoragecache) getSystemImageCount() (int, error) {
	sq := StoragecachedimageManager.Query("cachedimage_id").Equals("storagecache_id", sc.Id)
	q := CachedimageManager.Query().Equals("image_type", cloudprovider.ImageTypeSystem).In("id", sq.SubQuery())
	return q.CountWithError()
}

func (sc *SStoragecache) CheckCloudimages(ctx context.Context, userCred mcclient.TokenCredential, regionName, regionId string) error {
	lockman.LockRawObject(ctx, CachedimageManager.Keyword(), regionId)
	defer lockman.ReleaseRawObject(ctx, CachedimageManager.Keyword(), regionId)

	result := compare.SyncResult{}

	var err error
	result.DelCnt, err = sc.unlinkCloudimages(ctx, userCred, regionName, regionId)
	if err != nil {
		return errors.Wrapf(err, "unlinkCloudimages")
	}
	sc.updateSystemImageStatus()
	result.UpdateCnt, err = sc.getSystemImageCount()
	if err != nil {
		log.Errorf("getSystemImageCount error: %v", err)
	}
	result.AddCnt, err = sc.linkCloudimages(ctx, regionName, regionId)
	if err != nil {
		return errors.Wrapf(err, "linkCloudimages")
	}
	log.Infof("SycSystemImages for region %s(%s) storagecache %s result: %s", regionName, regionId, sc.Name, result.Result())
	return nil
}

func (manager *SStoragecacheManager) FetchStoragecaches(filter func(q *sqlchemy.SQuery) *sqlchemy.SQuery) ([]SStoragecache, error) {
	q := manager.Query()
	q = filter(q)

	storageCaches := make([]SStoragecache, 0)
	err := db.FetchModelObjects(manager, q, &storageCaches)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}

	return storageCaches, nil
}

func (manager *SStoragecacheManager) FetchStoragecachesByFilters(ctx context.Context, filters api.SStorageCacheFilters) ([]SStoragecache, error) {
	return manager.FetchStoragecaches(func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		storagesQ := StorageManager.Query().IsTrue("enabled").Equals("status", api.STORAGE_ONLINE)
		if len(filters.StorageType) > 0 || !filters.StorageTags.IsEmpty() {
			if len(filters.StorageType) > 0 {
				storagesQ = storagesQ.In("storage_type", filters.StorageType)
			}
			if !filters.StorageTags.IsEmpty() {
				storagesQ = db.ObjectIdQueryWithTagFilters(ctx, storagesQ, "id", StorageManager.Keyword(), filters.StorageTags)
			}
		}
		hostsQ := HostManager.Query().IsTrue("enabled").Equals("status", api.HOST_STATUS_RUNNING).Equals("host_status", api.HOST_ONLINE)
		if len(filters.HostType) > 0 || !filters.HostTags.IsEmpty() {
			if len(filters.HostType) > 0 {
				hostsQ = hostsQ.In("host_type", filters.HostType)
			}
			if !filters.HostTags.IsEmpty() {
				hostsQ = db.ObjectIdQueryWithTagFilters(ctx, hostsQ, "id", HostManager.Keyword(), filters.HostTags)
			}
		}
		hosts := hostsQ.SubQuery()
		hostStorages := HoststorageManager.Query().SubQuery()
		storages := storagesQ.SubQuery()
		q = q.Join(storages, sqlchemy.Equals(storages.Field("storagecache_id"), q.Field("id")))
		q = q.Join(hostStorages, sqlchemy.Equals(storages.Field("id"), hostStorages.Field("storage_id")))
		q = q.Join(hosts, sqlchemy.Equals(hostStorages.Field("host_id"), hosts.Field("id")))
		return q.Distinct()
	})
}
