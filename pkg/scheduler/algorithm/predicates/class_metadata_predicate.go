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

package predicates

import (
	"context"
	"database/sql"
	"fmt"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type ClassMetadataPredicate struct {
	BasePredicate
	cacheImage *models.SCachedimage
	//TODO GuestImage
	snapshot         *models.SSnapshot
	instanceSnapshot *models.SInstanceSnapshot
	backup           *models.SDiskBackup
	instanceBackup   *models.SInstanceBackup
}

func (p *ClassMetadataPredicate) Name() string {
	return "class_metadata"
}

func (p *ClassMetadataPredicate) Clone() core.FitPredicate {
	return &ClassMetadataPredicate{
		cacheImage:       p.cacheImage,
		snapshot:         p.snapshot,
		instanceSnapshot: p.instanceSnapshot,
		backup:           p.backup,
		instanceBackup:   p.instanceBackup,
	}
}

func (p *ClassMetadataPredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	info := u.SchedData()
	if len(info.InstanceSnapshotId) > 0 {
		obj, err := models.InstanceSnapshotManager.FetchById(info.InstanceSnapshotId)
		if err != nil {
			return false, errors.Wrapf(err, "unable to fetch instanceSnapshot %s", info.InstanceSnapshotId)
		}
		p.instanceSnapshot = obj.(*models.SInstanceSnapshot)
		return true, nil
	}
	if len(info.InstanceBackupId) > 0 {
		obj, err := models.InstanceBackupManager.FetchById(info.InstanceBackupId)
		if err != nil {
			return false, errors.Wrapf(err, "unable to fetch instanceBackup %s", info.InstanceBackupId)
		}
		p.instanceBackup = obj.(*models.SInstanceBackup)
		return true, nil
	}
	disks := info.Disks
	if len(disks) == 0 {
		return false, nil
	}
	switch {
	case disks[0].ImageId != "":
		obj, err := models.CachedimageManager.FetchById(disks[0].ImageId)
		if err != nil {
			// 忽略第一次上传到glance镜像后未缓存的记录
			if err == sql.ErrNoRows {
				return false, nil
			}
			return false, errors.Wrapf(err, "unable to fetch cachedimage %s", disks[0].ImageId)
		}
		p.cacheImage = obj.(*models.SCachedimage)
		return true, nil
	case disks[0].SnapshotId != "":
		obj, err := models.SnapshotManager.FetchById(disks[0].SnapshotId)
		if err != nil {
			return false, errors.Wrapf(err, "unable to fetch snapshot %s", disks[0].SnapshotId)
		}
		p.snapshot = obj.(*models.SSnapshot)
		return true, nil
	case disks[0].BackupId != "":
		obj, err := models.DiskBackupManager.FetchById(disks[0].BackupId)
		if err != nil {
			return false, errors.Wrapf(err, "unable to fetch diskbackup %s", disks[0].BackupId)
		}
		p.backup = obj.(*models.SDiskBackup)
		return true, nil
	}
	return false, nil
}

func (p *ClassMetadataPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := NewPredicateHelper(p, u, c)
	host := c.Getter().Host()
	ctx := context.Background()
	resourceDesc := ""
	var sara *db.SStandaloneAnonResourceBase
	switch {
	case p.cacheImage != nil:
		sara = &p.cacheImage.SStandaloneAnonResourceBase
		resourceDesc = fmt.Sprintf("image %s", p.cacheImage.GetName())
	case p.snapshot != nil:
		sara = &p.snapshot.SStandaloneAnonResourceBase
		resourceDesc = fmt.Sprintf("snapshot %s", p.snapshot.GetName())
	case p.backup != nil:
		sara = &p.backup.SStandaloneAnonResourceBase
		resourceDesc = fmt.Sprintf("backup %s", p.backup.GetName())
	case p.instanceBackup != nil:
		sara = &p.instanceBackup.SStandaloneAnonResourceBase
		resourceDesc = fmt.Sprintf("instance backup %s", p.instanceBackup.GetName())
	case p.instanceSnapshot != nil:
		sara = &p.instanceSnapshot.SStandaloneAnonResourceBase
		resourceDesc = fmt.Sprintf("instance snapshot %s", p.instanceSnapshot.GetName())
	}
	ic, err := host.IsInSameClass(ctx, sara)
	if err != nil {
		return false, nil, errors.Wrap(err, "unable to determine whether they are in a class")
	}
	if !ic {
		h.Exclude(fmt.Sprintf("The host doesn't have the same class metadata as the choosen %s.", resourceDesc))
	}
	return h.GetResult()
}
