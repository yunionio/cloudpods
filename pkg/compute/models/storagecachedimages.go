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
	"time"

	"github.com/serialx/hashring"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SStoragecachedimageManager struct {
	db.SJointResourceBaseManager
}

var StoragecachedimageManager *SStoragecachedimageManager

func init() {
	db.InitManager(func() {
		StoragecachedimageManager = &SStoragecachedimageManager{
			SJointResourceBaseManager: db.NewJointResourceBaseManager(
				SStoragecachedimage{},
				"storagecachedimages_tbl",
				"storagecachedimage",
				"storagecachedimages",
				StoragecacheManager,
				CachedimageManager,
			),
		}
		StoragecachedimageManager.SetVirtualObject(StoragecachedimageManager)
	})
}

type SStoragecachedimage struct {
	db.SJointResourceBase

	StoragecacheId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"`
	CachedimageId  string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"`

	ExternalId string `width:"256" charset:"utf8" nullable:"false" get:"admin"`

	Status         string    `width:"32" charset:"ascii" nullable:"false" default:"init" list:"admin" update:"admin" create:"admin_required"` // = Column(VARCHAR(32, charset='ascii'), nullable=False,
	Path           string    `width:"256" charset:"utf8" nullable:"true" list:"admin" update:"admin" create:"admin_optional"`                 // = Column(VARCHAR(256, charset='utf8'), nullable=True)
	LastDownload   time.Time `get:"admin"`                                                                                                    // = Column(DateTime)
	DownloadRefcnt int       `get:"admin"`                                                                                                    // = Column(Integer)
}

func (manager *SStoragecachedimageManager) GetMasterFieldName() string {
	return "storagecache_id"
}

func (manager *SStoragecachedimageManager) GetSlaveFieldName() string {
	return "cachedimage_id"
}

func (joint *SStoragecachedimage) Master() db.IStandaloneModel {
	return db.JointMaster(joint)
}

func (joint *SStoragecachedimage) Slave() db.IStandaloneModel {
	return db.JointSlave(joint)
}

func (self *SStoragecachedimageManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SStoragecachedimageManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SStoragecachedimage) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SStoragecachedimage) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SStoragecachedimage) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
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
	sc, err := self.GetStoragecache()
	if err != nil {
		return nil, errors.Wrap(err, "self.GetStoragecache")
	}
	return sc.GetHost()
}

func (self *SStoragecachedimage) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SJointResourceBase.GetCustomizeColumns(ctx, userCred, query)
	extra = db.JointModelExtra(self, extra)
	return self.getExtraDetails(ctx, extra)
}

func (self *SStoragecachedimage) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SJointResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	extra = db.JointModelExtra(self, extra)
	return self.getExtraDetails(ctx, extra), nil
}

func (manager *SStoragecachedimageManager) AllowListDescendent(ctx context.Context, userCred mcclient.TokenCredential, model db.IStandaloneModel, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, manager)
}

func (self *SStoragecachedimage) GetCachedimage() *SCachedimage {
	cachedImage, _ := CachedimageManager.FetchById(self.CachedimageId)
	if cachedImage != nil {
		return cachedImage.(*SCachedimage)
	}
	return nil
}

func (self *SStoragecachedimage) GetStoragecache() (*SStoragecache, error) {
	cache, err := StoragecacheManager.FetchById(self.StoragecacheId)
	if err != nil {
		return nil, errors.Wrap(err, "StoragecacheManager.FetchById")
	}
	return cache.(*SStoragecache), nil
}

func (self *SStoragecachedimage) getExtraDetails(ctx context.Context, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	storagecache, _ := self.GetStoragecache()
	if storagecache != nil {
		extra.Add(jsonutils.NewStringArray(storagecache.getStorageNames()), "storages")
		host, _ := storagecache.GetHost()
		if host != nil {
			extra.Add(host.GetShortDesc(ctx), "host")
		}
	}
	cachedImage := self.GetCachedimage()
	if cachedImage != nil {
		extra.Add(jsonutils.NewString(cachedImage.GetName()), "image")
		extra.Add(jsonutils.NewInt(cachedImage.Size), "size")
	}
	cnt, _ := self.getReferenceCount()
	extra.Add(jsonutils.NewInt(int64(cnt)), "reference")
	return extra
}

func (self *SStoragecachedimage) getCdromReferenceCount() (int, error) {
	cdroms := GuestcdromManager.Query().SubQuery()
	guests := GuestManager.Query().SubQuery()

	q := cdroms.Query()
	q = q.Join(guests, sqlchemy.Equals(cdroms.Field("id"), guests.Field("id")))
	q = q.Filter(sqlchemy.Equals(cdroms.Field("image_id"), self.CachedimageId))
	return q.CountWithError()
}

func (self *SStoragecachedimage) getDiskReferenceCount() (int, error) {
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
	q = q.Filter(sqlchemy.NOT(sqlchemy.In(disks.Field("status"), []string{api.DISK_ALLOC_FAILED, api.DISK_INIT})))

	return q.CountWithError()
}

func (self *SStoragecachedimage) getReferenceCount() (int, error) {
	totalCnt := 0
	cnt, err := self.getCdromReferenceCount()
	if err != nil {
		return -1, err
	}
	totalCnt += cnt
	cnt, err = self.getDiskReferenceCount()
	if err != nil {
		return -1, err
	}
	totalCnt += cnt
	return totalCnt, nil
}

func (manager *SStoragecachedimageManager) GetStoragecachedimage(cacheId string, imageId string) *SStoragecachedimage {
	obj, err := db.FetchJointByIds(manager, cacheId, imageId, nil)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("manager.FetchByIds %s %s error %s", cacheId, imageId, err)
		}
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
	cnt, err := self.getReferenceCount()
	if err != nil {
		return httperrors.NewInternalServerError("getReferenceCount fail %s", err)
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("Image is in use")
	}
	return self.SJointResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SStoragecachedimage) isCachedImageInUse() error {
	if !self.isDownloadSessionExpire() {
		return httperrors.NewResourceBusyError("Active download session not expired")
	}
	image := self.GetCachedimage()
	if image != nil && !image.canDeleteLastCache() {
		return httperrors.NewResourceBusyError("Cannot delete the last cache")
	}
	return nil
}

func (self *SStoragecachedimage) isDownloadSessionExpire() bool {
	if !self.LastDownload.IsZero() && time.Now().Sub(self.LastDownload) < api.DOWNLOAD_SESSION_LENGTH {
		return false
	} else {
		return true
	}
}

func (self *SStoragecachedimage) markDeleting(ctx context.Context, userCred mcclient.TokenCredential, isForce bool) error {
	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return err
	}
	if !isForce {
		err = self.isCachedImageInUse()
		if err != nil {
			return err
		}
	}

	cache, _ := self.GetStoragecache()
	image := self.GetCachedimage()

	if image != nil {
		lockman.LockJointObject(ctx, cache, image)
		defer lockman.ReleaseJointObject(ctx, cache, image)
	}

	if !isForce && !utils.IsInStringArray(self.Status,
		[]string{api.CACHED_IMAGE_STATUS_READY, api.CACHED_IMAGE_STATUS_DELETING, api.CACHED_IMAGE_STATUS_CACHE_FAILED}) {
		return httperrors.NewInvalidStatusError("Cannot uncache in status %s", self.Status)
	}
	_, err = db.Update(self, func() error {
		self.Status = api.CACHED_IMAGE_STATUS_DELETING
		return nil
	})
	return err
}

func (manager *SStoragecachedimageManager) Register(ctx context.Context, userCred mcclient.TokenCredential, cacheId, imageId string, status string) *SStoragecachedimage {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	cachedimage := manager.GetStoragecachedimage(cacheId, imageId)
	if cachedimage != nil {
		return cachedimage
	}

	cachedimage = &SStoragecachedimage{}
	cachedimage.SetModelManager(manager, cachedimage)

	cachedimage.StoragecacheId = cacheId
	cachedimage.CachedimageId = imageId
	if len(status) == 0 {
		status = api.CACHED_IMAGE_STATUS_INIT
	}
	cachedimage.Status = status

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
	_, err := db.Update(self, func() error {
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
	_, err := db.Update(self, func() error {
		self.DownloadRefcnt += 1
		self.LastDownload = time.Now()
		return nil
	})
	return err
}

func (self *SStoragecachedimage) SetExternalId(externalId string) error {
	_, err := db.Update(self, func() error {
		self.ExternalId = externalId
		return nil
	})
	return err
}

func (self SStoragecachedimage) GetExternalId() string {
	return self.ExternalId
}

func (self *SStoragecachedimage) syncRemoveCloudImage(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	image := self.GetCachedimage()
	err := self.Detach(ctx, userCred)
	if err != nil {
		return err
	}
	if image != nil {
		cnt, err := image.getStoragecacheCount()
		if err != nil {
			log.Errorf("getStoragecacheCount fail %s", err)
			return err
		}
		if cnt == 0 {
			err = image.Delete(ctx, userCred)
			if err != nil {
				log.Errorf("image delete error %s", err)
				return err
			}
		}
	}
	return nil
}

func (self *SStoragecachedimage) syncWithCloudImage(ctx context.Context, userCred mcclient.TokenCredential, image cloudprovider.ICloudImage) error {
	cachedImage := self.GetCachedimage()
	if len(cachedImage.ExternalId) > 0 {
		self.SetStatus(userCred, image.GetStatus(), "")
		return cachedImage.syncWithCloudImage(ctx, userCred, image)
	} else {
		return nil
	}
}

func (manager *SStoragecachedimageManager) newFromCloudImage(ctx context.Context, userCred mcclient.TokenCredential, image cloudprovider.ICloudImage, cache *SStoragecache) error {
	var cachedImage *SCachedimage
	imgObj, err := db.FetchByExternalId(CachedimageManager, image.GetGlobalId())
	if err != nil {
		if err != sql.ErrNoRows {
			// unhandled error
			log.Errorf("CachedimageManager.FetchByExternalId error %s", err)
			return err
		}
		// not found
		// first test if this image is uploaded by onecloud, if true, image name should be ID of onecloud image
		name := image.GetName()
		if utils.IsAscii(name) {
			if strings.HasPrefix(name, "img") {
				name = name[3:]
			}
			imgObj, err := CachedimageManager.FetchById(name)
			if err == nil && imgObj != nil {
				cachedImage = imgObj.(*SCachedimage)
			}
		}
		if cachedImage == nil {
			// no such image
			cachedImage, err = CachedimageManager.newFromCloudImage(ctx, userCred, image)
			if err != nil {
				log.Errorf("CachedimageManager.newFromCloudImage fail %s", err)
				return err
			}
		}
	} else {
		cachedImage = imgObj.(*SCachedimage)
	}
	if len(cachedImage.ExternalId) > 0 {
		cachedImage.syncWithCloudImage(ctx, userCred, image)
	}
	scimg := manager.Register(ctx, userCred, cache.GetId(), cachedImage.GetId(), image.GetStatus())
	if scimg == nil {
		return fmt.Errorf("register cached image fail")
	}
	return scimg.SetExternalId(image.GetGlobalId())
}
