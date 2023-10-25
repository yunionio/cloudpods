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
	"database/sql"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func (self *SCloudregion) purgeAll(ctx context.Context, managerId string) error {
	zones, err := self.GetZones()
	if err != nil {
		return errors.Wrapf(err, "GetZones")
	}
	for i := range zones {
		lockman.LockObject(ctx, &zones[i])
		defer lockman.ReleaseObject(ctx, &zones[i])

		err = zones[i].purgeAll(ctx, managerId)
		if err != nil {
			return errors.Wrapf(err, "zone purgeAll %s", zones[i].Name)
		}
	}
	err = self.purgeAccessGroups(ctx, managerId)
	if err != nil {
		return errors.Wrapf(err, "purgeAccessGroups")
	}
	err = self.purgeApps(ctx, managerId)
	if err != nil {
		return errors.Wrapf(err, "purgeApps")
	}
	err = self.purgeLoadbalancers(ctx, managerId)
	if err != nil {
		return errors.Wrapf(err, "purgeLoadbalancers")
	}
	err = self.purgeKubeClusters(ctx, managerId)
	if err != nil {
		return errors.Wrapf(err, "purgeKubeClusters")
	}
	err = self.purgeRds(ctx, managerId)
	if err != nil {
		return errors.Wrapf(err, "purgeRds")
	}
	err = self.purgeRedis(ctx, managerId)
	if err != nil {
		return errors.Wrapf(err, "purgeRedis")
	}
	err = self.purgeVpcs(ctx, managerId)
	if err != nil {
		return errors.Wrapf(err, "purgeVpcs")
	}
	err = self.purgeResources(ctx, managerId)
	if err != nil {
		return errors.Wrapf(err, "purgeResources")
	}

	cprCount, err := CloudproviderRegionManager.Query().Equals("cloudregion_id", self.Id).CountWithError()
	if err != nil {
		return errors.Wrapf(err, "cpr count")
	}
	// 部分cloudprovider依然有此region, 避免直接删除
	if cprCount > 0 {
		return nil
	}
	err = self.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return nil
	}
	// 资源清理完成后, 清理region下面的套餐
	err = self.purgeSkuResources(ctx)
	if err != nil {
		return errors.Wrapf(err, "purgeSkuResources")
	}
	return self.SEnabledStatusStandaloneResourceBase.Delete(ctx, nil)
}

func (self *SCloudregion) purgeSkuResources(ctx context.Context) error {
	rdsSkus := DBInstanceSkuManager.Query("id").Equals("cloudregion_id", self.Id)
	redisSkus := ElasticcacheSkuManager.Query("id").Equals("cloudregion_id", self.Id)
	artsSkus := ModelartsPoolSkuManager.Query("id").Equals("cloudregion_id", self.Id)
	nasSkus := NasSkuManager.Query("id").Equals("cloudregion_id", self.Id)
	natSkus := NatSkuManager.Query("id").Equals("cloudregion_id", self.Id)
	skus := ServerSkuManager.Query("id").Equals("cloudregion_id", self.Id)

	pairs := []purgePair{
		{manager: ServerSkuManager, key: "id", q: skus},
		{manager: NatSkuManager, key: "id", q: natSkus},
		{manager: NasSkuManager, key: "id", q: nasSkus},
		{manager: ModelartsPoolSkuManager, key: "id", q: artsSkus},
		{manager: ElasticcacheSkuManager, key: "id", q: redisSkus},
		{manager: DBInstanceSkuManager, key: "id", q: rdsSkus},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SCloudregion) purgeVpcs(ctx context.Context, managerId string) error {
	vpcs := VpcManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	wires := WireManager.Query("id").In("vpc_id", vpcs.SubQuery())
	networks := NetworkManager.Query("id").In("wire_id", wires.SubQuery())
	bns := HostnetworkManager.Query("row_id").In("network_id", networks.SubQuery())
	rdsnetworks := DBInstanceNetworkManager.Query("row_id").In("network_id", networks.SubQuery())
	groupnetworks := GroupnetworkManager.Query("row_id").In("network_id", networks.SubQuery())
	lbnetworks := LoadbalancernetworkManager.Query("row_id").In("network_id", networks.SubQuery())
	netmacs := NetworkIpMacManager.Query("id").In("network_id", networks.SubQuery())
	netaddrs := NetworkAddressManager.Query("id").In("network_id", networks.SubQuery())
	nis := NetworkinterfacenetworkManager.Query("row_id").In("network_id", networks.SubQuery())
	rips := ReservedipManager.Query("id").In("network_id", networks.SubQuery())
	sns := ScalingGroupNetworkManager.Query("row_id").In("network_id", networks.SubQuery())
	schedtags := NetworkschedtagManager.Query("row_id").In("network_id", networks.SubQuery())
	nats := NatGatewayManager.Query("id").In("vpc_id", vpcs.SubQuery())
	stables := NatSEntryManager.Query("id").In("natgateway_id", nats.SubQuery())
	secgroups := SecurityGroupManager.Query("id").In("vpc_id", vpcs.SubQuery())
	rules := SecurityGroupRuleManager.Query("id").In("secgroup_id", secgroups.SubQuery())

	dtables := NatDEntryManager.Query("id").In("natgateway_id", nats.SubQuery())
	routes := RouteTableManager.Query("id").In("vpc_id", vpcs.SubQuery())
	intervpcroutes := InterVpcNetworkRouteSetManager.Query("id").In("vpc_id", vpcs.SubQuery())
	peers := VpcPeeringConnectionManager.Query("id").In("vpc_id", vpcs.SubQuery())
	ipv6 := IPv6GatewayManager.Query("id").In("vpc_id", vpcs.SubQuery())

	pairs := []purgePair{
		{manager: SecurityGroupRuleManager, key: "id", q: rules},
		{manager: SecurityGroupManager, key: "id", q: secgroups},
		{manager: IPv6GatewayManager, key: "id", q: ipv6},
		{manager: VpcPeeringConnectionManager, key: "id", q: peers},
		{manager: InterVpcNetworkRouteSetManager, key: "id", q: intervpcroutes},
		{manager: RouteTableManager, key: "id", q: routes},
		{manager: NatDEntryManager, key: "id", q: dtables},
		{manager: NatSEntryManager, key: "id", q: stables},
		{manager: NatGatewayManager, key: "id", q: nats},
		{manager: NetworkschedtagManager, key: "row_id", q: schedtags},
		{manager: ScalingGroupNetworkManager, key: "row_id", q: sns},
		{manager: ReservedipManager, key: "id", q: rips},
		{manager: NetworkinterfacenetworkManager, key: "row_id", q: nis},
		{manager: NetworkAddressManager, key: "id", q: netaddrs},
		{manager: NetworkIpMacManager, key: "id", q: netmacs},
		{manager: LoadbalancernetworkManager, key: "row_id", q: lbnetworks},
		{manager: GroupnetworkManager, key: "row_id", q: groupnetworks},
		{manager: DBInstanceNetworkManager, key: "row_id", q: rdsnetworks},
		{manager: HostnetworkManager, key: "row_id", q: bns},
		{manager: NetworkManager, key: "id", q: networks},
		{manager: WireManager, key: "id", q: wires},
		{manager: VpcManager, key: "id", q: vpcs},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SVpc) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	wires := WireManager.Query("id").Equals("vpc_id", self.Id)
	networks := NetworkManager.Query("id").In("wire_id", wires.SubQuery())
	bns := HostnetworkManager.Query("row_id").In("network_id", networks.SubQuery())
	rdsnetworks := DBInstanceNetworkManager.Query("row_id").In("network_id", networks.SubQuery())
	groupnetworks := GroupnetworkManager.Query("row_id").In("network_id", networks.SubQuery())
	lbnetworks := LoadbalancernetworkManager.Query("row_id").In("network_id", networks.SubQuery())
	netmacs := NetworkIpMacManager.Query("id").In("network_id", networks.SubQuery())
	netaddrs := NetworkAddressManager.Query("id").In("network_id", networks.SubQuery())
	nis := NetworkinterfacenetworkManager.Query("row_id").In("network_id", networks.SubQuery())
	rips := ReservedipManager.Query("id").In("network_id", networks.SubQuery())
	sns := ScalingGroupNetworkManager.Query("row_id").In("network_id", networks.SubQuery())
	schedtags := NetworkschedtagManager.Query("row_id").In("network_id", networks.SubQuery())
	nats := NatGatewayManager.Query("id").Equals("vpc_id", self.Id)
	stables := NatSEntryManager.Query("id").In("natgateway_id", nats.SubQuery())
	dtables := NatDEntryManager.Query("id").In("natgateway_id", nats.SubQuery())
	routes := RouteTableManager.Query("id").Equals("vpc_id", self.Id)
	dnszones := DnsZoneVpcManager.Query("row_id").Equals("vpc_id", self.Id)
	intervpcroutes := InterVpcNetworkRouteSetManager.Query("id").Equals("vpc_id", self.Id)
	ipv6 := IPv6GatewayManager.Query("id").Equals("vpc_id", self.Id)
	secgroups := SecurityGroupManager.Query("id").Equals("vpc_id", self.Id)
	rules := SecurityGroupRuleManager.Query("id").In("secgroup_id", secgroups.SubQuery())

	pairs := []purgePair{
		{manager: SecurityGroupRuleManager, key: "id", q: rules},
		{manager: SecurityGroupManager, key: "id", q: secgroups},
		{manager: IPv6GatewayManager, key: "id", q: ipv6},
		{manager: InterVpcNetworkRouteSetManager, key: "id", q: intervpcroutes},
		{manager: DnsZoneVpcManager, key: "row_id", q: dnszones},
		{manager: RouteTableManager, key: "id", q: routes},
		{manager: NatDEntryManager, key: "id", q: dtables},
		{manager: NatSEntryManager, key: "id", q: stables},
		{manager: NatGatewayManager, key: "id", q: nats},
		{manager: NetworkschedtagManager, key: "row_id", q: schedtags},
		{manager: ScalingGroupNetworkManager, key: "row_id", q: sns},
		{manager: ReservedipManager, key: "id", q: rips},
		{manager: NetworkinterfacenetworkManager, key: "row_id", q: nis},
		{manager: NetworkAddressManager, key: "id", q: netaddrs},
		{manager: NetworkIpMacManager, key: "id", q: netmacs},
		{manager: LoadbalancernetworkManager, key: "row_id", q: lbnetworks},
		{manager: GroupnetworkManager, key: "row_id", q: groupnetworks},
		{manager: DBInstanceNetworkManager, key: "row_id", q: rdsnetworks},
		{manager: HostnetworkManager, key: "row_id", q: bns},
		{manager: NetworkManager, key: "id", q: networks},
		{manager: WireManager, key: "id", q: wires},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}
	return self.SEnabledStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SNetworkInterface) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	nis := NetworkinterfacenetworkManager.Query("row_id").In("networkinterface_id", self.Id)
	pairs := []purgePair{
		{manager: NetworkinterfacenetworkManager, key: "row_id", q: nis},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}
	return self.SModelBase.Delete(ctx, userCred)
}

func (self *SNatGateway) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	stables := NatSEntryManager.Query("id").Equals("natgateway_id", self.Id)
	dtables := NatDEntryManager.Query("id").Equals("natgateway_id", self.Id)

	pairs := []purgePair{
		{manager: NatDEntryManager, key: "id", q: dtables},
		{manager: NatSEntryManager, key: "id", q: stables},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}
	return self.SInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SCloudregion) purgeResources(ctx context.Context, managerId string) error {
	buckets := BucketManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	ess := ElasticSearchManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	eips := ElasticipManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	kafkas := KafkaManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	misc := MiscResourceManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	arts := ModelartsPoolManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	mongodbs := MongoDBManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	nics := NetworkInterfaceManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	nicips := NetworkinterfacenetworkManager.Query("row_id").In("networkinterface_id", nics.SubQuery())
	secgroups := SecurityGroupManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	rules := SecurityGroupRuleManager.Query("id").In("secgroup_id", secgroups.SubQuery())
	policycaches := SnapshotPolicyCacheManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	snapshots := SnapshotManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	tables := TablestoreManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	wafs := WafInstanceManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	ipsetcaches := WafIPSetCacheManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	regsetcaches := WafRegexSetCacheManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	wafgroups := WafRuleGroupCacheManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	cprs := CloudproviderRegionManager.Query("row_id").Equals("cloudprovider_id", managerId).Equals("cloudregion_id", self.Id)

	pairs := []purgePair{
		{manager: CloudproviderRegionManager, key: "row_id", q: cprs},
		{manager: WafRuleGroupCacheManager, key: "id", q: wafgroups},
		{manager: WafRegexSetCacheManager, key: "id", q: regsetcaches},
		{manager: WafIPSetCacheManager, key: "id", q: ipsetcaches},
		{manager: WafInstanceManager, key: "id", q: wafs},
		{manager: TablestoreManager, key: "id", q: tables},
		{manager: SnapshotManager, key: "id", q: snapshots},
		{manager: SnapshotPolicyCacheManager, key: "id", q: policycaches},
		{manager: SecurityGroupRuleManager, key: "id", q: rules},
		{manager: SecurityGroupManager, key: "id", q: secgroups},
		{manager: NetworkinterfacenetworkManager, key: "row_id", q: nicips},
		{manager: NetworkInterfaceManager, key: "id", q: nics},
		{manager: MongoDBManager, key: "id", q: mongodbs},
		{manager: ModelartsPoolManager, key: "id", q: arts},
		{manager: MiscResourceManager, key: "id", q: misc},
		{manager: KafkaManager, key: "id", q: kafkas},
		{manager: ElasticipManager, key: "id", q: eips},
		{manager: ElasticSearchManager, key: "id", q: ess},
		{manager: BucketManager, key: "id", q: buckets},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SCloudregion) purgeRedis(ctx context.Context, managerId string) error {
	vpcs := VpcManager.Query("id").Equals("cloudregion_id", self.Id).Equals("manager_id", managerId)
	redis := ElasticcacheManager.Query("id").In("vpc_id", vpcs.SubQuery())
	redisAcls := ElasticcacheAclManager.Query("id").In("elasticcache_id", redis.SubQuery())
	backups := ElasticcacheBackupManager.Query("id").In("elasticcache_id", redis.SubQuery())
	parameters := ElasticcacheParameterManager.Query("id").In("elasticcache_id", redis.SubQuery())
	secgroups := ElasticcachesecgroupManager.Query("row_id").In("elasticcache_id", redis.SubQuery())
	accounts := ElasticcacheAccountManager.Query("id").In("elasticcache_id", redis.SubQuery())

	pairs := []purgePair{
		{manager: ElasticcacheAccountManager, key: "id", q: accounts},
		{manager: ElasticcachesecgroupManager, key: "row_id", q: secgroups},
		{manager: ElasticcacheParameterManager, key: "id", q: parameters},
		{manager: ElasticcacheBackupManager, key: "id", q: backups},
		{manager: ElasticcacheAclManager, key: "id", q: redisAcls},
		{manager: ElasticcacheManager, key: "id", q: redis},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SCloudregion) purgeRds(ctx context.Context, managerId string) error {
	rds := DBInstanceManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	rdsNetworks := DBInstanceNetworkManager.Query("row_id").In("dbinstance_id", rds.SubQuery())
	rdsBackups := DBInstanceBackupManager.Query("id").In("dbinstance_id", rds.SubQuery())
	rdsAccounts := DBInstanceAccountManager.Query("id").In("dbinstance_id", rds.SubQuery())
	rdsSecgroups := DBInstanceSecgroupManager.Query("row_id").In("dbinstance_id", rds.SubQuery())
	rdsDbs := DBInstanceDatabaseManager.Query("id").In("dbinstance_id", rds.SubQuery())
	rdsParamenters := DBInstanceParameterManager.Query("id").In("dbinstance_id", rds.SubQuery())
	rdsPrivileges := DBInstancePrivilegeManager.Query("id").In("dbinstanceaccount_id", rdsAccounts.SubQuery())

	pairs := []purgePair{
		{manager: DBInstancePrivilegeManager, key: "id", q: rdsPrivileges},
		{manager: DBInstanceParameterManager, key: "id", q: rdsParamenters},
		{manager: DBInstanceDatabaseManager, key: "id", q: rdsDbs},
		{manager: DBInstanceSecgroupManager, key: "row_id", q: rdsSecgroups},
		{manager: DBInstanceAccountManager, key: "id", q: rdsAccounts},
		{manager: DBInstanceBackupManager, key: "id", q: rdsBackups},
		{manager: DBInstanceNetworkManager, key: "row_id", q: rdsNetworks},
		{manager: DBInstanceManager, key: "id", q: rds},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SDBInstance) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	rdsNetworks := DBInstanceNetworkManager.Query("row_id").Equals("dbinstance_id", self.Id)
	rdsBackups := DBInstanceBackupManager.Query("id").Equals("dbinstance_id", self.Id)
	rdsAccounts := DBInstanceAccountManager.Query("id").Equals("dbinstance_id", self.Id)
	rdsSecgroups := DBInstanceSecgroupManager.Query("row_id").Equals("dbinstance_id", self.Id)
	rdsDbs := DBInstanceDatabaseManager.Query("id").Equals("dbinstance_id", self.Id)
	rdsParamenters := DBInstanceParameterManager.Query("id").Equals("dbinstance_id", self.Id)
	rdsPrivileges := DBInstancePrivilegeManager.Query("id").In("dbinstanceaccount_id", rdsAccounts.SubQuery())

	pairs := []purgePair{
		{manager: DBInstancePrivilegeManager, key: "id", q: rdsPrivileges},
		{manager: DBInstanceParameterManager, key: "id", q: rdsParamenters},
		{manager: DBInstanceDatabaseManager, key: "id", q: rdsDbs},
		{manager: DBInstanceSecgroupManager, key: "row_id", q: rdsSecgroups},
		{manager: DBInstanceAccountManager, key: "id", q: rdsAccounts},
		{manager: DBInstanceBackupManager, key: "id", q: rdsBackups},
		{manager: DBInstanceNetworkManager, key: "row_id", q: rdsNetworks},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}
	return self.SVirtualResourceBase.Delete(ctx, userCred)
}

func (self *SCloudregion) purgeKubeClusters(ctx context.Context, managerId string) error {
	kubeClusters := KubeClusterManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	kubeNodes := KubeNodeManager.Query("id").In("cloud_kube_cluster_id", kubeClusters.SubQuery())
	kubePools := KubeNodePoolManager.Query("id").In("cloud_kube_cluster_id", kubeClusters.SubQuery())

	pairs := []purgePair{
		{manager: KubeNodePoolManager, key: "id", q: kubePools},
		{manager: KubeNodeManager, key: "id", q: kubeNodes},
		{manager: KubeClusterManager, key: "id", q: kubeClusters},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SCloudregion) purgeLoadbalancers(ctx context.Context, managerId string) error {
	cacheAcls := CachedLoadbalancerAclManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	cacheCerts := CachedLoadbalancerCertificateManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	lbs := LoadbalancerManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	lbnetworks := LoadbalancernetworkManager.Query("row_id").In("loadbalancer_id", lbs.SubQuery())
	lblis := LoadbalancerListenerManager.Query("id").In("loadbalancer_id", lbs.SubQuery())
	lbbgs := LoadbalancerBackendGroupManager.Query("id").In("loadbalancer_id", lbs.SubQuery())
	backends := LoadbalancerBackendManager.Query("id").In("backend_group_id", lbbgs.SubQuery())
	lbrules := LoadbalancerListenerRuleManager.Query("id").In("listener_id", lblis.SubQuery())

	pairs := []purgePair{
		{manager: LoadbalancerBackendManager, key: "id", q: backends},
		{manager: LoadbalancerListenerRuleManager, key: "id", q: lbrules},
		{manager: LoadbalancerBackendGroupManager, key: "id", q: lbbgs},
		{manager: LoadbalancerListenerManager, key: "id", q: lblis},
		{manager: LoadbalancernetworkManager, key: "row_id", q: lbnetworks},
		{manager: CachedLoadbalancerCertificateManager, key: "id", q: cacheCerts},
		{manager: CachedLoadbalancerAclManager, key: "id", q: cacheAcls},
		{manager: LoadbalancerManager, key: "id", q: lbs},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SCloudregion) purgeApps(ctx context.Context, managerId string) error {
	apps := AppManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	envs := AppEnvironmentManager.Query("id").In("app_id", apps.SubQuery())

	pairs := []purgePair{
		{manager: AppEnvironmentManager, key: "id", q: envs},
		{manager: AppManager, key: "id", q: apps},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SCloudregion) purgeAccessGroups(ctx context.Context, managerId string) error {
	fs := FileSystemManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	ags := AccessGroupCacheManager.Query("id").Equals("manager_id", managerId).Equals("cloudregion_id", self.Id)
	mts := MountTargetManager.Query("id").In("file_system_id", fs.SubQuery())

	pairs := []purgePair{
		{manager: MountTargetManager, key: "id", q: mts},
		{manager: AccessGroupCacheManager, key: "id", q: ags},
		{manager: FileSystemManager, key: "id", q: fs},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SZone) purgeAll(ctx context.Context, managerId string) error {
	err := self.purgeStorages(ctx, managerId)
	if err != nil {
		return errors.Wrapf(err, "purgeStorages")
	}
	err = self.purgeHosts(ctx, managerId)
	if err != nil {
		return errors.Wrapf(err, "purgeHosts")
	}
	err = self.purgeWires(ctx, managerId)
	if err != nil {
		return errors.Wrapf(err, "purgeWires")
	}
	err = self.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return nil
	}

	return self.SStandaloneResourceBase.Delete(ctx, nil)
}

type purgePair struct {
	manager db.IModelManager
	key     string
	q       *sqlchemy.SQuery
}

func (self *purgePair) queryIds() ([]string, error) {
	ids := []string{}
	sq := self.q.SubQuery()
	q := sq.Query(sq.Field(self.key)).Distinct()
	rows, err := q.Rows()
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return ids, nil
		}
		return ids, errors.Wrap(err, "Query")
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		err := rows.Scan(&id)
		if err != nil {
			return ids, errors.Wrap(err, "rows.Scan")
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (self *purgePair) purgeAll(ctx context.Context) error {
	purgeIds, err := self.queryIds()
	if err != nil {
		return errors.Wrapf(err, "Query ids")
	}
	if len(purgeIds) == 0 {
		return nil
	}
	var purge = func(ids []string) error {
		vars := []interface{}{}
		placeholders := make([]string, len(ids))
		for i := range placeholders {
			placeholders[i] = "?"
			vars = append(vars, ids[i])
		}
		placeholder := strings.Join(placeholders, ",")
		sql := fmt.Sprintf(
			"update %s set deleted = true, deleted_at = ? where %s in (%s) and deleted = false",
			self.manager.TableSpec().Name(), self.key, placeholder,
		)
		switch self.manager.Keyword() {
		case GuestcdromManager.Keyword(), GuestFloppyManager.Keyword():
			sql = fmt.Sprintf(
				"update %s set image_id = null, updated_at = ? where %s in (%s) and image_id is not null",
				self.manager.TableSpec().Name(), self.key, placeholder,
			)
			vars = append([]interface{}{time.Now()}, vars...)
		case NetInterfaceManager.Keyword():
			sql = fmt.Sprintf(
				"delete from %s where %s in (%s)",
				self.manager.TableSpec().Name(), self.key, placeholder,
			)
		// sku需要直接删除，避免数据库积累数据导致查询缓慢
		case DBInstanceSkuManager.Keyword(), ElasticcacheSkuManager.Keyword(), ServerSkuManager.Keyword():
			sql = fmt.Sprintf(
				"delete from %s where %s in (%s)",
				self.manager.TableSpec().Name(), self.key, placeholder,
			)
		case SecurityGroupRuleManager.Keyword():
			sql = fmt.Sprintf(
				"delete from %s where %s in (%s)",
				self.manager.TableSpec().Name(), self.key, placeholder,
			)
		case NetworkAdditionalWireManager.Keyword():
			sql = fmt.Sprintf("delete from `%s` where `wire_id` in (%s)",
				self.manager.TableSpec().Name(), placeholder,
			)
		default:
			vars = append([]interface{}{time.Now()}, vars...)
		}
		lockman.LockRawObject(ctx, self.manager.Keyword(), "purge")
		defer lockman.ReleaseRawObject(ctx, self.manager.Keyword(), "purge")
		_, err = sqlchemy.GetDB().Exec(
			sql, vars...,
		)
		if err != nil {
			return errors.Wrapf(err, strings.ReplaceAll(sql, "?", "%s"), vars...)
		}
		return nil
	}
	var splitByLen = func(data []string, splitLen int) [][]string {
		var result [][]string
		for i := 0; i < len(data); i += splitLen {
			end := i + splitLen
			if end > len(data) {
				end = len(data)
			}
			result = append(result, data[i:end])
		}
		return result
	}
	idsArr := splitByLen(purgeIds, 100)
	for i := range idsArr {
		err = purge(idsArr[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SZone) purgeStorages(ctx context.Context, managerId string) error {
	storages := StorageManager.Query("id").Equals("manager_id", managerId).Equals("zone_id", self.Id)
	schedtags := StorageschedtagManager.Query("row_id").In("storage_id", storages.SubQuery())
	snapshots := SnapshotManager.Query("id").In("storage_id", storages.SubQuery()).IsTrue("fake_deleted")
	hoststorages := HoststorageManager.Query("row_id").In("storage_id", storages.SubQuery())
	disks := DiskManager.Query("id").In("storage_id", storages.SubQuery())
	diskbackups := DiskBackupManager.Query("id").In("disk_id", disks.SubQuery())
	guestdisks := GuestdiskManager.Query("row_id").In("disk_id", disks.SubQuery())
	diskpolicies := SnapshotPolicyDiskManager.Query("row_id").In("disk_id", disks.SubQuery())

	pairs := []purgePair{
		{manager: SnapshotPolicyDiskManager, key: "row_id", q: diskpolicies},
		{manager: GuestdiskManager, key: "row_id", q: guestdisks},
		{manager: SnapshotManager, key: "id", q: snapshots},
		{manager: DiskBackupManager, key: "id", q: diskbackups},
		{manager: DiskManager, key: "id", q: disks},
		{manager: StorageschedtagManager, key: "row_id", q: schedtags},
		{manager: HoststorageManager, key: "row_id", q: hoststorages},
		{manager: StorageManager, key: "id", q: storages},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SStorage) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	schedtags := StorageschedtagManager.Query("row_id").Equals("storage_id", self.Id)
	snapshots := SnapshotManager.Query("id").Equals("storage_id", self.Id).IsTrue("fake_deleted")
	hoststorages := HoststorageManager.Query("row_id").Equals("storage_id", self.Id)
	disks := DiskManager.Query("id").Equals("storage_id", self.Id)
	diskbackups := DiskBackupManager.Query("id").In("disk_id", disks.SubQuery())
	guestdisks := GuestdiskManager.Query("row_id").In("disk_id", disks.SubQuery())
	diskpolicies := SnapshotPolicyDiskManager.Query("row_id").In("disk_id", disks.SubQuery())

	pairs := []purgePair{
		{manager: GuestdiskManager, key: "row_id", q: guestdisks},
		{manager: SnapshotPolicyDiskManager, key: "row_id", q: diskpolicies},
		{manager: SnapshotManager, key: "id", q: snapshots},
		{manager: DiskBackupManager, key: "id", q: diskbackups},
		{manager: DiskManager, key: "id", q: disks},
		{manager: StorageschedtagManager, key: "row_id", q: schedtags},
		{manager: HoststorageManager, key: "row_id", q: hoststorages},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}
	return self.SEnabledStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SZone) purgeHosts(ctx context.Context, managerId string) error {
	hosts := HostManager.Query("id").Equals("manager_id", managerId).Equals("zone_id", self.Id)
	isolateds := IsolatedDeviceManager.Query("id").In("host_id", hosts.SubQuery())
	hoststorages := HoststorageManager.Query("row_id").In("host_id", hosts.SubQuery())
	hostwires := HostwireManagerDeprecated.Query("row_id").In("host_id", hosts.SubQuery())
	guests := GuestManager.Query("id").In("host_id", hosts.SubQuery())
	guestdisks := GuestdiskManager.Query("row_id").In("guest_id", guests.SubQuery())
	guestnetworks := GuestnetworkManager.Query("row_id").In("guest_id", guests.SubQuery())
	guestcdroms := GuestcdromManager.Query("row_id").In("id", guests.SubQuery())
	guestvfd := GuestFloppyManager.Query("row_id").In("id", guests.SubQuery())
	guestgroups := GroupguestManager.Query("row_id").In("guest_id", guests.SubQuery())
	guestsecgroups := GuestsecgroupManager.Query("row_id").In("guest_id", guests.SubQuery())
	instancesnapshots := InstanceSnapshotManager.Query("id").In("guest_id", guests.SubQuery())
	instancebackups := InstanceBackupManager.Query("id").In("guest_id", guests.SubQuery())
	publicIps := ElasticipManager.Query("id").Equals("mode", api.EIP_MODE_INSTANCE_PUBLICIP).
		Equals("associate_type", api.EIP_ASSOCIATE_TYPE_SERVER).In("associate_id", guests.SubQuery())
	tapService := NetTapServiceManager.Query("id").Equals("type", api.TapServiceHost).In("target_id", hosts.SubQuery())
	tapFlows := NetTapFlowManager.Query("id").In("tap_id", tapService.SubQuery())
	tapNics := NetTapFlowManager.Query("id").Equals("type", api.TapFlowVSwitch).In("source_id", hosts.SubQuery())
	backends := LoadbalancerBackendManager.Query("id").In("backend_id", hosts.SubQuery())
	hostnetworks := HostnetworkManager.Query("row_id").In("baremetal_id", hosts.SubQuery())
	netinterfaces := NetInterfaceManager.Query("baremetal_id").In("baremetal_id", hosts.SubQuery())
	storageIds := HoststorageManager.Query("storage_id").In("host_id", hosts.SubQuery())
	storages := StorageManager.Query("id").Equals("storage_type", api.STORAGE_BAREMETAL).In("id", storageIds.SubQuery())
	guestTapService := NetTapServiceManager.Query("id").Equals("type", api.TapServiceGuest).In("target_id", guests.SubQuery())
	guestTapFlows := NetTapFlowManager.Query("id").In("tap_id", tapService.SubQuery())
	guestTapNics := NetTapFlowManager.Query("id").Equals("type", api.TapFlowGuestNic).In("source_id", guests.SubQuery())
	guestBackends := LoadbalancerBackendManager.Query("id").In("backend_id", guests.SubQuery())

	pairs := []purgePair{
		{manager: LoadbalancerBackendManager, key: "id", q: guestBackends},
		{manager: NetTapFlowManager, key: "id", q: guestTapNics},
		{manager: NetTapFlowManager, key: "id", q: guestTapFlows},
		{manager: NetTapServiceManager, key: "id", q: guestTapService},
		{manager: StorageManager, key: "id", q: storages},
		{manager: NetInterfaceManager, key: "baremetal_id", q: netinterfaces},
		{manager: HostnetworkManager, key: "row_id", q: hostnetworks},
		{manager: LoadbalancerBackendManager, key: "id", q: backends},
		{manager: NetTapFlowManager, key: "id", q: tapNics},
		{manager: NetTapFlowManager, key: "id", q: tapFlows},
		{manager: NetTapServiceManager, key: "id", q: tapService},
		{manager: ElasticipManager, key: "id", q: publicIps},
		{manager: GuestsecgroupManager, key: "row_id", q: guestsecgroups},
		{manager: GroupguestManager, key: "row_id", q: guestgroups},
		{manager: GuestcdromManager, key: "row_id", q: guestcdroms},
		{manager: GuestFloppyManager, key: "row_id", q: guestvfd},
		{manager: GuestnetworkManager, key: "row_id", q: guestnetworks},
		{manager: GuestdiskManager, key: "row_id", q: guestdisks},
		{manager: InstanceSnapshotManager, key: "id", q: instancesnapshots},
		{manager: InstanceBackupManager, key: "id", q: instancebackups},
		{manager: GuestManager, key: "id", q: guests},
		{manager: HoststorageManager, key: "row_id", q: hoststorages},
		{manager: HostwireManagerDeprecated, key: "row_id", q: hostwires},
		{manager: IsolatedDeviceManager, key: "id", q: isolateds},
		{manager: HostManager, key: "id", q: hosts},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SHost) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	isolateds := IsolatedDeviceManager.Query("id").Equals("host_id", self.Id)
	hoststorages := HoststorageManager.Query("row_id").Equals("host_id", self.Id)
	hostwires := HostwireManagerDeprecated.Query("row_id").Equals("host_id", self.Id)
	guests := GuestManager.Query("id").Equals("host_id", self.Id)
	guestdisks := GuestdiskManager.Query("row_id").In("guest_id", guests.SubQuery())
	guestnetworks := GuestnetworkManager.Query("row_id").In("guest_id", guests.SubQuery())
	guestcdroms := GuestcdromManager.Query("row_id").In("id", guests.SubQuery())
	guestvfd := GuestFloppyManager.Query("row_id").In("id", guests.SubQuery())
	guestgroups := GroupguestManager.Query("row_id").In("guest_id", guests.SubQuery())
	guestsecgroups := GuestsecgroupManager.Query("row_id").In("guest_id", guests.SubQuery())
	instancesnapshots := InstanceSnapshotManager.Query("id").In("guest_id", guests.SubQuery())
	instancebackups := InstanceBackupManager.Query("id").In("guest_id", guests.SubQuery())
	publicIps := ElasticipManager.Query("id").Equals("mode", api.EIP_MODE_INSTANCE_PUBLICIP).
		Equals("associate_type", api.EIP_ASSOCIATE_TYPE_SERVER).In("associate_id", guests.SubQuery())
	tapService := NetTapServiceManager.Query("id").Equals("type", api.TapServiceHost).Equals("target_id", self.Id)
	tapFlows := NetTapFlowManager.Query("id").In("tap_id", tapService.SubQuery())
	tapNics := NetTapFlowManager.Query("id").Equals("type", api.TapFlowVSwitch).Equals("source_id", self.Id)
	backends := LoadbalancerBackendManager.Query("id").Equals("backend_id", self.Id)
	hostnetworks := HostnetworkManager.Query("row_id").Equals("baremetal_id", self.Id)
	netinterfaces := NetInterfaceManager.Query("baremetal_id").Equals("baremetal_id", self.Id)
	storageIds := HoststorageManager.Query("storage_id").Equals("host_id", self.Id)
	storages := StorageManager.Query("id").Equals("storage_type", api.STORAGE_BAREMETAL).In("id", storageIds.SubQuery())

	pairs := []purgePair{
		{manager: StorageManager, key: "id", q: storages},
		{manager: NetInterfaceManager, key: "baremetal_id", q: netinterfaces},
		{manager: HostnetworkManager, key: "row_id", q: hostnetworks},
		{manager: LoadbalancerBackendManager, key: "id", q: backends},
		{manager: NetTapFlowManager, key: "id", q: tapNics},
		{manager: NetTapFlowManager, key: "id", q: tapFlows},
		{manager: NetTapServiceManager, key: "id", q: tapService},
		{manager: ElasticipManager, key: "id", q: publicIps},
		{manager: GuestsecgroupManager, key: "row_id", q: guestsecgroups},
		{manager: GroupguestManager, key: "row_id", q: guestgroups},
		{manager: GuestcdromManager, key: "row_id", q: guestcdroms},
		{manager: GuestFloppyManager, key: "row_id", q: guestvfd},
		{manager: GuestnetworkManager, key: "row_id", q: guestnetworks},
		{manager: GuestdiskManager, key: "row_id", q: guestdisks},
		{manager: InstanceSnapshotManager, key: "id", q: instancesnapshots},
		{manager: InstanceBackupManager, key: "id", q: instancebackups},
		{manager: GuestManager, key: "id", q: guests},
		{manager: HoststorageManager, key: "row_id", q: hoststorages},
		{manager: HostwireManagerDeprecated, key: "row_id", q: hostwires},
		{manager: IsolatedDeviceManager, key: "id", q: isolateds},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}
	defer func() {
		HostManager.ClearSchedDescCache(self.Id)
	}()
	return self.SEnabledStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SGuest) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	guestdisks := GuestdiskManager.Query("row_id").Equals("guest_id", self.Id)
	guestnetworks := GuestnetworkManager.Query("row_id").Equals("guest_id", self.Id)
	guestcdroms := GuestcdromManager.Query("row_id").Equals("id", self.Id)
	guestvfd := GuestFloppyManager.Query("row_id").Equals("id", self.Id)
	guestgroups := GroupguestManager.Query("row_id").Equals("guest_id", self.Id)
	guestsecgroups := GuestsecgroupManager.Query("row_id").Equals("guest_id", self.Id)
	instancesnapshots := InstanceSnapshotManager.Query("id").Equals("guest_id", self.Id)
	instancebackups := InstanceBackupManager.Query("id").Equals("guest_id", self.Id)
	publicIps := ElasticipManager.Query("id").Equals("mode", api.EIP_MODE_INSTANCE_PUBLICIP).
		Equals("associate_type", api.EIP_ASSOCIATE_TYPE_SERVER).Equals("associate_id", self.Id)
	tapService := NetTapServiceManager.Query("id").Equals("type", api.TapServiceGuest).Equals("target_id", self.Id)
	tapFlows := NetTapFlowManager.Query("id").In("tap_id", tapService.SubQuery())
	tapNics := NetTapFlowManager.Query("id").Equals("type", api.TapFlowGuestNic).Equals("source_id", self.Id)
	backends := LoadbalancerBackendManager.Query("id").Equals("backend_id", self.Id)

	pairs := []purgePair{
		{manager: LoadbalancerBackendManager, key: "id", q: backends},
		{manager: NetTapFlowManager, key: "id", q: tapNics},
		{manager: NetTapFlowManager, key: "id", q: tapFlows},
		{manager: NetTapServiceManager, key: "id", q: tapService},
		{manager: ElasticipManager, key: "id", q: publicIps},
		{manager: GuestsecgroupManager, key: "row_id", q: guestsecgroups},
		{manager: GroupguestManager, key: "row_id", q: guestgroups},
		{manager: GuestcdromManager, key: "row_id", q: guestcdroms},
		{manager: GuestFloppyManager, key: "row_id", q: guestvfd},
		{manager: GuestnetworkManager, key: "row_id", q: guestnetworks},
		{manager: GuestdiskManager, key: "row_id", q: guestdisks},
		{manager: InstanceSnapshotManager, key: "id", q: instancesnapshots},
		{manager: InstanceBackupManager, key: "id", q: instancebackups},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}
	return self.SVirtualResourceBase.Delete(ctx, userCred)
}

func (self *SZone) purgeWires(ctx context.Context, managerId string) error {
	wires := WireManager.Query("id").Equals("zone_id", self.Id).Equals("manager_id", managerId)
	// vpcs := VpcManager.Query().SubQuery()
	// wires = wires.Join(vpcs, sqlchemy.Equals(wires.Field("vpc_id"), vpcs.Field("id"))).
	// Filter(sqlchemy.Equals(vpcs.Field("manager_id"), managerId))

	hostwires := HostwireManagerDeprecated.Query("row_id").In("wire_id", wires.SubQuery())
	isolateds := IsolatedDeviceManager.Query("id").In("wire_id", wires.SubQuery())
	networks := NetworkManager.Query("id").In("wire_id", wires.SubQuery())
	bns := HostnetworkManager.Query("row_id").In("network_id", networks.SubQuery())
	rdsnetworks := DBInstanceNetworkManager.Query("row_id").In("network_id", networks.SubQuery())
	groupnetworks := GroupnetworkManager.Query("row_id").In("network_id", networks.SubQuery())
	lbnetworks := LoadbalancernetworkManager.Query("row_id").In("network_id", networks.SubQuery())
	netmacs := NetworkIpMacManager.Query("id").In("network_id", networks.SubQuery())
	netaddrs := NetworkAddressManager.Query("id").In("network_id", networks.SubQuery())
	nis := NetworkinterfacenetworkManager.Query("row_id").In("network_id", networks.SubQuery())
	rips := ReservedipManager.Query("id").In("network_id", networks.SubQuery())
	sns := ScalingGroupNetworkManager.Query("row_id").In("network_id", networks.SubQuery())
	schedtags := NetworkschedtagManager.Query("row_id").In("network_id", networks.SubQuery())

	pairs := []purgePair{
		{manager: NetworkschedtagManager, key: "row_id", q: schedtags},
		{manager: ScalingGroupNetworkManager, key: "row_id", q: sns},
		{manager: ReservedipManager, key: "id", q: rips},
		{manager: NetworkinterfacenetworkManager, key: "row_id", q: nis},
		{manager: NetworkAddressManager, key: "id", q: netaddrs},
		{manager: NetworkIpMacManager, key: "id", q: netmacs},
		{manager: LoadbalancernetworkManager, key: "row_id", q: lbnetworks},
		{manager: GroupnetworkManager, key: "row_id", q: groupnetworks},
		{manager: DBInstanceNetworkManager, key: "row_id", q: rdsnetworks},
		{manager: HostnetworkManager, key: "row_id", q: bns},
		{manager: NetworkManager, key: "id", q: networks},
		{manager: IsolatedDeviceManager, key: "id", q: isolateds},
		{manager: HostwireManagerDeprecated, key: "row_id", q: hostwires},
		{manager: NetworkAdditionalWireManager, key: "id", q: wires},
		{manager: WireManager, key: "id", q: wires},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cprvd *SCloudprovider) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	cprs := CloudproviderRegionManager.Query("row_id").Equals("cloudprovider_id", cprvd.Id)
	capability := CloudproviderCapabilityManager.Query("cloudprovider_id").Equals("cloudprovider_id", cprvd.Id)
	cdn := CDNDomainManager.Query("id").Equals("manager_id", cprvd.Id)
	vpcs := GlobalVpcManager.Query("id").Equals("manager_id", cprvd.Id)
	secgroups := SecurityGroupManager.Query("id").In("globalvpc_id", vpcs.SubQuery())
	rules := SecurityGroupRuleManager.Query("id").In("secgroup_id", secgroups.SubQuery())
	intervpcs := InterVpcNetworkManager.Query("id").Equals("manager_id", cprvd.Id)
	intervpcnetworks := InterVpcNetworkVpcManager.Query("row_id").In("inter_vpc_network_id", intervpcs.SubQuery())
	dnszones := DnsZoneManager.Query("id").Equals("manager_id", cprvd.Id)
	records := DnsRecordManager.Query("id").In("dns_zone_id", dnszones.SubQuery())
	dnsVpcs := DnsZoneVpcManager.Query("row_id").In("dns_zone_id", dnszones.SubQuery())

	pairs := []purgePair{
		{manager: DnsZoneVpcManager, key: "row_id", q: dnsVpcs},
		{manager: DnsRecordManager, key: "id", q: records},
		{manager: DnsZoneManager, key: "id", q: dnszones},
		{manager: InterVpcNetworkVpcManager, key: "row_id", q: intervpcnetworks},
		{manager: InterVpcNetworkManager, key: "id", q: intervpcs},
		{manager: SecurityGroupRuleManager, key: "id", q: rules},
		{manager: SecurityGroupManager, key: "id", q: secgroups},
		{manager: GlobalVpcManager, key: "id", q: vpcs},
		{manager: CDNDomainManager, key: "id", q: cdn},
		{manager: CloudproviderRegionManager, key: "row_id", q: cprs},
		{manager: CloudproviderCapabilityManager, key: "cloudprovider_id", q: capability},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}
	return cprvd.SEnabledStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (caccount *SCloudaccount) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	projects := ExternalProjectManager.Query("id").Equals("cloudaccount_id", caccount.Id)

	pairs := []purgePair{
		{manager: ExternalProjectManager, key: "id", q: projects},
	}
	for i := range pairs {
		err := pairs[i].purgeAll(ctx)
		if err != nil {
			return err
		}
	}
	return caccount.SEnabledStatusInfrasResourceBase.Delete(ctx, userCred)
}
