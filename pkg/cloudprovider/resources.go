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
	"yunion.io/x/pkg/tristate"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type ICloudResource interface {
	GetId() string
	GetName() string
	GetGlobalId() string

	GetStatus() string

	Refresh() error

	IsEmulated() bool
	GetMetadata() *jsonutils.JSONDict
}

type IVirtualResource interface {
	ICloudResource

	GetProjectId() string
}

type IBillingResource interface {
	GetBillingType() string
	GetCreatedAt() time.Time
	GetExpiredAt() time.Time
}

type ICloudRegion interface {
	ICloudResource

	// GetLatitude() float32
	// GetLongitude() float32
	GetGeographicInfo() SGeographicInfo

	GetIZones() ([]ICloudZone, error)
	GetIVpcs() ([]ICloudVpc, error)
	GetIEips() ([]ICloudEIP, error)
	GetIVpcById(id string) (ICloudVpc, error)
	GetIZoneById(id string) (ICloudZone, error)
	GetIEipById(id string) (ICloudEIP, error)
	// Esxi没有zone，需要通过region确认vm是否被删除
	GetIVMById(id string) (ICloudVM, error)
	GetIDiskById(id string) (ICloudDisk, error)

	GetISecurityGroupById(secgroupId string) (ICloudSecurityGroup, error)
	GetISecurityGroupByName(vpcId string, name string) (ICloudSecurityGroup, error)
	CreateISecurityGroup(conf *SecurityGroupCreateInput) (ICloudSecurityGroup, error)

	CreateIVpc(name string, desc string, cidr string) (ICloudVpc, error)
	CreateEIP(eip *SEip) (ICloudEIP, error)

	GetISnapshots() ([]ICloudSnapshot, error)
	GetISnapshotById(snapshotId string) (ICloudSnapshot, error)

	CreateSnapshotPolicy(*SnapshotPolicyInput) (string, error)
	UpdateSnapshotPolicy(*SnapshotPolicyInput, string) error
	DeleteSnapshotPolicy(string) error
	ApplySnapshotPolicyToDisks(snapshotPolicyId string, diskId string) error
	CancelSnapshotPolicyToDisks(snapshotPolicyId string, diskId string) error
	GetISnapshotPolicies() ([]ICloudSnapshotPolicy, error)
	GetISnapshotPolicyById(snapshotPolicyId string) (ICloudSnapshotPolicy, error)

	GetIHosts() ([]ICloudHost, error)
	GetIHostById(id string) (ICloudHost, error)

	GetIStorages() ([]ICloudStorage, error)
	GetIStorageById(id string) (ICloudStorage, error)

	GetIStoragecaches() ([]ICloudStoragecache, error)
	GetIStoragecacheById(id string) (ICloudStoragecache, error)

	GetILoadBalancers() ([]ICloudLoadbalancer, error)
	GetILoadBalancerAcls() ([]ICloudLoadbalancerAcl, error)
	GetILoadBalancerCertificates() ([]ICloudLoadbalancerCertificate, error)
	GetILoadBalancerBackendGroups() ([]ICloudLoadbalancerBackendGroup, error) // for aws only

	GetILoadBalancerById(loadbalancerId string) (ICloudLoadbalancer, error)
	GetILoadBalancerAclById(aclId string) (ICloudLoadbalancerAcl, error)
	GetILoadBalancerCertificateById(certId string) (ICloudLoadbalancerCertificate, error)

	CreateILoadBalancer(loadbalancer *SLoadbalancer) (ICloudLoadbalancer, error)
	CreateILoadBalancerAcl(acl *SLoadbalancerAccessControlList) (ICloudLoadbalancerAcl, error)
	CreateILoadBalancerCertificate(cert *SLoadbalancerCertificate) (ICloudLoadbalancerCertificate, error)

	GetISkus() ([]ICloudSku, error)
	CreateISku(name string, vCpu int, memoryMb int) error

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

	CreateIDBInstance(desc *SManagedDBInstanceCreateConfig) (ICloudDBInstance, error)

	GetIElasticcaches() ([]ICloudElasticcache, error)
	GetIElasticcacheById(id string) (ICloudElasticcache, error)
	CreateIElasticcaches(ec *SCloudElasticCacheInput) (ICloudElasticcache, error)

	GetCloudEnv() string
	GetProvider() string

	GetICloudEvents(start time.Time, end time.Time, withReadEvent bool) ([]ICloudEvent, error) //获取公有云操作日志接口
}

type ICloudZone interface {
	ICloudResource

	GetIRegion() ICloudRegion

	GetIHosts() ([]ICloudHost, error)
	GetIHostById(id string) (ICloudHost, error)

	GetIStorages() ([]ICloudStorage, error)
	GetIStorageById(id string) (ICloudStorage, error)
}

type ICloudImage interface {
	ICloudResource

	Delete(ctx context.Context) error
	GetIStoragecache() ICloudStoragecache

	GetSizeByte() int64
	GetImageType() string
	GetImageStatus() string
	GetOsType() string
	GetOsDist() string
	GetOsVersion() string
	GetOsArch() string
	GetMinOsDiskSizeGb() int
	GetMinRamSizeMb() int
	GetImageFormat() string
	GetCreatedAt() time.Time
}

type ICloudStoragecache interface {
	ICloudResource

	GetIImages() ([]ICloudImage, error)
	GetIImageById(extId string) (ICloudImage, error)

	GetPath() string

	CreateIImage(snapshotId, imageName, osType, imageDesc string) (ICloudImage, error)

	DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error)

	UploadImage(ctx context.Context, userCred mcclient.TokenCredential, image *SImageCreateOption, isForce bool) (string, error)
}

type ICloudStorage interface {
	ICloudResource

	GetIStoragecache() ICloudStoragecache

	GetIZone() ICloudZone
	GetIDisks() ([]ICloudDisk, error)

	GetStorageType() string
	GetMediumType() string
	GetCapacityMB() int64 // MB
	GetStorageConf() jsonutils.JSONObject
	GetEnabled() bool

	CreateIDisk(name string, sizeGb int, desc string) (ICloudDisk, error)

	GetIDiskById(idStr string) (ICloudDisk, error)

	GetMountPoint() string

	IsSysDiskStore() bool
}

type ICloudHost interface {
	ICloudResource

	GetIVMs() ([]ICloudVM, error)
	GetIVMById(id string) (ICloudVM, error)

	GetIWires() ([]ICloudWire, error)
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
	GetMemSizeMB() int
	GetMemCmtbound() float32
	GetReservedMemoryMb() int
	GetStorageSizeMB() int
	GetStorageType() string
	GetHostType() string

	GetIsMaintenance() bool
	GetVersion() string

	CreateVM(desc *SManagedVMCreateConfig) (ICloudVM, error)
	GetIHostNics() ([]ICloudHostNetInterface, error)
}

type ICloudVM interface {
	IBillingResource
	IVirtualResource

	GetIHost() ICloudHost
	GetIHostId() string

	GetIDisks() ([]ICloudDisk, error)
	GetINics() ([]ICloudNic, error)

	GetIEIP() (ICloudEIP, error)

	// GetStatus() string
	// GetRemoteStatus() string

	GetVcpuCount() int
	GetVmemSizeMB() int //MB
	GetBootOrder() string
	GetVga() string
	GetVdi() string
	GetOSType() string
	GetOSName() string
	GetBios() string
	GetMachine() string
	GetInstanceType() string

	GetSecurityGroupIds() ([]string, error)
	AssignSecurityGroup(secgroupId string) error
	SetSecurityGroups(secgroupIds []string) error

	GetHypervisor() string

	// GetSecurityGroup() ICloudSecurityGroup

	StartVM(ctx context.Context) error
	StopVM(ctx context.Context, isForce bool) error
	DeleteVM(ctx context.Context) error

	UpdateVM(ctx context.Context, name string) error

	UpdateUserData(userData string) error

	RebuildRoot(ctx context.Context, imageId string, passwd string, publicKey string, sysSizeGB int) (string, error)

	DeployVM(ctx context.Context, name string, username string, password string, publicKey string, deleteKeypair bool, description string) error

	ChangeConfig(ctx context.Context, config *SManagedVMChangeConfig) error

	GetVNCInfo() (jsonutils.JSONObject, error)
	AttachDisk(ctx context.Context, diskId string) error
	DetachDisk(ctx context.Context, diskId string) error

	CreateDisk(ctx context.Context, sizeMb int, uuid string, driver string) error

	Renew(bc billing.SBillingCycle) error

	GetError() error
}

type ICloudNic interface {
	GetIP() string
	GetMAC() string
	InClassicNetwork() bool
	GetDriver() string
	GetINetwork() ICloudNetwork
}

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
	ICloudResource

	GetDescription() string
	GetRules() ([]SecurityRule, error)
	GetVpcId() string

	SyncRules(common, inAdds, outAdds, inDels, outDels []SecurityRule) error
	Delete() error
}

type ICloudRouteTable interface {
	ICloudResource

	GetDescription() string
	GetRegionId() string
	GetVpcId() string
	GetType() string
	GetIRoutes() ([]ICloudRoute, error)
}

type ICloudRoute interface {
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

	GetDriver() string
	GetCacheMode() string
	GetMountpoint() string

	GetAccessPath() string

	Delete(ctx context.Context) error

	CreateISnapshot(ctx context.Context, name string, desc string) (ICloudSnapshot, error)
	GetISnapshot(idStr string) (ICloudSnapshot, error)
	GetISnapshots() ([]ICloudSnapshot, error)

	GetExtSnapshotPolicyIds() ([]string, error)

	Resize(ctx context.Context, newSizeMB int64) error
	Reset(ctx context.Context, snapshotId string) (string, error)

	Rebuild(ctx context.Context) error
}

type ICloudSnapshot interface {
	IVirtualResource

	GetSizeMb() int32
	GetDiskId() string
	GetDiskType() string
	Delete() error
}

type ICloudSnapshotPolicy interface {
	IVirtualResource

	IsActivated() bool
	GetRetentionDays() int
	GetRepeatWeekdays() ([]int, error)
	GetTimePoints() ([]int, error)
}

type ICloudVpc interface {
	// GetGlobalId() // 若vpc属于globalvpc,此函数返回格式必须是 'region.GetGlobalId()/vpc.GetGlobalId()'
	ICloudResource

	GetRegion() ICloudRegion
	GetIsDefault() bool
	GetCidrBlock() string
	// GetStatus() string
	GetIWires() ([]ICloudWire, error)
	GetISecurityGroups() ([]ICloudSecurityGroup, error)
	GetIRouteTables() ([]ICloudRouteTable, error)

	Delete() error

	GetIWireById(wireId string) (ICloudWire, error)
	GetINatGateways() ([]ICloudNatGateway, error)
}

type ICloudWire interface {
	ICloudResource
	GetIVpc() ICloudVpc
	GetIZone() ICloudZone
	GetINetworks() ([]ICloudNetwork, error)
	GetBandwidth() int

	GetINetworkById(netid string) (ICloudNetwork, error)

	CreateINetwork(name string, cidr string, desc string) (ICloudNetwork, error)
}

type ICloudNetwork interface {
	IVirtualResource

	GetIWire() ICloudWire
	// GetStatus() string
	GetIpStart() string
	GetIpEnd() string
	GetIpMask() int8
	GetGateway() string
	GetServerType() string
	GetIsPublic() bool
	GetPublicScope() rbacutils.TRbacScope

	Delete() error

	GetAllocTimeoutSeconds() int
}

type ICloudHostNetInterface interface {
	GetDevice() string
	GetDriver() string
	GetMac() string
	GetIndex() int8
	IsLinkUp() tristate.TriState
	GetIpAddr() string
	GetMtu() int32
	GetNicType() string
}

type ICloudLoadbalancer interface {
	IVirtualResource

	GetAddress() string
	GetAddressType() string
	GetNetworkType() string
	GetNetworkIds() []string
	GetVpcId() string
	GetZoneId() string
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

	CreateILoadBalancerListener(ctx context.Context, listener *SLoadbalancerListener) (ICloudLoadbalancerListener, error)
	GetILoadBalancerListenerById(listenerId string) (ICloudLoadbalancerListener, error)
}

type ICloudLoadbalancerListener interface {
	IVirtualResource

	GetListenerType() string
	GetListenerPort() int
	GetScheduler() string
	GetAclStatus() string
	GetAclType() string
	GetAclId() string

	GetEgressMbps() int

	GetHealthCheck() string
	GetHealthCheckType() string
	GetHealthCheckTimeout() int
	GetHealthCheckInterval() int
	GetHealthCheckRise() int
	GetHealthCheckFail() int

	GetHealthCheckReq() string
	GetHealthCheckExp() string

	GetBackendGroupId() string
	GetBackendServerPort() int

	// HTTP && HTTPS
	GetHealthCheckDomain() string
	GetHealthCheckURI() string
	GetHealthCheckCode() string
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

	Start() error
	Stop() error
	Sync(ctx context.Context, listener *SLoadbalancerListener) error

	Delete(ctx context.Context) error
}

type ICloudLoadbalancerListenerRule interface {
	IVirtualResource

	IsDefault() bool
	GetDomain() string
	GetPath() string
	GetCondition() string
	GetBackendGroupId() string

	Delete(ctx context.Context) error
}

type ICloudLoadbalancerBackendGroup interface {
	IVirtualResource

	IsDefault() bool
	GetType() string
	GetLoadbalancerId() string
	GetILoadbalancerBackends() ([]ICloudLoadbalancerBackend, error)
	GetILoadbalancerBackendById(backendId string) (ICloudLoadbalancerBackend, error)
	GetProtocolType() string                                // huawei only .后端云服务器组的后端协议。
	GetScheduler() string                                   // huawei only
	GetHealthCheck() (*SLoadbalancerHealthCheck, error)     // huawei only
	GetStickySession() (*SLoadbalancerStickySession, error) // huawei only
	AddBackendServer(serverId string, weight int, port int) (ICloudLoadbalancerBackend, error)
	RemoveBackendServer(serverId string, weight int, port int) error

	Delete(ctx context.Context) error
	Sync(ctx context.Context, group *SLoadbalancerBackendGroup) error
}

type ICloudLoadbalancerBackend interface {
	IVirtualResource

	GetWeight() int
	GetPort() int
	GetBackendType() string
	GetBackendRole() string
	GetBackendId() string
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
	GetGpuCount() int
	GetGpuMaxCount() int

	Delete() error
}

type ICloudProject interface {
	ICloudResource
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
	GetSecurityGroupId() string
	GetPort() int
	GetEngine() string
	GetEngineVersion() string
	//实例规格
	GetInstanceType() string

	GetVcpuCount() int
	GetVmemSizeMB() int //MB
	GetDiskSizeGB() int
	//基础版、高可用？
	GetCategory() string
	GetStorageType() string

	GetMaintainTime() string

	GetConnectionStr() string
	GetInternalConnectionStr() string
	GetIZoneId() string
	GetIVpcId() string

	GetDBNetwork() (*SDBInstanceNetwork, error)
	GetIDBInstanceParameters() ([]ICloudDBInstanceParameter, error)
	GetIDBInstanceDatabases() ([]ICloudDBInstanceDatabase, error)
	GetIDBInstanceAccounts() ([]ICloudDBInstanceAccount, error)
	GetIDBInstanceBackups() ([]ICloudDBInstanceBackup, error)

	ChangeConfig(ctx context.Context, config *SManagedDBInstanceChangeConfig) error
	Renew(bc billing.SBillingCycle) error

	OpenPublicConnection() error
	ClosePublicConnection() error

	CreateDatabase(conf *SDBInstanceDatabaseCreateConfig) error
	CreateAccount(conf *SDBInstanceAccountCreateConfig) error

	CreateIBackup(conf *SDBInstanceBackupCreateConfig) (string, error)

	RecoveryFromBackup(conf *SDBInstanceRecoveryConfig) error

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

	Delete() error
}

type ICloudDBInstanceDatabase interface {
	ICloudResource

	GetCharacterSet() string

	Delete() error
}

type ICloudDBInstanceAccount interface {
	ICloudResource

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

	GetPrivateDNS() string
	GetPrivateIpAddr() string
	GetPrivateConnectPort() int
	GetPublicDNS() string
	GetPublicIpAddr() string
	GetPublicConnectPort() int

	GetMaintainStartTime() string
	GetMaintainEndTime() string

	GetAuthMode() string

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
	CreateBackup() (ICloudElasticcacheBackup, error)
	FlushInstance() error
	UpdateAuthMode(noPasswordAccess bool) error
	UpdateInstanceParameters(config jsonutils.JSONObject) error
	UpdateBackupPolicy(config SCloudElasticCacheBackupPolicyUpdateInput) error
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
