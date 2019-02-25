package models

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

const (
	CACHED_IMAGE_REFRESH_SECONDS                  = 900   // 15 minutes
	CACHED_IMAGE_REFERENCE_SESSION_EXPIRE_SECONDS = 86400 // 1 day
)

type SCachedimageManager struct {
	db.SStandaloneResourceBaseManager
}

var CachedimageManager *SCachedimageManager

func init() {
	CachedimageManager = &SCachedimageManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SCachedimage{},
			"cachedimages_tbl",
			"cachedimage",
			"cachedimages",
		),
	}
}

type SCachedimage struct {
	db.SStandaloneResourceBase
	// SManagedResourceBase

	Size int64 `nullable:"false" list:"user" update:"admin" create:"admin_required"` // = Column(BigInteger, nullable=False) # in Byte
	// virtual_size = Column(BigInteger, nullable=False) # in Byte
	Info jsonutils.JSONObject `nullable:"true" list:"user" update:"admin" create:"admin_required"` // Column(JSONEncodedDict, nullable=True)

	LastSync time.Time `list:"admin"`            // = Column(DateTime)
	LastRef  time.Time `list:"admin"`            // = Column(DateTime)
	RefCount int       `default:"0" list:"user"` // = Column(Integer, default=0, server_default='0')

	ImageType string `width:"16" default:"customized" list:"user"`
}

func (self *SCachedimageManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SCachedimageManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SCachedimage) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SCachedimage) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SCachedimage) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (self *SCachedimage) ValidateDeleteCondition(ctx context.Context) error {
	if self.getStoragecacheCount() > 0 {
		return httperrors.NewNotEmptyError("The image has been cached on storages")
	}
	if self.GetStatus() == "active" && !self.isReferenceSessionExpire() {
		return httperrors.NewConflictError("the image reference session has not been expired!")
	}
	return self.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SCachedimage) isReferenceSessionExpire() bool {
	if !self.LastRef.IsZero() && time.Now().Sub(self.LastRef) < CACHED_IMAGE_REFERENCE_SESSION_EXPIRE_SECONDS*time.Second {
		return false
	} else {
		return true
	}
}

func (self *SCachedimage) isRefreshSessionExpire() bool {
	if len(self.ExternalId) > 0 { // external image info never expires
		return false
	}
	if !self.LastRef.IsZero() && time.Now().Sub(self.LastRef) < CACHED_IMAGE_REFRESH_SECONDS*time.Second {
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

func (self *SCachedimage) getStoragecacheQuery() *sqlchemy.SQuery {
	q := StoragecachedimageManager.Query().Equals("cachedimage_id", self.Id)
	return q
}

func (self *SCachedimage) getStoragecacheCount() int {
	return self.getStoragecacheQuery().Count()
}

func (self *SCachedimage) GetImage() (*cloudprovider.SImage, error) {
	image := cloudprovider.SImage{}

	err := self.Info.Unmarshal(&image)
	if err != nil {
		return nil, err
	} else {
		// hack, make cached image ID consistent
		image.Id = self.Id
		return &image, nil
	}
}

func (manager *SCachedimageManager) cacheGlanceImageInfo(ctx context.Context, userCred mcclient.TokenCredential, info jsonutils.JSONObject) (*SCachedimage, error) {
	lockman.LockClass(ctx, manager, userCred.GetProjectId())
	defer lockman.ReleaseClass(ctx, manager, userCred.GetProjectId())

	imgId, _ := info.GetString("id")
	if len(imgId) == 0 {
		return nil, fmt.Errorf("invalid image info")
	}

	imageCache := SCachedimage{}
	imageCache.SetModelManager(manager)

	size, _ := info.Int("size")
	name, _ := info.GetString("name")
	if len(name) == 0 {
		name = imgId
	}

	name = db.GenerateName(manager, "", name)

	err := manager.Query().Equals("id", imgId).First(&imageCache)
	if err != nil {
		if err == sql.ErrNoRows { // insert
			imageCache.Id = imgId
			imageCache.Name = name
			imageCache.Size = size
			imageCache.Info = info
			imageCache.LastSync = timeutils.UtcNow()

			err = manager.TableSpec().Insert(&imageCache)
			if err != nil {
				return nil, err
			}
			db.OpsLog.LogEvent(&imageCache, db.ACT_CREATE, info, userCred)

			return &imageCache, nil
		} else {
			log.Errorf("fetching image cache (%s) failed: %s", imgId, err)
			return nil, err
		}
	} else { // update
		diff, err := manager.TableSpec().Update(&imageCache, func() error {
			imageCache.Size = size
			imageCache.Info = info
			imageCache.LastSync = timeutils.UtcNow()
			return nil
		})
		if err != nil {
			return nil, err
		}

		db.OpsLog.LogEvent(&imageCache, db.ACT_UPDATE, sqlchemy.UpdateDiffString(diff), userCred)

		return &imageCache, nil
	}
}

func (manager *SCachedimageManager) GetImageById(ctx context.Context, userCred mcclient.TokenCredential, imageId string, refresh bool) (*cloudprovider.SImage, error) {
	imgObj, _ := manager.FetchById(imageId)
	if imgObj != nil {
		cachedImage := imgObj.(*SCachedimage)
		if !refresh && cachedImage.GetStatus() == cloudprovider.IMAGE_STATUS_ACTIVE && len(cachedImage.GetOSType()) > 0 && cachedImage.isRefreshSessionExpire() {
			return cachedImage.GetImage()
		} else if len(cachedImage.ExternalId) > 0 { // external image, request refresh
			return cachedImage.requestRefreshExternalImage(ctx, userCred)
		}
	}
	s := auth.GetAdminSession(ctx, options.Options.Region, "")
	obj, err := modules.Images.Get(s, imageId, nil)
	if err != nil {
		log.Errorf("GetImageById %s error %s", imageId, err)
		return nil, err
	}
	cachedImage, err := manager.cacheGlanceImageInfo(ctx, userCred, obj)
	if err != nil {
		return nil, err
	}
	return cachedImage.GetImage()
}

func (manager *SCachedimageManager) getImageByName(ctx context.Context, userCred mcclient.TokenCredential, imageId string, refresh bool) (*cloudprovider.SImage, error) {
	imgObj, _ := manager.FetchByName(userCred, imageId)
	if imgObj != nil {
		cachedImage := imgObj.(*SCachedimage)
		if !refresh && cachedImage.GetStatus() == cloudprovider.IMAGE_STATUS_ACTIVE && len(cachedImage.GetOSType()) > 0 && cachedImage.isRefreshSessionExpire() {
			return cachedImage.GetImage()
		} else if len(cachedImage.ExternalId) > 0 { // external image, request refresh
			return cachedImage.requestRefreshExternalImage(ctx, userCred)
		}
	}
	s := auth.GetSession(ctx, userCred, options.Options.Region, "")
	obj, err := modules.Images.GetByName(s, imageId, nil)
	if err != nil {
		return nil, err
	}
	cachedImage, err := manager.cacheGlanceImageInfo(ctx, userCred, obj)
	if err != nil {
		return nil, err
	}
	return cachedImage.GetImage()
}

func (manager *SCachedimageManager) getImageInfo(ctx context.Context, userCred mcclient.TokenCredential, imageId string, refresh bool) (*cloudprovider.SImage, error) {
	img, err := manager.GetImageById(ctx, userCred, imageId, refresh)
	if err == nil {
		return img, nil
	}
	log.Errorf("getImageInfoById %s fail %s", imageId, err)
	return manager.getImageByName(ctx, userCred, imageId, refresh)
}

func (self *SCachedimage) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	extra = self.getMoreDetails(extra)
	return extra, nil
}

func (self *SCachedimage) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	extra = self.getMoreDetails(extra)
	return extra
}

func (self *SCachedimage) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	// extra.Add(jsonutils.NewString(self.GetName()), "name")
	//extra.Add(jsonutils.NewString(self.GetOwner()), "owner")
	//extra.Add(jsonutils.NewString(self.GetFormat()), "format")
	extra.Add(jsonutils.NewString(self.GetStatus()), "status")
	for _, k := range []string{"os_type", "os_distribution", "os_version", "hypervisor"} {
		val, _ := self.Info.GetString("properties", k)
		if len(val) > 0 {
			extra.Add(jsonutils.NewString(val), k)
		}
	}
	extra.Add(jsonutils.NewInt(int64(self.getStoragecacheCount())), "cached_count")
	return extra
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

func (self *SCachedimage) addRefCount() {
	if self.GetStatus() != "active" {
		return
	}
	_, err := CachedimageManager.TableSpec().Update(self, func() error {
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
		Join(storage, sqlchemy.AND(sqlchemy.Equals(storage.Field("storagecache_id"), storageCachedImage.Field("storagecache_id")))).
		Join(hostStorage, sqlchemy.AND(sqlchemy.Equals(hostStorage.Field("storage_id"), storage.Field("id")))).
		Join(host, sqlchemy.AND(sqlchemy.Equals(hostStorage.Field("host_id"), host.Field("id")))).
		Filter(sqlchemy.Equals(storageCachedImage.Field("cachedimage_id"), self.Id)).
		Filter(sqlchemy.Equals(storageCachedImage.Field("status"), CACHED_IMAGE_STATUS_READY)).
		Filter(sqlchemy.Equals(host.Field("status"), HOST_STATUS_RUNNING)).
		Filter(sqlchemy.IsTrue(host.Field("enabled"))).
		Filter(sqlchemy.Equals(host.Field("host_status"), HOST_ONLINE))

	if len(excludes) > 0 {
		q = q.Filter(sqlchemy.NotIn(host.Field("id"), excludes))
	}
	if len(hostType) > 0 {
		q = q.Filter(sqlchemy.Equals(host.Field("host_type"), hostType))
	}

	for _, rangeObj := range rangeObjs {
		switch v := rangeObj.(type) {
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

	rand.Seed(time.Now().Unix())
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
	if self.getStoragecacheCount() != 1 {
		return true
	}
	if self.isReferenceSessionExpire() {
		return true
	}
	return false
}

func (self *SCachedimage) syncWithCloudImage(ctx context.Context, userCred mcclient.TokenCredential, image cloudprovider.ICloudImage) error {
	diff, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.Name = image.GetName()
		self.Size = image.GetSize()
		self.ExternalId = image.GetGlobalId()
		self.ImageType = image.GetImageType()
		sImage := cloudprovider.CloudImage2Image(image)
		self.Info = jsonutils.Marshal(&sImage)
		self.LastSync = time.Now().UTC()
		return nil
	})
	db.OpsLog.LogEvent(self, db.ACT_UPDATE, diff, userCred)
	return err
}

func (manager *SCachedimageManager) newFromCloudImage(ctx context.Context, userCred mcclient.TokenCredential, image cloudprovider.ICloudImage) (*SCachedimage, error) {
	cachedImage := SCachedimage{}
	cachedImage.SetModelManager(manager)

	cachedImage.Name = image.GetName()
	cachedImage.Size = image.GetSize()
	sImage := cloudprovider.CloudImage2Image(image)
	cachedImage.Info = jsonutils.Marshal(&sImage)
	cachedImage.LastSync = time.Now().UTC()
	cachedImage.ImageType = image.GetImageType()
	cachedImage.ExternalId = image.GetGlobalId()

	err := manager.TableSpec().Insert(&cachedImage)
	if err != nil {
		return nil, err
	}

	return &cachedImage, nil
}

func (image *SCachedimage) requestRefreshExternalImage(ctx context.Context, userCred mcclient.TokenCredential) (*cloudprovider.SImage, error) {
	caches := image.getValidStoragecache()
	if caches == nil || len(caches) == 0 {
		log.Errorf("requestRefreshExternalImage: no valid storage cache")
		return nil, fmt.Errorf("no valid storage cache")
	}
	iCache, err := caches[0].GetIStorageCache()
	if err != nil {
		log.Errorf("GetIStorageCache fail %s", err)
		return nil, err
	}
	iImage, err := iCache.GetIImageById(image.ExternalId)
	if err != nil {
		log.Errorf("iCache.GetIImageById fail %s", err)
		return nil, err
	}
	err = image.syncWithCloudImage(ctx, userCred, iImage)
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
	q = q.Filter(sqlchemy.Equals(providers.Field("status"), CLOUD_PROVIDER_CONNECTED))
	q = q.Filter(sqlchemy.Equals(storagecacheimages.Field("cachedimage_id"), image.Id))

	caches := make([]SStoragecache, 0)
	err := db.FetchModelObjects(StoragecacheManager, q, &caches)
	if err != nil {
		log.Errorf("getValidStoragecache fail %s", err)
		return nil
	}
	return caches
}

func (manager *SCachedimageManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	providerStr := jsonutils.GetAnyString(query, []string{"provider"})
	if len(providerStr) > 0 {
		cachedImages := CachedimageManager.Query().SubQuery()
		storagecachedImages := StoragecachedimageManager.Query().SubQuery()
		storageCaches := StoragecacheManager.Query().SubQuery()
		cloudproviders := CloudproviderManager.Query().SubQuery()

		subq := cachedImages.Query(cachedImages.Field("id"))
		subq = subq.Join(storagecachedImages, sqlchemy.Equals(cachedImages.Field("id"), storagecachedImages.Field("cachedimage_id")))
		subq = subq.Join(storageCaches, sqlchemy.Equals(storagecachedImages.Field("storagecache_id"), storageCaches.Field("id")))
		subq = subq.Join(cloudproviders, sqlchemy.Equals(storageCaches.Field("manager_id"), cloudproviders.Field("id")))
		subq = subq.Filter(sqlchemy.Equals(cloudproviders.Field("provider"), providerStr))

		q = q.Filter(sqlchemy.In(q.Field("id"), subq.SubQuery()))
	}

	accountStr := jsonutils.GetAnyString(query, []string{"account", "account_id"})
	if len(accountStr) > 0 {
		accountObj, err := CloudaccountManager.FetchByIdOrName(nil, accountStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudaccountManager.Keyword(), accountStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		cachedImages := CachedimageManager.Query().SubQuery()
		storagecachedImages := StoragecachedimageManager.Query().SubQuery()
		storageCaches := StoragecacheManager.Query().SubQuery()
		cloudproviders := CloudproviderManager.Query().SubQuery()

		subq := cachedImages.Query(cachedImages.Field("id"))
		subq = subq.Join(storagecachedImages, sqlchemy.Equals(cachedImages.Field("id"), storagecachedImages.Field("cachedimage_id")))
		subq = subq.Join(storageCaches, sqlchemy.Equals(storagecachedImages.Field("storagecache_id"), storageCaches.Field("id")))
		subq = subq.Join(cloudproviders, sqlchemy.Equals(storageCaches.Field("manager_id"), cloudproviders.Field("id")))
		subq = subq.Filter(sqlchemy.Equals(cloudproviders.Field("cloudaccount_id"), accountObj.GetId()))

		q = q.Filter(sqlchemy.In(q.Field("id"), subq.SubQuery()))
	}

	managerStr := jsonutils.GetAnyString(query, []string{"manager", "manager_id"})
	if len(managerStr) > 0 {
		managerObj, err := CloudproviderManager.FetchByIdOrName(nil, managerStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudproviderManager.Keyword(), managerStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		cachedImages := CachedimageManager.Query().SubQuery()
		storagecachedImages := StoragecachedimageManager.Query().SubQuery()
		storageCaches := StoragecacheManager.Query().SubQuery()

		subq := cachedImages.Query(cachedImages.Field("id"))
		subq = subq.Join(storagecachedImages, sqlchemy.Equals(cachedImages.Field("id"), storagecachedImages.Field("cachedimage_id")))
		subq = subq.Join(storageCaches, sqlchemy.Equals(storagecachedImages.Field("storagecache_id"), storageCaches.Field("id")))
		subq = subq.Filter(sqlchemy.Equals(storageCaches.Field("manager_id"), managerObj.GetId()))

		q = q.Filter(sqlchemy.In(q.Field("id"), subq.SubQuery()))
	}

	regionStr := jsonutils.GetAnyString(query, []string{"region", "region_id", "cloudregion", "cloudregion_id"})
	if len(regionStr) > 0 {
		regionObj, err := CloudregionManager.FetchByIdOrName(nil, regionStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudregionManager.Keyword(), regionStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		cachedImages := CachedimageManager.Query().SubQuery()
		storagecachedImages := StoragecachedimageManager.Query().SubQuery()
		storageCaches := StoragecacheManager.Query().SubQuery()
		storages := StorageManager.Query().SubQuery()
		zones := ZoneManager.Query().SubQuery()

		subq := cachedImages.Query(cachedImages.Field("id"))
		subq = subq.Join(storagecachedImages, sqlchemy.Equals(cachedImages.Field("id"), storagecachedImages.Field("cachedimage_id")))
		subq = subq.Join(storageCaches, sqlchemy.Equals(storagecachedImages.Field("storagecache_id"), storageCaches.Field("id")))
		subq = subq.Join(storages, sqlchemy.Equals(storageCaches.Field("id"), storages.Field("storagecache_id")))
		subq = subq.Join(zones, sqlchemy.Equals(storages.Field("zone_id"), zones.Field("id")))
		subq = subq.Filter(sqlchemy.Equals(zones.Field("cloudregion_id"), regionObj.GetId()))

		q = q.Filter(sqlchemy.In(q.Field("id"), subq.SubQuery()))
	}

	zoneStr := jsonutils.GetAnyString(query, []string{"zone", "zone_id"})
	if len(zoneStr) > 0 {
		zoneObj, err := ZoneManager.FetchByIdOrName(nil, zoneStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(ZoneManager.Keyword(), zoneStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		cachedImages := CachedimageManager.Query().SubQuery()
		storagecachedImages := StoragecachedimageManager.Query().SubQuery()
		storageCaches := StoragecacheManager.Query().SubQuery()
		storages := StorageManager.Query().SubQuery()

		subq := cachedImages.Query(cachedImages.Field("id"))
		subq = subq.Join(storagecachedImages, sqlchemy.Equals(cachedImages.Field("id"), storagecachedImages.Field("cachedimage_id")))
		subq = subq.Join(storageCaches, sqlchemy.Equals(storagecachedImages.Field("storagecache_id"), storageCaches.Field("id")))
		subq = subq.Join(storages, sqlchemy.Equals(storageCaches.Field("id"), storages.Field("storagecache_id")))
		subq = subq.Filter(sqlchemy.Equals(storages.Field("zone_id"), zoneObj.GetId()))

		q = q.Filter(sqlchemy.In(q.Field("id"), subq.SubQuery()))
	}

	return q, nil
}
