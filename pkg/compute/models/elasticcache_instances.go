package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SElasticcacheManager struct {
	db.SVirtualResourceBaseManager
}

var ElasticcacheManager *SElasticcacheManager

func init() {
	ElasticcacheManager = &SElasticcacheManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SElasticcache{},
			"elasticcacheinstances_tbl",
			"elasticcache",
			"elasticcaches",
		),
	}
	ElasticcacheManager.SetVirtualObject(ElasticcacheManager)
}

type SElasticcache struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase
	SBillingResourceBase
	SManagedResourceBase

	SCloudregionResourceBase
	SZoneResourceBase

	InstanceType  string `width:"64" charset:"ascii" nullable:"true" list:"user" create:"optional"`  // redis.master.micro.default
	CapacityMB    int    `nullable:"false" list:"user" create:"optional"`                            //  1024
	ArchType      string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional"` // 集群版 | 标准版 | 读写分离版 | 单机 ？
	NodeType      string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional"` // STAND_ALONE（单节点） MASTER_SLAVE（多节点) ？
	Engine        string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"` // Redis | Memcache
	EngineVersion string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"` // 4.0 5.0

	VpcId       string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	NetworkType string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional"` // CLASSIC（经典网络）  VPC（专有网络）
	NetworkId   string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`

	PrivateDNS         string `width:"256" charset:"ascii" nullable:"false" list:"user" create:"optional"` //  内网DNS
	PrivateIpAddr      string `width:"17" charset:"ascii" list:"user" create:"optional"`                   //  内网IP地址
	PrivateConnectPort int    `nullable:"false" list:"user" create:"optional"`                             // 内网访问端口
	PublicDNS          string `width:"256" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	PublicIpAddr       string `width:"17" charset:"ascii" list:"user" create:"optional"` //
	PublicConnectPort  int    `nullable:"false" list:"user" create:"optional"`           // 外网访问端口

	MaintainStartTime string `width:"8" charset:"ascii" nullable:"false" list:"user" create:"optional"` // HH:mmZ eg. 02:00Z
	MaintainEndTime   string `width:"8" charset:"ascii" nullable:"false" list:"user" create:"optional"`

	// AutoRenew // 自动续费
	// AutoRenewPeriod // 自动续费周期
}

func (self *SElasticcache) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return extra
}

func (self *SElasticcache) GetElasticcacheParameters() ([]SElasticcacheParameter, error) {
	ret := []SElasticcacheParameter{}
	q := ElasticcacheParameterManager.Query().Equals("elasticcache_id", self.Id)
	err := db.FetchModelObjects(ElasticcacheParameterManager, q, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "GetElasticcacheParameters.FetchModelObjects for elastic cache %s", self.Id)
	}
	return ret, nil
}

func (self *SElasticcache) GetElasticcacheAccounts() ([]SElasticcacheAccount, error) {
	ret := []SElasticcacheAccount{}
	q := ElasticcacheAccountManager.Query().Equals("elasticcache_id", self.Id)
	err := db.FetchModelObjects(ElasticcacheAccountManager, q, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "GetElasticcacheAccounts.FetchModelObjects for elastic cache %s", self.Id)
	}
	return ret, nil
}

func (self *SElasticcache) GetElasticcacheAcls() ([]SElasticcacheAcl, error) {
	ret := []SElasticcacheAcl{}
	q := ElasticcacheAclManager.Query().Equals("elasticcache_id", self.Id)
	err := db.FetchModelObjects(ElasticcacheAclManager, q, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "GetElasticcacheAcls.FetchModelObjects for elastic cache %s", self.Id)
	}
	return ret, nil
}

func (self *SElasticcache) GetElasticcacheBackups() ([]SElasticcacheBackup, error) {
	ret := []SElasticcacheBackup{}
	q := ElasticcacheBackupManager.Query().Equals("elasticcache_id", self.Id)
	err := db.FetchModelObjects(ElasticcacheBackupManager, q, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "GetElasticcacheBackups.FetchModelObjects for elastic cache %s", self.Id)
	}
	return ret, nil
}

func (manager *SElasticcacheManager) SyncElasticcaches(ctx context.Context, userCred mcclient.TokenCredential, syncOwnerId mcclient.IIdentityProvider, provider *SCloudprovider, region *SCloudregion, cloudElasticcaches []cloudprovider.ICloudElasticcache) ([]SElasticcache, []cloudprovider.ICloudElasticcache, compare.SyncResult) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, provider.GetOwnerId()))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, provider.GetOwnerId()))

	localElasticcaches := []SElasticcache{}
	remoteElasticcaches := []cloudprovider.ICloudElasticcache{}
	syncResult := compare.SyncResult{}

	dbInstances, err := region.GetElasticcaches(provider)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := make([]SElasticcache, 0)
	commondb := make([]SElasticcache, 0)
	commonext := make([]cloudprovider.ICloudElasticcache, 0)
	added := make([]cloudprovider.ICloudElasticcache, 0)
	if err := compare.CompareSets(dbInstances, cloudElasticcaches, &removed, &commondb, &commonext, &added); err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemoveCloudElasticcache(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i++ {
		err := commondb[i].SyncWithCloudElasticcache(ctx, userCred, provider, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
			continue
		}
		syncMetadata(ctx, userCred, &commondb[i], commonext[i])
		localElasticcaches = append(localElasticcaches, commondb[i])
		remoteElasticcaches = append(remoteElasticcaches, commonext[i])
		syncResult.Update()
	}

	for i := 0; i < len(added); i++ {
		instance, err := manager.newFromCloudElasticcache(ctx, userCred, syncOwnerId, provider, region, added[i])
		if err != nil {
			syncResult.AddError(err)
			continue
		}
		syncMetadata(ctx, userCred, instance, added[i])
		localElasticcaches = append(localElasticcaches, *instance)
		remoteElasticcaches = append(remoteElasticcaches, added[i])
		syncResult.Add()
	}
	return localElasticcaches, remoteElasticcaches, syncResult
}

func (self *SElasticcache) syncRemoveCloudElasticcache(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return self.SetStatus(userCred, api.ELASTIC_CACHE_STATUS_ERROR, "sync to delete")
	}
	return self.Delete(ctx, userCred)
}

func (self *SElasticcache) SyncWithCloudElasticcache(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extInstance cloudprovider.ICloudElasticcache) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = extInstance.GetStatus()
		self.InstanceType = extInstance.GetInstanceType()
		self.CapacityMB = extInstance.GetCapacityMB()
		self.ArchType = extInstance.GetArchType()
		self.NodeType = extInstance.GetNodeType()
		self.Engine = extInstance.GetEngine()
		self.EngineVersion = extInstance.GetEngineVersion()

		self.NetworkType = extInstance.GetNetworkType()
		self.PrivateDNS = extInstance.GetPrivateDNS()
		self.PrivateIpAddr = extInstance.GetPrivateIpAddr()
		self.PrivateConnectPort = extInstance.GetPrivateConnectPort()
		self.PublicDNS = extInstance.GetPublicDNS()
		self.PublicIpAddr = extInstance.GetPublicIpAddr()
		self.PublicConnectPort = extInstance.GetPublicConnectPort()
		self.MaintainStartTime = extInstance.GetMaintainStartTime()
		self.MaintainEndTime = extInstance.GetMaintainEndTime()

		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "syncWithCloudElasticcache.Update")
	}

	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (manager *SElasticcacheManager) newFromCloudElasticcache(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, provider *SCloudprovider, region *SCloudregion, extInstance cloudprovider.ICloudElasticcache) (*SElasticcache, error) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	instance := SElasticcache{}
	instance.SetModelManager(manager, &instance)

	newName, err := db.GenerateName(manager, ownerId, extInstance.GetName())
	if err != nil {
		return nil, err
	}
	instance.Name = newName
	instance.ExternalId = extInstance.GetGlobalId()
	instance.CloudregionId = region.Id
	instance.ManagerId = provider.Id
	instance.IsEmulated = extInstance.IsEmulated()
	instance.Status = extInstance.GetStatus()

	instance.InstanceType = extInstance.GetInstanceType()
	instance.CapacityMB = extInstance.GetCapacityMB()
	instance.ArchType = extInstance.GetArchType()
	instance.NodeType = extInstance.GetNodeType()
	instance.Engine = extInstance.GetEngine()
	instance.EngineVersion = extInstance.GetEngineVersion()

	instance.NetworkType = extInstance.GetNetworkType()
	instance.PrivateDNS = extInstance.GetPrivateDNS()
	instance.PrivateIpAddr = extInstance.GetPrivateIpAddr()
	instance.PrivateConnectPort = extInstance.GetPrivateConnectPort()
	instance.PublicDNS = extInstance.GetPublicDNS()
	instance.PublicIpAddr = extInstance.GetPublicIpAddr()
	instance.PublicConnectPort = extInstance.GetPublicConnectPort()
	instance.MaintainStartTime = extInstance.GetMaintainStartTime()
	instance.MaintainEndTime = extInstance.GetMaintainEndTime()

	if zoneId := extInstance.GetZoneId(); len(zoneId) > 0 {
		zone, err := db.FetchByExternalId(ZoneManager, zoneId)
		if err != nil {
			return nil, errors.Wrapf(err, "newFromCloudElasticcache.FetchZoneId")
		}
		instance.ZoneId = zone.GetId()
	}

	if vpcId := extInstance.GetVpcId(); len(vpcId) > 0 {
		vpc, err := db.FetchByExternalId(VpcManager, vpcId)
		if err != nil {
			return nil, errors.Wrapf(err, "newFromCloudElasticcache.FetchVpcId")
		}
		instance.VpcId = vpc.GetId()
	}

	if networkId := extInstance.GetNetworkId(); len(networkId) > 0 {
		network, err := db.FetchByExternalId(NetworkManager, networkId)
		if err != nil {
			return nil, errors.Wrapf(err, "newFromCloudElasticcache.FetchNetworkId")
		}
		instance.NetworkId = network.GetId()
	}

	if createdAt := extInstance.GetCreatedAt(); !createdAt.IsZero() {
		instance.CreatedAt = createdAt
	}

	factory, err := provider.GetProviderFactory()
	if err != nil {
		return nil, errors.Wrap(err, "newFromCloudElasticcache.GetProviderFactory")
	}

	if factory.IsSupportPrepaidResources() {
		instance.BillingType = extInstance.GetBillingType()
		instance.ExpiredAt = extInstance.GetExpiredAt()
	}

	err = manager.TableSpec().Insert(&instance)
	if err != nil {
		return nil, errors.Wrapf(err, "newFromCloudElasticcache.Insert")
	}

	SyncCloudProject(userCred, &instance, ownerId, extInstance, instance.ManagerId)
	db.OpsLog.LogEvent(&instance, db.ACT_CREATE, instance.GetShortDesc(ctx), userCred)

	return &instance, nil
}
