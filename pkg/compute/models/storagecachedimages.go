package models

import (
	"context"
	"fmt"
	"time"

	"github.com/serialx/hashring"
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/lockman"
	"github.com/yunionio/onecloud/pkg/httperrors"
	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/pkg/utils"
	"github.com/yunionio/sqlchemy"
)

const (
	CACHED_IMAGE_STATUS_INIT         = "init"
	CACHED_IMAGE_STATUS_SAVING       = "saving"
	CACHED_IMAGE_STATUS_CACHING      = "caching"
	CACHED_IMAGE_STATUS_READY        = "ready"
	CACHED_IMAGE_STATUS_DELETING     = "deleting"
	CACHED_IMAGE_STATUS_CACHE_FAILED = "cache_fail"

	DOWNLOAD_SESSION_LENGTH = 3600 * 3 // 3 hour
)

type SStoragecachedimageManager struct {
	db.SJointResourceBaseManager
	SInfrastructureManager
}

var StoragecachedimageManager *SStoragecachedimageManager

func init() {
	db.InitManager(func() {
		StoragecachedimageManager = &SStoragecachedimageManager{SJointResourceBaseManager: db.NewJointResourceBaseManager(SStoragecachedimage{}, "storagecachedimages_tbl", "storagecachedimage", "storagecachedimages", StoragecacheManager, CachedimageManager)}
	})
}

type SStoragecachedimage struct {
	db.SJointResourceBase
	SInfrastructure

	StoragecacheId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required" key_index:"true"`
	CachedimageId  string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required" key_index:"true"`

	ExternalId string `width:"64" charset:"ascii" nullable:"false" get:"admin"`

	Status         string    `width:"32" charset:"ascii" nullable:"false" default:"init" list:"admin" update:"admin" create:"admin_required"` // = Column(VARCHAR(32, charset='ascii'), nullable=False,
	Path           string    `width:"256" charset:"utf8" nullable:"true" list:"admin" update:"admin" create:"admin_optional"`                 // = Column(VARCHAR(256, charset='utf8'), nullable=True)
	LastDownload   time.Time `get:"admin"`                                                                                                    // = Column(DateTime)
	DownloadRefcnt int       `get:"admin"`                                                                                                    // = Column(Integer)
}

func (joint *SStoragecachedimage) Master() db.IStandaloneModel {
	return db.JointMaster(joint)
}

func (joint *SStoragecachedimage) Slave() db.IStandaloneModel {
	return db.JointSlave(joint)
}

func (self *SStoragecachedimage) getStorageHostId() (string, error) {
	var s SStorage
	storage := StorageManager.Query()
	err := storage.Filter(sqlchemy.Equals(storage.Field("storagecache_id"), self.StoragecacheId)).First(&s)
	if err != nil {
		return "", err
	}

	hosts := s.GetAllAttachingHosts()
	var hostIds = make([]string, 0)
	for _, host := range hosts {
		hostIds = append(hostIds, host.Id)
	}
	if len(hostIds) == 0 {
		return "", nil
	}

	ring := hashring.New(hostIds)
	ret, _ := ring.GetNode(self.StoragecacheId)
	return ret, nil
}

func (self *SStoragecachedimage) GetHost() (*SHost, error) {
	hostId, err := self.getStorageHostId()
	if err != nil {
		return nil, err
	} else if len(hostId) == 0 {
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

func (self *SStoragecachedimage) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SJointResourceBase.GetCustomizeColumns(ctx, userCred, query)
	extra = db.JointModelExtra(self, extra)
	return self.getExtraDetails(extra)
}

func (self *SStoragecachedimage) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SJointResourceBase.GetExtraDetails(ctx, userCred, query)
	extra = db.JointModelExtra(self, extra)
	return self.getExtraDetails(extra)
}

func (manager *SStoragecachedimageManager) AllowListDescendent(ctx context.Context, userCred mcclient.TokenCredential, model db.IStandaloneModel, query jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (self *SStoragecachedimage) getCachedimage() *SCachedimage {
	cachedImage, _ := CachedimageManager.FetchById(self.CachedimageId)
	if cachedImage != nil {
		return cachedImage.(*SCachedimage)
	}
	return nil
}

func (self *SStoragecachedimage) getStoragecache() *SStoragecache {
	cache, _ := StoragecacheManager.FetchById(self.StoragecacheId)
	if cache != nil {
		return cache.(*SStoragecache)
	}
	return nil
}

func (self *SStoragecachedimage) getExtraDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	storagecache := self.getStoragecache()
	if storagecache != nil {
		extra.Add(jsonutils.NewStringArray(storagecache.getStorageNames()), "storages")
	}
	cachedImage := self.getCachedimage()
	if cachedImage != nil {
		extra.Add(jsonutils.NewString(cachedImage.getName()), "image")
		extra.Add(jsonutils.NewInt(cachedImage.Size), "size")
	}
	extra.Add(jsonutils.NewInt(int64(self.getReferenceCount())), "reference")
	return extra
}

func (self *SStoragecachedimage) getCdromReferenceCount() int {
	// TODO
	return 0
}

func (self *SStoragecachedimage) getDiskReferenceCount() int {
	guestdisks := GuestdiskManager.Query().SubQuery()
	disks := DiskManager.Query().SubQuery()
	storages := StorageManager.Query().SubQuery()

	q := guestdisks.Query()
	q = q.Join(disks, sqlchemy.AND(sqlchemy.Equals(disks.Field("id"), guestdisks.Field("disk_id")),
		sqlchemy.IsFalse(disks.Field("deleted"))))
	q = q.Join(storages, sqlchemy.AND(sqlchemy.Equals(disks.Field("storage_id"), storages.Field("id")),
		sqlchemy.IsFalse(storages.Field("deleted"))))
	q = q.Filter(sqlchemy.Equals(storages.Field("storagecache_id"), self.StoragecacheId))
	q = q.Filter(sqlchemy.Equals(disks.Field("template_id"), self.CachedimageId))
	q = q.Filter(sqlchemy.NOT(sqlchemy.In(disks.Field("status"), []string{DISK_ALLOC_FAILED, DISK_INIT})))

	return q.Count()
}

func (self *SStoragecachedimage) getReferenceCount() int {
	return self.getCdromReferenceCount() + self.getDiskReferenceCount()
}

func (manager *SStoragecachedimageManager) GetStoragecachedimage(cacheId string, imageId string) *SStoragecachedimage {
	obj, err := manager.FetchByIds(cacheId, imageId)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	return obj.(*SStoragecachedimage)
}

func (self *SStoragecachedimage) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SStoragecachedimage) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

func (self *SStoragecachedimage) ValidateDeleteCondition(ctx context.Context) error {
	if self.getReferenceCount() > 0 {
		return httperrors.NewNotEmptyError("Image is in use")
	}
	if !self.isDownloadSessionExpire() {
		return httperrors.NewResourceBusyError("Active download session not expired")
	}
	image := self.getCachedimage()
	if !image.canDeleteLastCache() {
		return httperrors.NewResourceBusyError("Cannot delete the last cache")
	}
	return self.SJointResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SStoragecachedimage) isDownloadSessionExpire() bool {
	if !self.LastDownload.IsZero() && time.Now().Sub(self.LastDownload) < DOWNLOAD_SESSION_LENGTH {
		return false
	} else {
		return true
	}
}

func (self *SStoragecachedimage) markDeleting(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}

	cache := self.getStoragecache()
	image := self.getCachedimage()

	lockman.LockJointObject(ctx, cache, image)
	defer lockman.ReleaseJointObject(ctx, cache, image)

	if utils.IsInStringArray(self.Status, []string{CACHED_IMAGE_STATUS_READY, CACHED_IMAGE_STATUS_DELETING}) {
		return httperrors.NewInvalidStatusError("Cannot uncache in status %s", self.Status)
	}
	_, err = self.GetModelManager().TableSpec().Update(self, func() error {
		self.Status = CACHED_IMAGE_STATUS_DELETING
		return nil
	})
	return err
}

func (manager *SStoragecachedimageManager) Register(ctx context.Context, userCred mcclient.TokenCredential, cacheId, imageId string) *SStoragecachedimage {
	lockman.LockClass(ctx, manager, userCred.GetProjectId())
	defer lockman.ReleaseClass(ctx, manager, userCred.GetProjectId())

	cachedimage := manager.GetStoragecachedimage(cacheId, imageId)
	if cachedimage != nil {
		return cachedimage
	}

	cachedimage = &SStoragecachedimage{}
	cachedimage.SetModelManager(manager)

	cachedimage.StoragecacheId = cacheId
	cachedimage.CachedimageId = imageId
	cachedimage.Status = CACHED_IMAGE_STATUS_INIT

	err := manager.TableSpec().Insert(cachedimage)

	if err != nil {
		log.Errorf("insert error %s", err)
		return nil
	}

	return cachedimage
}

func (self *SStoragecachedimage) SetStatus(userCred mcclient.TokenCredential, status string, reason string) error {
	if self.Status == status {
		return nil
	}
	oldStatus := self.Status
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.Status = status
		return nil
	})
	if err != nil {
		return err
	}
	if userCred != nil {
		notes := fmt.Sprintf("%s=>%s", oldStatus, status)
		if len(reason) > 0 {
			notes = fmt.Sprintf("%s: %s", notes, reason)
		}
		db.OpsLog.LogEvent(self, db.ACT_UPDATE_STATUS, notes, userCred)
	}
	return nil
}

func (self *SStoragecachedimage) AddDownloadRefcount() error {
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.DownloadRefcnt += 1
		self.LastDownload = time.Now()
		return nil
	})
	return err
}

func (self *SStoragecachedimage) SetExternalId(externalId string) error {
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.ExternalId = externalId
		return nil
	})
	return err
}
