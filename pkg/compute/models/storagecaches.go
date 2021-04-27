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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
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

func (self *SStoragecache) GetEsxiAgentHostDesc() (*jsonutils.JSONDict, error) {
	if !strings.Contains(self.Name, "esxiagent") {
		return nil, nil
	}
	obj, err := BaremetalagentManager.FetchById(self.ExternalId)
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

func (self *SStoragecache) GetHost() (*SHost, error) {
	hostId, err := self.getHostId()
	if err != nil {
		return nil, errors.Wrap(err, "self.getHostId")
	}
	if len(hostId) == 0 {
		return nil, nil
	}

	host, err := HostManager.FetchById(hostId)
	if err != nil {
		return nil, errors.Wrap(err, "HostManager.FetchById")
	} else if host == nil {
		return nil, nil
	}
	return host.(*SHost), nil
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

func (self *SStoragecache) GetHosts() ([]SHost, error) {
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
		Join(storages, sqlchemy.AND(sqlchemy.Equals(storages.Field("storagecache_id"), self.Id),
			sqlchemy.In(storages.Field("status"), []string{api.STORAGE_ENABLED, api.STORAGE_ONLINE}),
			sqlchemy.IsTrue(storages.Field("enabled")))).
		Filter(sqlchemy.Equals(hoststorages.Field("storage_id"), storages.Field("id"))).All(&hosts)
	if err != nil {
		return nil, err
	}
	return hosts, nil
}

func (self *SStoragecache) getHostId() (string, error) {
	hosts, err := self.GetHosts()
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
	ret, _ := ring.GetNode(self.Id)
	return ret, nil
}

func (manager *SStoragecacheManager) SyncWithCloudStoragecache(ctx context.Context, userCred mcclient.TokenCredential, cloudCache cloudprovider.ICloudStoragecache, provider *SCloudprovider) (*SStoragecache, bool, error) {
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
		localCache.syncWithCloudStoragecache(ctx, userCred, cloudCache, provider)
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

func (self *SStoragecache) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.StoragecacheDetails, error) {
	return api.StoragecacheDetails{}, nil
}

func (self *SStoragecache) getMoreDetails(ctx context.Context, out api.StoragecacheDetails) api.StoragecacheDetails {
	out.Storages = self.getStorageNames()
	out.Size = self.getCachedImageSize()
	out.Count = self.getCachedImageCount()

	host, _ := self.GetHost()
	if host != nil {
		out.Host = host.GetShortDesc(ctx)
	}
	return out
}

func (self *SStoragecache) getCachedImageList(excludeIds []string, imageType string, status []string) []SCachedimage {
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

func (self *SStoragecache) getCachedImages() ([]SStoragecachedimage, error) {
	images := make([]SStoragecachedimage, 0)
	q := StoragecachedimageManager.Query().Equals("storagecache_id", self.Id)
	err := db.FetchModelObjects(StoragecachedimageManager, q, &images)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return images, nil
}

func (self *SStoragecache) getCustomdCachedImages() ([]SStoragecachedimage, error) {
	images := make([]SStoragecachedimage, 0)
	sq := CachedimageManager.Query("id").Equals("image_type", "customized").SubQuery()
	q := StoragecachedimageManager.Query().Equals("storagecache_id", self.Id).In("cachedimage_id", sq)
	err := db.FetchModelObjects(StoragecachedimageManager, q, &images)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return images, nil
}

func (self *SStoragecache) getCachedImageCount() int {
	images, _ := self.getCachedImages()
	return len(images)
}

func (self *SStoragecache) getCachedImageSize() int64 {
	images, _ := self.getCachedImages()
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

func (self *SStoragecache) StartImageCacheTask(ctx context.Context, userCred mcclient.TokenCredential, imageId string, format string, isForce bool, parentTaskId string) error {
	return self.StartImageCacheTaskFromHost(ctx, userCred, imageId, format, isForce, "", parentTaskId)
}

func (self *SStoragecache) StartImageCacheTaskFromHost(ctx context.Context, userCred mcclient.TokenCredential, imageId string, format string, isForce bool, srcHostId string, parentTaskId string) error {
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

	if srcHostId != "" {
		data.Add(jsonutils.NewString(srcHostId), "source_host_id")
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
	storages := self.getStorages()
	if len(storages) > 0 {
		return httperrors.NewNotEmptyError("referered by storages")
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
		if cloudprovider.TImageType(cachedImage.ImageType) != cloudprovider.ImageTypeCustomized && !isForce {
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

func (self *SStoragecache) SyncCloudImages(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	iStoragecache cloudprovider.ICloudStoragecache,
	region *SCloudregion,
) compare.SyncResult {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	lockman.LockRawObject(ctx, "cachedimages", self.Id)
	defer lockman.ReleaseRawObject(ctx, "cachedimages", self.Id)

	result := compare.SyncResult{}

	driver, err := self.GetProviderFactory()
	if err != nil {
		result.Error(errors.Wrapf(err, "GetProviderFactory(%s)", region.Provider))
		return result
	}
	if driver.IsPublicCloud() {
		err = func() error {
			err := region.SyncCloudImages(ctx, userCred, false)
			if err != nil {
				return errors.Wrapf(err, "SyncCloudImages")
			}
			err = self.CheckCloudimages(ctx, userCred, region.Name, region.Id)
			if err != nil {
				return errors.Wrapf(err, "CheckCloudimages")
			}
			return nil
		}()
		if err != nil {
			log.Errorf("sync public image error: %v", err)
		}

		log.Debugln("localCachedImages started")
		localCachedImages, err := self.getCustomdCachedImages()
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
		result = self.syncCloudImages(ctx, userCred, localCachedImages, remoteImages)
	} else {
		log.Debugln("localCachedImages started")
		localCachedImages, err := self.getCachedImages()
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
		result = self.syncCloudImages(ctx, userCred, localCachedImages, remoteImages)
	}

	return result
}

func (cache *SStoragecache) syncCloudImages(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	localCachedImages []SStoragecachedimage,
	remoteImages []cloudprovider.ICloudImage,
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
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].syncWithCloudImage(ctx, userCred, syncOwnerId, commonext[i], cache.ManagerId)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncResult.Update()
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

func (self *SStoragecache) IsReachCapacityLimit(imageId string) bool {
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
	cachedImages := self.getCachedImageList(nil, string(cloudprovider.ImageTypeCustomized), []string{api.CACHED_IMAGE_STATUS_ACTIVE})
	for i := range cachedImages {
		if cachedImages[i].Id == imageId {
			// already cached
			log.Debugf("image %s has been cached in storage cache %s(%s)", imageId, self.Id, self.Name)
			return false
		}
	}
	host, _ := self.GetHost()
	return host.GetHostDriver().IsReachStoragecacheCapacityLimit(host, cachedImages)
}

func (self *SStoragecache) StartRelinquishLeastUsedCachedImageTask(ctx context.Context, userCred mcclient.TokenCredential, imageId string, parentTaskId string) error {
	cachedImages := self.getCachedImageList([]string{imageId}, string(cloudprovider.ImageTypeCustomized), []string{api.CACHED_IMAGE_STATUS_ACTIVE})
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

func (self *SStoragecache) linkCloudimages(ctx context.Context, regionName, regionId string) (int, error) {
	cloudimages := CloudimageManager.Query("external_id").Equals("cloudregion_id", regionId).SubQuery()
	sq := StoragecachedimageManager.Query("cachedimage_id").Equals("storagecache_id", self.Id).SubQuery()
	q := CachedimageManager.Query().Equals("image_type", cloudprovider.ImageTypeSystem).In("external_id", cloudimages).NotIn("id", sq)
	images := []SCachedimage{}
	err := db.FetchModelObjects(CachedimageManager, q, &images)
	if err != nil {
		return 0, errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range images {
		sci := &SStoragecachedimage{}
		sci.SetModelManager(StoragecachedimageManager, sci)
		sci.StoragecacheId = self.Id
		sci.CachedimageId = images[i].Id
		sci.Status = api.CACHED_IMAGE_STATUS_ACTIVE
		err = StoragecachedimageManager.TableSpec().Insert(ctx, sci)
		if err != nil {
			return 0, errors.Wrapf(err, "Insert")
		}
	}
	return len(images), nil
}

func (self *SStoragecache) unlinkCloudimages(ctx context.Context, userCred mcclient.TokenCredential, regionName, regionId string) (int, error) {
	cloudimages := CloudimageManager.Query("external_id").Equals("cloudregion_id", regionId).SubQuery()
	sq := CachedimageManager.Query("id").Equals("image_type", cloudprovider.ImageTypeSystem).NotIn("external_id", cloudimages).SubQuery()
	q := StoragecachedimageManager.Query().Equals("storagecache_id", self.Id).In("cachedimage_id", sq)
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

func (self *SStoragecache) getSystemImageCount() (int, error) {
	sq := StoragecachedimageManager.Query("cachedimage_id").Equals("storagecache_id", self.Id)
	q := CachedimageManager.Query().Equals("image_type", cloudprovider.ImageTypeSystem).In("id", sq.SubQuery())
	return q.CountWithError()
}

func (self *SStoragecache) CheckCloudimages(ctx context.Context, userCred mcclient.TokenCredential, regionName, regionId string) error {
	lockman.LockRawObject(ctx, "cachedimages", regionId)
	defer lockman.ReleaseRawObject(ctx, "cachedimages", regionId)

	result := compare.SyncResult{}

	var err error
	result.DelCnt, err = self.unlinkCloudimages(ctx, userCred, regionName, regionId)
	if err != nil {
		return errors.Wrapf(err, "unlinkCloudimages")
	}
	result.UpdateCnt, err = self.getSystemImageCount()
	if err != nil {
		log.Errorf("getSystemImageCount error: %v", err)
	}
	result.AddCnt, err = self.linkCloudimages(ctx, regionName, regionId)
	if err != nil {
		return errors.Wrapf(err, "linkCloudimages")
	}
	log.Infof("SycSystemImages for region %s(%s) storagecache %s result: %s", regionName, regionId, self.Name, result.Result())
	return nil
}
