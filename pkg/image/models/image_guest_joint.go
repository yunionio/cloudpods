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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
)

type SGuestImageJointManager struct {
	db.SJointResourceBaseManager
}

type SGuestImageJoint struct {
	db.SJointResourceBase

	GuestImageId string `width:"128" charset:"ascii" create:"required"`
	ImageId      string `width:"128" charset:"ascii" create:"required"`
}

var GuestImageJointManager *SGuestImageJointManager

func init() {
	GuestImageJointManager = &SGuestImageJointManager{
		db.NewJointResourceBaseManager(
			SGuestImageJoint{},
			"guest_image_tbl",
			"guestimagejoint",
			"guestimagejoints",
			GuestImageManager,
			ImageManager,
		),
	}
	GuestImageJointManager.SetVirtualObject(GuestImageJointManager)
}

func (gm *SGuestImageJointManager) GetByGuestImageId(guestImageId string) ([]SGuestImageJoint, error) {
	q := gm.Query().Equals("guest_image_id", guestImageId)
	ret := make([]SGuestImageJoint, 0, 1)
	err := db.FetchModelObjects(gm, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (gm *SGuestImageJointManager) GetByImageId(imageId string) (*SGuestImageJoint, error) {
	q := gm.Query().Equals("image_id", imageId)
	ret := SGuestImageJoint{}
	err := q.First(&ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (gm *SGuestImageJointManager) GetGuestImageByImageId(imageId string) (*SGuestImage, error) {
	gits, err := gm.GetByImageId(imageId)
	if err != nil {
		return nil, err
	}
	model, err := GuestImageManager.FetchById(gits.GuestImageId)
	if err != nil {
		return nil, err
	}
	return model.(*SGuestImage), nil
}

func (gm *SGuestImageJointManager) GetImagesByGuestImageId(guestImageId string) ([]SImage, error) {
	giJoints, err := gm.GetByGuestImageId(guestImageId)
	if err != nil {
		return nil, errors.Wrap(err, "get joints of guest and image failed")
	}
	if len(giJoints) == 0 {
		return []SImage{}, nil
	}
	imageIds := make([]string, len(giJoints))
	for i := range giJoints {
		imageIds[i] = giJoints[i].ImageId
	}
	q := ImageManager.Query().In("id", imageIds)
	images := make([]SImage, 0, len(imageIds))
	err = db.FetchModelObjects(ImageManager, q, &images)
	if err != nil {
		return nil, errors.Wrap(err, "fetch images failed")
	}
	return images, nil
}

func (gt *SGuestImageJoint) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, gt)
}

func (gt *SGuestImageJointManager) CreateGuestImageJoint(ctx context.Context, guestImageId,
	imageId string) (*SGuestImageJoint, error) {

	gi := SGuestImageJoint{}
	gi.GuestImageId = guestImageId
	gi.ImageId = imageId

	//
	if err := gt.TableSpec().Insert(&gi); err != nil {
		return nil, errors.Wrapf(err, "insert guestimage joint error")
	}
	gi.SetVirtualObject(gt)
	return &gi, nil
}
