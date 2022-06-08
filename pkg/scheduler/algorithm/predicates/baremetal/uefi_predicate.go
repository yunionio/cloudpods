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

package baremetal

import (
	"context"
	"database/sql"
	"fmt"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type UEFIImagePredicate struct {
	predicates.BasePredicate
	cacheImage *models.SCachedimage
}

func (f *UEFIImagePredicate) Name() string {
	return "uefi_image"
}

func (f *UEFIImagePredicate) Clone() core.FitPredicate {
	return new(UEFIImagePredicate)
}

func (f *UEFIImagePredicate) PreExecute(ctx context.Context, u *core.Unit, cs []core.Candidater) (bool, error) {
	disks := u.SchedData().Disks
	if len(disks) == 0 {
		return false, nil
	}
	imageId := disks[0].ImageId
	if len(imageId) == 0 {
		return false, nil
	}
	obj, err := models.CachedimageManager.FetchById(imageId)
	if err != nil {
		// 忽略第一次上传到glance镜像后未缓存的记录
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, errors.Wrapf(err, "fetch cached image %q", imageId)
	}
	cacheImage := obj.(*models.SCachedimage)
	f.cacheImage = cacheImage
	return true, nil
}

func (f *UEFIImagePredicate) isImageUEFI() bool {
	if f.cacheImage.UEFI.Bool() {
		return true
	}
	support, err := f.cacheImage.Info.Bool("properties", "uefi_support")
	if err != nil {
		return false
	}
	return support
}

func (f *UEFIImagePredicate) Execute(ctx context.Context, u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(f, u, c)
	imgName := f.cacheImage.GetName()
	hostName := c.Getter().Name()
	imageMsg := fmt.Sprintf("image %s is not UEFI", imgName)
	hostMsg := fmt.Sprintf("host %s is not UEFI boot", hostName)
	isUEFIImage := f.isImageUEFI()
	if isUEFIImage {
		imageMsg = fmt.Sprintf("image %s is UEFI", imgName)
	}
	isUEFIHost := c.Getter().Host().IsUEFIBoot()
	if isUEFIHost {
		hostMsg = fmt.Sprintf("host %s is UEFI boot", hostName)
	}
	if isUEFIImage != isUEFIHost {
		h.Exclude(imageMsg + " but " + hostMsg)
	}
	return h.GetResult()
}
