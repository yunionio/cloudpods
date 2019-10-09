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

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// SElasticcache.Parameter
type SElasticcacheParameterManager struct {
	db.SStandaloneResourceBaseManager
}

var ElasticcacheParameterManager *SElasticcacheParameterManager

func init() {
	ElasticcacheParameterManager = &SElasticcacheParameterManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SElasticcacheParameter{},
			"elasticcacheparameters_tbl",
			"elasticcacheparameter",
			"elasticcacheparameters",
		),
	}
	ElasticcacheParameterManager.SetVirtualObject(ElasticcacheParameterManager)
}

type SElasticcacheParameter struct {
	db.SStandaloneResourceBase
	db.SExternalizedResourceBase

	ElasticcacheId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"` // elastic cache instance id

	Key          string `width:"64" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required"`
	Value        string `width:"256" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required"`
	ValueRange   string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional"` // 校验代码，参数的可选范围。
	Modifiable   bool   `nullable:"true" list:"user" create:"optional"`                             // True（可修改）   False（不可修改）
	ForceRestart bool   `nullable:"true" list:"user" create:"optional"`                             // True（重启生效） False（无需重启，提交后即生效）
}

func (manager *SElasticcacheParameterManager) SyncElasticcacheParameters(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, cloudElasticcacheParameters []cloudprovider.ICloudElasticcacheParameter) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, elasticcache.GetOwnerId()))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, elasticcache.GetOwnerId()))

	syncResult := compare.SyncResult{}

	dbParameters, err := elasticcache.GetElasticcacheParameters()
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := make([]SElasticcacheParameter, 0)
	commondb := make([]SElasticcacheParameter, 0)
	commonext := make([]cloudprovider.ICloudElasticcacheParameter, 0)
	added := make([]cloudprovider.ICloudElasticcacheParameter, 0)
	if err := compare.CompareSets(dbParameters, cloudElasticcacheParameters, &removed, &commondb, &commonext, &added); err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemoveCloudElasticcacheParameter(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i++ {
		err := commondb[i].SyncWithCloudElasticcacheParameter(ctx, userCred, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
			continue
		}

		syncResult.Update()
	}

	for i := 0; i < len(added); i++ {
		_, err := manager.newFromCloudElasticcacheParameter(ctx, userCred, elasticcache, added[i])
		if err != nil {
			syncResult.AddError(err)
			continue
		}

		syncResult.Add()
	}
	return syncResult
}

func (self *SElasticcacheParameter) syncRemoveCloudElasticcacheParameter(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return errors.Wrapf(err, "newFromCloudElasticcacheParameter.Remove")
	}
	return self.Delete(ctx, userCred)
}

func (self *SElasticcacheParameter) SyncWithCloudElasticcacheParameter(ctx context.Context, userCred mcclient.TokenCredential, extParameter cloudprovider.ICloudElasticcacheParameter) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		self.Key = extParameter.GetParameterKey()
		self.Value = extParameter.GetParameterValue()
		self.Modifiable = extParameter.GetModifiable()
		self.ForceRestart = extParameter.GetForceRestart()
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "SyncWithCloudElasticcacheParameter.UpdateWithLock")
	}

	return nil
}

func (manager *SElasticcacheParameterManager) newFromCloudElasticcacheParameter(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, extParameter cloudprovider.ICloudElasticcacheParameter) (*SElasticcacheParameter, error) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	parameter := SElasticcacheParameter{}
	parameter.SetModelManager(manager, &parameter)

	parameter.ElasticcacheId = elasticcache.Id
	parameter.Name = extParameter.GetName()
	parameter.ExternalId = extParameter.GetGlobalId()
	parameter.Key = extParameter.GetParameterKey()
	parameter.Value = extParameter.GetParameterValue()
	parameter.ValueRange = extParameter.GetParameterValueRange()
	parameter.Modifiable = extParameter.GetModifiable()
	parameter.ForceRestart = extParameter.GetForceRestart()
	parameter.Description = extParameter.GetDescription()

	err := manager.TableSpec().Insert(&parameter)
	if err != nil {
		return nil, errors.Wrapf(err, "newFromCloudElasticcacheParameter.Insert")
	}

	return &parameter, nil
}
