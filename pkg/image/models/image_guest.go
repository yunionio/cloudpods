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
	"fmt"
	"sort"
	"time"

	"golang.org/x/sync/errgroup"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/image/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SGuestImageManager struct {
	db.SSharableVirtualResourceBaseManager
}

var GuestImageManager *SGuestImageManager

func init() {
	GuestImageManager = &SGuestImageManager{
		db.NewSharableVirtualResourceBaseManager(
			SGuestImage{},
			"guestimages_tbl",
			"guestimage",
			"guestimages",
		),
	}
	GuestImageManager.SetVirtualObject(GuestImageManager)
}

type SGuestImage struct {
	db.SSharableVirtualResourceBase

	Protected tristate.TriState `nullable:"false" default:"true" list:"user" get:"user" create:"optional" update:"user"`
}

func (manager *SGuestImageManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {

	if !data.Contains("image_number") {
		return nil, httperrors.NewMissingParameterError("image_number")
	}
	imageNum, _ := data.Int("image_number")

	pendingUsage := SQuota{Image: int(imageNum)}
	keys := quotas.OwnerIdQuotaKeys(rbacutils.ScopeProject, ownerId)
	pendingUsage.SetKeys(keys)
	if err := quotas.CheckSetPendingQuota(ctx, userCred, &pendingUsage); err != nil {

		return nil, httperrors.NewOutOfQuotaError("%s", err)
	}
	return data, nil
}

func (gi *SGuestImage) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {

	err := gi.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
	if err != nil {
		return err
	}
	gi.Status = api.IMAGE_STATUS_QUEUED
	return nil
}

func (gi *SGuestImage) PostCreate(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {

	kwargs := data.(*jsonutils.JSONDict)
	// deal public params
	kwargs.Remove("size")
	kwargs.Remove("image_number")
	kwargs.Remove("name")
	if !kwargs.Contains("images") {
		return
	}
	images, _ := kwargs.GetArray("images")
	kwargs.Remove("images")
	kwargs.Add(jsonutils.NewString(gi.Id), "guest_image_id")

	imageIds := jsonutils.NewArray()
	suc := true

	// HACK
	appParams := appsrv.AppContextGetParams(ctx)
	appParams.Request.ContentLength = 0

	for i := 0; i < len(images); i++ {
		params := jsonutils.DeepCopy(kwargs).(*jsonutils.JSONDict)
		image := images[i].(*jsonutils.JSONDict)
		for _, key := range image.SortedKeys() {
			tmp, _ := image.Get(key)
			params.Add(tmp, key)
		}
		if i == len(images)-1 {
			params.Add(jsonutils.NewString(fmt.Sprintf("%s-%s", gi.Name, "root")), "generate_name")
		} else {
			params.Add(jsonutils.NewString(fmt.Sprintf("%s-%s-%d", gi.Name, "data", i)), "generate_name")
			params.Add(jsonutils.JSONTrue, "is_data")
		}
		model, err := db.DoCreate(ImageManager, ctx, userCred, query, params, ownerId)
		if err != nil {
			imageIds.Add(jsonutils.NewString(""))
			suc = false
			break
		} else {
			func() {
				lockman.LockObject(ctx, model)
				defer lockman.ReleaseObject(ctx, model)

				model.PostCreate(ctx, userCred, ownerId, query, data)
			}()
			imageIds.Add(jsonutils.NewString(model.GetId()))
		}
	}

	imageNumber, _ := data.Int("image_number")
	pendingUsage := SQuota{Image: int(imageNumber)}
	keys := quotas.OwnerIdQuotaKeys(rbacutils.ScopeProject, ownerId)
	pendingUsage.SetKeys(keys)
	quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage)

	if !suc {
		gi.SetStatus(userCred, api.IMAGE_STATUS_KILLED, "create subimage failed")
	}

	// HACK
	tmp := query.(*jsonutils.JSONDict)
	tmp.Add(imageIds, "image_ids")
}

func (gi *SGuestImage) ValidateDeleteCondition(ctx context.Context) error {
	if gi.Protected.IsTrue() {
		return httperrors.NewForbiddenError("image is protected")
	}
	return nil
}

func (gi *SGuestImage) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("image delete to nothing")
	return nil
}

func (gi *SGuestImage) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	// delete joint
	guestJoints, err := GuestImageJointManager.GetByGuestImageId(gi.Id)
	if err != nil {
		return errors.Wrap(err, "get guest image joint failed")
	}
	for i := range guestJoints {
		guestJoints[i].Delete(ctx, userCred)
	}
	return gi.SVirtualResourceBase.Delete(ctx, userCred)
}

func (gi *SGuestImage) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) error {

	images, err := GuestImageJointManager.GetImagesByGuestImageId(gi.Id)
	if err != nil {
		return errors.Wrap(err, "get images of guest images failed")
	}
	if len(images) == 0 {
		return gi.RealDelete(ctx, userCred)
	}
	overridePendingDelete := false
	purge := false
	if query != nil {
		overridePendingDelete = jsonutils.QueryBoolean(query, "override_pending_delete", false)
		purge = jsonutils.QueryBoolean(query, "purge", false)
	}
	if gi.Status == api.IMAGE_STATUS_QUEUED {
		gi.checkStatus(ctx, userCred)
	}
	if utils.IsInStringArray(gi.Status, []string{api.IMAGE_STATUS_QUEUED, api.IMAGE_STATUS_KILLED}) {
		overridePendingDelete = true
	}
	return gi.startDeleteTask(ctx, userCred, "", purge, overridePendingDelete)
}

func (gi *SGuestImage) startDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string,
	isPurge bool, overridePendingDelete bool) error {

	params := jsonutils.NewDict()
	if isPurge {
		params.Add(jsonutils.JSONTrue, "purge")
	}
	if overridePendingDelete {
		params.Add(jsonutils.JSONTrue, "override_pending_delete")
	}
	params.Add(jsonutils.NewString(gi.Status), "image_status")
	gi.SetStatus(userCred, api.IMAGE_STATUS_DEACTIVATED, "")
	if task, err := taskman.TaskManager.NewTask(ctx, "GuestImageDeleteTask", gi, userCred, params, parentTaskId, "",
		nil); err != nil {

		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (gi *SGuestImage) AllowPerformCancelDelete(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) bool {

	return db.IsAdminAllowPerform(userCred, gi, "cancel-delete")
}

func (gi *SGuestImage) PerformCancelDelete(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	if gi.PendingDeleted {
		err := gi.DoCancelPendingDelete(ctx, userCred)
		return nil, err
	}
	return nil, nil
}

func (gi *SGuestImage) DoCancelPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	subImages, err := GuestImageJointManager.GetImagesByGuestImageId(gi.Id)
	for i := range subImages {
		err = subImages[i].DoCancelPendingDelete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "subimage %s cancel delete error", subImages[i].GetId())
		}
	}
	err = gi.SVirtualResourceBase.DoCancelPendingDelete(ctx, userCred)
	if err != nil {
		return err
	}
	_, err = db.Update(gi, func() error {
		gi.Status = api.IMAGE_STATUS_ACTIVE
		return nil
	})
	return errors.Wrap(err, "guest image cancel delete error")
}

type sPair struct {
	ID         string
	Name       string
	MinDiskMB  int32
	DiskFormat string
	Size       int64
	Status     string
}

func (self *SGuestImage) getMoreDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	extra *jsonutils.JSONDict) *jsonutils.JSONDict {

	if self.Status != api.IMAGE_STATUS_ACTIVE {
		self.checkStatus(ctx, userCred)
		extra.Add(jsonutils.NewString(self.Status), "status")
	}
	images, err := GuestImageJointManager.GetImagesByGuestImageId(self.Id)
	if err != nil {
		return extra
	}
	var size int64 = 0
	if len(images) == 0 {
		extra.Add(jsonutils.NewInt(size), "size")
		return extra
	}
	dataImages := make([]sPair, 0, len(images)-1)
	var rootImage sPair
	for i := range images {
		image := images[i]
		size += image.Size
		if !image.IsData.IsTrue() {
			rootImage = sPair{image.Id, images[i].Name, image.MinDiskMB, image.DiskFormat, image.Size, image.Status}
			extra.Add(jsonutils.NewInt(int64(image.MinRamMB)), "min_ram_mb")
			extra.Add(jsonutils.NewString(image.DiskFormat), "disk_format")
			continue
		}
		dataImages = append(dataImages, sPair{image.Id, image.Name, image.MinDiskMB, image.DiskFormat, image.Size,
			image.Status})
	}
	// make sure that the sort of dataimage is fixed
	sort.Slice(dataImages, func(i, j int) bool {
		return dataImages[i].Name < dataImages[j].Name
	})
	extra.Add(jsonutils.NewInt(size), "size")
	extra.Add(jsonutils.Marshal(rootImage), "root_image")
	extra.Add(jsonutils.Marshal(dataImages), "data_images")
	// properties of root image
	properties, _ := ImagePropertyManager.GetProperties(rootImage.ID)
	if err != nil {
		return extra
	}
	propJson := jsonutils.NewDict()
	for k, v := range properties {
		propJson.Add(jsonutils.NewString(v), k)
	}
	extra.Add(propJson, "properties")
	return extra
}

func (self *SGuestImage) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) *jsonutils.JSONDict {

	extra := self.SSharableVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(ctx, userCred, query, extra)
}

func (self *SGuestImage) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {

	extra, err := self.SSharableVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	if query.Contains("image_ids") {
		imageIds, _ := query.Get("image_ids")
		extra.Add(imageIds, "image_ids")
		return extra, nil
	}

	return self.getMoreDetails(ctx, userCred, query, extra), nil
}

func (self *SGuestImage) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) {

	lockman.LockClass(ctx, ImageManager, db.GetLockClassKey(ImageManager, userCred))
	defer lockman.ReleaseClass(ctx, ImageManager, db.GetLockClassKey(ImageManager, userCred))

	err := self.UpdateSubImage(ctx, userCred, data)
	if err != nil {
		logclient.AddSimpleActionLog(self, logclient.ACT_UPDATE, nil, userCred, false)
	}

}

func (self *SGuestImage) UpdateSubImage(ctx context.Context, userCred mcclient.TokenCredential,
	data jsonutils.JSONObject) error {

	subImages, err := GuestImageJointManager.GetImagesByFilter(self.GetId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Asc("name")
	})
	if err != nil {
		return err
	}
	dict := data.(*jsonutils.JSONDict)
	var g errgroup.Group
	for i := range subImages {
		if f, ok := self.genUpdateImage(ctx, userCred, &subImages[i], i, dict); ok {
			g.Go(f)
		}
	}
	return g.Wait()
}

func (self *SGuestImage) genUpdateImage(ctx context.Context, userCred mcclient.TokenCredential, image *SImage,
	index int, dict *jsonutils.JSONDict) (func() error, bool) {

	if !dict.Contains("name") && image.IsData.IsTrue() {
		return nil, false
	}
	if image.IsData.IsTrue() {
		if !dict.Contains("name") {
			return nil, false
		}
		return func() error {
			name, _ := dict.GetString("name")
			name, err := db.GenerateName(ImageManager, userCred, fmt.Sprintf("%s-%s-%d", name, "data", index))
			if err != nil {
				return errors.Wrap(err, "fail to generate unique name")
			}
			_, err = db.Update(image, func() error {
				image.Name = name
				return nil
			})
			if err != nil {
				return errors.Wrap(err, "modify subimage's name failed")
			}
			return nil
		}, true
	}

	return func() error {
		if dict.Contains("name") {
			name, _ := dict.GetString("name")
			name, err := db.GenerateName(ImageManager, userCred, fmt.Sprintf("%s-%s", name, "root"))
			if err != nil {
				return errors.Wrap(err, "fail to generate unique name")
			}
			_, err = db.Update(image, func() error {
				image.Name = name
				return nil
			})
			if err != nil {
				return errors.Wrap(err, "modify subimage's name failed")
			}
		}
		if dict.Contains("properties") {
			props, _ := dict.Get("properties")
			err := ImagePropertyManager.SaveProperties(ctx, userCred, image.GetId(), props)
			if err != nil {
				return errors.Wrap(err, "save properties error")
			}
		}
		return nil
	}, true
}

var checkStatus = map[string]int{
	api.IMAGE_STATUS_ACTIVE:      1,
	api.IMAGE_STATUS_QUEUED:      2,
	api.IMAGE_STATUS_SAVING:      3,
	api.IMAGE_STATUS_DEACTIVATED: 4,
	api.IMAGE_STATUS_KILLED:      5,
}

func (self *SGuestImage) checkStatus(ctx context.Context, userCred mcclient.TokenCredential) error {
	images, err := GuestImageJointManager.GetImagesByGuestImageId(self.Id)
	if err != nil {
		return err
	}
	if len(images) == 0 {
		return nil
	}

	status := api.IMAGE_STATUS_ACTIVE

	for i := range images {
		if checkStatus[images[i].Status] > checkStatus[status] {
			status = images[i].Status
		}
	}
	if self.Status != status {
		self.SetStatus(userCred, status, "")
		self.Status = status
	}
	return nil
}

func (self *SGuestImage) getSize(ctx context.Context, userCred mcclient.TokenCredential) (int64, error) {
	images, err := GuestImageJointManager.GetImagesByGuestImageId(self.Id)
	if err != nil {
		return 0, err
	}
	var size int64 = 0
	for i := range images {
		size += images[i].Size
	}
	return size, nil
}

func (self *SGuestImageManager) getExpiredPendingDeleteImages() []SGuestImage {
	deadline := time.Now().Add(time.Duration(-options.Options.PendingDeleteExpireSeconds) * time.Second)

	// there are so many common images of one guest image, so that batch shrink three times
	q := self.Query().IsTrue("pending_deleted").LT("pending_deleted_at",
		deadline).Limit(options.Options.PendingDeleteMaxCleanBatchSize / 3)

	images := make([]SGuestImage, 0)
	err := db.FetchModelObjects(self, q, &images)
	if err != nil {
		log.Errorf("fetch guest images error %s", err)
		return nil
	}
	return images
}

func (self *SGuestImageManager) CleanPendingDeleteImages(ctx context.Context, userCred mcclient.TokenCredential,
	isStart bool) {

	images := self.getExpiredPendingDeleteImages()
	if images == nil {
		return
	}
	for i := range images {
		images[i].startDeleteTask(ctx, userCred, "", false, true)
	}
}

func (self *SGuestImage) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	images, err := GuestImageJointManager.GetImagesByGuestImageId(self.Id)
	if err != nil {
		return nil, errors.Wrap(err, "fail to fetch subimages of guest image")
	}
	for i := range images {
		_, err := images[i].PerformPublic(ctx, userCred, query, data)
		if err != nil {
			return nil, errors.Wrapf(err, "fail to public subimage %s", images[i].GetId())
		}
	}
	return self.SSharableVirtualResourceBase.PerformPublic(ctx, userCred, query, data)
}

func (self *SGuestImage) PerformPrivate(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	images, err := GuestImageJointManager.GetImagesByGuestImageId(self.Id)
	if err != nil {
		return nil, errors.Wrap(err, "fail to fetch subimages of guest image")
	}
	for i := range images {
		_, err := images[i].PerformPrivate(ctx, userCred, query, data)
		if err != nil {
			return nil, errors.Wrapf(err, "fail to private subimage %s", images[i].GetId())
		}
	}
	return self.SSharableVirtualResourceBase.PerformPrivate(ctx, userCred, query, data)
}
