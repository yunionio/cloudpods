package models

import (
	"context"
	"database/sql"

	"github.com/serialx/hashring"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/imagetools"
)

type SStoragecacheManager struct {
	db.SStandaloneResourceBaseManager
	SInfrastructureManager
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
}

type SStoragecache struct {
	db.SStandaloneResourceBase
	SInfrastructure
	SManagedResourceBase

	Path string `width:"256" charset:"utf8" nullable:"true" list:"admin" update:"admin" create:"admin_optional"` // = Column(VARCHAR(256, charset='utf8'), nullable=True)
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

func (self *SStoragecache) getHostId() (string, error) {
	hoststorages := HoststorageManager.Query().SubQuery()
	storages := StorageManager.Query().SubQuery()

	hosts := make([]SHost, 0)
	host := HostManager.Query().SubQuery()
	q := host.Query(host.Field("id"))
	err := q.Join(hoststorages, sqlchemy.AND(sqlchemy.Equals(hoststorages.Field("host_id"), host.Field("id")),
		sqlchemy.Equals(host.Field("host_status"), HOST_ONLINE),
		sqlchemy.IsTrue(host.Field("enabled")))).
		Join(storages, sqlchemy.AND(sqlchemy.Equals(storages.Field("storagecache_id"), self.Id),
			sqlchemy.Equals(storages.Field("status"), STORAGE_ONLINE),
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

func (manager *SStoragecacheManager) SyncWithCloudStoragecache(cloudCache cloudprovider.ICloudStoragecache) (*SStoragecache, error) {
	localCacheObj, err := manager.FetchByExternalId(cloudCache.GetGlobalId())
	if err != nil {
		if err == sql.ErrNoRows {
			return manager.newFromCloudStoragecache(cloudCache)
		} else {
			log.Errorf("%s", err)
			return nil, err
		}
	} else {
		localCache := localCacheObj.(*SStoragecache)
		localCache.syncWithCloudStoragecache(cloudCache)
		return localCache, nil
	}
}

func (manager *SStoragecacheManager) newFromCloudStoragecache(cloudCache cloudprovider.ICloudStoragecache) (*SStoragecache, error) {
	local := SStoragecache{}
	local.SetModelManager(manager)

	local.Name = cloudCache.GetName()
	local.ExternalId = cloudCache.GetGlobalId()

	local.IsEmulated = cloudCache.IsEmulated()
	local.ManagerId = cloudCache.GetManagerId()

	err := manager.TableSpec().Insert(&local)
	if err != nil {
		return nil, err
	}

	return &local, nil
}

func (self *SStoragecache) syncWithCloudStoragecache(cloudCache cloudprovider.ICloudStoragecache) error {
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.Name = cloudCache.GetName()

		self.IsEmulated = cloudCache.IsEmulated()
		self.ManagerId = cloudCache.GetManagerId()

		return nil
	})
	return err
}

func (self *SStoragecache) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	extra = self.getMoreDetails(extra)
	return extra
}

func (self *SStoragecache) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	extra = self.getMoreDetails(extra)
	return extra
}

func (self *SStoragecache) getCachedImages() []SStoragecachedimage {
	images := make([]SStoragecachedimage, 0)
	q := StoragecachedimageManager.Query().Equals("storagecache_id", self.Id)
	err := q.All(&images)
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
		size += imginfo.Size
	}
	return size
}

func (self *SStoragecache) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	extra.Add(jsonutils.NewStringArray(self.getStorageNames()), "storages")
	extra.Add(jsonutils.NewInt(self.getCachedImageSize()), "size")
	extra.Add(jsonutils.NewInt(int64(self.getCachedImageCount())), "count")
	return extra
}

func (self *SStoragecache) StartImageCacheTask(ctx context.Context, userCred mcclient.TokenCredential, imageId string, isForce bool, parentTaskId string) error {
	StoragecachedimageManager.Register(ctx, userCred, self.Id, imageId)
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(imageId), "image_id")

	image, _ := CachedimageManager.GetImageById(ctx, userCred, imageId, false)

	if image != nil {
		imgInfo := imagetools.NormalizeImageInfo(image.Name, image.Properties["os_arch"], image.Properties["os_type"],
			image.Properties["os_distribution"])
		data.Add(jsonutils.NewString(imgInfo.OsType), "os_type")
		data.Add(jsonutils.NewString(imgInfo.OsArch), "os_arch")
		data.Add(jsonutils.NewString(imgInfo.OsDistro), "os_distribution")
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

func (self *SStoragecache) StartImageUncacheTask(ctx context.Context, userCred mcclient.TokenCredential, imageId string, isForce bool, parentTaskId string) error {
	if !isForce {
		err := self.ValidateDeleteCondition(ctx)
		if err != nil {
			return err
		}
	}

	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(imageId), "image_id")
	if isForce {
		data.Add(jsonutils.JSONTrue, "is_force")
	}
	task, err := taskman.TaskManager.NewTask(ctx, "StorageUncacheImageTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SStoragecache) GetIStorageCache() (cloudprovider.ICloudStoragecache, error) {
	provider, err := self.GetDriver()
	if err != nil {
		log.Errorf("fail to find cloud provider")
		return nil, err
	}
	return provider.GetIStoragecacheById(self.GetExternalId())
}

func (manager *SStoragecacheManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	managerStr := jsonutils.GetAnyString(query, []string{"manager", "provider", "manager_id", "provider_id"})
	if len(managerStr) > 0 {
		provider := CloudproviderManager.FetchCloudproviderByIdOrName(managerStr)
		if provider == nil {
			return nil, httperrors.NewResourceNotFoundError("provider %s not found", managerStr)
		}
		q = q.Filter(sqlchemy.Equals(q.Field("manager_id"), provider.GetId()))
	}

	return q, nil
}

func (self *SStoragecache) ValidateDeleteCondition(ctx context.Context) error {
	if self.getCachedImageCount() > 0 {
		return httperrors.NewNotEmptyError("storage cache not empty")
	}
	return self.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SStoragecache) AllowPerformUncacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (self *SStoragecache) PerformUncacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
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

	scimg := StoragecachedimageManager.GetStoragecachedimage(self.Id, image.Id)
	if scimg == nil {
		return nil, httperrors.NewResourceNotFoundError("storage not cache image")
	}

	if scimg.Status == CACHED_IMAGE_STATUS_INIT {
		err = scimg.Detach(ctx, userCred)
		return nil, err
	}

	err = scimg.markDeleting(ctx, userCred, isForce)
	if err != nil {
		return nil, httperrors.NewInvalidStatusError("Fail to mark cache status: %s", err)
	}

	err = self.StartImageUncacheTask(ctx, userCred, image.Id, isForce, "")

	return nil, err
}

func (self *SStoragecache) AllowPerformCacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
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

	err = self.StartImageCacheTask(ctx, userCred, image.Id, isForce, "")
	return nil, err
}
