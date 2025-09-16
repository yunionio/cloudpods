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

	"github.com/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/compare"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func syncElasticcaches(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	syncResults SSyncResultSet,
	provider *SCloudprovider,
	localRegion *SCloudregion,
	remoteRegion cloudprovider.ICloudRegion,
	syncRange *SSyncRange,
) {
	extCacheDBs, err := func() ([]cloudprovider.ICloudElasticcache, error) {
		defer syncResults.AddRequestCost(ElasticcacheManager)()
		return remoteRegion.GetIElasticcaches()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetIElasticcaches for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return
	}

	localInstances, remoteInstances, result := func() ([]SElasticcache, []cloudprovider.ICloudElasticcache, compare.SyncResult) {
		defer syncResults.AddSqlCost(ElasticcacheManager)()
		return localRegion.SyncElasticcaches(ctx, userCred, provider.GetOwnerId(), provider, extCacheDBs, syncRange.Xor)
	}()

	syncResults.Add(ElasticcacheManager, result)

	msg := result.Result()
	log.Infof("SyncElasticcaches for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_CLOUD_ELASTIC_CACHE, msg, userCred)
	for i := 0; i < len(localInstances); i++ {
		func() {
			lockman.LockObject(ctx, &localInstances[i])
			defer lockman.ReleaseObject(ctx, &localInstances[i])

			syncElasticcacheParameters(ctx, userCred, syncResults, &localInstances[i], remoteInstances[i])
			syncElasticcacheAccounts(ctx, userCred, syncResults, &localInstances[i], remoteInstances[i])
			syncElasticcacheAcls(ctx, userCred, syncResults, &localInstances[i], remoteInstances[i])
			syncElasticcacheBackups(ctx, userCred, syncResults, &localInstances[i], remoteInstances[i])
			syncElasticcacheSecgroups(ctx, userCred, syncResults, &localInstances[i], remoteInstances[i])
		}()
	}
}

func syncElasticcacheParameters(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, localInstance *SElasticcache, remoteInstance cloudprovider.ICloudElasticcache) {
	parameters, err := func() ([]cloudprovider.ICloudElasticcacheParameter, error) {
		defer syncResults.AddRequestCost(ElasticcacheParameterManager)()
		return remoteInstance.GetICloudElasticcacheParameters()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetIElasticcacheParameters for dbinstance %s failed %s", remoteInstance.GetName(), err)
		log.Errorf(msg)
		return
	}

	func() {
		defer syncResults.AddSqlCost(ElasticcacheParameterManager)()
		result := ElasticcacheParameterManager.SyncElasticcacheParameters(ctx, userCred, localInstance, parameters)
		syncResults.Add(ElasticcacheParameterManager, result)

		msg := result.Result()
		log.Infof("SyncElasticcacheParameters for dbinstance %s result: %s", localInstance.Name, msg)
		if result.IsError() {
			return
		}
	}()
}

func syncElasticcacheAccounts(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, localInstance *SElasticcache, remoteInstance cloudprovider.ICloudElasticcache) {
	accounts, err := func() ([]cloudprovider.ICloudElasticcacheAccount, error) {
		defer syncResults.AddRequestCost(ElasticcacheAccountManager)()
		return remoteInstance.GetICloudElasticcacheAccounts()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetIElasticcacheAccounts for dbinstance %s failed %s", remoteInstance.GetName(), err)
		log.Errorf(msg)
		return
	}

	func() {
		defer syncResults.AddSqlCost(ElasticcacheAccountManager)()
		result := ElasticcacheAccountManager.SyncElasticcacheAccounts(ctx, userCred, localInstance, accounts)
		syncResults.Add(ElasticcacheAccountManager, result)

		msg := result.Result()
		log.Infof("SyncElasticcacheAccounts for dbinstance %s result: %s", localInstance.Name, msg)
		if result.IsError() {
			return
		}
	}()
}

func syncElasticcacheAcls(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, localInstance *SElasticcache, remoteInstance cloudprovider.ICloudElasticcache) {
	acls, err := func() ([]cloudprovider.ICloudElasticcacheAcl, error) {
		defer syncResults.AddRequestCost(ElasticcacheAclManager)()
		return remoteInstance.GetICloudElasticcacheAcls()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetIElasticcacheAcls for dbinstance %s failed %s", remoteInstance.GetName(), err)
		if errors.Cause(err) == cloudprovider.ErrNotSupported {
			log.Warningf(msg)
		} else {
			log.Errorf(msg)
		}
		return
	}

	func() {
		defer syncResults.AddSqlCost(ElasticcacheAclManager)()
		result := ElasticcacheAclManager.SyncElasticcacheAcls(ctx, userCred, localInstance, acls)
		syncResults.Add(ElasticcacheAclManager, result)

		msg := result.Result()
		log.Infof("SyncElasticcacheAcls for dbinstance %s result: %s", localInstance.Name, msg)
		if result.IsError() {
			return
		}
	}()
}

func syncElasticcacheBackups(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, localInstance *SElasticcache, remoteInstance cloudprovider.ICloudElasticcache) {
	backups, err := func() ([]cloudprovider.ICloudElasticcacheBackup, error) {
		defer syncResults.AddRequestCost(ElasticcacheBackupManager)()
		return remoteInstance.GetICloudElasticcacheBackups()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetIElasticcacheBackups for dbinstance %s failed %s", remoteInstance.GetName(), err)
		log.Errorf(msg)
		return
	}

	func() {
		defer syncResults.AddSqlCost(ElasticcacheBackupManager)()
		result := ElasticcacheBackupManager.SyncElasticcacheBackups(ctx, userCred, localInstance, backups)
		syncResults.Add(ElasticcacheBackupManager, result)

		msg := result.Result()
		log.Infof("SyncElasticcacheBackups for dbinstance %s result: %s", localInstance.Name, msg)
		if result.IsError() {
			return
		}
	}()
}

func syncElasticcacheSecgroups(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, localInstance *SElasticcache, remoteInstance cloudprovider.ICloudElasticcache) {
	secgroupIds, err := func() ([]string, error) {
		defer syncResults.AddRequestCost(ElasticcachesecgroupManager)()
		return remoteInstance.GetSecurityGroupIds()
	}()
	if err != nil {
		msg := fmt.Sprintf("Elasticcache.GetSecurityGroupIds for dbinstance %s failed %s", remoteInstance.GetName(), err)
		if errors.Cause(err) == cloudprovider.ErrNotSupported {
			log.Warningf(msg)
		} else {
			log.Errorf(msg)
		}
		return
	}

	func() {
		defer syncResults.AddSqlCost(ElasticcachesecgroupManager)()
		result := localInstance.SyncElasticcacheSecgroups(ctx, userCred, secgroupIds)
		syncResults.Add(ElasticcachesecgroupManager, result)

		msg := result.Result()
		log.Infof("SyncElasticcacheSecgroups for dbinstance %s result: %s", localInstance.Name, msg)
		if result.IsError() {
			return
		}
	}()
}
