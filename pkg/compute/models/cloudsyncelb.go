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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func syncRegionLoadbalancerCertificates(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	syncResults SSyncResultSet,
	provider *SCloudprovider,
	localRegion *SCloudregion,
	remoteRegion cloudprovider.ICloudRegion,
	syncRange *SSyncRange,
) {
	certificates, err := func() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
		defer syncResults.AddRequestCost(LoadbalancerCertificateManager)()
		return remoteRegion.GetILoadBalancerCertificates()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetILoadBalancerCertificates for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorln(msg)
		return
	}
	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(LoadbalancerCertificateManager)()
		return localRegion.SyncLoadbalancerCertificates(ctx, userCred, provider, certificates, syncRange.Xor)
	}()

	syncResults.Add(LoadbalancerCertificateManager, result)

	msg := result.Result()
	log.Infof("SyncLoadbalancerCertificates for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return
	}
}

func syncRegionLoadbalancerAcls(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	syncResults SSyncResultSet,
	provider *SCloudprovider,
	localRegion *SCloudregion,
	remoteRegion cloudprovider.ICloudRegion,
	syncRange *SSyncRange,
) {
	acls, err := func() ([]cloudprovider.ICloudLoadbalancerAcl, error) {
		defer syncResults.AddRequestCost(LoadbalancerAclManager)()
		return remoteRegion.GetILoadBalancerAcls()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetILoadBalancerAcls for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorln(msg)
		return
	}
	result := func() compare.SyncResult {
		defer syncResults.AddSqlCost(LoadbalancerAclManager)()
		return localRegion.SyncLoadbalancerAcls(ctx, userCred, provider, acls, syncRange.Xor)
	}()

	syncResults.Add(LoadbalancerAclManager, result)

	msg := result.Result()
	log.Infof("SyncLoadbalancerAcls for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return
	}
}

func syncRegionLoadbalancers(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	syncResults SSyncResultSet,
	provider *SCloudprovider,
	localRegion *SCloudregion,
	remoteRegion cloudprovider.ICloudRegion,
	syncRange *SSyncRange,
) {
	lbs, err := func() ([]cloudprovider.ICloudLoadbalancer, error) {
		defer syncResults.AddRequestCost(LoadbalancerManager)()
		return remoteRegion.GetILoadBalancers()
	}()
	if err != nil {
		msg := fmt.Sprintf("GetILoadBalancers for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorln(msg)
		return
	}
	func() {
		defer syncResults.AddSqlCost(LoadbalancerManager)()

		localLbs, remoteLbs, result := LoadbalancerManager.SyncLoadbalancers(ctx, userCred, provider, localRegion, lbs, syncRange.Xor)

		syncResults.Add(LoadbalancerManager, result)

		msg := result.Result()
		log.Infof("SyncLoadbalancers for region %s result: %s", localRegion.Name, msg)
		if result.IsError() {
			return
		}
		db.OpsLog.LogEvent(provider, db.ACT_SYNC_LB_COMPLETE, msg, userCred)

		for i := 0; i < len(localLbs); i++ {
			func() {
				lockman.LockObject(ctx, &localLbs[i])
				defer lockman.ReleaseObject(ctx, &localLbs[i])

				syncLbPeripherals(ctx, userCred, provider, &localLbs[i], remoteLbs[i])
			}()
		}
	}()
}

func syncLbPeripherals(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, local *SLoadbalancer, remote cloudprovider.ICloudLoadbalancer) {
	err := syncLoadbalancerEip(ctx, userCred, provider, local, remote)
	if err != nil {
		log.Errorf("syncLoadbalancerEip error %s", err)
	}
	err = syncLoadbalancerBackendgroups(ctx, userCred, SSyncResultSet{}, provider, local, remote)
	if err != nil {
		log.Errorf("syncLoadbalancerBackendgroups error: %v", err)
	}
	err = syncLoadbalancerListeners(ctx, userCred, SSyncResultSet{}, provider, local, remote)
	if err != nil {
		log.Errorf("syncLoadbalancerListeners error: %v", err)
	}

	err = syncLoadbalancerSecurityGroups(ctx, userCred, local, remote)
	if err != nil {
		log.Errorf("syncLoadbalancerSecurityGroups error: %v", err)
	}
}

func syncLoadbalancerSecurityGroups(ctx context.Context, userCred mcclient.TokenCredential, localLb *SLoadbalancer, remoteLb cloudprovider.ICloudLoadbalancer) error {
	secIds, err := remoteLb.GetSecurityGroupIds()
	if err != nil {
		return errors.Wrapf(err, "GetSecurityGroupIds")
	}
	result := localLb.SyncSecurityGroups(ctx, userCred, secIds)
	msg := result.Result()
	log.Infof("SyncSecurityGroups for Loadbalancer %s result: %s", localLb.Name, msg)
	if result.IsError() {
		return result.AllError()
	}
	return nil
}

func syncLoadbalancerEip(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localLb *SLoadbalancer, remoteLb cloudprovider.ICloudLoadbalancer) error {
	eip, err := remoteLb.GetIEIP()
	if err != nil {
		return errors.Wrapf(err, "GetIEIP")
	}
	result := localLb.SyncLoadbalancerEip(ctx, userCred, provider, eip)
	msg := result.Result()
	log.Infof("SyncEip for Loadbalancer %s result: %s", localLb.Name, msg)
	if result.IsError() {
		return result.AllError()
	}
	return nil
}

func syncLoadbalancerListeners(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localLoadbalancer *SLoadbalancer, remoteLoadbalancer cloudprovider.ICloudLoadbalancer) error {
	remoteListeners, err := remoteLoadbalancer.GetILoadBalancerListeners()
	if err != nil {
		return errors.Wrapf(err, "GetILoadBalancerListeners")
	}
	localListeners, remoteListeners, result := LoadbalancerListenerManager.SyncLoadbalancerListeners(ctx, userCred, provider, localLoadbalancer, remoteListeners)

	syncResults.Add(LoadbalancerListenerManager, result)

	msg := result.Result()
	log.Infof("SyncLoadbalancerListeners for loadbalancer %s result: %s", localLoadbalancer.Name, msg)
	if result.IsError() {
		return result.AllError()
	}
	for i := 0; i < len(localListeners); i++ {
		func() {
			lockman.LockObject(ctx, &localListeners[i])
			defer lockman.ReleaseObject(ctx, &localListeners[i])

			syncLoadbalancerListenerRules(ctx, userCred, syncResults, provider, &localListeners[i], remoteListeners[i])
		}()
	}
	return nil
}

func syncLoadbalancerListenerRules(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localListener *SLoadbalancerListener, remoteListener cloudprovider.ICloudLoadbalancerListener) {
	remoteRules, err := remoteListener.GetILoadbalancerListenerRules()
	if err != nil {
		msg := fmt.Sprintf("GetILoadbalancerListenerRules for listener %s failed %s", localListener.Name, err)
		log.Errorln(msg)
		return
	}
	result := LoadbalancerListenerRuleManager.SyncLoadbalancerListenerRules(ctx, userCred, provider, localListener, remoteRules)

	syncResults.Add(LoadbalancerListenerRuleManager, result)

	msg := result.Result()
	log.Infof("SyncLoadbalancerListenerRules for listener %s result: %s", localListener.Name, msg)
	if result.IsError() {
		return
	}
}

func syncLoadbalancerBackendgroups(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, local *SLoadbalancer, remote cloudprovider.ICloudLoadbalancer) error {
	exts, err := remote.GetILoadBalancerBackendGroups()
	if err != nil {
		return errors.Wrapf(err, "GetILoadBalancerBackendGroups")
	}
	localLbbgs, remoteLbbgs, result := local.SyncLoadbalancerBackendgroups(ctx, userCred, provider, exts)
	syncResults.Add(LoadbalancerBackendGroupManager, result)

	msg := result.Result()
	log.Infof("SyncLoadbalancerBackendgroups for loadbalancer %s result: %s", local.Name, msg)
	if result.IsError() {
		return result.AllError()
	}
	for i := 0; i < len(localLbbgs); i++ {
		func() {
			lockman.LockObject(ctx, &localLbbgs[i])
			defer lockman.ReleaseObject(ctx, &localLbbgs[i])

			syncLoadbalancerBackends(ctx, userCred, syncResults, provider, &localLbbgs[i], remoteLbbgs[i])
		}()
	}
	return nil
}

func syncLoadbalancerBackends(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, local *SLoadbalancerBackendGroup, remote cloudprovider.ICloudLoadbalancerBackendGroup) {
	exts, err := remote.GetILoadbalancerBackends()
	if err != nil {
		msg := fmt.Sprintf("GetILoadbalancerBackends for lbbg %s failed %s", local.Name, err)
		log.Errorln(msg)
		return
	}
	result := local.SyncLoadbalancerBackends(ctx, userCred, provider, exts)
	syncResults.Add(LoadbalancerBackendManager, result)

	msg := result.Result()
	log.Infof("SyncLoadbalancerBackends for LoadbalancerBackendgroup %s result: %s", local.Name, msg)
	if result.IsError() {
		return
	}
}
