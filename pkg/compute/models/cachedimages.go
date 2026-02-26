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
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/image"
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
	CachedimageManager.TableSpec().AddIndex(false, "deleted", "domain_id", "tenant_id", "image_type")
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

func (manager *SCachedimageManager) GetResourceCount() ([]db.SScopeResourceCount, error) {
	return []db.SScopeResourceCount{}, nil
}

func (cachedImage SCachedimage) GetGlobalId() string {
	return cachedImage.ExternalId
}

func (cachedImage *SCachedimage) ValidateDeleteCondition(ctx context.Context, info *api.CachedimageDetails) error {
	if gotypes.IsNil(info) {
		info = &api.CachedimageDetails{}
		count, err := CachedimageManager.TotalResourceCount([]string{cachedImage.Id})
		if err != nil {
			return err
		}
		info.CachedimageUsage, _ = count[cachedImage.Id]
	}
	if info.CachedCount > 0 {
		return httperrors.NewNotEmptyError("The image has been cached on storages")
	}
	if cachedImage.GetStatus() == api.CACHED_IMAGE_STATUS_ACTIVE && !cachedImage.isReferenceSessionExpire() {
		return httperrors.NewConflictError("the image reference session has not been expired!")
	}
	return cachedImage.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (cachedImage *SCachedimage) isReferenceSessionExpire() bool {
	if !cachedImage.LastRef.IsZero() && time.Now().Sub(cachedImage.LastRef) < api.CACHED_IMAGE_REFERENCE_SESSION_EXPIRE_SECONDS*time.Second {
		return false
	} else {
		return true
	}
}

func (cachedImage *SCachedimage) isRefreshSessionExpire() bool {
	if len(cachedImage.ExternalId) > 0 { // external image info never expires
		return false
	}
	if !cachedImage.LastRef.IsZero() && time.Now().Sub(cachedImage.LastRef) < api.CACHED_IMAGE_REFRESH_SECONDS*time.Second {
		return false
	} else {
		return true
	}
}

func (cachedImage *SCachedimage) GetHosts() ([]SHost, error) {
	q := HostManager.Query().Distinct()
	hs := HoststorageManager.Query().SubQuery()
	storages := StorageManager.Query().SubQuery()
	scis := StoragecachedimageManager.Query().Equals("cachedimage_id", cachedImage.Id).Equals("status", api.CACHED_IMAGE_STATUS_ACTIVE).SubQuery()

	q = q.Join(hs, sqlchemy.Equals(hs.Field("host_id"), q.Field("id")))
	q = q.Join(storages, sqlchemy.Equals(hs.Field("storage_id"), storages.Field("id")))
	q = q.Join(scis, sqlchemy.Equals(storages.Field("storagecache_id"), scis.Field("storagecache_id")))
	ret := []SHost{}
	err := db.FetchModelObjects(HostManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, err
}

func (cachedImage *SCachedimage) GetName() string {
	name, _ := cachedImage.Info.GetString("name")
	return name
}

func (cachedImage *SCachedimage) GetOwner() string {
	owner, _ := cachedImage.Info.GetString("owner")
	return owner
}

func (cachedImage *SCachedimage) GetFormat() string {
	format, _ := cachedImage.Info.GetString("disk_format")
	return format
}

func (cachedImage *SCachedimage) GetStatus() string {
	status, _ := cachedImage.Info.GetString("status")
	return status
}

func (cachedImage *SCachedimage) GetOSType() string {
	osType, _ := cachedImage.Info.GetString("properties", "os_type")
	return osType
}

func (cachedImage *SCachedimage) GetOSDistribution() string {
	osType, _ := cachedImage.Info.GetString("properties", "os_distribution")
	return osType
}

func (cachedImage *SCachedimage) GetOSVersion() string {
	osType, _ := cachedImage.Info.GetString("properties", "os_version")
	return osType
}

func (cachedImage *SCachedimage) GetHypervisor() string {
	osType, _ := cachedImage.Info.GetString("properties", "hypervisor")
	return osType
}

func (cachedImage *SCachedimage) GetChecksum() string {
	checksum, _ := cachedImage.Info.GetString("checksum")
	return checksum
}

func (cachedImage *SCachedimage) getStoragecacheQuery() *sqlchemy.SQuery {
	q := StoragecachedimageManager.Query().Equals("cachedimage_id", cachedImage.Id)
	return q
}

func (cachedImage *SCachedimage) getStoragecacheCount() (int, error) {
	return cachedImage.getStoragecacheQuery().CountWithError()
}

func (cachedImage *SCachedimage) GetImage() (*cloudprovider.SImage, error) {
	image := cloudprovider.SImage{
		ExternalId: cachedImage.ExternalId,
	}

	err := cachedImage.Info.Unmarshal(&image)
	if err != nil {
		log.Errorf("unmarshal %s fail %s", cachedImage.Info, err)
		return nil, errors.Wrap(err, "cachedImage.Info.Unmarshal")
	} else {
		// hack, make cached image ID consistent
		image.Id = cachedImage.Id
		return &image, nil
	}
}

func (cachedImage *SCachedimage) syncClassMetadata(ctx context.Context, userCred mcclient.TokenCredential) error {
	session := auth.GetSessionWithInternal(ctx, userCred, "")
	ret, err := image.Images.GetSpecific(session, cachedImage.Id, "class-metadata", nil)
	if err != nil {
		return errors.Wrap(err, "unable to get class_metadata")
	}
	classMetadata := make(map[string]string, 0)
	err = ret.Unmarshal(&classMetadata)
	if err != nil {
		return err
	}

	return cachedImage.SetClassMetadataAll(ctx, classMetadata, userCred)
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
		DomainId    string
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

	err = manager.RawQuery().Equals("id", img.Id).First(&imageCache)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows { // insert
			imageCache.Id = img.Id
			imageCache.Name = img.Name
			imageCache.Size = img.Size
			imageCache.Info = info
			imageCache.Status = img.Status
			imageCache.IsPublic = img.IsPublic
			imageCache.PublicScope = img.PublicScope
			imageCache.ProjectId = img.ProjectId
			imageCache.DomainId = img.DomainId
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
			imageCache.ProjectId = img.ProjectId
			imageCache.DomainId = img.DomainId
			if imageCache.Deleted {
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

func (manager *SCachedimageManager) RecoverCachedImage(ctx context.Context, userCred mcclient.TokenCredential, imgId string) (*SCachedimage, error) {
	lockman.LockRawObject(ctx, manager.Keyword(), "name")
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

	imageCache := SCachedimage{}
	imageCache.SetModelManager(manager, &imageCache)

	err := manager.RawQuery().Equals("id", imgId).First(&imageCache)
	if err != nil {
		return nil, err
	}
	diff, err := db.Update(&imageCache, func() error {
		imageCache.Status = api.CACHED_IMAGE_STATUS_ACTIVE
		imageCache.LastSync = timeutils.UtcNow()
		if imageCache.Deleted {
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

func (manager *SCachedimageManager) GetCachedimageById(ctx context.Context, userCred mcclient.TokenCredential, imageId string, refresh bool) (*SCachedimage, error) {
	img, err := manager.FetchById(imageId)
	if err == nil {
		cachedImage := img.(*SCachedimage)
		oSTypeOk := options.Options.NoCheckOsTypeForCachedImage || len(cachedImage.GetOSType()) > 0
		if (!refresh && cachedImage.GetStatus() == cloudprovider.IMAGE_STATUS_ACTIVE && oSTypeOk && !cachedImage.isRefreshSessionExpire()) || options.Options.ProhibitRefreshingCloudImage || len(cachedImage.ExternalId) > 0 {
			return cachedImage, nil
		}
	}
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, err
	}
	s := auth.GetAdminSession(ctx, options.Options.Region)
	obj, err := image.Images.Get(s, imageId, nil)
	if err != nil {
		log.Errorf("GetImageById %s error %s", imageId, err)
		return nil, errors.Wrap(err, "modules.Images.Get")
	}
	cachedImage, err := manager.cacheGlanceImageInfo(ctx, userCred, obj)
	if err != nil {
		return nil, errors.Wrap(err, "manager.cacheGlanceImageInfo")
	}
	return cachedImage, nil
}

func (manager *SCachedimageManager) GetImageById(ctx context.Context, userCred mcclient.TokenCredential, imageId string, refresh bool) (*cloudprovider.SImage, error) {
	imgObj, _ := manager.FetchById(imageId)
	if imgObj != nil {
		cachedImage := imgObj.(*SCachedimage)
		oSTypeOk := options.Options.NoCheckOsTypeForCachedImage || len(cachedImage.GetOSType()) > 0
		if !refresh && cachedImage.GetStatus() == cloudprovider.IMAGE_STATUS_ACTIVE && oSTypeOk && !cachedImage.isRefreshSessionExpire() {
			return cachedImage.GetImage()
		} else if options.Options.ProhibitRefreshingCloudImage {
			return cachedImage.GetImage()
		} else if len(cachedImage.ExternalId) > 0 { // external image, request refresh
			return cachedImage.requestRefreshExternalImage(ctx, userCred)
		}
	}
	s := auth.GetAdminSession(ctx, options.Options.Region)
	obj, err := image.Images.Get(s, imageId, nil)
	if err != nil {
		return nil, errors.Wrap(err, "modules.Images.Get")
	}
	cachedImage, err := manager.cacheGlanceImageInfo(ctx, userCred, obj)
	if err != nil {
		return nil, errors.Wrap(err, "manager.cacheGlanceImageInfo")
	}
	return cachedImage.GetImage()
}

func (manager *SCachedimageManager) getImageByName(ctx context.Context, userCred mcclient.TokenCredential, imageId string, refresh bool) (*cloudprovider.SImage, error) {
	imgObj, _ := manager.FetchByName(ctx, userCred, imageId)
	if imgObj != nil {
		cachedImage := imgObj.(*SCachedimage)
		if !refresh && cachedImage.GetStatus() == cloudprovider.IMAGE_STATUS_ACTIVE && len(cachedImage.GetOSType()) > 0 && !cachedImage.isRefreshSessionExpire() {
			return cachedImage.GetImage()
		} else if options.Options.ProhibitRefreshingCloudImage {
			return cachedImage.GetImage()
		} else if len(cachedImage.ExternalId) > 0 { // external image, request refresh
			return cachedImage.requestRefreshExternalImage(ctx, userCred)
		}
	}
	s := auth.GetSession(ctx, userCred, options.Options.Region)
	obj, err := image.Images.GetByName(s, imageId, nil)
	if err != nil {
		if httputils.ErrorCode(err) == 404 {
			err = httperrors.ErrNotFound
		}
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

func (cm *SCachedimageManager) query(manager db.IModelManager, field string, cacheIds []string, filter func(*sqlchemy.SQuery) *sqlchemy.SQuery) *sqlchemy.SSubQuery {
	q := manager.Query()

	if filter != nil {
		q = filter(q)
	}

	sq := q.SubQuery()

	return sq.Query(
		sq.Field("cachedimage_id"),
		sqlchemy.COUNT(field),
	).In("cachedimage_id", cacheIds).GroupBy(sq.Field("cachedimage_id")).SubQuery()
}

type CachedimageUsageCount struct {
	Id string
	api.CachedimageUsage
}

func (manager *SCachedimageManager) TotalResourceCount(cacheIds []string) (map[string]api.CachedimageUsage, error) {
	ret := map[string]api.CachedimageUsage{}

	scSQ := manager.query(StoragecachedimageManager, "cached_cnt", cacheIds, nil)

	caches := manager.Query().SubQuery()
	cachesQ := caches.Query(
		sqlchemy.SUM("cached_count", scSQ.Field("cached_cnt")),
	)

	cachesQ.AppendField(cachesQ.Field("id"))

	cachesQ = cachesQ.LeftJoin(scSQ, sqlchemy.Equals(cachesQ.Field("id"), scSQ.Field("cachedimage_id")))

	cachesQ = cachesQ.Filter(sqlchemy.In(cachesQ.Field("id"), cacheIds)).GroupBy(cachesQ.Field("id"))

	counts := []CachedimageUsageCount{}
	err := cachesQ.All(&counts)
	if err != nil {
		return nil, errors.Wrapf(err, "cachesQ.All")
	}
	for i := range counts {
		ret[counts[i].Id] = counts[i].CachedimageUsage
	}

	return ret, nil
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
	cacheIds := make([]string, len(objs))
	for i := range rows {
		ci := objs[i].(*SCachedimage)
		rows[i] = api.CachedimageDetails{
			SharableVirtualResourceDetails: virtRows[i],
			OsType:                         ci.GetOSType(),
			OsDistribution:                 ci.GetOSDistribution(),
			OsVersion:                      ci.GetOSVersion(),
			Hypervisor:                     ci.GetHypervisor(),
		}
		cacheIds[i] = ci.Id
	}
	usage, err := manager.TotalResourceCount(cacheIds)
	if err != nil {
		log.Errorf("TotalResourceCount error: %v", err)
		return rows
	}
	for i := range rows {
		rows[i].CachedimageUsage, _ = usage[cacheIds[i]]
	}
	return rows
}

func (cachedImage *SCachedimage) PerformRefresh(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	img, err := CachedimageManager.GetImageById(ctx, userCred, cachedImage.Id, true)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(img), nil
}

// 清除镜像缓存
func (cachedImage *SCachedimage) PerformUncacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.CachedImageUncacheImageInput) (jsonutils.JSONObject, error) {
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
	return storagecache.PerformUncacheImage(ctx, userCred, query, jsonutils.Marshal(map[string]interface{}{"image": cachedImage.Id, "is_force": input.IsForce}))
}

func (manager *SCachedimageManager) PerformCacheImage(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.CachedImageManagerCacheImageInput,
) (jsonutils.JSONObject, error) {
	if len(input.ImageId) == 0 {
		return nil, httperrors.NewMissingParameterError("image_id")
	}
	s := auth.GetAdminSession(ctx, options.Options.Region)
	obj, err := image.Images.Get(s, input.ImageId, nil)
	if err != nil {
		return nil, errors.Wrap(err, "modules.Images.Get")
	}
	cimg, err := manager.cacheGlanceImageInfo(ctx, userCred, obj)
	if err != nil {
		return nil, errors.Wrap(err, "manager.cacheGlanceImageInfo")
	}
	if input.AutoCache {
		storageCaches, err := StoragecacheManager.FetchStoragecachesByFilters(ctx, input.SStorageCacheFilters)
		if err != nil {
			return nil, errors.Wrap(err, "FetchStoragecachesByFilters")
		}
		err = StoragecacheManager.StartImageCacheTask(ctx, userCred, storageCaches, api.CacheImageInput{
			ImageId:  cimg.Id,
			PreCache: true,
		})
		if err != nil {
			return nil, errors.Wrap(err, "StoragecacheManager.StartImageCacheTask")
		}
	}

	return nil, nil
}

func (cachedImage *SCachedimage) addRefCount() {
	if cachedImage.GetStatus() != api.CACHED_IMAGE_STATUS_ACTIVE {
		return
	}
	_, err := db.Update(cachedImage, func() error {
		cachedImage.RefCount += 1
		cachedImage.LastRef = timeutils.UtcNow()
		return nil
	})
	if err != nil {
		log.Errorf("addRefCount fail %s", err)
	}
}

func (cachedImage *SCachedimage) ChooseSourceStoragecacheInRange(hostType string, excludes []string, rangeObjs []interface{}) (*SStoragecachedimage, error) {
	storageCachedImage := StoragecachedimageManager.Query().SubQuery()
	storage := StorageManager.Query().SubQuery()
	hostStorage := HoststorageManager.Query().SubQuery()
	host := HostManager.Query().SubQuery()

	scimgs := make([]SStoragecachedimage, 0)
	q := storageCachedImage.Query().
		Join(storage, sqlchemy.Equals(storage.Field("storagecache_id"), storageCachedImage.Field("storagecache_id"))).
		Join(hostStorage, sqlchemy.Equals(hostStorage.Field("storage_id"), storage.Field("id"))).
		Join(host, sqlchemy.Equals(hostStorage.Field("host_id"), host.Field("id"))).
		Filter(sqlchemy.Equals(storageCachedImage.Field("cachedimage_id"), cachedImage.Id)).
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

func (cachedImage *SCachedimage) canDeleteLastCache() bool {
	cnt, err := cachedImage.getStoragecacheCount()
	if err != nil {
		// database error
		return false
	}
	if cnt != 1 {
		return true
	}
	if cachedImage.isReferenceSessionExpire() {
		return true
	}
	return false
}

func (cachedImage *SCachedimage) syncWithCloudImage(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, image cloudprovider.ICloudImage, provider *SCloudprovider) error {
	diff, err := db.UpdateWithLock(ctx, cachedImage, func() error {
		if options.Options.EnableSyncName {
			newName, err := db.GenerateAlterName(cachedImage, image.GetName())
			if err != nil {
				return errors.Wrap(err, "GenerateAlterName")
			}
			cachedImage.Name = newName
		}
		cachedImage.Size = image.GetSizeByte()
		cachedImage.ExternalId = image.GetGlobalId()
		cachedImage.ImageType = string(image.GetImageType())
		cachedImage.PublicScope = string(image.GetPublicScope())
		cachedImage.Status = image.GetStatus()
		if image.GetPublicScope() == rbacscope.ScopeSystem {
			cachedImage.IsPublic = true
		}
		cachedImage.UEFI = tristate.NewFromBool(cloudprovider.IsUEFI(image))
		sImage := cloudprovider.CloudImage2Image(image)
		cachedImage.Info = jsonutils.Marshal(&sImage)
		cachedImage.LastSync = time.Now().UTC()
		return nil
	})
	db.OpsLog.LogSyncUpdate(cachedImage, diff, userCred)

	if provider != nil {
		SyncCloudProject(ctx, userCred, cachedImage, ownerId, image, provider)
	}
	return err
}

func (manager *SCachedimageManager) newFromCloudImage(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, image cloudprovider.ICloudImage, provider *SCloudprovider) (*SCachedimage, error) {
	cachedImage := SCachedimage{}
	cachedImage.SetModelManager(manager, &cachedImage)

	cachedImage.Size = image.GetSizeByte()
	cachedImage.UEFI = tristate.NewFromBool(cloudprovider.IsUEFI(image))
	sImage := cloudprovider.CloudImage2Image(image)
	cachedImage.Info = jsonutils.Marshal(&sImage)
	cachedImage.LastSync = time.Now().UTC()
	cachedImage.ImageType = string(image.GetImageType())
	cachedImage.ExternalId = image.GetGlobalId()
	cachedImage.Status = image.GetStatus()
	cachedImage.ProjectId = ownerId.GetProjectId()
	cachedImage.DomainId = ownerId.GetProjectDomainId()
	cachedImage.PublicScope = string(image.GetPublicScope())
	switch image.GetPublicScope() {
	case rbacscope.ScopeNone:
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

	if provider != nil {
		SyncCloudProject(ctx, userCred, &cachedImage, ownerId, image, provider)
	}

	return &cachedImage, nil
}

func (image *SCachedimage) requestRefreshExternalImage(ctx context.Context, userCred mcclient.TokenCredential) (*cloudprovider.SImage, error) {
	caches := image.getValidStoragecache()
	if len(caches) == 0 {
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
	iCache, err := cache.GetIStorageCache(ctx)
	if err != nil {
		log.Errorf("GetIStorageCache fail %s", err)
		return nil, err
	}
	iImage, err := iCache.GetIImageById(cachedImage.ExternalId)
	if err != nil {
		log.Errorf("iCache.GetIImageById fail %s", err)
		return nil, err
	}
	err = image.syncWithCloudImage(ctx, userCred, nil, iImage, nil)
	if err != nil {
		log.Errorf("image.syncWithCloudImage fail %s", err)
		return nil, err
	}
	return image.GetImage()
}

func (image *SCachedimage) getValidStoragecache() []SStoragecache {
	storagecaches := StoragecacheManager.Query().SubQuery()
	storagecacheimages := StoragecachedimageManager.Query().SubQuery()
	providers := usableCloudProviders().SubQuery()

	q := storagecaches.Query()
	q = q.Join(providers, sqlchemy.Equals(providers.Field("id"), storagecaches.Field("manager_id")))
	q = q.Join(storagecacheimages, sqlchemy.Equals(storagecaches.Field("id"), storagecacheimages.Field("storagecache_id")))
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

func (image *SCachedimage) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	db.SharedResourceManager.CleanModelShares(ctx, userCred, image.GetISharableVirtualModel())
	return db.RealDeleteModel(ctx, userCred, image)
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
	providers := usableCloudProviders().SubQuery()

	q := zones.Query(zones.Field("id"))
	q = q.Join(storages, sqlchemy.Equals(q.Field("id"), storages.Field("zone_id")))
	q = q.Join(storagecaches, sqlchemy.Equals(storages.Field("storagecache_id"), storagecaches.Field("id")))
	q = q.Join(providers, sqlchemy.Equals(providers.Field("id"), storagecaches.Field("manager_id")))
	q = q.Join(storagecacheimages, sqlchemy.Equals(storagecaches.Field("id"), storagecacheimages.Field("storagecache_id")))
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
		storagecachedImages := StoragecachedimageManager.Query("cachedimage_id").Equals("status", api.CACHED_IMAGE_STATUS_ACTIVE) // .SubQuery()
		storagecachedImages = storagecachedImages.Snapshot()

		storagesQ := StorageManager.Query("storagecache_id")
		storagesQ = storagesQ.Snapshot()
		if query.Valid {
			storagesQ = storagesQ.In("status", []string{api.STORAGE_ENABLED, api.STORAGE_ONLINE}).IsTrue("enabled")
		}
		if len(query.CloudproviderId) > 0 {
			storagesQ = storagesQ.In("manager_id", query.CloudproviderId)
		}
		if len(query.CloudEnv) > 0 {
			switch query.CloudEnv {
			case api.CLOUD_ENV_PUBLIC_CLOUD:
				pubQ := CloudproviderManager.GetPublicProviderIdsQuery()
				storagesQ = storagesQ.In("manager_id", pubQ)
			case api.CLOUD_ENV_PRIVATE_CLOUD:
				privQ := CloudproviderManager.GetPrivateProviderIdsQuery()
				storagesQ = storagesQ.In("manager_id", privQ)
			case api.CLOUD_ENV_ON_PREMISE:
				storagesQ = storagesQ.IsNullOrEmpty("manager_id")
			case api.CLOUD_ENV_PRIVATE_ON_PREMISE:
				privQ := CloudproviderManager.GetPrivateProviderIdsQuery()
				storagesQ = storagesQ.Filter(
					sqlchemy.OR(
						sqlchemy.In(storagesQ.Field("manager_id"), privQ),
						sqlchemy.IsNullOrEmpty(storagesQ.Field("manager_id")),
					),
				)
			}
		}
		if len(query.HostSchedtagId) > 0 {
			hostschedtags := HostschedtagManager.Query("host_id").Equals("schedtag_id", query.HostSchedtagId)
			hoststorages := HoststorageManager.Query("storage_id").In("host_id", hostschedtags.Distinct().SubQuery())
			storagesQ = storagesQ.In("id", hoststorages.Distinct().SubQuery())
		}

		zonesQ := ZoneManager.Query("id")
		zonesQ = zonesQ.Snapshot()
		if len(query.ZoneId) > 0 {
			zonesQ = zonesQ.Equals("id", query.ZoneId)
		}
		if len(query.CloudregionId) > 0 {
			zonesQ = zonesQ.In("cloudregion_id", query.CloudregionId)
		}
		if zonesQ.IsAltered() {
			storagesQ = storagesQ.Filter(
				sqlchemy.In(storagesQ.Field("zone_id"), zonesQ.Distinct().SubQuery()),
			)
		}

		if storagesQ.IsAltered() {
			storagecachedImages = storagecachedImages.Filter(
				sqlchemy.In(storagecachedImages.Field("storagecache_id"), storagesQ.Distinct().SubQuery()),
			)
		}

		if storagecachedImages.IsAltered() {
			q = q.In("id", storagecachedImages.Distinct().SubQuery())
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

// 清理已经删除的镜像缓存
func (manager *SCachedimageManager) AutoCleanImageCaches(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	defer func() {
		err := manager.cleanExternalImages()
		if err != nil {
			log.Errorf("cleanExternalImages error: %v", err)
		}
		err = manager.cleanStoragecachedimages()
		if err != nil {
			log.Errorf("cleanStoragecachedimages error: %v", err)
		}
	}()
	lastSync := time.Now().Add(time.Duration(-1*api.CACHED_IMAGE_REFERENCE_SESSION_EXPIRE_SECONDS) * time.Second)
	q := manager.Query()
	q = q.LT("last_sync", lastSync).Equals("status", api.CACHED_IMAGE_STATUS_ACTIVE).IsNullOrEmpty("external_id").Limit(50)
	caches := []SCachedimage{}
	err := db.FetchModelObjects(manager, q, &caches)
	if err != nil {
		return
	}
	s := auth.GetAdminSession(ctx, options.Options.Region)
	for i := range caches {
		_, err := image.Images.Get(s, caches[i].Id, nil)
		if err != nil {
			if e, ok := err.(*httputils.JSONClientError); ok && e.Code == 404 {
				e := caches[i].ValidateDeleteCondition(ctx, nil)
				if e == nil {
					caches[i].Delete(ctx, userCred)
					continue
				}
				deleteMark := "-deleted@"
				db.Update(&caches[i], func() error {
					if !strings.Contains(caches[i].Name, deleteMark) {
						caches[i].Name = fmt.Sprintf("%s%s%s", caches[i].Name, deleteMark, timeutils.ShortDate(time.Now()))
					}
					return nil
				})
			}
			continue
		}
		db.Update(&caches[i], func() error {
			caches[i].LastSync = time.Now()
			return nil
		})
	}
}

func (manager *SCachedimageManager) getExpireExternalImageIds() ([]string, error) {
	ids := []string{}
	templatedIds := DiskManager.Query("template_id").IsNotEmpty("template_id").Distinct().SubQuery()
	cachedimageIds := StoragecachedimageManager.Query("cachedimage_id").Distinct().SubQuery()
	externalIds := CloudimageManager.Query("external_id").Distinct().SubQuery()
	q := manager.RawQuery("id")
	q = q.Filter(
		sqlchemy.AND(
			sqlchemy.IsNotEmpty(q.Field("external_id")),
			sqlchemy.NotIn(q.Field("id"), templatedIds),
			sqlchemy.NotIn(q.Field("id"), cachedimageIds),
			sqlchemy.NotIn(q.Field("external_id"), externalIds),
		),
	)
	rows, err := q.Rows()
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return ids, nil
		}
		return nil, errors.Wrap(err, "Query")
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		err := rows.Scan(&id)
		if err != nil {
			return nil, errors.Wrapf(err, "rows.Scan")
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (manager *SCachedimageManager) cleanExternalImages() error {
	ids, err := manager.getExpireExternalImageIds()
	if err != nil {
		return errors.Wrapf(err, "getExpireExternalImageIds")
	}

	err = db.Purge(manager, "id", ids, true)
	if err != nil {
		return errors.Wrapf(err, "purge")
	}

	log.Debugf("clean %d expired external images", len(ids))
	return nil
}

func (manager *SCachedimageManager) cleanStoragecachedimages() error {
	ids, err := db.FetchField(StoragecachedimageManager, "row_id", func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		sq := manager.Query("id").Distinct().SubQuery()
		return q.NotIn("cachedimage_id", sq)
	})
	if err != nil {
		return errors.Wrapf(err, "getExpireExternalImageIds")
	}
	err = db.Purge(StoragecachedimageManager, "row_id", ids, true)
	if err != nil {
		return errors.Wrapf(err, "purge")
	}

	log.Debugf("clean %d invalid storagecachedimages", len(ids))
	return nil
}

func (image *SCachedimage) GetAllClassMetadata() (map[string]string, error) {
	meta, err := image.SSharableVirtualResourceBase.GetAllClassMetadata()
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBase.GetAllClassMetadata")
	}
	metaDict, _ := image.Info.GetMap("metadata")
	for k, v := range metaDict {
		if !strings.HasPrefix(k, db.CLASS_TAG_PREFIX) {
			continue
		}
		meta[k[len(db.CLASS_TAG_PREFIX):]], _ = v.GetString()
	}
	return meta, nil
}
