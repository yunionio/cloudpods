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

package cloudprovider

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/samlutils"
	"yunion.io/x/pkg/util/secrules"
)

type ICloudResource interface {
	GetId() string
	GetName() string
	GetGlobalId() string
	GetCreatedAt() time.Time
	GetDescription() string

	GetStatus() string

	Refresh() error

	IsEmulated() bool

	GetSysTags() map[string]string
	GetTags() (map[string]string, error)
	SetTags(tags map[string]string, replace bool) error
}

type ICloudEnabledResource interface {
	ICloudResource
	GetEnabled() bool
}

type IVirtualResource interface {
	ICloudResource

	GetProjectId() string
}

type IBillingResource interface {
	GetBillingType() string
	GetExpiredAt() time.Time
	SetAutoRenew(bc billing.SBillingCycle) error
	Renew(bc billing.SBillingCycle) error
	IsAutoRenew() bool
}

type ICloudI18nResource interface {
	GetI18n() SModelI18nTable
}

type ICloudRegion interface {
	ICloudResource
	ICloudI18nResource

	GetGeographicInfo() SGeographicInfo

	GetIZones() ([]ICloudZone, error)
	GetIVpcs() ([]ICloudVpc, error)
	GetIEips() ([]ICloudEIP, error)
	GetIVpcById(id string) (ICloudVpc, error)
	GetIZoneById(id string) (ICloudZone, error)
	GetIEipById(id string) (ICloudEIP, error)
	// ICoudVM 的 GetGlobalId 接口不能panic
	GetIVMs() ([]ICloudVM, error)
	// Esxi没有zone，需要通过region确认vm是否被删除
	GetIVMById(id string) (ICloudVM, error)
	GetIDiskById(id string) (ICloudDisk, error)

	// 仅返回region级别的安全组, vpc下面的安全组需要在ICloudVpc底下返回
	GetISecurityGroups() ([]ICloudSecurityGroup, error)
	GetISecurityGroupById(secgroupId string) (ICloudSecurityGroup, error)
	CreateISecurityGroup(opts *SecurityGroupCreateInput) (ICloudSecurityGroup, error)

	CreateIVpc(opts *VpcCreateOptions) (ICloudVpc, error)
	CreateInternetGateway() (ICloudInternetGateway, error)
	CreateEIP(eip *SEip) (ICloudEIP, error)

	GetISnapshots() ([]ICloudSnapshot, error)
	GetISnapshotById(snapshotId string) (ICloudSnapshot, error)

	CreateSnapshotPolicy(*SnapshotPolicyInput) (string, error)
	GetISnapshotPolicies() ([]ICloudSnapshotPolicy, error)
	GetISnapshotPolicyById(id string) (ICloudSnapshotPolicy, error)

	GetIHosts() ([]ICloudHost, error)
	GetIHostById(id string) (ICloudHost, error)

	GetIStorages() ([]ICloudStorage, error)
	GetIStorageById(id string) (ICloudStorage, error)

	GetIStoragecaches() ([]ICloudStoragecache, error)
	GetIStoragecacheById(id string) (ICloudStoragecache, error)

	GetILoadBalancers() ([]ICloudLoadbalancer, error)
	GetILoadBalancerAcls() ([]ICloudLoadbalancerAcl, error)
	GetILoadBalancerCertificates() ([]ICloudLoadbalancerCertificate, error)

	GetILoadBalancerById(loadbalancerId string) (ICloudLoadbalancer, error)
	GetILoadBalancerAclById(aclId string) (ICloudLoadbalancerAcl, error)
	GetILoadBalancerCertificateById(certId string) (ICloudLoadbalancerCertificate, error)

	CreateILoadBalancer(loadbalancer *SLoadbalancerCreateOptions) (ICloudLoadbalancer, error)
	CreateILoadBalancerAcl(acl *SLoadbalancerAccessControlList) (ICloudLoadbalancerAcl, error)
	CreateILoadBalancerCertificate(cert *SLoadbalancerCertificate) (ICloudLoadbalancerCertificate, error)

	GetISkus() ([]ICloudSku, error)
	CreateISku(opts *SServerSkuCreateOption) (ICloudSku, error)

	GetICloudNatSkus() ([]ICloudNatSku, error)

	GetINetworkInterfaces() ([]ICloudNetworkInterface, error)

	GetIBuckets() ([]ICloudBucket, error)
	CreateIBucket(name string, storageClassStr string, acl string) error
	DeleteIBucket(name string) error
	IBucketExist(name string) (bool, error)
	GetIBucketById(name string) (ICloudBucket, error)
	GetIBucketByName(name string) (ICloudBucket, error)

	GetIDBInstances() ([]ICloudDBInstance, error)
	GetIDBInstanceById(instanceId string) (ICloudDBInstance, error)
	GetIDBInstanceBackups() ([]ICloudDBInstanceBackup, error)
	GetIDBInstanceBackupById(backupId string) (ICloudDBInstanceBackup, error)
	GetIDBInstanceSkus() ([]ICloudDBInstanceSku, error)

	CreateIDBInstance(desc *SManagedDBInstanceCreateConfig) (ICloudDBInstance, error)

	GetIElasticcaches() ([]ICloudElasticcache, error)
	GetIElasticcacheSkus() ([]ICloudElasticcacheSku, error)
	GetIElasticcacheById(id string) (ICloudElasticcache, error)
	CreateIElasticcaches(ec *SCloudElasticCacheInput) (ICloudElasticcache, error)

	GetCloudEnv() string
	GetProvider() string

	GetICloudEvents(start time.Time, end time.Time, withReadEvent bool) ([]ICloudEvent, error) //获取公有云操作日志接口

	GetCapabilities() []string

	GetICloudQuotas() ([]ICloudQuota, error)

	GetICloudFileSystems() ([]ICloudFileSystem, error)
	GetICloudFileSystemById(id string) (ICloudFileSystem, error)

	CreateICloudFileSystem(opts *FileSystemCraeteOptions) (ICloudFileSystem, error)

	GetICloudAccessGroups() ([]ICloudAccessGroup, error)
	CreateICloudAccessGroup(opts *SAccessGroup) (ICloudAccessGroup, error)
	GetICloudAccessGroupById(id string) (ICloudAccessGroup, error)

	GetICloudWafIPSets() ([]ICloudWafIPSet, error)
	GetICloudWafRegexSets() ([]ICloudWafRegexSet, error)
	GetICloudWafInstances() ([]ICloudWafInstance, error)
	GetICloudWafInstanceById(id string) (ICloudWafInstance, error)
	CreateICloudWafInstance(opts *WafCreateOptions) (ICloudWafInstance, error)
	GetICloudWafRuleGroups() ([]ICloudWafRuleGroup, error)

	GetICloudMongoDBs() ([]ICloudMongoDB, error)
	GetICloudMongoDBById(id string) (ICloudMongoDB, error)

	GetIElasticSearchs() ([]ICloudElasticSearch, error)
	GetIElasticSearchById(id string) (ICloudElasticSearch, error)

	GetICloudKafkas() ([]ICloudKafka, error)
	GetICloudKafkaById(id string) (ICloudKafka, error)

	GetICloudApps() ([]ICloudApp, error)
	GetICloudAppById(id string) (ICloudApp, error)

	GetICloudKubeClusters() ([]ICloudKubeCluster, error)
	GetICloudKubeClusterById(id string) (ICloudKubeCluster, error)
	CreateIKubeCluster(opts *KubeClusterCreateOptions) (ICloudKubeCluster, error)

	GetICloudTablestores() ([]ICloudTablestore, error)

	GetIModelartsPools() ([]ICloudModelartsPool, error)
	GetIModelartsPoolById(id string) (ICloudModelartsPool, error)
	CreateIModelartsPool(pool *ModelartsPoolCreateOption, callback func(externalId string)) (ICloudModelartsPool, error)
	GetIModelartsPoolSku() ([]ICloudModelartsPoolSku, error)

	GetIMiscResources() ([]ICloudMiscResource, error)

	GetISSLCertificates() ([]ICloudSSLCertificate, error)
}

type ICloudZone interface {
	ICloudResource
	ICloudI18nResource

	GetIRegion() ICloudRegion

	GetIHosts() ([]ICloudHost, error)
	GetIHostById(id string) (ICloudHost, error)

	GetIStorages() ([]ICloudStorage, error)
	GetIStorageById(id string) (ICloudStorage, error)
}

type ICloudImage interface {
	IVirtualResource

	IOSInfo

	Delete(ctx context.Context) error
	GetIStoragecache() ICloudStoragecache

	GetSizeByte() int64
	GetImageType() TImageType
	GetImageStatus() string

	GetMinOsDiskSizeGb() int
	GetMinRamSizeMb() int
	GetImageFormat() string

	GetPublicScope() rbacscope.TRbacScope
	GetSubImages() []SSubImage

	Export(opts *SImageExportOptions) ([]SImageExportInfo, error)
}

type ICloudStoragecache interface {
	ICloudResource

	// 私有云需要实现
	GetICloudImages() ([]ICloudImage, error)
	// 公有云需要实现
	GetICustomizedCloudImages() ([]ICloudImage, error)
	GetIImageById(extId string) (ICloudImage, error)

	GetPath() string

	UploadImage(ctx context.Context, image *SImageCreateOption, callback func(float32)) (string, error)
}

type ICloudStorage interface {
	ICloudResource

	GetIStoragecache() ICloudStoragecache

	GetIZone() ICloudZone
	GetIDisks() ([]ICloudDisk, error)

	GetStorageType() string
	GetMediumType() string
	GetCapacityMB() int64 // MB
	GetCapacityUsedMB() int64
	GetStorageConf() jsonutils.JSONObject
	GetEnabled() bool

	CreateIDisk(conf *DiskCreateConfig) (ICloudDisk, error)

	GetIDiskById(idStr string) (ICloudDisk, error)

	GetMountPoint() string

	IsSysDiskStore() bool

	DisableSync() bool
}

type ICloudHost interface {
	ICloudResource

	GetIVMs() ([]ICloudVM, error)
	GetIVMById(id string) (ICloudVM, error)

	// GetIWires() ([]ICloudWire, error)
	GetIStorages() ([]ICloudStorage, error)
	GetIStorageById(id string) (ICloudStorage, error)

	// GetStatus() string     // os status
	GetEnabled() bool      // is enabled
	GetHostStatus() string // service status
	GetAccessIp() string   //
	GetAccessMac() string  //
	GetSysInfo() jsonutils.JSONObject
	GetSN() string
	GetCpuCount() int
	GetNodeCount() int8
	GetCpuDesc() string
	GetCpuMhz() int
	GetCpuCmtbound() float32
	GetCpuArchitecture() string

	GetMemSizeMB() int
	GetMemCmtbound() float32
	GetReservedMemoryMb() int
	GetStorageSizeMB() int64
	GetStorageType() string
	GetHostType() string

	GetIsMaintenance() bool
	GetVersion() string

	CreateVM(desc *SManagedVMCreateConfig) (ICloudVM, error)
	GetIHostNics() ([]ICloudHostNetInterface, error)

	GetSchedtags() ([]string, error)

	GetOvnVersion() string // just for cloudpods host
}

type ICloudVM interface {
	IBillingResource
	IVirtualResource

	IOSInfo

	ConvertPublicIpToEip() error

	GetHostname() string
	GetIHost() ICloudHost
	GetIHostId() string

	GetIDisks() ([]ICloudDisk, error)
	GetINics() ([]ICloudNic, error)

	GetIEIP() (ICloudEIP, error)

	GetInternetMaxBandwidthOut() int
	GetThroughput() int
	// GetStatus() string
	// GetRemoteStatus() string

	GetSerialOutput(port int) (string, error) // 目前仅谷歌云windows机器会使用到此接口

	GetCpuSockets() int
	GetVcpuCount() int
	GetVmemSizeMB() int //MB
	GetBootOrder() string
	GetVga() string
	GetVdi() string

	// GetOSArch() string
	// GetOsType() TOsType
	// GetOSName() string
	// GetBios() string

	GetMachine() string
	GetInstanceType() string

	GetSecurityGroupIds() ([]string, error)
	SetSecurityGroups(secgroupIds []string) error

	GetHypervisor() string

	StartVM(ctx context.Context) error
	StopVM(ctx context.Context, opts *ServerStopOptions) error
	DeleteVM(ctx context.Context) error

	UpdateVM(ctx context.Context, input SInstanceUpdateOptions) error

	UpdateUserData(userData string) error

	RebuildRoot(ctx context.Context, config *SManagedVMRebuildRootConfig) (string, error)

	DeployVM(ctx context.Context, opts *SInstanceDeployOptions) error

	ChangeConfig(ctx context.Context, config *SManagedVMChangeConfig) error

	GetVNCInfo(input *ServerVncInput) (*ServerVncOutput, error)
	AttachDisk(ctx context.Context, diskId string) error
	DetachDisk(ctx context.Context, diskId string) error

	CreateDisk(ctx context.Context, opts *GuestDiskCreateOptions) (string, error)

	MigrateVM(hostid string) error
	LiveMigrateVM(hostid string) error

	GetError() error

	CreateInstanceSnapshot(ctx context.Context, name string, desc string) (ICloudInstanceSnapshot, error)
	GetInstanceSnapshot(idStr string) (ICloudInstanceSnapshot, error)
	GetInstanceSnapshots() ([]ICloudInstanceSnapshot, error)
	ResetToInstanceSnapshot(ctx context.Context, idStr string) error

	SaveImage(opts *SaveImageOptions) (ICloudImage, error)

	AllocatePublicIpAddress() (string, error)
	GetPowerStates() string
}

type ICloudNic interface {
	GetId() string
	GetIP() string
	GetIP6() string
	GetMAC() string
	InClassicNetwork() bool
	GetDriver() string
	GetINetworkId() string

	// GetSubAddress returns non-primary/secondary/alias ipv4 addresses of
	// the network interface
	//
	// Implement it when any AssignXx ops methods are implemented
	GetSubAddress() ([]string, error)
	AssignNAddress(count int) ([]string, error)
	AssignAddress(ipAddrs []string) error
	// UnassignAddress should not return error if the network interface is
	// now not present, or the addresses is not assigned to the network
	// interface in the first place
	UnassignAddress(ipAddrs []string) error
}

const ErrAddressCountExceed = errors.Error("ErrAddressCountExceed")

type DummyICloudNic struct{}

var _ ICloudNic = DummyICloudNic{}

func (d DummyICloudNic) GetId() string          { panic(errors.ErrNotImplemented) }
func (d DummyICloudNic) GetIP() string          { panic(errors.ErrNotImplemented) }
func (d DummyICloudNic) GetIP6() string         { return "" }
func (d DummyICloudNic) GetMAC() string         { panic(errors.ErrNotImplemented) }
func (d DummyICloudNic) InClassicNetwork() bool { panic(errors.ErrNotImplemented) }
func (d DummyICloudNic) GetDriver() string      { panic(errors.ErrNotImplemented) }
func (d DummyICloudNic) GetINetworkId() string  { panic(errors.ErrNotImplemented) }
func (d DummyICloudNic) GetSubAddress() ([]string, error) {
	return nil, nil
}
func (d DummyICloudNic) AssignNAddress(count int) ([]string, error) {
	return nil, errors.ErrNotImplemented
}
func (d DummyICloudNic) AssignAddress(ipAddrs []string) error   { return errors.ErrNotImplemented }
func (d DummyICloudNic) UnassignAddress(ipAddrs []string) error { return errors.ErrNotImplemented }

type ICloudEIP interface {
	IBillingResource
	IVirtualResource

	GetIpAddr() string
	GetMode() string
	GetINetworkId() string
	GetAssociationType() string
	GetAssociationExternalId() string

	GetBandwidth() int

	GetInternetChargeType() string

	Delete() error

	Associate(conf *AssociateConfig) error
	Dissociate() error

	ChangeBandwidth(bw int) error
}

type ICloudSecurityGroup interface {
	IVirtualResource

	GetDescription() string
	// 返回的优先级字段(priority)要求数字越大优先级越高, 若有默认不可修改的allow规则依然需要返回
	GetRules() ([]ISecurityGroupRule, error)
	GetVpcId() string

	CreateRule(opts *SecurityGroupRuleCreateOptions) (ISecurityGroupRule, error)

	GetReferences() ([]SecurityGroupReference, error)
	Delete() error
}

type ISecurityGroupRule interface {
	GetGlobalId() string
	GetDirection() secrules.TSecurityRuleDirection
	GetPriority() int
	GetAction() secrules.TSecurityRuleAction
	GetProtocol() string
	GetPorts() string
	GetDescription() string
	GetCIDRs() []string

	Update(opts *SecurityGroupRuleUpdateOptions) error
	Delete() error
}

type ICloudRouteTable interface {
	ICloudResource

	GetAssociations() []RouteTableAssociation
	GetDescription() string
	GetRegionId() string
	GetVpcId() string
	GetType() RouteTableType
	GetIRoutes() ([]ICloudRoute, error)

	CreateRoute(route RouteSet) error
	UpdateRoute(route RouteSet) error
	RemoveRoute(route RouteSet) error
}

type ICloudRoute interface {
	ICloudResource
	GetType() string
	GetCidr() string
	GetNextHopType() string
	GetNextHop() string
}

type ICloudDisk interface {
	IBillingResource
	IVirtualResource

	GetIStorage() (ICloudStorage, error)
	GetIStorageId() string

	// GetStatus() string
	GetDiskFormat() string
	GetDiskSizeMB() int // MB
	GetIsAutoDelete() bool
	GetTemplateId() string
	GetDiskType() string
	GetFsFormat() string
	GetIsNonPersistent() bool
	GetIops() int

	GetDriver() string
	GetCacheMode() string
	GetMountpoint() string

	GetAccessPath() string

	Delete(ctx context.Context) error

	CreateISnapshot(ctx context.Context, name string, desc string) (ICloudSnapshot, error)
	GetISnapshots() ([]ICloudSnapshot, error)

	Resize(ctx context.Context, newSizeMB int64) error
	Reset(ctx context.Context, snapshotId string) (string, error)

	Rebuild(ctx context.Context) error

	GetPreallocation() string
}

type ICloudSnapshot interface {
	IVirtualResource

	GetSizeMb() int32
	GetDiskId() string
	GetDiskType() string
	Delete() error
}

type ICloudInstanceSnapshot interface {
	IVirtualResource

	GetDescription() string
	Delete() error
}

type ICloudSnapshotPolicy interface {
	IVirtualResource

	GetRetentionDays() int
	GetRepeatWeekdays() ([]int, error)
	GetTimePoints() ([]int, error)
	Delete() error
	ApplyDisks(ids []string) error
	CancelDisks(ids []string) error
	GetApplyDiskIds() ([]string, error)
}

type ICloudGlobalVpc interface {
	ICloudResource

	GetISecurityGroups() ([]ICloudSecurityGroup, error)
	CreateISecurityGroup(opts *SecurityGroupCreateInput) (ICloudSecurityGroup, error)

	Delete() error
}

type ICloudIPv6Gateway interface {
	IVirtualResource

	GetInstanceType() string
}

type ICloudVpc interface {
	ICloudResource

	GetGlobalVpcId() string
	IsSupportSetExternalAccess() bool // 是否支持Attach互联网网关.
	GetExternalAccessMode() string
	AttachInternetGateway(igwId string) error

	GetRegion() ICloudRegion
	GetIsDefault() bool
	GetCidrBlock() string
	GetCidrBlock6() string
	GetIWires() ([]ICloudWire, error)
	CreateIWire(opts *SWireCreateOptions) (ICloudWire, error)
	GetISecurityGroups() ([]ICloudSecurityGroup, error)
	GetIRouteTables() ([]ICloudRouteTable, error)
	GetIRouteTableById(routeTableId string) (ICloudRouteTable, error)

	Delete() error

	GetIWireById(wireId string) (ICloudWire, error)
	GetINatGateways() ([]ICloudNatGateway, error)
	CreateINatGateway(opts *NatGatewayCreateOptions) (ICloudNatGateway, error)

	GetICloudVpcPeeringConnections() ([]ICloudVpcPeeringConnection, error)
	GetICloudAccepterVpcPeeringConnections() ([]ICloudVpcPeeringConnection, error)
	GetICloudVpcPeeringConnectionById(id string) (ICloudVpcPeeringConnection, error)
	CreateICloudVpcPeeringConnection(opts *VpcPeeringConnectionCreateOptions) (ICloudVpcPeeringConnection, error)
	AcceptICloudVpcPeeringConnection(id string) error

	GetAuthorityOwnerId() string

	ProposeJoinICloudInterVpcNetwork(opts *SVpcJointInterVpcNetworkOption) error

	GetICloudIPv6Gateways() ([]ICloudIPv6Gateway, error)
}

type ICloudInternetGateway interface {
	ICloudResource
}

type ICloudWire interface {
	ICloudResource
	GetIVpc() ICloudVpc
	GetIZone() ICloudZone
	GetINetworks() ([]ICloudNetwork, error)
	GetBandwidth() int

	GetINetworkById(netid string) (ICloudNetwork, error)

	CreateINetwork(opts *SNetworkCreateOptions) (ICloudNetwork, error)
}

type ICloudNetwork interface {
	IVirtualResource

	GetIWire() ICloudWire

	GetIpStart() string
	GetIpEnd() string
	GetIpMask() int8
	GetGateway() string

	// IPv6
	GetIp6Start() string
	GetIp6End() string
	GetIp6Mask() uint8
	GetGateway6() string

	GetServerType() string
	//GetIsPublic() bool
	// 仅私有云有用，公有云无效
	// 1. scope = none 非共享, network仅会属于一个项目,并且私有
	// 2. scope = system 系统共享 云账号共享会跟随云账号共享，云账号非共享,会共享到network所在域
	GetPublicScope() rbacscope.TRbacScope

	Delete() error

	GetAllocTimeoutSeconds() int
}

type ICloudHostNetInterface interface {
	GetDevice() string
	GetDriver() string
	GetMac() string
	GetVlanId() int
	GetIndex() int8
	IsLinkUp() tristate.TriState
	GetIpAddr() string
	GetMtu() int32
	GetNicType() string
	GetBridge() string
	GetIWire() ICloudWire
}

type ICloudLoadbalancer interface {
	IVirtualResource

	GetAddress() string
	GetAddressType() string
	GetNetworkType() string
	GetNetworkIds() []string
	GetVpcId() string
	GetZoneId() string
	GetZone1Id() string // first slave zone
	GetLoadbalancerSpec() string
	GetChargeType() string
	GetEgressMbps() int

	GetIEIP() (ICloudEIP, error)

	Delete(ctx context.Context) error

	Start() error
	Stop() error

	GetILoadBalancerListeners() ([]ICloudLoadbalancerListener, error)
	GetILoadBalancerBackendGroups() ([]ICloudLoadbalancerBackendGroup, error)

	CreateILoadBalancerBackendGroup(group *SLoadbalancerBackendGroup) (ICloudLoadbalancerBackendGroup, error)
	GetILoadBalancerBackendGroupById(groupId string) (ICloudLoadbalancerBackendGroup, error)

	CreateILoadBalancerListener(ctx context.Context, listener *SLoadbalancerListenerCreateOptions) (ICloudLoadbalancerListener, error)
	GetILoadBalancerListenerById(listenerId string) (ICloudLoadbalancerListener, error)
}

type ICloudLoadbalancerRedirect interface {
	GetRedirect() string
	GetRedirectCode() int64
	GetRedirectScheme() string
	GetRedirectHost() string
	GetRedirectPath() string
}

type ICloudloadbalancerHealthCheck interface {
	GetHealthCheck() string
	GetHealthCheckType() string
	GetHealthCheckTimeout() int
	GetHealthCheckInterval() int
	GetHealthCheckRise() int
	GetHealthCheckFail() int

	GetHealthCheckReq() string
	GetHealthCheckExp() string

	// HTTP && HTTPS
	GetHealthCheckDomain() string
	GetHealthCheckURI() string
	GetHealthCheckCode() string
}

type ICloudLoadbalancerListener interface {
	ICloudResource

	GetListenerType() string
	GetListenerPort() int
	GetScheduler() string
	GetAclStatus() string
	GetAclType() string
	GetAclId() string

	GetEgressMbps() int
	GetBackendGroupId() string
	GetBackendServerPort() int

	GetClientIdleTimeout() int
	GetBackendConnectTimeout() int

	// HTTP && HTTPS
	CreateILoadBalancerListenerRule(rule *SLoadbalancerListenerRule) (ICloudLoadbalancerListenerRule, error)
	GetILoadBalancerListenerRuleById(ruleId string) (ICloudLoadbalancerListenerRule, error)
	GetILoadbalancerListenerRules() ([]ICloudLoadbalancerListenerRule, error)
	GetStickySession() string
	GetStickySessionType() string
	GetStickySessionCookie() string
	GetStickySessionCookieTimeout() int
	XForwardedForEnabled() bool
	GzipEnabled() bool

	// HTTPS
	GetCertificateId() string
	GetTLSCipherPolicy() string
	HTTP2Enabled() bool

	// http redirect
	ICloudLoadbalancerRedirect
	ICloudloadbalancerHealthCheck

	Start() error
	Stop() error
	ChangeScheduler(ctx context.Context, opts *ChangeListenerSchedulerOptions) error
	SetHealthCheck(ctx context.Context, opts *ListenerHealthCheckOptions) error
	ChangeCertificate(ctx context.Context, opts *ListenerCertificateOptions) error
	SetAcl(ctx context.Context, opts *ListenerAclOptions) error

	Delete(ctx context.Context) error
}

type ICloudLoadbalancerListenerRule interface {
	ICloudResource
	// http redirect
	ICloudLoadbalancerRedirect

	IsDefault() bool
	GetDomain() string
	GetPath() string
	GetCondition() string
	GetBackendGroupId() string

	Delete(ctx context.Context) error
}

type ICloudLoadbalancerBackendGroup interface {
	ICloudResource

	IsDefault() bool
	GetType() string
	GetILoadbalancerBackends() ([]ICloudLoadbalancerBackend, error)
	GetILoadbalancerBackendById(backendId string) (ICloudLoadbalancerBackend, error)
	AddBackendServer(serverId string, weight int, port int) (ICloudLoadbalancerBackend, error)
	RemoveBackendServer(serverId string, weight int, port int) error

	Delete(ctx context.Context) error
	Sync(ctx context.Context, group *SLoadbalancerBackendGroup) error
}

type ICloudLoadbalancerBackend interface {
	ICloudResource

	GetWeight() int
	GetPort() int
	GetBackendType() string
	GetBackendRole() string
	GetBackendId() string
	GetIpAddress() string // backend type is ip
	SyncConf(ctx context.Context, port, weight int) error
}

type ICloudLoadbalancerCertificate interface {
	IVirtualResource

	Sync(name, privateKey, publickKey string) error
	Delete() error

	GetCommonName() string
	GetSubjectAlternativeNames() string
	GetFingerprint() string // return value format: <algo>:<fingerprint>，比如sha1:7454a14fdb8ae1ea8b2f72e458a24a76bd23ec19
	GetExpireTime() time.Time
	GetPublickKey() string
	GetPrivateKey() string
}

type ICloudLoadbalancerAcl interface {
	IVirtualResource

	GetAclListenerID() string // huawei only
	GetAclEntries() []SLoadbalancerAccessControlListEntry
	Sync(acl *SLoadbalancerAccessControlList) error
	Delete() error
}

type ICloudSku interface {
	ICloudResource

	GetInstanceTypeFamily() string
	GetInstanceTypeCategory() string

	GetPrepaidStatus() string
	GetPostpaidStatus() string

	GetCpuArch() string
	GetCpuCoreCount() int
	GetMemorySizeMB() int

	GetOsName() string

	GetSysDiskResizable() bool
	GetSysDiskType() string
	GetSysDiskMinSizeGB() int
	GetSysDiskMaxSizeGB() int

	GetAttachedDiskType() string
	GetAttachedDiskSizeGB() int
	GetAttachedDiskCount() int

	GetDataDiskTypes() string
	GetDataDiskMaxCount() int

	GetNicType() string
	GetNicMaxCount() int

	GetGpuAttachable() bool
	GetGpuSpec() string
	GetGpuCount() string
	GetGpuMaxCount() int

	Delete() error
}

type ICloudProject interface {
	ICloudResource

	GetDomainId() string
	GetDomainName() string

	GetAccountId() string
}

type ICloudNatGateway interface {
	ICloudResource
	IBillingResource

	// 获取 NAT 规格
	GetNatSpec() string
	GetIEips() ([]ICloudEIP, error)
	GetINatDTable() ([]ICloudNatDEntry, error)
	GetINatSTable() ([]ICloudNatSEntry, error)

	// ID is the ID of snat entry/rule or dnat entry/rule.
	GetINatDEntryByID(id string) (ICloudNatDEntry, error)
	GetINatSEntryByID(id string) (ICloudNatSEntry, error)

	// Read the description of these two structures before using.
	CreateINatDEntry(rule SNatDRule) (ICloudNatDEntry, error)
	CreateINatSEntry(rule SNatSRule) (ICloudNatSEntry, error)

	GetINetworkId() string
	GetBandwidthMb() int
	GetIpAddr() string

	Delete() error
}

// ICloudNatDEntry describe a DNat rule which transfer externalIp:externalPort to
// internalIp:internalPort with IpProtocol(tcp/udp)
type ICloudNatDEntry interface {
	ICloudResource

	GetIpProtocol() string
	GetExternalIp() string
	GetExternalPort() int

	GetInternalIp() string
	GetInternalPort() int

	Delete() error
}

// ICloudNatSEntry describe a SNat rule which transfer internalIp(GetIP()) to externalIp which from sourceCIDR
type ICloudNatSEntry interface {
	ICloudResource

	GetIP() string
	GetSourceCIDR() string
	GetNetworkId() string

	Delete() error
}

type ICloudNetworkInterface interface {
	ICloudResource

	GetMacAddress() string
	GetAssociateType() string
	GetAssociateId() string

	GetICloudInterfaceAddresses() ([]ICloudInterfaceAddress, error)
}

type ICloudInterfaceAddress interface {
	GetGlobalId() string //返回IP即可

	GetINetworkId() string
	GetIP() string
	IsPrimary() bool
}

type ICloudDBInstance interface {
	IVirtualResource
	IBillingResource

	Reboot() error

	GetMasterInstanceId() string
	GetSecurityGroupIds() ([]string, error)
	SetSecurityGroups(ids []string) error
	GetPort() int
	GetEngine() string
	GetEngineVersion() string
	//实例规格
	GetInstanceType() string

	GetVcpuCount() int
	GetVmemSizeMB() int //MB
	GetDiskSizeGB() int
	GetDiskSizeUsedMB() int
	//基础版、高可用？
	GetCategory() string
	GetStorageType() string

	GetMaintainTime() string

	GetConnectionStr() string
	GetInternalConnectionStr() string
	GetZone1Id() string
	GetZone2Id() string
	GetZone3Id() string
	GetIVpcId() string
	GetIops() int

	GetDBNetworks() ([]SDBInstanceNetwork, error)
	GetIDBInstanceParameters() ([]ICloudDBInstanceParameter, error)
	GetIDBInstanceDatabases() ([]ICloudDBInstanceDatabase, error)
	GetIDBInstanceAccounts() ([]ICloudDBInstanceAccount, error)
	GetIDBInstanceBackups() ([]ICloudDBInstanceBackup, error)

	ChangeConfig(ctx context.Context, config *SManagedDBInstanceChangeConfig) error

	OpenPublicConnection() error
	ClosePublicConnection() error

	CreateDatabase(conf *SDBInstanceDatabaseCreateConfig) error
	CreateAccount(conf *SDBInstanceAccountCreateConfig) error

	CreateIBackup(conf *SDBInstanceBackupCreateConfig) (string, error)

	RecoveryFromBackup(conf *SDBInstanceRecoveryConfig) error

	Update(ctx context.Context, input SDBInstanceUpdateOptions) error

	Delete() error
}

type ICloudDBInstanceParameter interface {
	GetGlobalId() string
	GetKey() string
	GetValue() string
	GetDescription() string
}

type ICloudDBInstanceBackup interface {
	IVirtualResource

	GetEngine() string
	GetEngineVersion() string
	GetDBInstanceId() string
	GetStartTime() time.Time
	GetEndTime() time.Time
	GetBackupSizeMb() int
	GetDBNames() string
	GetBackupMode() string
	GetBackupMethod() TBackupMethod

	CreateICloudDBInstance(opts *SManagedDBInstanceCreateConfig) (ICloudDBInstance, error)

	Delete() error
}

type ICloudDBInstanceDatabase interface {
	ICloudResource

	GetCharacterSet() string

	Delete() error
}

type ICloudDBInstanceAccount interface {
	GetName() string
	GetStatus() string
	GetHost() string

	GetIDBInstanceAccountPrivileges() ([]ICloudDBInstanceAccountPrivilege, error)

	ResetPassword(password string) error
	GrantPrivilege(database, privilege string) error
	RevokePrivilege(database string) error

	Delete() error
}

type ICloudDBInstanceAccountPrivilege interface {
	GetGlobalId() string

	GetPrivilege() string
	GetDBName() string
}

type ICloudElasticcacheSku interface {
	GetName() string
	GetGlobalId() string
	GetZoneId() string
	GetSlaveZoneId() string
	GetEngineArch() string
	GetLocalCategory() string
	GetPrepaidStatus() string
	GetPostpaidStatus() string
	GetEngine() string
	GetEngineVersion() string
	GetCpuArch() string
	GetStorageType() string
	GetMemorySizeMb() int
	GetPerformanceType() string
	GetNodeType() string
	GetDiskSizeGb() int
	GetShardNum() int
	GetMaxShardNum() int
	GetReplicasNum() int
	GetMaxReplicasNum() int
	GetMaxClients() int
	GetMaxConnections() int
	GetMaxInBandwidthMb() int
	GetMaxMemoryMb() int
	GetQps() int
}

type ICloudElasticcache interface {
	IVirtualResource
	IBillingResource

	GetInstanceType() string
	GetCapacityMB() int
	GetArchType() string
	GetNodeType() string
	GetEngine() string
	GetEngineVersion() string

	GetVpcId() string
	GetZoneId() string
	GetNetworkType() string
	GetNetworkId() string
	GetBandwidth() int
	GetConnections() int

	GetPrivateDNS() string
	GetPrivateIpAddr() string
	GetPrivateConnectPort() int
	GetPublicDNS() string
	GetPublicIpAddr() string
	GetPublicConnectPort() int

	GetMaintainStartTime() string
	GetMaintainEndTime() string

	GetAuthMode() string
	GetSecurityGroupIds() ([]string, error)

	GetICloudElasticcacheAccounts() ([]ICloudElasticcacheAccount, error)
	GetICloudElasticcacheAcls() ([]ICloudElasticcacheAcl, error)
	GetICloudElasticcacheBackups() ([]ICloudElasticcacheBackup, error)
	GetICloudElasticcacheParameters() ([]ICloudElasticcacheParameter, error)

	GetICloudElasticcacheAccount(accountId string) (ICloudElasticcacheAccount, error)
	GetICloudElasticcacheAcl(aclId string) (ICloudElasticcacheAcl, error)
	GetICloudElasticcacheBackup(backupId string) (ICloudElasticcacheBackup, error)

	Restart() error
	Delete() error
	ChangeInstanceSpec(spec string) error
	SetMaintainTime(maintainStartTime, maintainEndTime string) error
	AllocatePublicConnection(port int) (string, error) // return url & error info
	ReleasePublicConnection() error

	CreateAccount(account SCloudElasticCacheAccountInput) (ICloudElasticcacheAccount, error)
	CreateAcl(aclName, securityIps string) (ICloudElasticcacheAcl, error)
	CreateBackup(desc string) (ICloudElasticcacheBackup, error)
	FlushInstance(input SCloudElasticCacheFlushInstanceInput) error
	UpdateAuthMode(noPasswordAccess bool, password string) error
	UpdateInstanceParameters(config jsonutils.JSONObject) error
	UpdateBackupPolicy(config SCloudElasticCacheBackupPolicyUpdateInput) error

	UpdateSecurityGroups(secgroupIds []string) error
}

type ICloudElasticcacheAccount interface {
	ICloudResource

	GetAccountType() string
	GetAccountPrivilege() string

	Delete() error
	ResetPassword(input SCloudElasticCacheAccountResetPasswordInput) error
	UpdateAccount(input SCloudElasticCacheAccountUpdateInput) error
}

type ICloudElasticcacheAcl interface {
	ICloudResource

	GetIpList() string

	Delete() error
	UpdateAcl(securityIps string) error
}

type ICloudElasticcacheBackup interface {
	ICloudResource

	GetBackupSizeMb() int
	GetBackupType() string
	GetBackupMode() string
	GetDownloadURL() string

	GetStartTime() time.Time
	GetEndTime() time.Time

	Delete() error
	RestoreInstance(instanceId string) error
}

type ICloudElasticcacheParameter interface {
	ICloudResource

	GetParameterKey() string
	GetParameterValue() string
	GetParameterValueRange() string
	GetDescription() string
	GetModifiable() bool
	GetForceRestart() bool
}

type ICloudEvent interface {
	GetName() string
	GetService() string
	GetAction() string
	GetResourceType() string
	GetRequestId() string
	GetRequest() jsonutils.JSONObject
	GetAccount() string
	IsSuccess() bool

	GetCreatedAt() time.Time
}

type ICloudQuota interface {
	GetGlobalId() string
	GetDesc() string
	GetQuotaType() string
	GetMaxQuotaCount() int
	GetCurrentQuotaUsedCount() int
}

// 公有云子账号
type IClouduser interface {
	GetGlobalId() string
	GetName() string

	GetEmailAddr() string
	GetInviteUrl() string

	GetICloudgroups() ([]ICloudgroup, error)

	GetISystemCloudpolicies() ([]ICloudpolicy, error)
	GetICustomCloudpolicies() ([]ICloudpolicy, error)

	AttachSystemPolicy(policyName string) error
	DetachSystemPolicy(policyName string) error

	AttachCustomPolicy(policyName string) error
	DetachCustomPolicy(policyName string) error

	Delete() error

	ResetPassword(password string) error
	IsConsoleLogin() bool

	CreateAccessKey(name string) (*SAccessKey, error)
	DeleteAccessKey(accessKey string) error
	GetAccessKeys() ([]SAccessKey, error)
}

// 公有云子账号权限
type ICloudpolicy interface {
	GetGlobalId() string
	GetName() string
	GetDescription() string

	GetDocument() (*jsonutils.JSONDict, error)
	UpdateDocument(*jsonutils.JSONDict) error

	Delete() error
}

// 公有云用户组
type ICloudgroup interface {
	GetGlobalId() string
	GetName() string
	GetDescription() string
	GetISystemCloudpolicies() ([]ICloudpolicy, error)
	GetICustomCloudpolicies() ([]ICloudpolicy, error)
	GetICloudusers() ([]IClouduser, error)

	AddUser(name string) error
	RemoveUser(name string) error

	AttachSystemPolicy(policyName string) error
	DetachSystemPolicy(policyName string) error

	AttachCustomPolicy(policyName string) error
	DetachCustomPolicy(policyName string) error

	Delete() error
}

type ICloudDnsZone interface {
	IVirtualResource

	GetZoneType() TDnsZoneType

	GetICloudVpcIds() ([]string, error)
	AddVpc(*SPrivateZoneVpc) error
	RemoveVpc(*SPrivateZoneVpc) error

	GetIDnsRecords() ([]ICloudDnsRecord, error)
	GetIDnsRecordById(id string) (ICloudDnsRecord, error)

	AddDnsRecord(*DnsRecord) (string, error)

	Delete() error

	GetDnsProductType() TDnsProductType
}

type ICloudDnsRecord interface {
	GetGlobalId() string

	GetDnsName() string
	GetStatus() string
	GetEnabled() bool
	GetDnsType() TDnsType
	GetDnsValue() string
	GetTTL() int64
	GetMxPriority() int64

	Update(*DnsRecord) error

	Enable() error
	Disable() error

	GetPolicyType() TDnsPolicyType
	GetPolicyValue() TDnsPolicyValue
	Delete() error
}

type ICloudVpcPeeringConnection interface {
	ICloudResource

	GetPeerVpcId() string
	GetPeerAccountId() string
	GetEnabled() bool
	Delete() error
}

type ICloudSAMLProvider interface {
	ICloudResource

	GetMetadataDocument() (*samlutils.EntityDescriptor, error)
	UpdateMetadata(samlutils.EntityDescriptor) error

	GetAuthUrl(apiServer string) string
	Delete() error
}

type ICloudrole interface {
	GetGlobalId() string
	GetName() string

	GetDocument() *jsonutils.JSONDict
	GetSAMLProvider() string

	GetICloudpolicies() ([]ICloudpolicy, error)
	AttachPolicy(id string) error
	DetachPolicy(id string) error

	Delete() error
}

type ICloudInterVpcNetwork interface {
	ICloudResource
	GetAuthorityOwnerId() string
	GetICloudVpcIds() ([]string, error)
	AttachVpc(opts *SInterVpcNetworkAttachVpcOption) error
	DetachVpc(opts *SInterVpcNetworkDetachVpcOption) error
	Delete() error
	GetIRoutes() ([]ICloudInterVpcNetworkRoute, error)
	EnableRouteEntry(routeId string) error
	DisableRouteEntry(routeId string) error
}

type ICloudInterVpcNetworkRoute interface {
	ICloudResource
	GetInstanceId() string
	GetInstanceType() string
	GetInstanceRegionId() string

	GetEnabled() bool
	GetCidr() string
}

type ICloudFileSystem interface {
	ICloudResource
	IBillingResource

	GetFileSystemType() string
	GetStorageType() string
	GetProtocol() string
	GetCapacityGb() int64
	GetUsedCapacityGb() int64
	GetMountTargetCountLimit() int

	GetZoneId() string

	GetMountTargets() ([]ICloudMountTarget, error)
	CreateMountTarget(opts *SMountTargetCreateOptions) (ICloudMountTarget, error)

	Delete() error
}

type ICloudMountTarget interface {
	GetGlobalId() string
	GetName() string
	GetAccessGroupId() string
	GetDomainName() string
	GetNetworkType() string
	GetVpcId() string
	GetNetworkId() string
	GetStatus() string

	Delete() error
}

type ICloudAccessGroup interface {
	GetGlobalId() string
	GetName() string
	GetDesc() string
	GetSupporedUserAccessTypes() []TUserAccessType
	GetNetworkType() string
	GetFileSystemType() string
	GetMountTargetCount() int

	GetRules() ([]IAccessGroupRule, error)
	CreateRule(opts *AccessGroupRule) (IAccessGroupRule, error)

	Delete() error
}

type IAccessGroupRule interface {
	GetGlobalId() string
	GetPriority() int
	GetRWAccessType() TRWAccessType
	GetUserAccessType() TUserAccessType
	GetSource() string

	Delete() error
}

type ICloudWafIPSet interface {
	GetName() string
	GetDesc() string
	GetType() TWafType
	GetGlobalId() string
	GetAddresses() WafAddresses

	Delete() error
}

type ICloudWafRegexSet interface {
	GetName() string
	GetDesc() string
	GetType() TWafType
	GetGlobalId() string
	GetRegexPatterns() WafRegexPatterns

	Delete() error
}

type ICloudWafInstance interface {
	ICloudEnabledResource

	GetWafType() TWafType
	GetDefaultAction() *DefaultAction
	GetRules() ([]ICloudWafRule, error)
	AddRule(opts *SWafRule) (ICloudWafRule, error)

	// 绑定的资源列表
	GetCloudResources() ([]SCloudResource, error)

	Delete() error
}

type ICloudWafRuleGroup interface {
	GetName() string
	GetDesc() string
	GetGlobalId() string
	GetWafType() TWafType
	GetRules() ([]ICloudWafRule, error)
}

type ICloudWafRule interface {
	GetName() string
	GetDesc() string
	GetGlobalId() string
	GetPriority() int
	GetAction() *DefaultAction
	GetStatementCondition() TWafStatementCondition
	GetStatements() ([]SWafStatement, error)

	Update(opts *SWafRule) error
	Delete() error
}

type ICloudMongoDB interface {
	IVirtualResource
	IBillingResource

	GetVpcId() string
	GetNetworkId() string
	GetIpAddr() string
	GetVcpuCount() int
	GetVmemSizeMb() int
	GetDiskSizeMb() int
	GetZoneId() string
	GetReplicationNum() int
	GetCategory() string
	GetEngine() string
	GetEngineVersion() string
	GetInstanceType() string
	GetMaintainTime() string
	GetPort() int
	GetIops() int

	GetMaxConnections() int

	GetNetworkAddress() string

	GetIBackups() ([]SMongoDBBackup, error)

	Delete() error
}

type ICloudElasticSearch interface {
	IVirtualResource
	IBillingResource

	GetVersion() string
	GetStorageType() string
	GetDiskSizeGb() int
	GetCategory() string

	GetInstanceType() string
	GetVcpuCount() int
	GetVmemSizeGb() int

	GetVpcId() string
	GetNetworkId() string
	GetZoneId() string
	IsMultiAz() bool

	GetAccessInfo() (*ElasticSearchAccessInfo, error)

	Delete() error
}

type ICloudKafka interface {
	IVirtualResource
	IBillingResource

	GetNetworkId() string
	GetVpcId() string
	GetZoneId() string
	GetInstanceType() string

	GetVersion() string
	GetDiskSizeGb() int
	GetStorageType() string
	GetBandwidthMb() int
	GetEndpoint() string
	GetMsgRetentionMinute() int

	IsMultiAz() bool

	GetTopics() ([]SKafkaTopic, error)

	Delete() error
}

type ICloudApp interface {
	IVirtualResource
	GetEnvironments() ([]ICloudAppEnvironment, error)
	GetTechStack() string
	GetType() string
	GetKind() string
	GetOsType() TOsType
}

type ICloudAppEnvironment interface {
	IVirtualResource
	GetInstanceType() (string, error)
	GetInstanceNumber() (int, error)
}

type ICloudDBInstanceSku interface {
	GetName() string
	GetGlobalId() string
	GetStatus() string
	GetEngine() string
	GetEngineVersion() string
	GetStorageType() string
	GetDiskSizeStep() int
	GetMaxDiskSizeGb() int
	GetMinDiskSizeGb() int
	GetIOPS() int
	GetTPS() int
	GetQPS() int
	GetMaxConnections() int
	GetVcpuCount() int
	GetVmemSizeMb() int
	GetCategory() string
	GetZone1Id() string
	GetZone2Id() string
	GetZone3Id() string
	GetZoneId() string
}

type ICloudNatSku interface {
	GetName() string
	GetDesc() string
	GetGlobalId() string
	GetPrepaidStatus() string
	GetPostpaidStatus() string
}

type ICloudCDNDomain interface {
	IVirtualResource
	GetEnabled() bool

	GetArea() string
	GetServiceType() string
	GetCname() string
	GetOrigins() *SCdnOrigins

	// 是否忽略参数
	GetCacheKeys() (*SCDNCacheKeys, error)
	// 是否分片回源
	GetRangeOriginPull() (*SCDNRangeOriginPull, error)
	// 缓存配置
	GetCache() (*SCDNCache, error)
	// https配置
	GetHTTPS() (*SCDNHttps, error)
	// 强制跳转
	GetForceRedirect() (*SCDNForceRedirect, error)
	// 防盗链配置
	GetReferer() (*SCDNReferer, error)
	// 浏览器缓存配置
	GetMaxAge() (*SCDNMaxAge, error)

	Delete() error
}

type ICloudKubeCluster interface {
	ICloudEnabledResource

	GetKubeConfig(private bool, expireMinutes int) (*SKubeconfig, error)

	GetVersion() string
	GetVpcId() string
	GetNetworkIds() []string

	GetIKubeNodePools() ([]ICloudKubeNodePool, error)
	CreateIKubeNodePool(opts *KubeNodePoolCreateOptions) (ICloudKubeNodePool, error)
	GetIKubeNodes() ([]ICloudKubeNode, error)

	Delete(isRetain bool) error
}

type ICloudKubeNode interface {
	ICloudResource

	GetINodePoolId() string
}

type ICloudKubeNodePool interface {
	ICloudResource

	GetMinInstanceCount() int
	GetMaxInstanceCount() int
	GetDesiredInstanceCount() int
	GetRootDiskSizeGb() int

	GetInstanceTypes() []string
	GetNetworkIds() []string

	Delete() error
}

type ICloudTablestore interface {
	IVirtualResource
}

type ICloudMiscResource interface {
	IVirtualResource

	GetResourceType() string

	GetConfig() jsonutils.JSONObject
}

type ICloudSSLCertificate interface {
	IVirtualResource

	GetSans() string
	GetStartDate() time.Time
	GetProvince() string
	GetCommon() string
	GetCountry() string
	GetIssuer() string
	GetExpired() bool
	GetEndDate() time.Time
	GetFingerprint() string
	GetCity() string
	GetOrgName() string
	GetIsUpload() bool
	GetCert() string
	GetKey() string
}
