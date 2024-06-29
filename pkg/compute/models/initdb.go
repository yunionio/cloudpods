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
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/proxy"
	"yunion.io/x/onecloud/pkg/compute/models/baremetal"
)

func InitDB() error {
	for _, manager := range []db.IModelManager{
		/*
		 * Important!!!
		 * initialization order matters, do not change the order
		 */
		db.TenantCacheManager,
		db.Metadata,

		proxy.ProxySettingManager,

		QuotaManager,

		CloudaccountManager,
		CloudproviderManager,
		CloudregionManager,
		ZoneManager,
		VpcManager,
		WireManager,
		StorageManager,
		SecurityGroupManager,
		NetworkManager,
		NetworkAddressManager,
		NetworkIpMacManager,
		GuestManager,
		HostManager,
		LoadbalancerCertificateManager,
		LoadbalancerAclManager,
		LoadbalancerManager,
		LoadbalancerListenerManager,
		LoadbalancerListenerRuleManager,
		LoadbalancerBackendGroupManager,
		LoadbalancerBackendManager,
		LoadbalancerClusterManager,
		SchedtagManager,
		DynamicschedtagManager,
		ServerSkuManager,
		ElasticcacheSkuManager,

		ExternalProjectManager,
		CachedimageManager,
		StoragecachedimageManager,
		NetworkinterfacenetworkManager,
		DBInstanceNetworkManager,
		DBInstanceAccountManager,
		DBInstanceDatabaseManager,

		SnapshotPolicyDiskManager,
		AccessGroupManager,
		AccessGroupRuleManager,

		GroupnetworkManager,

		ElasticcacheManager,

		baremetal.BaremetalProfileManager,
	} {
		now := time.Now()
		err := manager.InitializeData()
		if err != nil {
			return errors.Wrapf(err, "%s InitializeData", manager.Keyword())
		}
		if cost := time.Now().Sub(now); cost > time.Duration(time.Second)*15 {
			log.Infof("%s InitializeData cost %s", manager.Keyword(), cost.Round(time.Second))
		}
	}
	return nil
}
