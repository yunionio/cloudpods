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

package guest

import (
	"database/sql"
	"fmt"

	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type ImagePredicate struct {
	predicates.BasePredicate
	cacheImage *models.SCachedimage
	zones      []string
}

func (f *ImagePredicate) Name() string {
	return "disk_image"
}

func (f *ImagePredicate) Clone() core.FitPredicate {
	return &ImagePredicate{}
}

func (f *ImagePredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	disks := u.SchedData().Disks
	if len(disks) == 0 {
		return false, nil
	}
	imageId := disks[0].ImageId
	if len(imageId) == 0 || u.SchedData().PreferZone != "" {
		return false, nil
	}
	if !utils.IsInStringArray(u.SchedData().Hypervisor, compute.PUBLIC_CLOUD_HYPERVISORS) && !utils.IsInStringArray(u.SchedData().Hypervisor, compute.PRIVATE_CLOUD_HYPERVISORS) {
		return false, nil
	}
	obj, err := models.CachedimageManager.FetchById(imageId)
	if err != nil {
		// 忽略第一次上传到glance镜像后未缓存的记录
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("Fetch CachedImage %s: %v", imageId, err)
	}
	cacheImage := obj.(*models.SCachedimage)
	if cacheImage.ImageType != cloudprovider.CachedImageTypeSystem {
		return false, nil
	}
	zones, err := cacheImage.GetUsableZoneIds()
	if err != nil {
		return false, fmt.Errorf("Fetch CachedImage %s zones: %v", cacheImage.GetName(), err)
	}
	f.cacheImage = cacheImage
	f.zones = zones
	return true, nil
}

func (f *ImagePredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(f, u, c)
	inZone := false
	hostZoneId := c.Getter().Zone().GetId()
	if utils.IsInStringArray(hostZoneId, f.zones) {
		inZone = true
	}
	if !inZone {
		h.Exclude(fmt.Sprintf("Host zone %s not in image usable zones %v", hostZoneId, f.zones))
	}
	return h.GetResult()
}
