package models

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/mcclient"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/cloudprovider"
)

type SStoragecacheManager struct {
	db.SStandaloneResourceBaseManager
	SInfrastructureManager
}

var StoragecacheManager *SStoragecacheManager

func init() {
	StoragecacheManager = &SStoragecacheManager{SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(SStoragecache{}, "storagecaches_tbl", "storagecache", "storagecaches")}
}

type SStoragecache struct {
	db.SStandaloneResourceBase
	SInfrastructure

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

	err := manager.TableSpec().Insert(&local)
	if err != nil {
		return nil, err
	}

	return &local, nil
}

func (self *SStoragecache) syncWithCloudStoragecache(cloudCache cloudprovider.ICloudStoragecache) error {
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.Name = cloudCache.GetName()
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
		imginfo := img.getCachedimage()
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

func (self *SStoragecache) StartImageUncacheTask(ctx context.Context, userCred mcclient.TokenCredential, imageId string, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(imageId), "image_id")
	task, err := taskman.TaskManager.NewTask(ctx, "StorageUncacheImageTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SStoragecache) GetIStorageCache() (cloudprovider.ICloudStoragecache, error) {
	storages := self.getStorages()
	if storages == nil {
		return nil, fmt.Errorf("no valid storages")
	}
	for i := 0; i < len(storages); i += 1 {
		istorage, err := storages[i].GetIStorage()
		if err == nil {
			return istorage.GetIStoragecache(), nil
		}
	}
	return nil, fmt.Errorf("not valid storage cache")
}
