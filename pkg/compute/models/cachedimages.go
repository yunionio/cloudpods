package models

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/onecloud/pkg/mcclient/auth"
	"github.com/yunionio/onecloud/pkg/mcclient/modules"
	"github.com/yunionio/onecloud/pkg/httperrors"
	"github.com/yunionio/pkg/util/timeutils"
	"github.com/yunionio/sqlchemy"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/lockman"
	"github.com/yunionio/onecloud/pkg/compute/options"
)

const (
	CACHED_IMAGE_REFRESH_SECONDS                  = 900   // 15 minutes
	CACHED_IMAGE_REFERENCE_SESSION_EXPIRE_SECONDS = 86400 // 1 day
)

type SCachedimageManager struct {
	db.SStandaloneResourceBaseManager
	SInfrastructureManager
}

var CachedimageManager *SCachedimageManager

func init() {
	CachedimageManager = &SCachedimageManager{SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(SCachedimage{}, "cachedimages_tbl", "cachedimage", "cachedimages")}
}

type SCachedimage struct {
	db.SStandaloneResourceBase
	SInfrastructure

	Size int64 `nullable:"false" list:"admin" update:"admin" create:"admin_required"` // = Column(BigInteger, nullable=False) # in Byte
	// virtual_size = Column(BigInteger, nullable=False) # in Byte
	Info     jsonutils.JSONObject `nullable:"true" get:"admin" update:"admin" create:"admin_required"` // Column(JSONEncodedDict, nullable=True)
	LastSync time.Time            `list:"admin"`                                                       // = Column(DateTime)
	LastRef  time.Time            `list:"admin"`                                                       // = Column(DateTime)
	RefCount int                  `default:"0" list:"admin"`                                           // = Column(Integer, default=0, server_default='0')
}

func (self *SCachedimage) ValidateDeleteCondition(ctx context.Context) error {
	if self.getStoragecacheCount() > 0 {
		return httperrors.NewNotEmptyError("The image has been cached on storages")
	}
	if self.getStatus() == "active" && !self.isReferenceSessionExpire() {
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
	if !self.LastRef.IsZero() && time.Now().Sub(self.LastRef) < CACHED_IMAGE_REFRESH_SECONDS*time.Second {
		return false
	} else {
		return true
	}
}

func (self *SCachedimage) getName() string {
	name, _ := self.Info.GetString("name")
	return name
}

func (self *SCachedimage) getOwner() string {
	owner, _ := self.Info.GetString("owner")
	return owner
}

func (self *SCachedimage) getFormat() string {
	format, _ := self.Info.GetString("disk_format")
	return format
}

func (self *SCachedimage) getStatus() string {
	status, _ := self.Info.GetString("status")
	return status
}

func (self *SCachedimage) getOSType() string {
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

func (self *SCachedimage) getImage() (*SImage, error) {
	image := SImage{}

	err := self.Info.Unmarshal(&image)
	if err != nil {
		return nil, err
	} else {
		return &image, nil
	}
}

func (manager *SCachedimageManager) cacheImageInfo(ctx context.Context, userCred mcclient.TokenCredential, info jsonutils.JSONObject) (*SCachedimage, error) {
	lockman.LockClass(ctx, manager, userCred.GetProjectId())
	defer lockman.ReleaseClass(ctx, manager, userCred.GetProjectId())

	imgId, _ := info.GetString("id")
	if len(imgId) == 0 {
		return nil, fmt.Errorf("invalid image info")
	}

	imageCache := SCachedimage{}
	imageCache.SetModelManager(manager)

	size, _ := info.Int("size")

	err := manager.Query().Equals("id", imgId).First(&imageCache)
	if err != nil {
		if err == sql.ErrNoRows { // insert
			imageCache.Id = imgId
			imageCache.Name = imgId
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
			log.Errorf("fail to query image cahe %d", imgId)
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

func (manager *SCachedimageManager) getImageById(ctx context.Context, userCred mcclient.TokenCredential, imageId string, refresh bool) (*SImage, error) {
	if !refresh {
		imgObj, _ := manager.FetchById(imageId)
		if imgObj != nil {
			cachedImage := imgObj.(*SCachedimage)
			if cachedImage.getStatus() == "active" && len(cachedImage.getOSType()) > 0 && cachedImage.isRefreshSessionExpire() {
				return cachedImage.getImage()
			}
		}
	}
	s := auth.GetAdminSession(options.Options.Region, "")
	obj, err := modules.Images.Get(s, imageId, nil)
	if err != nil {
		log.Errorf("GetImageById %s error %s", imageId, err)
		return nil, err
	}
	cachedImage, err := manager.cacheImageInfo(ctx, userCred, obj)
	if err != nil {
		return nil, err
	}
	return cachedImage.getImage()
}

func (manager *SCachedimageManager) getImageByName(ctx context.Context, userCred mcclient.TokenCredential, imageId string) (*SImage, error) {
	s := auth.GetSession(userCred, options.Options.Region, "")
	obj, err := modules.Images.GetByName(s, imageId, nil)
	if err != nil {
		return nil, err
	}
	cachedImage, err := manager.cacheImageInfo(ctx, userCred, obj)
	if err != nil {
		return nil, err
	}
	return cachedImage.getImage()
}

func (manager *SCachedimageManager) getImageInfo(ctx context.Context, userCred mcclient.TokenCredential, imageId string, refresh bool) (*SImage, error) {
	img, err := manager.getImageById(ctx, userCred, imageId, refresh)
	if err == nil {
		return img, nil
	}
	log.Errorf("getImageInfoById %s fail %s", imageId, err)
	return manager.getImageByName(ctx, userCred, imageId)
}

func (self *SCachedimage) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	extra.Add(jsonutils.NewString(self.getName()), "name")
	extra.Add(jsonutils.NewString(self.getOwner()), "owner")
	extra.Add(jsonutils.NewString(self.getFormat()), "format")
	extra.Add(jsonutils.NewString(self.getStatus()), "status")
	for _, k := range []string{"os_type", "os_distribution", "os_version", "hypervisor"} {
		val, _ := self.Info.GetString("properties", k)
		if len(val) > 0 {
			extra.Add(jsonutils.NewString(val), k)
		}
	}
	extra.Add(jsonutils.NewInt(int64(self.getStoragecacheCount())), "storage_cache_count")
	return extra
}

func (self *SCachedimage) AllowPerformRefresh(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (self *SCachedimage) PerformRefresh(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	img, err := CachedimageManager.getImageById(ctx, userCred, self.Id, true)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(img), nil
}

func (self *SCachedimage) addRefCount() {
	if self.getStatus() != "active" {
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

func (self *SCachedimage) ChooseSourceStoragecacheInRange(hostType string, excludes []string, rangeObjs interface{}) (*SStoragecachedimage, error) {
	storageCachedImage := StoragecachedimageManager.Query().SubQuery()
	storage := StorageManager.Query().SubQuery()
	hostStorage := HoststorageManager.Query().SubQuery()
	host := HostManager.Query().SubQuery()

	scimgs := make([]SStoragecachedimage, 0)
	q := storageCachedImage.Query().
		Join(storage, sqlchemy.AND(sqlchemy.Equals(storage.Field("storagecache_id"), storageCachedImage.Field("storagecache_id")))).
		Join(hostStorage, sqlchemy.AND(sqlchemy.Equals(hostStorage.Field("storage_id"), storage.Field("id")))).
		Join(host, sqlchemy.AND(sqlchemy.Equals(hostStorage.Field("host_id"), host.Field("id")))).
		Filter(sqlchemy.Equals(storageCachedImage.Field("cacheimage_id"), self.Id)).
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

	switch v := rangeObjs.(type) {
	case []*SZone:
		for _, obj := range v {
			q = q.Filter(sqlchemy.Equals(host.Field("zone_id"), obj.Id))
		}
	case []*SVCenter:
		for _, obj := range v {
			q = q.Filter(sqlchemy.Equals(host.Field("manager_id"), obj.Id))
		}
	}
	err := q.All(scimgs)
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
