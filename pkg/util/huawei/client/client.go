package client

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth/credentials"
	"yunion.io/x/onecloud/pkg/util/huawei/client/modules"
)

type Client struct {
	signer    auth.Signer
	regionId  string
	domainId  string
	projectId string

	debug bool

	// 标记初始化状态
	init bool

	Balances           *modules.SBalanceManager
	Bandwidths         *modules.SBandwidthManager
	Disks              *modules.SDiskManager
	Domains            *modules.SDomainManager
	Eips               *modules.SEipManager
	Flavors            *modules.SFlavorManager
	Images             *modules.SImageManager
	OpenStackImages    *modules.SImageManager
	Interface          *modules.SInterfaceManager
	Jobs               *modules.SJobManager
	Keypairs           *modules.SKeypairManager
	Orders             *modules.SOrderManager
	Port               *modules.SPortManager
	Projects           *modules.SProjectManager
	Regions            *modules.SRegionManager
	SecurityGroupRules *modules.SSecgroupRuleManager
	SecurityGroups     *modules.SSecurityGroupManager
	NovaSecurityGroups *modules.SSecurityGroupManager
	Servers            *modules.SServerManager
	NovaServers        *modules.SServerManager
	Snapshots          *modules.SSnapshotManager
	OsSnapshots        *modules.SSnapshotManager
	Subnets            *modules.SSubnetManager
	Users              *modules.SUserManager
	Vpcs               *modules.SVpcManager
	Zones              *modules.SZoneManager
}

func (self *Client) Init() error {
	// 从环境变量中初始化client
	return nil
}

func (self *Client) InitWithOptions(regionId, projectId string, credential auth.Credential) error {
	// 从signer中初始化
	signer, err := auth.NewSignerWithCredential(credential)
	if err != nil {
		return err
	}
	self.signer = signer
	self.regionId = regionId
	self.projectId = projectId
	// 暂时还未用到domainId
	self.domainId = ""
	// 初始化 resource manager
	self.initManagers()
	return err
}

func (self *Client) InitWithAccessKey(regionId, projectId, accessKey, secretKey string) error {
	// accessKey signer
	credential := &credentials.AccessKeyCredential{
		AccessKeyId:     accessKey,
		AccessKeySecret: secretKey,
	}

	return self.InitWithOptions(regionId, projectId, credential)
}

func (self *Client) initManagers() {
	if !self.init {
		self.Servers = modules.NewServerManager(self.regionId, self.projectId, self.signer, self.debug)
		self.NovaServers = modules.NewNovaServerManager(self.regionId, self.projectId, self.signer, self.debug)
		self.Snapshots = modules.NewSnapshotManager(self.regionId, self.projectId, self.signer, self.debug)
		self.OsSnapshots = modules.NewOsSnapshotManager(self.regionId, self.projectId, self.signer, self.debug)
		self.Images = modules.NewImageManager(self.regionId, self.projectId, self.signer, self.debug)
		self.OpenStackImages = modules.NewOpenstackImageManager(self.regionId, self.signer, self.debug)
		self.Projects = modules.NewProjectManager(self.signer, self.debug)
		self.Regions = modules.NewRegionManager(self.signer, self.debug)
		self.Zones = modules.NewZoneManager(self.regionId, self.projectId, self.signer, self.debug)
		self.Vpcs = modules.NewVpcManager(self.regionId, self.projectId, self.signer, self.debug)
		self.Eips = modules.NewEipManager(self.regionId, self.projectId, self.signer, self.debug)
		self.Disks = modules.NewDiskManager(self.regionId, self.projectId, self.signer, self.debug)
		self.Domains = modules.NewDomainManager(self.signer, self.debug)
		self.Keypairs = modules.NewKeypairManager(self.regionId, self.projectId, self.signer, self.debug)
		self.Orders = modules.NewOrderManager(self.regionId, self.signer, self.debug)
		self.SecurityGroupRules = modules.NewSecgroupRuleManager(self.regionId, self.projectId, self.signer, self.debug)
		self.SecurityGroups = modules.NewSecurityGroupManager(self.regionId, self.projectId, self.signer, self.debug)
		self.NovaSecurityGroups = modules.NewNovaSecurityGroupManager(self.regionId, self.projectId, self.signer, self.debug)
		self.Subnets = modules.NewSubnetManager(self.regionId, self.projectId, self.signer, self.debug)
		self.Users = modules.NewUserManager(self.signer, self.debug)
		self.Interface = modules.NewInterfaceManager(self.regionId, self.projectId, self.signer, self.debug)
		self.Jobs = modules.NewJobManager(self.regionId, self.projectId, self.signer, self.debug)
		self.Balances = modules.NewBalanceManager(self.signer, self.debug)
		self.Bandwidths = modules.NewBandwidthManager(self.regionId, self.projectId, self.signer, self.debug)
		self.Port = modules.NewPortManager(self.regionId, self.projectId, self.signer, self.debug)
		self.Flavors = modules.NewFlavorManager(self.regionId, self.projectId, self.signer, self.debug)
	}

	self.init = true
}

// todo: init from envrioment
func NewClient() (*Client, error) {
	return nil, nil
}

func NewClientWithAccessKey(regionId, projectId, accessKey, secretKey string, debug bool) (*Client, error) {
	c := &Client{debug: debug}
	err := c.InitWithAccessKey(regionId, projectId, accessKey, secretKey)
	return c, err
}
