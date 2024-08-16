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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SStoragecachedimageManager struct {
	db.SJointResourceBaseManager
	db.SExternalizedResourceBaseManager

	SStoragecacheResourceBaseManager
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
	db.SExternalizedResourceBase

	SStoragecacheResourceBase
	// 镜像缓存Id
	CachedimageId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"`

	// 外部Id
	// ExternalId string `width:"256" charset:"utf8" nullable:"false" get:"admin"`

	// 镜像状态
	Status string `width:"32" charset:"ascii" nullable:"false" default:"init" list:"admin" update:"admin" create:"admin_required"`
	Path   string `width:"256" charset:"utf8" nullable:"true" list:"admin" update:"admin" create:"admin_optional"`
	// 上次下载时间
	LastDownload time.Time `get:"admin"`
	// 下载引用次数
	DownloadRefcnt int `get:"admin"`
}

func (manager *SStoragecachedimageManager) GetMasterFieldName() string {
	return "storagecache_id"
}

func (manager *SStoragecachedimageManager) GetSlaveFieldName() string {
	return "cachedimage_id"
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
	sc := self.GetStoragecache()
	return sc.GetMasterHost()
}

func (manager *SStoragecachedimageManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.StoragecachedimageDetails {
	rows := make([]api.StoragecachedimageDetails, len(objs))

	storagecacheIds := make([]string, 0)
	imageIds := make([]string, 0)

	jointRows := manager.SJointResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	scRows := manager.SStoragecacheResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.StoragecachedimageDetails{
			JointResourceBaseDetails: jointRows[i],
			StoragecacheResourceInfo: scRows[i],
		}
		sci := objs[i].(*SStoragecachedimage)
		storagecacheIds = append(storagecacheIds, sci.StoragecacheId)
		imageIds = append(imageIds, sci.CachedimageId)
	}

	cachedImages := make(map[string]SCachedimage)
	err := db.FetchModelObjectsByIds(CachedimageManager, "id", imageIds, &cachedImages)
	if err != nil {
		log.Errorf("db.FetchModelObjectsByIds fail %s", err)
	}
	cdromRefs, err := manager.fetchCdromReferenceCounts(storagecacheIds, imageIds)
	if err != nil {
		log.Errorf("manager.fetchCdromReferenceCounts fail %s", err)
	}
	diskRefs, err := manager.fetchDiskReferenceCounts(storagecacheIds, imageIds)
	if err != nil {
		log.Errorf("manager.fetchDiskReferenceCounts fail %s", err)
	}

	for i := range rows {
		sci := objs[i].(*SStoragecachedimage)
		if cachedImages != nil {
			if cachedImage, ok := cachedImages[sci.CachedimageId]; ok {
				rows[i].Cachedimage = cachedImage.Name
				rows[i].Image = cachedImage.Name
				rows[i].Size = cachedImage.Size
			}
		}
		if cdromRefs != nil {
			if refMap, ok := cdromRefs[sci.StoragecacheId]; ok {
				rows[i].CdromReference = refMap[sci.CachedimageId]
			}
		}
		if diskRefs != nil {
			if refMap, ok := diskRefs[sci.StoragecacheId]; ok {
				rows[i].DiskReference = refMap[sci.CachedimageId]
				rows[i].Reference = rows[i].CdromReference + rows[i].DiskReference
			}
		}
	}

	return rows
}

func (self *SStoragecachedimage) GetCachedimage() *SCachedimage {
	cachedImage, _ := CachedimageManager.FetchById(self.CachedimageId)
	if cachedImage != nil {
		return cachedImage.(*SCachedimage)
	}
	return nil
}

/*func (self *SStoragecachedimage) getExtraDetails(ctx context.Context, out api.StoragecachedimageDetails) api.StoragecachedimageDetails {
	storagecache := self.GetStoragecache()
	if storagecache != nil {
		// out.Storagecache = storagecache.Name
		out.Storages = storagecache.getStorageNames()
		host, _ := storagecache.GetMasterHost()
		if host != nil {
			out.Host = host.GetShortDesc(ctx)
		} else {
			var err error
			hostDesc, err := storagecache.GetEsxiAgentHostDesc()
			if err != nil {
				log.Errorf("unable to GetEsxiAgentHostDesc of stroagecache: %s", err.Error())
			}
			if hostDesc != nil {
				out.Host = hostDesc
			}
		}
	}
	cachedImage := self.GetCachedimage()
	if cachedImage != nil {
		out.Cachedimage = cachedImage.Name
		out.Image = cachedImage.GetName()
		out.Size = cachedImage.Size
	}
	out.Reference, _ = self.getReferenceCount()
	return out
}*/

func (manager *SStoragecachedimageManager) fetchCdromReferenceCounts(storagecacheIds []string, imageIds []string) (map[string]map[string]int, error) {
	q := GuestcdromManager.Query()
	guests := GuestManager.Query().SubQuery()
	hostStorages := HoststorageManager.Query().SubQuery()
	storages := StorageManager.Query().SubQuery()

	q = q.Join(guests, sqlchemy.Equals(q.Field("id"), guests.Field("id")))
	q = q.Join(hostStorages, sqlchemy.Equals(guests.Field("host_id"), hostStorages.Field("host_id")))
	q = q.Join(storages, sqlchemy.Equals(hostStorages.Field("storage_id"), storages.Field("id")))

	q = q.GroupBy(q.Field("image_id"))
	q = q.GroupBy(storages.Field("storagecache_id"))

	q = q.Filter(sqlchemy.In(q.Field("image_id"), imageIds))
	q = q.Filter(sqlchemy.In(storages.Field("storagecache_id"), storagecacheIds))

	q = q.AppendField(sqlchemy.COUNT("ref_count"))
	q = q.AppendField(q.Field("image_id"))
	q = q.AppendField(storages.Field("storagecache_id"))

	return manager.fetchRefCount(q)
}

func (maanger *SStoragecachedimageManager) fetchRefCount(q *sqlchemy.SQuery) (map[string]map[string]int, error) {
	results := []struct {
		RefCount       int    `json:"ref_count"`
		ImageId        string `json:"image_id"`
		StoragecacheId string `json:"storagecache_id"`
	}{}

	err := q.All(&results)
	if err != nil {
		return nil, errors.Wrap(err, "Query")
	}

	ret := make(map[string]map[string]int)
	for _, r := range results {
		if _, ok := ret[r.StoragecacheId]; !ok {
			ret[r.StoragecacheId] = make(map[string]int)
		}
		ret[r.StoragecacheId][r.ImageId] = r.RefCount
	}

	return ret, nil
}

func (self *SStoragecachedimage) getCdromReferenceCount() (int, error) {
	cdroms := GuestcdromManager.Query().SubQuery()
	guests := GuestManager.Query().SubQuery()

	q := cdroms.Query()
	q = q.Join(guests, sqlchemy.Equals(cdroms.Field("id"), guests.Field("id")))
	q = q.Filter(sqlchemy.Equals(cdroms.Field("image_id"), self.CachedimageId))
	return q.CountWithError()
}

func (manager *SStoragecachedimageManager) fetchDiskReferenceCounts(storagecacheIds, imageIds []string) (map[string]map[string]int, error) {
	disks := DiskManager.Query().SubQuery()
	storages := StorageManager.Query().SubQuery()

	q := disks.Query()

	q = q.Join(storages, sqlchemy.Equals(disks.Field("storage_id"), storages.Field("id")))

	q = q.GroupBy(disks.Field("template_id"))
	q = q.GroupBy(storages.Field("storagecache_id"))

	q = q.Filter(sqlchemy.In(disks.Field("template_id"), imageIds))
	q = q.Filter(sqlchemy.In(storages.Field("storagecache_id"), storagecacheIds))

	q = q.AppendField(sqlchemy.COUNT("ref_count"))
	q = q.AppendField(disks.Field("template_id").Label("image_id"))
	q = q.AppendField(storages.Field("storagecache_id"))

	return manager.fetchRefCount(q)
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

func (manager *SStoragecachedimageManager) RecoverStoragecachedImage(
	ctx context.Context, userCred mcclient.TokenCredential, scId, imgId string,
) (*SStoragecachedimage, error) {
	lockman.LockRawObject(ctx, manager.Keyword(), "name")
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

	storagecachedImage := SStoragecachedimage{}
	storagecachedImage.SetModelManager(manager, &storagecachedImage)

	err := manager.RawQuery().Equals("storagecache_id", scId).Equals("cachedimage_id", imgId).First(&storagecachedImage)
	if err != nil {
		return nil, err
	}
	diff, err := db.Update(&storagecachedImage, func() error {
		storagecachedImage.Status = api.CACHED_IMAGE_STATUS_ACTIVE
		if storagecachedImage.Deleted == true {
			storagecachedImage.Deleted = false
			storagecachedImage.DeletedAt = time.Time{}
			storagecachedImage.UpdateVersion = 0
		}
		return nil
	})

	db.OpsLog.LogEvent(&storagecachedImage, db.ACT_UPDATE, diff, userCred)
	return &storagecachedImage, nil
}

func (self *SStoragecachedimage) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"delete from %s where row_id = ?",
			self.GetModelManager().TableSpec().Name(),
		), self.RowId,
	)
	return err
}

func (self *SStoragecachedimage) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

func (self *SStoragecachedimage) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if self.Status != api.CACHED_IMAGE_STATUS_CACHE_FAILED {
		cnt, err := self.getReferenceCount()
		if err != nil {
			return httperrors.NewInternalServerError("getReferenceCount fail %s", err)
		}
		if cnt > 0 {
			return httperrors.NewNotEmptyError("Image is in use")
		}
	}
	return self.SJointResourceBase.ValidateDeleteCondition(ctx, nil)
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
	err := self.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}
	if !isForce {
		err = self.isCachedImageInUse()
		if err != nil {
			return err
		}
	}

	cache := self.GetStoragecache()
	image := self.GetCachedimage()

	if image != nil {
		lockman.LockJointObject(ctx, cache, image)
		defer lockman.ReleaseJointObject(ctx, cache, image)
	}

	if !isForce && !utils.IsInStringArray(self.Status,
		[]string{api.CACHED_IMAGE_STATUS_ACTIVE, api.CACHED_IMAGE_STATUS_DELETING, api.CACHED_IMAGE_STATUS_CACHE_FAILED}) {
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

	err := manager.TableSpec().Insert(ctx, cachedimage)

	if err != nil {
		log.Errorf("insert error %s", err)
		return nil
	}

	return cachedimage
}

func (self *SStoragecachedimage) SetStatus(ctx context.Context, userCred mcclient.TokenCredential, status string, reason string) error {
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
	err := self.Delete(ctx, userCred)
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

func (self *SStoragecachedimage) syncWithCloudImage(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, image cloudprovider.ICloudImage, managerId string) error {
	cachedImage := self.GetCachedimage()
	if len(self.ExternalId) == 0 {
		self.SetExternalId(cachedImage.GetExternalId())
	}
	if len(cachedImage.ExternalId) > 0 {
		self.SetStatus(ctx, userCred, image.GetStatus(), "")
		return cachedImage.syncWithCloudImage(ctx, userCred, ownerId, image, nil)
	} else {
		return nil
	}
}

func (manager *SStoragecachedimageManager) newFromCloudImage(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, image cloudprovider.ICloudImage, cache *SStoragecache) error {
	var cachedImage *SCachedimage
	provider := cache.GetCloudprovider()
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
			cachedImage, err = CachedimageManager.newFromCloudImage(ctx, userCred, ownerId, image, provider)
			if err != nil {
				log.Errorf("CachedimageManager.newFromCloudImage fail %s", err)
				return err
			}
		}
	} else {
		cachedImage = imgObj.(*SCachedimage)
	}
	if len(cachedImage.ExternalId) > 0 {
		cachedImage.syncWithCloudImage(ctx, userCred, ownerId, image, provider)
	}
	scimg := manager.Register(ctx, userCred, cache.GetId(), cachedImage.GetId(), image.GetStatus())
	if scimg == nil {
		return fmt.Errorf("register cached image fail")
	}
	return scimg.SetExternalId(image.GetGlobalId())
}

func (manager *SStoragecachedimageManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.StoragecachedimageListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SJointResourceBaseManager.ListItemFilter(ctx, q, userCred, query.JointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSchedtagJointsManager.ListItemFilter")
	}
	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SStoragecacheResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StoragecacheFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStoragecacheResourceBaseManager.ListItemFilter")
	}

	if len(query.CachedimageId) > 0 {
		cachedImageObj, err := CachedimageManager.FetchByIdOrName(ctx, userCred, query.CachedimageId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CachedimageManager.Keyword(), query.CachedimageId)
			} else {
				return nil, errors.Wrap(err, "CachedimageManager.FetchByIdOrName")
			}
		}
		q = q.Equals("cachedimage_id", cachedImageObj.GetId())
	}

	if len(query.Status) > 0 {
		q = q.In("status", query.Status)
	}

	return q, nil
}

func (manager *SStoragecachedimageManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.StoragecachedimageListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SJointResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.JointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSchedtagJointsManager.OrderByExtraFields")
	}
	q, err = manager.SStoragecacheResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StoragecacheFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStoragecacheResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SStoragecachedimageManager) InitializeData() error {
	images := []SStoragecachedimage{}
	q := manager.Query().Equals("status", "ready")
	err := db.FetchModelObjects(manager, q, &images)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range images {
		_, err := db.Update(&images[i], func() error {
			images[i].Status = api.CACHED_IMAGE_STATUS_ACTIVE
			return nil
		})
		if err != nil {
			return errors.Wrapf(err, "db.Update(%d)", images[i].RowId)
		}
	}
	return nil
}
