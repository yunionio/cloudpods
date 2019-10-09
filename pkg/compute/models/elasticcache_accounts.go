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

// SElasticcache.Account
type SElasticcacheAccountManager struct {
	db.SStatusStandaloneResourceBaseManager
}

var ElasticcacheAccountManager *SElasticcacheAccountManager

func init() {
	ElasticcacheAccountManager = &SElasticcacheAccountManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SElasticcacheAccount{},
			"elasticcacheaccounts_tbl",
			"elasticcacheaccount",
			"elasticcacheaccounts",
		),
	}
	ElasticcacheAccountManager.SetVirtualObject(ElasticcacheAccountManager)
}

type SElasticcacheAccount struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	ElasticcacheId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"` // elastic cache instance id

	AccountType      string `width:"16" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"` // 账号类型 normal |admin
	AccountPrivilege string `width:"16" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"` // 账号权限 read | write
}

func (manager *SElasticcacheAccountManager) SyncElasticcacheAccounts(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, cloudElasticcacheAccounts []cloudprovider.ICloudElasticcacheAccount) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, elasticcache.GetOwnerId()))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, elasticcache.GetOwnerId()))

	syncResult := compare.SyncResult{}

	dbAccounts, err := elasticcache.GetElasticcacheAccounts()
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := make([]SElasticcacheAccount, 0)
	commondb := make([]SElasticcacheAccount, 0)
	commonext := make([]cloudprovider.ICloudElasticcacheAccount, 0)
	added := make([]cloudprovider.ICloudElasticcacheAccount, 0)
	if err := compare.CompareSets(dbAccounts, cloudElasticcacheAccounts, &removed, &commondb, &commonext, &added); err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemoveCloudElasticcacheAccount(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i++ {
		err := commondb[i].SyncWithCloudElasticcacheAccount(ctx, userCred, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
			continue
		}

		syncResult.Update()
	}

	for i := 0; i < len(added); i++ {
		_, err := manager.newFromCloudElasticcacheAccount(ctx, userCred, elasticcache, added[i])
		if err != nil {
			syncResult.AddError(err)
			continue
		}

		syncResult.Add()
	}
	return syncResult
}

func (self *SElasticcacheAccount) syncRemoveCloudElasticcacheAccount(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return errors.Wrapf(err, "newFromCloudElasticcacheAccount.Remove")
	}
	return self.Delete(ctx, userCred)
}

func (self *SElasticcacheAccount) SyncWithCloudElasticcacheAccount(ctx context.Context, userCred mcclient.TokenCredential, extAccount cloudprovider.ICloudElasticcacheAccount) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = extAccount.GetStatus()
		self.AccountType = extAccount.GetAccountType()
		self.AccountPrivilege = extAccount.GetAccountPrivilege()

		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "SyncWithCloudElasticcacheAccount.UpdateWithLock")
	}

	return nil
}

func (manager *SElasticcacheAccountManager) newFromCloudElasticcacheAccount(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, extAccount cloudprovider.ICloudElasticcacheAccount) (*SElasticcacheAccount, error) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	account := SElasticcacheAccount{}
	account.SetModelManager(manager, &account)

	account.ElasticcacheId = elasticcache.GetId()
	account.Name = extAccount.GetName()
	account.ExternalId = extAccount.GetGlobalId()
	account.Status = extAccount.GetStatus()
	account.AccountType = extAccount.GetAccountType()
	account.AccountPrivilege = extAccount.GetAccountPrivilege()

	err := manager.TableSpec().Insert(&account)
	if err != nil {
		return nil, errors.Wrapf(err, "newFromCloudElasticcacheAccount.Insert")
	}

	return &account, nil
}
