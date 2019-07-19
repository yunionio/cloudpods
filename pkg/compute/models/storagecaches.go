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

	"github.com/serialx/hashring"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/imagetools"
)

type SStoragecacheManager struct {
	db.SStandaloneResourceBaseManager
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

	Path string `width:"256" charset:"utf8" nullable:"true" list:"user" update:"admin" create:"admin_optional"` // = Column(VARCHAR(256, charset='utf8'), nullable=True)
}

func (self *SStoragecacheManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SStoragecacheManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SStoragecache) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SStoragecache) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SStoragecache) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (self *SStoragecache) getStorages() []SStorage {
	storages := make([]SStorage, 0)
	q := StorageManager.Query().Equals("storagecache_id", self.Id)
	err := db.FetchModelObjects(StorageManager, q, &storages)
	if err != nil {
		return nil
	}
	return storages
}

func (self *SStoragecache) getValidStorages() []SStorage {
	storages := []SStorage{}
	q := StorageManager.Query()
	q = q.Equals("storagecache_id", self.Id).
		Filter(sqlchemy.In(q.Field("status"), []string{api.STORAGE_ENABLED, api.STORAGE_ONLINE})).
		Filter(sqlchemy.IsTrue(q.Field("enabled")))
	err := db.FetchModelObjects(StorageManager, q, &storages)
	if err != nil {
		return nil
	}
	return storages
}

func (self *SStoragecache) getStorageNames() []string {
	storages := self.getStorages()
	if storages == nil {
		return nil
	}
	names := make([]string, len(storages))
	for i := 0; i < len(storages); i += 1 {
		names[i] = storages[i].Name
	}
	return names
}

func (self *SStoragecache) GetHost() (*SHost, error) {
	hostId, err := self.getHostId()
	if err != nil {
		return nil, err
	}
	if len(hostId) == 0 {
		return nil, nil
	}

	host, err := HostManager.FetchById(hostId)
	if err != nil {
		return nil, err
	} else if host == nil {
		return nil, nil
	}
	h, _ := host.(*SHost)
	return h, nil
}

func (self *SStoragecache) GetRegion() (*SCloudregion, error) {
	host, err := self.GetHost()
	if err != nil {
		return nil, err
	}
	region := host.GetRegion()
	if region == nil {
		return nil, fmt.Errorf("failed to get region for host %s(%s)", host.Name, host.Id)
	}
	return region, nil
}

func (self *SStoragecache) getHostId() (string, error) {
	hoststorages := HoststorageManager.Query().SubQuery()
	storages := StorageManager.Query().SubQuery()

	hosts := make([]SHost, 0)
	host := HostManager.Query().SubQuery()
	q := host.Query(host.Field("id"))
	err := q.Join(hoststorages, sqlchemy.AND(sqlchemy.Equals(hoststorages.Field("host_id"), host.Field("id")),
		sqlchemy.Equals(host.Field("host_status"), api.HOST_ONLINE),
		sqlchemy.IsTrue(host.Field("enabled")))).
		Join(storages, sqlchemy.AND(sqlchemy.Equals(storages.Field("storagecache_id"), self.Id),
			sqlchemy.In(storages.Field("status"), []string{api.STORAGE_ENABLED, api.STORAGE_ONLINE}),
			sqlchemy.IsTrue(storages.Field("enabled")))).
		Filter(sqlchemy.Equals(hoststorages.Field("storage_id"), storages.Field("id"))).All(&hosts)
	if err != nil {
		return "", err
	}

	hostIds := make([]string, 0)
	for _, h := range hosts {
		hostIds = append(hostIds, h.Id)
	}

	if len(hostIds) == 0 {
		return "", nil
	}
	ring := hashring.New(hostIds)
	ret, _ := ring.GetNode(self.Id)
	return ret, nil
}

func (manager *SStoragecacheManager) SyncWithCloudStoragecache(ctx context.Context, userCred mcclient.TokenCredential, cloudCache cloudprovider.ICloudStoragecache, provider *SCloudprovider) (*SStoragecache, bool, error) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	localCacheObj, err := db.FetchByExternalId(manager, cloudCache.GetGlobalId())
	if err != nil {
		if err == sql.ErrNoRows {
			localCache, err := manager.newFromCloudStoragecache(ctx, userCred, cloudCache, provider)
			if err != nil {
				return nil, false, err
			} else {
				return localCache, true, nil
			}
		} else {
			log.Errorf("%s", err)
			return nil, false, err
		}
	} else {
		localCache := localCacheObj.(*SStoragecache)
		localCache.syncWithCloudStoragecache(ctx, userCred, cloudCache, provider)
		return localCache, false, nil
	}
}

func (manager *SStoragecacheManager) newFromCloudStoragecache(ctx context.Context, userCred mcclient.TokenCredential, cloudCache cloudprovider.ICloudStoragecache, provider *SCloudprovider) (*SStoragecache, error) {
	local := SStoragecache{}
	local.SetModelManager(manager, &local)

	newName, err := db.GenerateName(manager, userCred, cloudCache.GetName())
	if err != nil {
		return nil, err
	}
	local.Name = newName
	local.ExternalId = cloudCache.GetGlobalId()

	local.IsEmulated = cloudCache.IsEmulated()
	local.ManagerId = provider.Id

	local.Path = cloudCache.GetPath()

	err = manager.TableSpec().Insert(&local)
	if err != nil {
		return nil, err
	}

	db.OpsLog.LogEvent(&local, db.ACT_CREATE, local.GetShortDesc(ctx), userCred)

	return &local, nil
}

func (self *SStoragecache) syncWithCloudStoragecache(ctx context.Context, userCred mcclient.TokenCredential, cloudCache cloudprovider.ICloudStoragecache, provider *SCloudprovider) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Name = cloudCache.GetName()

		self.Path = cloudCache.GetPath()

		self.IsEmulated = cloudCache.IsEmulated()
		self.ManagerId = provider.Id

		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (self *SStoragecache) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	extra = self.getMoreDetails(extra)
	return extra, nil
}

func (self *SStoragecache) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	extra = self.getMoreDetails(extra)
	return extra
}

func (self *SStoragecache) getCachedImageList(excludeIds []string, imageType string) []SCachedimage {
	images := make([]SCachedimage, 0)

	cachedImages := CachedimageManager.Query().SubQuery()
	storagecachedImages := StoragecachedimageManager.Query().SubQuery()

	q := cachedImages.Query()
	q = q.Join(storagecachedImages, sqlchemy.Equals(cachedImages.Field("id"), storagecachedImages.Field("cachedimage_id")))
	q = q.Filter(sqlchemy.Equals(storagecachedImages.Field("storagecache_id"), self.Id))

	if len(excludeIds) > 0 {
		q = q.Filter(sqlchemy.NotIn(cachedImages.Field("id"), excludeIds))
	}
	if len(imageType) > 0 {
		q = q.Filter(sqlchemy.Equals(cachedImages.Field("image_type"), imageType))
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

func (self *SStoragecache) getCachedImages() []SStoragecachedimage {
	images := make([]SStoragecachedimage, 0)
	q := StoragecachedimageManager.Query().Equals("storagecache_id", self.Id)
	err := db.FetchModelObjects(StoragecachedimageManager, q, &images)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	return images
}

func (self *SStoragecache) getCachedImageCount() int {
	images := self.getCachedImages()
	return len(images)
}

func (self *SStoragecache) getCachedImageSize() int64 {
	images := self.getCachedImages()
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

func (self *SStoragecache) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	extra.Add(jsonutils.NewStringArray(self.getStorageNames()), "storages")
	extra.Add(jsonutils.NewInt(self.getCachedImageSize()), "size")
	extra.Add(jsonutils.NewInt(int64(self.getCachedImageCount())), "count")
	return extra
}

func (self *SStoragecache) StartImageCacheTask(ctx context.Context, userCred mcclient.TokenCredential, imageId string, format string, isForce bool, parentTaskId string) error {
	StoragecachedimageManager.Register(ctx, userCred, self.Id, imageId, "")
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(imageId), "image_id")
	if len(format) > 0 {
		data.Add(jsonutils.NewString(format), "format")
	}

	image, _ := CachedimageManager.GetImageById(ctx, userCred, imageId, false)

	if image != nil {
		imgInfo := imagetools.NormalizeImageInfo(image.Name, image.Properties["os_arch"], image.Properties["os_type"],
			image.Properties["os_distribution"], image.Properties["os_version"])
		data.Add(jsonutils.NewString(imgInfo.OsType), "os_type")
		data.Add(jsonutils.NewString(imgInfo.OsArch), "os_arch")
		data.Add(jsonutils.NewString(imgInfo.OsDistro), "os_distribution")
		data.Add(jsonutils.NewString(imgInfo.OsVersion), "os_version")
		data.Add(jsonutils.NewString(imgInfo.OsFullVersion), "os_full_version")
		data.Add(jsonutils.NewString(image.Name), "image_name")
	}

	if isForce {
		data.Add(jsonutils.JSONTrue, "is_force")
	}
	task, err := taskman.TaskManager.NewTask(ctx, "StorageCacheImageTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		log.Errorf("create StorageCacheImageTask fail %s", err)
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SStoragecache) StartImageUncacheTask(ctx context.Context, userCred mcclient.TokenCredential, imageId string, isPurge bool, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(imageId), "image_id")
	if isPurge {
		data.Add(jsonutils.JSONTrue, "is_purge")
	}
	task, err := taskman.TaskManager.NewTask(ctx, "StorageUncacheImageTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SStoragecache) GetIStorageCache() (cloudprovider.ICloudStoragecache, error) {
	storages := self.getValidStorages()
	if len(storages) == 0 {
		msg := fmt.Sprintf("no storages for this storagecache %s(%s)???", self.Name, self.Id)
		log.Errorf(msg)
		return nil, fmt.Errorf(msg)
	}
	istorage, err := storages[0].GetIStorage()
	if err != nil {
		log.Errorf("fail to find istorage for storage %s", err)
		return nil, err
	}
	return istorage.GetIStoragecache(), nil
}

func (manager *SStoragecacheManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
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

	q, err = manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	return q, nil
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

func (self *SStoragecache) ValidateDeleteCondition(ctx context.Context) error {
	if self.getCachedImageCount() > 0 {
		return httperrors.NewNotEmptyError("storage cache not empty")
	}
	return self.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SStoragecache) AllowPerformUncacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "uncache-image")
}

func (self *SStoragecache) PerformUncacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	imageStr, _ := data.GetString("image")
	if len(imageStr) == 0 {
		return nil, httperrors.NewMissingParameterError("image")
	}

	isForce := jsonutils.QueryBoolean(data, "is_force", false)

	var imageId string

	imgObj, err := CachedimageManager.FetchByIdOrName(nil, imageStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError2(CachedimageManager.Keyword(), imageStr)
		} else {
			return nil, httperrors.NewGeneralError(err)
		}
	} else {
		cachedImage := imgObj.(*SCachedimage)
		if cachedImage.ImageType != cloudprovider.CachedImageTypeCustomized && !isForce {
			return nil, httperrors.NewForbiddenError("cannot uncache non-customized images")
		}
		imageId = imgObj.GetId()
		_, err := CachedimageManager.getImageInfo(ctx, userCred, imageStr, isForce)
		if err != nil {
			log.Infof("image %s not found %s", imageStr, err)
			if !isForce {
				return nil, httperrors.NewImageNotFoundError(imageStr)
			}
		}
	}

	scimg := StoragecachedimageManager.GetStoragecachedimage(self.Id, imageId)
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

	err = self.StartImageUncacheTask(ctx, userCred, imageId, isForce, "")

	return nil, err
}

func (self *SStoragecache) AllowPerformCacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "cache-image")
}

func (self *SStoragecache) PerformCacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	imageStr, _ := data.GetString("image")
	if len(imageStr) == 0 {
		return nil, httperrors.NewInputParameterError("missing image id or name")
	}
	isForce := jsonutils.QueryBoolean(data, "is_force", false)

	image, err := CachedimageManager.getImageInfo(ctx, userCred, imageStr, isForce)
	if err != nil {
		log.Infof("image %s not found %s", imageStr, err)
		return nil, httperrors.NewImageNotFoundError(imageStr)
	}

	if len(image.Checksum) == 0 {
		return nil, httperrors.NewInvalidStatusError("Cannot cache image with no checksum")
	}

	format, _ := data.GetString("format")

	err = self.StartImageCacheTask(ctx, userCred, image.Id, format, isForce, "")
	return nil, err
}

func (cache *SStoragecache) SyncCloudImages(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	iStoragecache cloudprovider.ICloudStoragecache,
) compare.SyncResult {
	lockman.LockObject(ctx, cache)
	defer lockman.ReleaseObject(ctx, cache)

	lockman.LockClass(ctx, StoragecachedimageManager, db.GetLockClassKey(StoragecachedimageManager, userCred))
	defer lockman.ReleaseClass(ctx, StoragecachedimageManager, db.GetLockClassKey(StoragecachedimageManager, userCred))

	syncResult := compare.SyncResult{}

	localCachedImages := cache.getCachedImages()
	log.Debugf("localCachedImages %d", len(localCachedImages))

	remoteImages, err := iStoragecache.GetIImages()
	if err != nil {
		log.Errorf("fail to get images %s", err)
		syncResult.Error(err)
		return syncResult
	}

	removed := make([]SStoragecachedimage, 0)
	commondb := make([]SStoragecachedimage, 0)
	commonext := make([]cloudprovider.ICloudImage, 0)
	added := make([]cloudprovider.ICloudImage, 0)

	err = compare.CompareSets(localCachedImages, remoteImages, &removed, &commondb, &commonext, &added)
	if err != nil {
		log.Errorf("compare.CompareSets error %s", err)
		syncResult.Error(err)
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
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].syncWithCloudImage(ctx, userCred, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		err = StoragecachedimageManager.newFromCloudImage(ctx, userCred, added[i], cache)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncResult.Add()
		}
	}

	return syncResult
}

func (self *SStoragecache) IsReachCapacityLimit(imageId string) bool {
	imgObj, _ := CachedimageManager.FetchById(imageId)
	if imgObj == nil {
		return false
	}
	cachedImage := imgObj.(*SCachedimage)
	if cachedImage.ImageType != cloudprovider.CachedImageTypeCustomized {
		// no need to cache
		return false
	}
	cachedImages := self.getCachedImageList([]string{imageId}, cloudprovider.CachedImageTypeCustomized)
	host, _ := self.GetHost()
	return host.GetHostDriver().IsReachStoragecacheCapacityLimit(host, cachedImages)
}

func (self *SStoragecache) StartRelinquishLeastUsedCachedImageTask(ctx context.Context, userCred mcclient.TokenCredential, imageId string, parentTaskId string) error {
	cachedImages := self.getCachedImageList([]string{imageId}, cloudprovider.CachedImageTypeCustomized)
	leastUsedIdx := -1
	leastRefCount := -1
	for i := range cachedImages {
		if leastRefCount < 0 || leastRefCount > cachedImages[i].RefCount {
			leastRefCount = cachedImages[i].RefCount
			leastUsedIdx = i
		}
	}
	return self.StartImageUncacheTask(ctx, userCred, cachedImages[leastUsedIdx].GetId(), false, parentTaskId)
}
