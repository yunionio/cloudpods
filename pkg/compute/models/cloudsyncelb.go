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

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func syncRegionLoadbalancerCertificates(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) {
	certificates, err := remoteRegion.GetILoadBalancerCertificates()
	if err != nil {
		msg := fmt.Sprintf("GetILoadBalancerCertificates for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorln(msg)
		return
	}
	result := CachedLoadbalancerCertificateManager.SyncLoadbalancerCertificates(ctx, userCred, provider, localRegion, certificates, syncRange)

	syncResults.Add(CachedLoadbalancerCertificateManager, result)

	msg := result.Result()
	log.Infof("SyncLoadbalancerCachedCertificates for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return
	}
}

func syncRegionLoadbalancerAcls(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) {
	acls, err := remoteRegion.GetILoadBalancerAcls()
	if err != nil {
		msg := fmt.Sprintf("GetILoadBalancerAcls for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorln(msg)
		return
	}
	result := CachedLoadbalancerAclManager.SyncLoadbalancerAcls(ctx, userCred, provider, localRegion, acls, syncRange)

	syncResults.Add(CachedLoadbalancerAclManager, result)

	msg := result.Result()
	log.Infof("SyncLoadbalancerCachedAcls for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return
	}
}

func syncRegionLoadbalancers(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) {
	lbs, err := remoteRegion.GetILoadBalancers()
	if err != nil {
		msg := fmt.Sprintf("GetILoadBalancers for region %s failed %s", remoteRegion.GetName(), err)
		log.Errorln(msg)
		return
	}
	localLbs, remoteLbs, result := LoadbalancerManager.SyncLoadbalancers(ctx, userCred, provider, localRegion, lbs, syncRange)

	syncResults.Add(LoadbalancerManager, result)

	msg := result.Result()
	log.Infof("SyncLoadbalancers for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_LB_COMPLETE, msg, userCred)
	// 同步未关联负载均衡的后端服务器组
	regionDriver := localRegion.GetDriver()
	if regionDriver == nil {
		msg := fmt.Sprintf("GetRegionDriver %s failed", localRegion.GetName())
		log.Errorln(msg)
		return
	}

	regionDriver.RequestPullRegionLoadbalancerBackendGroup(ctx, userCred, syncResults, provider, localRegion, remoteRegion, syncRange)

	for i := 0; i < len(localLbs); i++ {
		func() {
			lockman.LockObject(ctx, &localLbs[i])
			defer lockman.ReleaseObject(ctx, &localLbs[i])

			syncLoadbalancerEip(ctx, userCred, provider, &localLbs[i], remoteLbs[i])
			regionDriver.RequestPullLoadbalancerBackendGroup(ctx, userCred, syncResults, provider, &localLbs[i], remoteLbs[i], syncRange)
			syncLoadbalancerListeners(ctx, userCred, syncResults, provider, &localLbs[i], remoteLbs[i], syncRange)
		}()
	}
}

func syncLoadbalancerEip(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, localLb *SLoadbalancer, remoteLb cloudprovider.ICloudLoadbalancer) {
	eip, err := remoteLb.GetIEIP()
	if err != nil {
		msg := fmt.Sprintf("GetIEIP for Loadbalancer %s failed %s", remoteLb.GetName(), err)
		log.Errorf(msg)
		return
	}
	result := localLb.SyncLoadbalancerEip(ctx, userCred, provider, eip)
	msg := result.Result()
	log.Infof("SyncEip for Loadbalancer %s result: %s", localLb.Name, msg)
	if result.IsError() {
		return
	}
}

func syncLoadbalancerListeners(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localLoadbalancer *SLoadbalancer, remoteLoadbalancer cloudprovider.ICloudLoadbalancer, syncRange *SSyncRange) {
	remoteListeners, err := remoteLoadbalancer.GetILoadBalancerListeners()
	if err != nil {
		msg := fmt.Sprintf("GetILoadBalancerListeners for loadbalancer %s failed %s", localLoadbalancer.Name, err)
		log.Errorln(msg)
		return
	}
	localListeners, remoteListeners, result := LoadbalancerListenerManager.SyncLoadbalancerListeners(ctx, userCred, provider, localLoadbalancer, remoteListeners, syncRange)

	syncResults.Add(LoadbalancerListenerManager, result)

	msg := result.Result()
	log.Infof("SyncLoadbalancerListeners for loadbalancer %s result: %s", localLoadbalancer.Name, msg)
	if result.IsError() {
		return
	}
	for i := 0; i < len(localListeners); i++ {
		func() {
			lockman.LockObject(ctx, &localListeners[i])
			defer lockman.ReleaseObject(ctx, &localListeners[i])

			syncLoadbalancerListenerRules(ctx, userCred, syncResults, provider, &localListeners[i], remoteListeners[i], syncRange)

		}()
	}
}

func syncLoadbalancerListenerRules(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localListener *SLoadbalancerListener, remoteListener cloudprovider.ICloudLoadbalancerListener, syncRange *SSyncRange) {
	remoteRules, err := remoteListener.GetILoadbalancerListenerRules()
	if err != nil {
		msg := fmt.Sprintf("GetILoadbalancerListenerRules for listener %s failed %s", localListener.Name, err)
		log.Errorln(msg)
		return
	}
	result := LoadbalancerListenerRuleManager.SyncLoadbalancerListenerRules(ctx, userCred, provider, localListener, remoteRules, syncRange)

	syncResults.Add(LoadbalancerListenerRuleManager, result)

	msg := result.Result()
	log.Infof("SyncLoadbalancerListenerRules for listener %s result: %s", localListener.Name, msg)
	if result.IsError() {
		return
	}
}

func SyncLoadbalancerBackendgroups(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localLoadbalancer *SLoadbalancer, remoteLoadbalancer cloudprovider.ICloudLoadbalancer, syncRange *SSyncRange) {
	remoteBackendgroups, err := remoteLoadbalancer.GetILoadBalancerBackendGroups()
	if err != nil {
		msg := fmt.Sprintf("GetILoadBalancerBackendGroups for loadbalancer %s failed %s", localLoadbalancer.Name, err)
		log.Errorln(msg)
		return
	}
	localLbbgs, remoteLbbgs, result := LoadbalancerBackendGroupManager.SyncLoadbalancerBackendgroups(ctx, userCred, provider, localLoadbalancer, remoteBackendgroups, syncRange)

	syncResults.Add(LoadbalancerBackendGroupManager, result)

	msg := result.Result()
	log.Infof("SyncLoadbalancerBackendgroups for loadbalancer %s result: %s", localLoadbalancer.Name, msg)
	if result.IsError() {
		return
	}
	for i := 0; i < len(localLbbgs); i++ {
		func() {
			lockman.LockObject(ctx, &localLbbgs[i])
			defer lockman.ReleaseObject(ctx, &localLbbgs[i])

			syncLoadbalancerBackends(ctx, userCred, syncResults, provider, &localLbbgs[i], remoteLbbgs[i], syncRange)
		}()
	}
}

func syncLoadbalancerBackends(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localLbbg *SLoadbalancerBackendGroup, remoteLbbg cloudprovider.ICloudLoadbalancerBackendGroup, syncRange *SSyncRange) {
	remoteLbbs, err := remoteLbbg.GetILoadbalancerBackends()
	if err != nil {
		msg := fmt.Sprintf("GetILoadbalancerBackends for lbbg %s failed %s", localLbbg.Name, err)
		log.Errorln(msg)
		return
	}
	result := LoadbalancerBackendManager.SyncLoadbalancerBackends(ctx, userCred, provider, localLbbg, remoteLbbs, syncRange)

	syncResults.Add(LoadbalancerBackendManager, result)

	msg := result.Result()
	log.Infof("SyncLoadbalancerBackends for LoadbalancerBackendgroup %s result: %s", localLbbg.Name, msg)
	if result.IsError() {
		return
	}
}

/*huawei elb sync*/
func SyncHuaweiLoadbalancerBackendgroups(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localLoadbalancer *SLoadbalancer, remoteLoadbalancer cloudprovider.ICloudLoadbalancer, syncRange *SSyncRange) {
	remoteBackendgroups, err := remoteLoadbalancer.GetILoadBalancerBackendGroups()
	if err != nil {
		msg := fmt.Sprintf("GetILoadBalancerBackendGroups for loadbalancer %s failed %s", localLoadbalancer.Name, err)
		log.Errorln(msg)
		return
	}
	localLbbgs, remoteLbbgs, result := HuaweiCachedLbbgManager.SyncLoadbalancerBackendgroups(ctx, userCred, provider, localLoadbalancer, remoteBackendgroups, syncRange)

	syncResults.Add(HuaweiCachedLbbgManager, result)

	msg := result.Result()
	log.Infof("SyncLoadbalancerBackendgroups for loadbalancer %s result: %s", localLoadbalancer.Name, msg)
	if result.IsError() {
		return
	}
	for i := 0; i < len(localLbbgs); i++ {
		func() {
			lockman.LockObject(ctx, &localLbbgs[i])
			defer lockman.ReleaseObject(ctx, &localLbbgs[i])

			syncHuaweiLoadbalancerBackends(ctx, userCred, syncResults, provider, &localLbbgs[i], remoteLbbgs[i], syncRange)
		}()
	}
}

func syncHuaweiLoadbalancerBackends(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localLbbg *SHuaweiCachedLbbg, remoteLbbg cloudprovider.ICloudLoadbalancerBackendGroup, syncRange *SSyncRange) {
	remoteLbbs, err := remoteLbbg.GetILoadbalancerBackends()
	if err != nil {
		msg := fmt.Sprintf("GetILoadbalancerBackends for lbbg %s failed %s", localLbbg.Name, err)
		log.Errorln(msg)
		return
	}
	result := HuaweiCachedLbManager.SyncLoadbalancerBackends(ctx, userCred, provider, localLbbg, remoteLbbs, syncRange)

	syncResults.Add(LoadbalancerBackendManager, result)

	msg := result.Result()
	log.Infof("SyncLoadbalancerBackends for LoadbalancerBackendgroup %s result: %s", localLbbg.Name, msg)
	if result.IsError() {
		return
	}
}

/*aws elb sync*/
func SyncAwsLoadbalancerBackendgroups(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) {
	remoteBackendgroups, err := remoteRegion.GetILoadBalancerBackendGroups()
	if err != nil {
		msg := fmt.Sprintf("GetILoadBalancerBackendGroups for region %s failed %s", localRegion.Name, err)
		log.Errorln(msg)
		return
	}
	localLbbgs, remoteLbbgs, result := AwsCachedLbbgManager.SyncLoadbalancerBackendgroups(ctx, userCred, provider, localRegion, remoteBackendgroups, syncRange)

	syncResults.Add(LoadbalancerBackendGroupManager, result)

	msg := result.Result()
	log.Infof("SyncLoadbalancerBackendgroups for region %s result: %s", localRegion.Name, msg)
	if result.IsError() {
		return
	}
	for i := 0; i < len(localLbbgs); i++ {
		func() {
			lockman.LockObject(ctx, &localLbbgs[i])
			defer lockman.ReleaseObject(ctx, &localLbbgs[i])

			syncAwsLoadbalancerBackends(ctx, userCred, syncResults, provider, &localLbbgs[i], remoteLbbgs[i], syncRange)
		}()
	}
}

func syncAwsLoadbalancerBackends(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localLbbg *SAwsCachedLbbg, remoteLbbg cloudprovider.ICloudLoadbalancerBackendGroup, syncRange *SSyncRange) {
	remoteLbbs, err := remoteLbbg.GetILoadbalancerBackends()
	if err != nil {
		msg := fmt.Sprintf("GetILoadbalancerBackends for lbbg %s failed %s", localLbbg.Name, err)
		log.Errorln(msg)
		return
	}
	result := AwsCachedLbManager.SyncLoadbalancerBackends(ctx, userCred, provider, localLbbg, remoteLbbs, syncRange)

	syncResults.Add(LoadbalancerBackendManager, result)

	msg := result.Result()
	log.Infof("SyncLoadbalancerBackends for LoadbalancerBackendgroup %s result: %s", localLbbg.Name, msg)
	if result.IsError() {
		return
	}
}

/*qcloud elb sync*/
func SyncQcloudLoadbalancerBackendgroups(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localLoadbalancer *SLoadbalancer, remoteLoadbalancer cloudprovider.ICloudLoadbalancer, syncRange *SSyncRange) {
	remoteBackendgroups, err := remoteLoadbalancer.GetILoadBalancerBackendGroups()
	if err != nil {
		msg := fmt.Sprintf("GetILoadBalancerBackendGroups for loadbalancer %s failed %s", localLoadbalancer.Name, err)
		log.Errorln(msg)
		return
	}
	localLbbgs, remoteLbbgs, result := QcloudCachedLbbgManager.SyncLoadbalancerBackendgroups(ctx, userCred, provider, localLoadbalancer, remoteBackendgroups, syncRange)

	syncResults.Add(QcloudCachedLbbgManager, result)

	msg := result.Result()
	log.Infof("SyncLoadbalancerBackendgroups for loadbalancer %s result: %s", localLoadbalancer.Name, msg)
	if result.IsError() {
		return
	}
	for i := 0; i < len(localLbbgs); i++ {
		func() {
			lockman.LockObject(ctx, &localLbbgs[i])
			defer lockman.ReleaseObject(ctx, &localLbbgs[i])

			syncQcloudLoadbalancerBackends(ctx, userCred, syncResults, provider, &localLbbgs[i], remoteLbbgs[i], syncRange)
		}()
	}
}

func syncQcloudLoadbalancerBackends(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localLbbg *SQcloudCachedLbbg, remoteLbbg cloudprovider.ICloudLoadbalancerBackendGroup, syncRange *SSyncRange) {
	remoteLbbs, err := remoteLbbg.GetILoadbalancerBackends()
	if err != nil {
		msg := fmt.Sprintf("GetILoadbalancerBackends for lbbg %s failed %s", localLbbg.Name, err)
		log.Errorln(msg)
		return
	}
	result := QcloudCachedLbManager.SyncLoadbalancerBackends(ctx, userCred, provider, localLbbg, remoteLbbs, syncRange)

	syncResults.Add(LoadbalancerBackendManager, result)

	msg := result.Result()
	log.Infof("SyncLoadbalancerBackends for LoadbalancerBackendgroup %s result: %s", localLbbg.Name, msg)
	if result.IsError() {
		return
	}
}
