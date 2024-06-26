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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type ClassMetadataPredicate struct {
	BasePredicate

	guestSource *ResourceWithClassMetadata
	tenant      *ResourceWithClassMetadata
}

type ResourceWithClassMetadata struct {
	keyword     string
	name        string
	classMedata map[string]string
}

func (rcm *ResourceWithClassMetadata) GetAllClassMetadata() (map[string]string, error) {
	return rcm.classMedata, nil
}

func (rcm *ResourceWithClassMetadata) GetDescription() string {
	return fmt.Sprintf("%s %s", rcm.keyword, rcm.name)
}

func (p *ClassMetadataPredicate) Name() string {
	return "class_metadata"
}

func (p *ClassMetadataPredicate) Clone() core.FitPredicate {
	return &ClassMetadataPredicate{
		guestSource: p.guestSource,
		tenant:      p.tenant,
	}
}

func (p *ClassMetadataPredicate) PreExecute(ctx context.Context, u *core.Unit, cs []core.Candidater) (bool, error) {
	info := u.SchedData()
	if info.ResetCpuNumaPin {
		return false, nil
	}

	tenant, err := db.TenantCacheManager.FetchTenantById(ctx, info.Project)
	if err != nil {
		return false, errors.Wrapf(err, "unable to fetch tenant %s", info.Project)
	}
	tcm, err := tenant.GetAllClassMetadata()
	if err != nil {
		return false, errors.Wrapf(err, "unable to GetAllClassMetadata of project %s", info.Project)
	}
	p.tenant = &ResourceWithClassMetadata{
		classMedata: tcm,
		keyword:     tenant.Keyword(),
		name:        tenant.GetName(),
	}

	// guest source
	guestSource := &ResourceWithClassMetadata{}
	disks := info.Disks
	var stand db.IStandaloneModel
	// TODO GuestImage
	switch {
	case len(info.InstanceBackupId) > 0:
		obj, err := models.InstanceBackupManager.FetchById(info.InstanceBackupId)
		if err != nil {
			return false, errors.Wrapf(err, "unable to fetch instanceBackup %s", info.InstanceBackupId)
		}
		stand = obj.(db.IStandaloneModel)
	case len(info.InstanceSnapshotId) > 0:
		obj, err := models.InstanceSnapshotManager.FetchById(info.InstanceSnapshotId)
		if err != nil {
			return false, errors.Wrapf(err, "unable to fetch instanceSnapshot %s", info.InstanceSnapshotId)
		}
		stand = obj.(db.IStandaloneModel)
	case len(disks) == 0:
		break
	case disks[0].ImageId != "":
		obj, err := models.CachedimageManager.GetCachedimageById(ctx, auth.AdminCredential(), disks[0].ImageId, false)
		if err == nil {
			// no check if image if system public image
			public := jsonutils.QueryBoolean(obj.Info, "is_public", false)
			publicScope, _ := obj.Info.GetString("public_scope")
			if !public || publicScope != string(rbacscope.ScopeSystem) {
				stand = obj
				guestSource.keyword = "image"
			}
		}
	case disks[0].SnapshotId != "":
		obj, err := models.SnapshotManager.FetchById(disks[0].SnapshotId)
		if err == nil {
			stand = obj.(db.IStandaloneModel)
		}
	case disks[0].BackupId != "":
		obj, err := models.DiskBackupManager.FetchById(disks[0].BackupId)
		if err != nil {
			return false, errors.Wrapf(err, "unable to fetch diskbackup %s", disks[0].BackupId)
		}
		stand = obj.(db.IStandaloneModel)
	}
	if stand == nil {
		return true, nil
	}
	cm, err := stand.GetAllClassMetadata()
	if err != nil {
		return false, errors.Wrapf(err, "unable to GetAllClassMetadata %s", stand.GetId())
	}
	guestSource.classMedata = cm
	if guestSource.keyword == "" {
		guestSource.keyword = stand.Keyword()
	}
	guestSource.name = stand.GetName()
	p.guestSource = guestSource
	return true, nil
}

func (p *ClassMetadataPredicate) Execute(ctx context.Context, u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := NewPredicateHelper(p, u, c)
	for _, resource := range []*ResourceWithClassMetadata{p.tenant, p.guestSource} {
		if resource == nil {
			continue
		}
		ic, err := db.IsInSameClass(ctx, c.Getter(), resource)
		if err != nil {
			return false, nil, errors.Wrap(err, "unable to determine whether they are in a class")
		}
		if !ic {
			h.Exclude(fmt.Sprintf("The host doesn't have the same class metadata as the choosen %s.", resource.GetDescription()))
			break
		}
	}
	return h.GetResult()
}
