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

package client

import (
	"net/http"

	"yunion.io/x/cloudmux/pkg/multicloud/huawei/client/auth"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei/client/auth/credentials"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei/client/modules"
)

type Client struct {
	cfg *SClientConfig
	// 标记初始化状态
	init bool

	Credentials          *modules.SCredentialManager
	Disks                *modules.SDiskManager
	Domains              *modules.SDomainManager
	Elasticcache         *modules.SElasticcacheManager
	DcsAvailableZone     *modules.SDcsAvailableZoneManager
	Flavors              *modules.SFlavorManager
	Images               *modules.SImageManager
	OpenStackImages      *modules.SImageManager
	Interface            *modules.SInterfaceManager
	Jobs                 *modules.SJobManager
	Keypairs             *modules.SKeypairManager
	Orders               *modules.SOrderManager
	Port                 *modules.SPortManager
	Projects             *modules.SProjectManager
	Regions              *modules.SRegionManager
	Servers              *modules.SServerManager
	ServersV2            *modules.SServerManager
	NovaServers          *modules.SServerManager
	Snapshots            *modules.SSnapshotManager
	OsSnapshots          *modules.SSnapshotManager
	Subnets              *modules.SSubnetManager
	Users                *modules.SUserManager
	Vpcs                 *modules.SVpcManager
	Zones                *modules.SZoneManager
	VpcRoutes            *modules.SVpcRouteManager
	SNatRules            *modules.SNatSRuleManager
	DNatRules            *modules.SNatDRuleManager
	NatGateways          *modules.SNatGatewayManager
	VpcPeerings          *modules.SVpcPeeringManager
	DBInstance           *modules.SDBInstanceManager
	DBInstanceBackup     *modules.SDBInstanceBackupManager
	DBInstanceFlavor     *modules.SDBInstanceFlavorManager
	DBInstanceJob        *modules.SDBInstanceJobManager
	Traces               *modules.STraceManager
	CloudEye             *modules.SCloudEyeManager
	Quotas               *modules.SQuotaManager
	EnterpriseProjects   *modules.SEnterpriseProjectManager
	Roles                *modules.SRoleManager
	Groups               *modules.SGroupManager
	SAMLProviders        *modules.SAMLProviderManager
	SAMLProviderMappings *modules.SAMLProviderMappingManager
	SfsTurbos            *modules.SfsTurboManager
	RouteTables          *modules.SRouteTableManager
}

type SClientConfig struct {
	signer    auth.Signer
	endpoint  string // myhuaweicloud.com
	regionId  string
	domainId  string
	projectId string

	debug bool
}

func (self *SClientConfig) GetSigner() auth.Signer {
	return self.signer
}

func (self *SClientConfig) GetEndpoint() string {
	return self.endpoint
}

func (self *SClientConfig) GetRegionId() string {
	return self.regionId
}

func (self *SClientConfig) GetDomainId() string {
	return self.domainId
}

func (self *SClientConfig) GetProjectId() string {
	return self.projectId
}

func (self *SClientConfig) GetDebug() bool {
	return self.debug
}

func (self *Client) SetHttpClient(httpClient *http.Client) {
	self.Credentials.SetHttpClient(httpClient)
	self.Servers.SetHttpClient(httpClient)
	self.ServersV2.SetHttpClient(httpClient)
	self.NovaServers.SetHttpClient(httpClient)
	self.Snapshots.SetHttpClient(httpClient)
	self.OsSnapshots.SetHttpClient(httpClient)
	self.Images.SetHttpClient(httpClient)
	self.OpenStackImages.SetHttpClient(httpClient)
	self.Projects.SetHttpClient(httpClient)
	self.Regions.SetHttpClient(httpClient)
	self.Zones.SetHttpClient(httpClient)
	self.Vpcs.SetHttpClient(httpClient)
	self.Elasticcache.SetHttpClient(httpClient)
	self.DcsAvailableZone.SetHttpClient(httpClient)
	self.Disks.SetHttpClient(httpClient)
	self.Domains.SetHttpClient(httpClient)
	self.Keypairs.SetHttpClient(httpClient)
	self.Orders.SetHttpClient(httpClient)
	self.Subnets.SetHttpClient(httpClient)
	self.Users.SetHttpClient(httpClient)
	self.Interface.SetHttpClient(httpClient)
	self.Jobs.SetHttpClient(httpClient)
	self.Port.SetHttpClient(httpClient)
	self.Flavors.SetHttpClient(httpClient)
	self.VpcRoutes.SetHttpClient(httpClient)
	self.SNatRules.SetHttpClient(httpClient)
	self.DNatRules.SetHttpClient(httpClient)
	self.NatGateways.SetHttpClient(httpClient)
	self.DBInstance.SetHttpClient(httpClient)
	self.DBInstanceBackup.SetHttpClient(httpClient)
	self.DBInstanceFlavor.SetHttpClient(httpClient)
	self.DBInstanceJob.SetHttpClient(httpClient)
	self.Traces.SetHttpClient(httpClient)
	self.CloudEye.SetHttpClient(httpClient)
	self.EnterpriseProjects.SetHttpClient(httpClient)
	self.Roles.SetHttpClient(httpClient)
	self.Groups.SetHttpClient(httpClient)
	self.SAMLProviders.SetHttpClient(httpClient)
	self.SAMLProviderMappings.SetHttpClient(httpClient)
	self.SfsTurbos.SetHttpClient(httpClient)
	self.RouteTables.SetHttpClient(httpClient)
}

func (self *Client) InitWithAccessKey(endpoint, regionId, domainId, projectId, accessKey, secretKey string, debug bool) error {
	// accessKey signer
	credential := &credentials.AccessKeyCredential{
		AccessKeyId:     accessKey,
		AccessKeySecret: secretKey,
	}

	// 从signer中初始化
	signer, err := auth.NewSignerWithCredential(credential)
	if err != nil {
		return err
	}
	self.cfg = &SClientConfig{
		signer:    signer,
		endpoint:  endpoint,
		regionId:  regionId,
		domainId:  domainId,
		projectId: projectId,
		debug:     debug,
	}

	// 初始化 resource manager
	self.initManagers()
	return err
}

func (self *Client) initManagers() {
	if !self.init {
		self.Servers = modules.NewServerManager(self.cfg)
		self.ServersV2 = modules.NewServerV2Manager(self.cfg)
		self.NovaServers = modules.NewNovaServerManager(self.cfg)
		self.Snapshots = modules.NewSnapshotManager(self.cfg)
		self.OsSnapshots = modules.NewOsSnapshotManager(self.cfg)
		self.Images = modules.NewImageManager(self.cfg)
		self.OpenStackImages = modules.NewOpenstackImageManager(self.cfg)
		self.Projects = modules.NewProjectManager(self.cfg)
		self.Regions = modules.NewRegionManager(self.cfg)
		self.Zones = modules.NewZoneManager(self.cfg)
		self.Vpcs = modules.NewVpcManager(self.cfg)
		self.Elasticcache = modules.NewElasticcacheManager(self.cfg)
		self.DcsAvailableZone = modules.NewDcsAvailableZoneManager(self.cfg)
		self.Disks = modules.NewDiskManager(self.cfg)
		self.Domains = modules.NewDomainManager(self.cfg)
		self.Keypairs = modules.NewKeypairManager(self.cfg)
		self.Orders = modules.NewOrderManager(self.cfg)
		self.Subnets = modules.NewSubnetManager(self.cfg)
		self.Users = modules.NewUserManager(self.cfg)
		self.Interface = modules.NewInterfaceManager(self.cfg)
		self.Jobs = modules.NewJobManager(self.cfg)
		self.Credentials = modules.NewCredentialManager(self.cfg)
		self.Port = modules.NewPortManager(self.cfg)
		self.Flavors = modules.NewFlavorManager(self.cfg)
		self.VpcRoutes = modules.NewVpcRouteManager(self.cfg)
		self.SNatRules = modules.NewNatSManager(self.cfg)
		self.DNatRules = modules.NewNatDManager(self.cfg)
		self.NatGateways = modules.NewNatGatewayManager(self.cfg)
		self.VpcPeerings = modules.NewVpcPeeringManager(self.cfg)
		self.DBInstance = modules.NewDBInstanceManager(self.cfg)
		self.DBInstanceBackup = modules.NewDBInstanceBackupManager(self.cfg)
		self.DBInstanceFlavor = modules.NewDBInstanceFlavorManager(self.cfg)
		self.DBInstanceJob = modules.NewDBInstanceJobManager(self.cfg)
		self.Traces = modules.NewTraceManager(self.cfg)
		self.CloudEye = modules.NewCloudEyeManager(self.cfg)
		self.Quotas = modules.NewQuotaManager(self.cfg)
		self.EnterpriseProjects = modules.NewEnterpriseProjectManager(self.cfg)
		self.Roles = modules.NewRoleManager(self.cfg)
		self.Groups = modules.NewGroupManager(self.cfg)
		self.SAMLProviders = modules.NewSAMLProviderManager(self.cfg)
		self.SAMLProviderMappings = modules.NewSAMLProviderMappingManager(self.cfg)
		self.SfsTurbos = modules.NewSfsTurboManager(self.cfg)
		self.RouteTables = modules.NewRouteTableManager(self.cfg)
	}

	self.init = true
}

func NewClientWithAccessKey(endpoint, regionId, domainId, projectId, accessKey, secretKey string, debug bool) (*Client, error) {
	c := &Client{}
	err := c.InitWithAccessKey(endpoint, regionId, domainId, projectId, accessKey, secretKey, debug)
	return c, err
}

func NewPublicCloudClientWithAccessKey(regionId, domainId, projectId, accessKey, secretKey string, debug bool) (*Client, error) {
	return NewClientWithAccessKey("myhuaweicloud.com", regionId, domainId, projectId, accessKey, secretKey, debug)
}
