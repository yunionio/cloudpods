package models

import (
	"context"
	"fmt"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func syncElasticcaches(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) {
	extCacheDBs, err := remoteRegion.GetIElasticcaches()
	if err != nil {
		msg := fmt.Sprintf("GetIElasticcaches for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorf(msg)
		return
	}

	localInstances, remoteInstances, result := ElasticcacheManager.SyncElasticcaches(ctx, userCred, provider.GetOwnerId(), provider, localRegion, extCacheDBs)

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
		}()
	}
}

func syncElasticcacheParameters(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, localInstance *SElasticcache, remoteInstance cloudprovider.ICloudElasticcache) {
	parameters, err := remoteInstance.GetICloudElasticcacheParameters()
	if err != nil {
		msg := fmt.Sprintf("GetIElasticcacheParameters for dbinstance %s failed %s", remoteInstance.GetName(), err)
		log.Errorf(msg)
		return
	}

	result := ElasticcacheParameterManager.SyncElasticcacheParameters(ctx, userCred, localInstance, parameters)
	syncResults.Add(ElasticcacheParameterManager, result)

	msg := result.Result()
	log.Infof("SyncElasticcacheParameters for dbinstance %s result: %s", localInstance.Name, msg)
	if result.IsError() {
		return
	}
}

func syncElasticcacheAccounts(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, localInstance *SElasticcache, remoteInstance cloudprovider.ICloudElasticcache) {
	accounts, err := remoteInstance.GetICloudElasticcacheAccounts()
	if err != nil {
		msg := fmt.Sprintf("GetIElasticcacheAccounts for dbinstance %s failed %s", remoteInstance.GetName(), err)
		log.Errorf(msg)
		return
	}

	result := ElasticcacheAccountManager.SyncElasticcacheAccounts(ctx, userCred, localInstance, accounts)
	syncResults.Add(ElasticcacheAccountManager, result)

	msg := result.Result()
	log.Infof("SyncElasticcacheAccounts for dbinstance %s result: %s", localInstance.Name, msg)
	if result.IsError() {
		return
	}
}

func syncElasticcacheAcls(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, localInstance *SElasticcache, remoteInstance cloudprovider.ICloudElasticcache) {
	acls, err := remoteInstance.GetICloudElasticcacheAcls()
	if err != nil {
		msg := fmt.Sprintf("GetIElasticcacheAcls for dbinstance %s failed %s", remoteInstance.GetName(), err)
		log.Errorf(msg)
		return
	}

	result := ElasticcacheAclManager.SyncElasticcacheAcls(ctx, userCred, localInstance, acls)
	syncResults.Add(ElasticcacheAclManager, result)

	msg := result.Result()
	log.Infof("SyncElasticcacheAcls for dbinstance %s result: %s", localInstance.Name, msg)
	if result.IsError() {
		return
	}
}

func syncElasticcacheBackups(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, localInstance *SElasticcache, remoteInstance cloudprovider.ICloudElasticcache) {
	backups, err := remoteInstance.GetICloudElasticcacheBackups()
	if err != nil {
		msg := fmt.Sprintf("GetIElasticcacheBackups for dbinstance %s failed %s", remoteInstance.GetName(), err)
		log.Errorf(msg)
		return
	}

	result := ElasticcacheBackupManager.SyncElasticcacheBackups(ctx, userCred, localInstance, backups)
	syncResults.Add(ElasticcacheBackupManager, result)

	msg := result.Result()
	log.Infof("SyncElasticcacheBackups for dbinstance %s result: %s", localInstance.Name, msg)
	if result.IsError() {
		return
	}
}
