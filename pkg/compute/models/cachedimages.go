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
	"math/rand"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SCachedimageManager struct {
	db.SSharableVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
}

var CachedimageManager *SCachedimageManager

func init() {
	CachedimageManager = &SCachedimageManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SCachedimage{},
			"cachedimages_tbl",
			"cachedimage",
			"cachedimages",
		),
	}
	CachedimageManager.SetVirtualObject(CachedimageManager)
}

type SCachedimage struct {
	db.SSharableVirtualResourceBase
	db.SExternalizedResourceBase

	// 镜像大小单位: Byte
	// example: 53687091200
	Size int64 `nullable:"false" list:"user" update:"admin" create:"admin_required"`

	// 镜像详情信息
	// example: {"deleted":false,"disk_format":"qcow2","id":"img-a6uucnfl","is_public":true,"min_disk":51200,"min_ram":0,"name":"FreeBSD 11.1 64bit","properties":{"os_arch":"x86_64","os_distribution":"FreeBSD","os_type":"FreeBSD","os_version":"11"},"protected":true,"size":53687091200,"status":"active"}
	Info jsonutils.JSONObject `nullable:"true" list:"user" update:"admin" create:"admin_required"`

	// 上此同步时间
	// example: 2020-01-17T05:28:54.000000Z
	LastSync time.Time `list:"admin"`

	// 最近一次缓存引用时间
	// 2020-01-17T05:20:54.000000Z
	LastRef time.Time `list:"admin"`

	// 引用次数
	// example: 0
	RefCount int `default:"0" list:"user"`

	// 是否支持UEFI
	// example: false
	UEFI tristate.TriState `default:"false" list:"user"`

	// 镜像类型, system: 公有云镜像, customized: 自定义镜像
	// example: system
	ImageType string `width:"16" default:"customized" list:"user" index:"true"`
}

func (self SCachedimage) GetGlobalId() string {
	return self.ExternalId
}

func (self *SCachedimage) ValidateDeleteCondition(ctx context.Context) error {
	cnt, err := self.getStoragecacheCount()
	if err != nil {
		return httperrors.NewInternalServerError("ValidateDeleteCondition error %s", err)
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("The image has been cached on storages")
	}
	if self.GetStatus() == api.CACHED_IMAGE_STATUS_ACTIVE && !self.isReferenceSessionExpire() {
		return httperrors.NewConflictError("the image reference session has not been expired!")
	}
	return self.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SCachedimage) isReferenceSessionExpire() bool {
	if !self.LastRef.IsZero() && time.Now().Sub(self.LastRef) < api.CACHED_IMAGE_REFERENCE_SESSION_EXPIRE_SECONDS*time.Second {
		return false
	} else {
		return true
	}
}

func (self *SCachedimage) isRefreshSessionExpire() bool {
	if len(self.ExternalId) > 0 { // external image info never expires
		return false
	}
	if !self.LastRef.IsZero() && time.Now().Sub(self.LastRef) < api.CACHED_IMAGE_REFRESH_SECONDS*time.Second {
		return false
	} else {
		return true
	}
}

func (self *SCachedimage) GetName() string {
	name, _ := self.Info.GetString("name")
	return name
}

func (self *SCachedimage) GetOwner() string {
	owner, _ := self.Info.GetString("owner")
	return owner
}

func (self *SCachedimage) GetFormat() string {
	format, _ := self.Info.GetString("disk_format")
	return format
}

func (self *SCachedimage) GetStatus() string {
	status, _ := self.Info.GetString("status")
	return status
}

func (self *SCachedimage) GetOSType() string {
	osType, _ := self.Info.GetString("properties", "os_type")
	return osType
}

func (self *SCachedimage) GetOSDistribution() string {
	osType, _ := self.Info.GetString("properties", "os_distribution")
	return osType
}

func (self *SCachedimage) GetOSVersion() string {
	osType, _ := self.Info.GetString("properties", "os_version")
	return osType
}

func (self *SCachedimage) GetHypervisor() string {
	osType, _ := self.Info.GetString("properties", "hypervisor")
	return osType
}

func (self *SCachedimage) GetChecksum() string {
	checksum, _ := self.Info.GetString("checksum")
	return checksum
}

func (self *SCachedimage) getStoragecacheQuery() *sqlchemy.SQuery {
	q := StoragecachedimageManager.Query().Equals("cachedimage_id", self.Id)
	return q
}

func (self *SCachedimage) getStoragecacheCount() (int, error) {
	return self.getStoragecacheQuery().CountWithError()
}

func (self *SCachedimage) GetImage() (*cloudprovider.SImage, error) {
	image := cloudprovider.SImage{
		ExternalId: self.ExternalId,
	}

	err := self.Info.Unmarshal(&image)
	if err != nil {
		log.Errorf("unmarshal %s fail %s", self.Info, err)
		return nil, errors.Wrap(err, "self.Info.Unmarshal")
	} else {
		// hack, make cached image ID consistent
		image.Id = self.Id
		return &image, nil
	}
}

func (manager *SCachedimageManager) cacheGlanceImageInfo(ctx context.Context, userCred mcclient.TokenCredential, info jsonutils.JSONObject) (*SCachedimage, error) {
	lockman.LockRawObject(ctx, manager.Keyword(), "name")
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

	img := struct {
		Id          string
		Size        int64
		Name        string
		Status      string
		IsPublic    bool
		ProjectId   string `json:"tenant_id"`
		PublicScope string
	}{}
	err := info.Unmarshal(&img)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal %s", info.String())
	}
	if len(img.Id) == 0 {
		return nil, fmt.Errorf("invalid image info")
	}
	if len(img.Name) == 0 {
		img.Name = img.Id
	}

	imageCache := SCachedimage{}
	imageCache.SetModelManager(manager, &imageCache)

	img.Name, err = db.GenerateName(ctx, manager, nil, img.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GenerateName(%s)", img.Name)
	}

	err = manager.RawQuery().Equals("id", img.Id).First(&imageCache)
	if err != nil {
		if err == sql.ErrNoRows { // insert
			imageCache.Id = img.Id
			imageCache.Name = img.Name
			imageCache.Size = img.Size
			imageCache.Info = info
			imageCache.Status = img.Status
			imageCache.IsPublic = img.IsPublic
			imageCache.PublicScope = img.PublicScope
			imageCache.LastSync = timeutils.UtcNow()

			err = manager.TableSpec().Insert(ctx, &imageCache)
			if err != nil {
				return nil, err
			}
			db.OpsLog.LogEvent(&imageCache, db.ACT_CREATE, info, userCred)

			return &imageCache, nil
		} else {
			log.Errorf("fetching image cache (%s) failed: %s", img.Id, err)
			return nil, err
		}
	} else { // update
		diff, err := db.Update(&imageCache, func() error {
			imageCache.Size = img.Size
			imageCache.Info = info
			imageCache.Status = img.Status
			imageCache.IsPublic = img.IsPublic
			imageCache.PublicScope = img.PublicScope
			imageCache.LastSync = timeutils.UtcNow()
			if imageCache.Deleted == true {
				imageCache.Deleted = false
				imageCache.DeletedAt = time.Time{}
				imageCache.RefCount = 0
				imageCache.UpdateVersion = 0
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		db.OpsLog.LogEvent(&imageCache, db.ACT_UPDATE, diff, userCred)

		return &imageCache, nil
	}
}

func (image *SCachedimage) GetStorages() ([]SStorage, error) {
	sq := StorageManager.Query()
	storagecacheimageSubq := StoragecachedimageManager.Query("storagecache_id").Equals("cachedimage_id", image.GetId()).SubQuery()
	sq.Join(storagecacheimageSubq, sqlchemy.Equals(sq.Field("storagecache_id"), storagecacheimageSubq.Field("storagecache_id")))
	storages := make([]SStorage, 0, 1)
	err := db.FetchModelObjects(StorageManager, sq, &storages)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return storages, nil
}

func (manager *SCachedimageManager) GetImageById(ctx context.Context, userCred mcclient.TokenCredential, imageId string, refresh bool) (*cloudprovider.SImage, error) {
	imgObj, _ := manager.FetchById(imageId)
	if imgObj != nil {
		cachedImage := imgObj.(*SCachedimage)
		oSTypeOk := options.Options.NoCheckOsTypeForCachedImage || len(cachedImage.GetOSType()) > 0
		if !refresh && cachedImage.GetStatus() == cloudprovider.IMAGE_STATUS_ACTIVE && oSTypeOk && !cachedImage.isRefreshSessionExpire() {
			return cachedImage.GetImage()
		} else if len(cachedImage.ExternalId) > 0 { // external image, request refresh
			return cachedImage.requestRefreshExternalImage(ctx, userCred)
		}
	}
	s := auth.GetAdminSession(ctx, options.Options.Region, "")
	obj, err := modules.Images.Get(s, imageId, nil)
	if err != nil {
		log.Errorf("GetImageById %s error %s", imageId, err)
		return nil, errors.Wrap(err, "modules.Images.Get")
	}
	cachedImage, err := manager.cacheGlanceImageInfo(ctx, userCred, obj)
	if err != nil {
		return nil, errors.Wrap(err, "manager.cacheGlanceImageInfo")
	}
	return cachedImage.GetImage()
}

func (manager *SCachedimageManager) getImageByName(ctx context.Context, userCred mcclient.TokenCredential, imageId string, refresh bool) (*cloudprovider.SImage, error) {
	imgObj, _ := manager.FetchByName(userCred, imageId)
	if imgObj != nil {
		cachedImage := imgObj.(*SCachedimage)
		if !refresh && cachedImage.GetStatus() == cloudprovider.IMAGE_STATUS_ACTIVE && len(cachedImage.GetOSType()) > 0 && !cachedImage.isRefreshSessionExpire() {
			return cachedImage.GetImage()
		} else if len(cachedImage.ExternalId) > 0 { // external image, request refresh
			return cachedImage.requestRefreshExternalImage(ctx, userCred)
		}
	}
	s := auth.GetSession(ctx, userCred, options.Options.Region, "")
	obj, err := modules.Images.GetByName(s, imageId, nil)
	if err != nil {
		return nil, errors.Wrap(err, "modules.Images.GetByName")
	}
	cachedImage, err := manager.cacheGlanceImageInfo(ctx, userCred, obj)
	if err != nil {
		return nil, errors.Wrap(err, "manager.cacheGlanceImageInfo")
	}
	return cachedImage.GetImage()
}

func (manager *SCachedimageManager) GetImageInfo(ctx context.Context, userCred mcclient.TokenCredential, imageId string, refresh bool) (*cloudprovider.SImage, error) {
	return manager.getImageInfo(ctx, userCred, imageId, refresh)
}

func (manager *SCachedimageManager) getImageInfo(ctx context.Context, userCred mcclient.TokenCredential, imageId string, refresh bool) (*cloudprovider.SImage, error) {
	img, err := manager.GetImageById(ctx, userCred, imageId, refresh)
	if err == nil {
		return img, nil
	}
	log.Errorf("getImageInfoById %s fail %s", imageId, err)
	return manager.getImageByName(ctx, userCred, imageId, refresh)
}

func (self *SCachedimage) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.CachedimageDetails, error) {
	return api.CachedimageDetails{}, nil
}

func (manager *SCachedimageManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CachedimageDetails {
	rows := make([]api.CachedimageDetails, len(objs))
	virtRows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		ci := objs[i].(*SCachedimage)
		rows[i] = api.CachedimageDetails{
			SharableVirtualResourceDetails: virtRows[i],
			OsType:                         ci.GetOSType(),
			OsDistribution:                 ci.GetOSDistribution(),
			OsVersion:                      ci.GetOSVersion(),
			Hypervisor:                     ci.GetHypervisor(),
		}
		rows[i].CachedCount, _ = ci.getStoragecacheCount()
	}
	return rows
}

func (self *SCachedimage) AllowPerformRefresh(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "refresh")
}

func (self *SCachedimage) PerformRefresh(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	img, err := CachedimageManager.GetImageById(ctx, userCred, self.Id, true)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(img), nil
}

func (self *SCachedimage) AllowPerformUncacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "uncache-image")
}

// 清除镜像缓存
func (self *SCachedimage) PerformUncacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CachedImageUncacheImageInput) (jsonutils.JSONObject, error) {
	if len(input.StoragecacheId) == 0 {
		return nil, httperrors.NewMissingParameterError("storagecache_id")
	}
	_storagecache, err := StoragecacheManager.FetchById(input.StoragecacheId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError("failed to found storagecache %s", input.StoragecacheId)
		}
		return nil, errors.Wrap(err, "StoragecacheManager.FetchById")
	}
	storagecache := _storagecache.(*SStoragecache)
	return storagecache.PerformUncacheImage(ctx, userCred, query, jsonutils.Marshal(map[string]interface{}{"image": self.Id, "is_force": input.IsForce}))
}

func (self *SCachedimage) addRefCount() {
	if self.GetStatus() != api.CACHED_IMAGE_STATUS_ACTIVE {
		return
	}
	_, err := db.Update(self, func() error {
		self.RefCount += 1
		self.LastRef = timeutils.UtcNow()
		return nil
	})
	if err != nil {
		log.Errorf("addRefCount fail %s", err)
	}
}

func (self *SCachedimage) ChooseSourceStoragecacheInRange(hostType string, excludes []string, rangeObjs []interface{}) (*SStoragecachedimage, error) {
	storageCachedImage := StoragecachedimageManager.Query().SubQuery()
	storage := StorageManager.Query().SubQuery()
	hostStorage := HoststorageManager.Query().SubQuery()
	host := HostManager.Query().SubQuery()

	scimgs := make([]SStoragecachedimage, 0)
	q := storageCachedImage.Query().
		Join(storage, sqlchemy.Equals(storage.Field("storagecache_id"), storageCachedImage.Field("storagecache_id"))).
		Join(hostStorage, sqlchemy.Equals(hostStorage.Field("storage_id"), storage.Field("id"))).
		Join(host, sqlchemy.Equals(hostStorage.Field("host_id"), host.Field("id"))).
		Filter(sqlchemy.Equals(storageCachedImage.Field("cachedimage_id"), self.Id)).
		Filter(sqlchemy.Equals(storageCachedImage.Field("status"), api.CACHED_IMAGE_STATUS_ACTIVE)).
		Filter(sqlchemy.Equals(host.Field("status"), api.HOST_STATUS_RUNNING)).
		Filter(sqlchemy.IsTrue(host.Field("enabled"))).
		Filter(sqlchemy.Equals(host.Field("host_status"), api.HOST_ONLINE)).
		Filter(sqlchemy.IsTrue(storage.Field("enabled"))).
		Filter(sqlchemy.In(storage.Field("status"), []string{api.STORAGE_ENABLED, api.STORAGE_ONLINE})).
		Filter(sqlchemy.Equals(storage.Field("storage_type"), api.STORAGE_LOCAL))

	if len(excludes) > 0 {
		q = q.Filter(sqlchemy.NotIn(host.Field("id"), excludes))
	}
	if len(hostType) > 0 {
		q = q.Filter(sqlchemy.Equals(host.Field("host_type"), hostType))
	}

	for _, rangeObj := range rangeObjs {
		switch v := rangeObj.(type) {
		case *SHost:
			q = q.Filter(sqlchemy.Equals(host.Field("id"), v.Id))
		case *SZone:
			q = q.Filter(sqlchemy.Equals(host.Field("zone_id"), v.Id))
		case *SCloudprovider:
			q = q.Filter(sqlchemy.Equals(host.Field("manager_id"), v.Id))
		}
	}

	err := db.FetchModelObjects(StoragecachedimageManager, q, &scimgs)
	if err != nil {
		return nil, err
	}
	if len(scimgs) == 0 {
		return nil, nil
	}

	return &scimgs[rand.Intn(len(scimgs))], nil
}

func (manager *SCachedimageManager) ImageAddRefCount(imageId string) {
	cachedObj, _ := manager.FetchById(imageId)
	if cachedObj != nil {
		cachedImage := (cachedObj).(*SCachedimage)
		cachedImage.addRefCount()
	}
}

func (self *SCachedimage) canDeleteLastCache() bool {
	cnt, err := self.getStoragecacheCount()
	if err != nil {
		// database error
		return false
	}
	if cnt != 1 {
		return true
	}
	if self.isReferenceSessionExpire() {
		return true
	}
	return false
}

func (self *SCachedimage) syncWithCloudImage(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, image cloudprovider.ICloudImage, managerId string) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		newName, err := db.GenerateAlterName(self, image.GetName())
		if err != nil {
			return errors.Wrap(err, "GenerateAlterName")
		}
		self.Name = newName
		self.Size = image.GetSizeByte()
		self.ExternalId = image.GetGlobalId()
		self.ImageType = string(image.GetImageType())
		self.PublicScope = string(image.GetPublicScope())
		self.Status = image.GetStatus()
		if image.GetPublicScope() == rbacutils.ScopeSystem {
			self.IsPublic = true
		}
		self.UEFI = tristate.NewFromBool(image.UEFI())
		sImage := cloudprovider.CloudImage2Image(image)
		self.Info = jsonutils.Marshal(&sImage)
		self.LastSync = time.Now().UTC()
		return nil
	})
	db.OpsLog.LogSyncUpdate(self, diff, userCred)

	SyncCloudProject(userCred, self, ownerId, image, managerId)
	return err
}

func (manager *SCachedimageManager) newFromCloudImage(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, image cloudprovider.ICloudImage, managerId string) (*SCachedimage, error) {
	cachedImage := SCachedimage{}
	cachedImage.SetModelManager(manager, &cachedImage)

	cachedImage.Size = image.GetSizeByte()
	cachedImage.UEFI = tristate.NewFromBool(image.UEFI())
	sImage := cloudprovider.CloudImage2Image(image)
	cachedImage.Info = jsonutils.Marshal(&sImage)
	cachedImage.LastSync = time.Now().UTC()
	cachedImage.ImageType = string(image.GetImageType())
	cachedImage.ExternalId = image.GetGlobalId()
	cachedImage.Status = image.GetStatus()
	cachedImage.PublicScope = string(image.GetPublicScope())
	switch image.GetPublicScope() {
	case rbacutils.ScopeNone:
	default:
		cachedImage.IsPublic = true
	}

	var err error
	err = func() error {
		lockman.LockRawObject(ctx, manager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

		cachedImage.Name, err = db.GenerateName(ctx, manager, nil, image.GetName())
		if err != nil {
			return err
		}
		return manager.TableSpec().Insert(ctx, &cachedImage)
	}()
	if err != nil {
		return nil, err
	}

	SyncCloudProject(userCred, &cachedImage, ownerId, image, managerId)

	return &cachedImage, nil
}

func (image *SCachedimage) requestRefreshExternalImage(ctx context.Context, userCred mcclient.TokenCredential) (*cloudprovider.SImage, error) {
	caches := image.getValidStoragecache()
	if caches == nil || len(caches) == 0 {
		log.Errorf("requestRefreshExternalImage: no valid storage cache")
		return nil, fmt.Errorf("no valid storage cache")
	}
	var cache *SStoragecache
	var cachedImage *SStoragecachedimage
	for i := range caches {
		ci := StoragecachedimageManager.GetStoragecachedimage(caches[i].Id, image.Id)
		if ci != nil {
			cache = &caches[i]
			cachedImage = ci
			break
		}
	}
	if cache == nil {
		return nil, fmt.Errorf("no cached image found")
	}
	iCache, err := cache.GetIStorageCache()
	if err != nil {
		log.Errorf("GetIStorageCache fail %s", err)
		return nil, err
	}
	iImage, err := iCache.GetIImageById(cachedImage.ExternalId)
	if err != nil {
		log.Errorf("iCache.GetIImageById fail %s", err)
		return nil, err
	}
	err = image.syncWithCloudImage(ctx, userCred, nil, iImage, "")
	if err != nil {
		log.Errorf("image.syncWithCloudImage fail %s", err)
		return nil, err
	}
	return image.GetImage()
}

func (image *SCachedimage) getValidStoragecache() []SStoragecache {
	storagecaches := StoragecacheManager.Query().SubQuery()
	storagecacheimages := StoragecachedimageManager.Query().SubQuery()
	providers := CloudproviderManager.Query().SubQuery()

	q := storagecaches.Query()
	q = q.Join(providers, sqlchemy.Equals(providers.Field("id"), storagecaches.Field("manager_id")))
	q = q.Join(storagecacheimages, sqlchemy.Equals(storagecaches.Field("id"), storagecacheimages.Field("storagecache_id")))
	q = q.Filter(sqlchemy.IsTrue(providers.Field("enabled")))
	q = q.Filter(sqlchemy.In(providers.Field("status"), api.CLOUD_PROVIDER_VALID_STATUS))
	q = q.Filter(sqlchemy.In(providers.Field("health_status"), api.CLOUD_PROVIDER_VALID_HEALTH_STATUS))
	q = q.Filter(sqlchemy.Equals(storagecacheimages.Field("cachedimage_id"), image.Id))
	q = q.Filter(sqlchemy.Equals(storagecacheimages.Field("status"), api.CACHED_IMAGE_STATUS_ACTIVE))

	caches := make([]SStoragecache, 0)
	err := db.FetchModelObjects(StoragecacheManager, q, &caches)
	if err != nil {
		log.Errorf("getValidStoragecache fail %s", err)
		return nil
	}
	return caches
}

func (image *SCachedimage) GetRegions() ([]SCloudregion, error) {
	regions := []SCloudregion{}
	caches := image.getValidStoragecache()
	for _, cache := range caches {
		region, err := cache.GetRegion()
		if err != nil {
			return nil, err
		}
		regions = append(regions, *region)
	}
	return regions, nil
}

func (image *SCachedimage) GetUsableZoneIds() ([]string, error) {
	zones := ZoneManager.Query().SubQuery()
	storages := StorageManager.Query().SubQuery()
	storagecaches := StoragecacheManager.Query().SubQuery()
	storagecacheimages := StoragecachedimageManager.Query().SubQuery()
	providers := CloudproviderManager.Query().SubQuery()

	q := zones.Query(zones.Field("id"))
	q = q.Join(storages, sqlchemy.Equals(q.Field("id"), storages.Field("zone_id")))
	q = q.Join(storagecaches, sqlchemy.Equals(storages.Field("storagecache_id"), storagecaches.Field("id")))
	q = q.Join(providers, sqlchemy.Equals(providers.Field("id"), storagecaches.Field("manager_id")))
	q = q.Join(storagecacheimages, sqlchemy.Equals(storagecaches.Field("id"), storagecacheimages.Field("storagecache_id")))
	q = q.Filter(sqlchemy.IsTrue(providers.Field("enabled")))
	q = q.Filter(sqlchemy.In(providers.Field("status"), api.CLOUD_PROVIDER_VALID_STATUS))
	q = q.Filter(sqlchemy.In(providers.Field("health_status"), api.CLOUD_PROVIDER_VALID_HEALTH_STATUS))
	q = q.Filter(sqlchemy.Equals(storagecacheimages.Field("cachedimage_id"), image.Id))
	q = q.Filter(sqlchemy.Equals(storagecacheimages.Field("status"), api.CACHED_IMAGE_STATUS_ACTIVE))
	q = q.Filter(sqlchemy.Equals(q.Field("status"), api.ZONE_ENABLE))

	result := []string{}
	rows, err := q.Rows()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var zoneId string
		if err := rows.Scan(&zoneId); err != nil {
			return nil, err
		}
		result = append(result, zoneId)
	}

	return result, nil
}

func (image *SCachedimage) GetCloudprovider() (*SCloudprovider, error) {
	caches := image.getValidStoragecache()
	if len(caches) == 0 {
		return nil, fmt.Errorf("no valid storagecache for image %s(%s)", image.Name, image.Id)
	}
	cloudprovider := caches[0].GetCloudprovider()
	if cloudprovider == nil {
		return nil, fmt.Errorf("failed to found cloudprovider for storagecache %s(%s)", caches[0].Name, caches[0].Id)
	}
	return cloudprovider, nil
}

// 缓存镜像列表
func (manager *SCachedimageManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CachedimageListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SSharableBaseResourceManager.ListItemFilter(ctx, q, userCred, query.SharableResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SSharableBaseResourceManager.ListItemFilter")
	}

	q, err = manager.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	{
		var idFilter bool
		storagecachedImages := StoragecachedimageManager.Query().SubQuery()
		storageCaches := StoragecacheManager.Query().SubQuery()
		var storages *sqlchemy.SSubQuery

		if query.Valid == nil {
			storages = StorageManager.Query().SubQuery()
		} else if *query.Valid {
			idFilter = true
			storages = StorageManager.Query().In("status", []string{api.STORAGE_ENABLED, api.STORAGE_ONLINE}).IsTrue("enabled").SubQuery()
		} else {
			idFilter = true
			stroage := StorageManager.Query()
			storages = stroage.Filter(sqlchemy.OR(sqlchemy.NotIn(stroage.Field("status"), []string{}), sqlchemy.IsFalse(stroage.Field("enabled")))).SubQuery()
		}
		zones := ZoneManager.Query().SubQuery()

		subq := storagecachedImages.Query(storagecachedImages.Field("cachedimage_id"))
		subq = subq.Join(storageCaches, sqlchemy.Equals(storagecachedImages.Field("storagecache_id"), storageCaches.Field("id")))
		subq = subq.Join(storages, sqlchemy.Equals(storageCaches.Field("id"), storages.Field("storagecache_id")))
		subq = subq.Join(zones, sqlchemy.Equals(storages.Field("zone_id"), zones.Field("id")))

		if len(query.HostSchedtagId) > 0 {
			idFilter = true
			schedTagObj, err := SchedtagManager.FetchByIdOrName(userCred, query.HostSchedtagId)
			if err != nil {
				if errors.Cause(err) == sql.ErrNoRows {
					return nil, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", SchedtagManager.Keyword(), query.HostSchedtagId)
				} else {
					return nil, errors.Wrap(err, "SchedtagManager.FetchByIdOrName")
				}
			}
			hoststorages := HoststorageManager.Query("host_id", "storage_id").SubQuery()
			hostschedtags := HostschedtagManager.Query().Equals("schedtag_id", schedTagObj.GetId()).SubQuery()
			subq = subq.Join(hoststorages, sqlchemy.Equals(hoststorages.Field("storage_id"), storages.Field("id")))
			subq = subq.Join(hostschedtags, sqlchemy.Equals(hostschedtags.Field("host_id"), hoststorages.Field("host_id")))
		}
		subq = subq.Filter(sqlchemy.Equals(storagecachedImages.Field("status"), api.CACHED_IMAGE_STATUS_ACTIVE))

		subq = subq.Snapshot()

		subq, err = managedResourceFilterByAccount(subq, query.ManagedResourceListInput, "", nil)
		if err != nil {
			return nil, errors.Wrap(err, "managedResourceFilterByAccount")
		}

		subq, err = managedResourceFilterByRegion(subq, query.RegionalFilterListInput, "", nil)
		if err != nil {
			return nil, errors.Wrap(err, "_managedResourceFilterByRegion")
		}

		subq, err = managedResourceFilterByZone(subq, query.ZonalFilterListInput, "", nil)
		if err != nil {
			return nil, errors.Wrap(err, "_managedResourceFilterByZone")
		}

		if subq.IsAltered() {
			idFilter = true
		}

		if idFilter {
			q = q.In("id", subq)
		}
	}

	if len(query.ImageType) > 0 {
		q = q.Equals("image_type", query.ImageType)
	}

	return q, nil
}

func (manager *SCachedimageManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CachedimageListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SCachedimageManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SCachedimageManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	q, err := manager.SSharableVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrapf(err, "SSharableVirtualResourceBaseManager.L.ListItemExportKeys")
	}
	return q, nil
}

func (manager *SCachedimageManager) InitializeData() error {
	images := []SCachedimage{}
	q := manager.Query().IsNullOrEmpty("tenant_id")
	err := db.FetchModelObjects(manager, q, &images)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range images {
		_, err := db.Update(&images[i], func() error {
			images[i].IsPublic = true
			images[i].PublicScope = string(rbacutils.ScopeSystem)
			images[i].ProjectId = "system"
			if len(images[i].ExternalId) > 0 {
				images[i].Status = api.CACHED_IMAGE_STATUS_ACTIVE
			} else {
				images[i].Status = images[i].GetStatus()
			}
			return nil
		})
		if err != nil {
			return errors.Wrapf(err, "db.Update(%s)", images[i].Id)
		}
	}
	return nil
}
