package clients

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

	Bandwidths         *modules.SBandwidthManager
	Disks              *modules.SDiskManager
	Eips               *modules.SEipManager
	Images             *modules.SImageManager
	Interface          *modules.SInterfaceManager
	Keypairs           *modules.SKeypairManager
	Port               *modules.SPortManager
	Projects           *modules.SProjectManager
	Regions            *modules.SRegionManager
	SecurityGroupRules *modules.SSecgroupRuleManager
	SecurityGroups     *modules.SSecurityGroupManager
	Servers            *modules.SServerManager
	Snapshots          *modules.SSnapshotManager
	Subnets            *modules.SSubnetManager
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
	if self.Servers == nil {
		self.Servers = modules.NewServerManager(self.regionId, self.projectId, self.signer)
	}

	if self.Snapshots == nil {
		self.Snapshots = modules.NewSnapshotManager(self.regionId, self.projectId, self.signer)
	}

	if self.Images == nil {
		self.Images = modules.NewImageManager(self.regionId, self.signer)
	}

	if self.Projects == nil {
		self.Projects = modules.NewProjectManager(self.signer)
	}

	if self.Regions == nil {
		self.Regions = modules.NewRegionManager(self.signer)
	}

	if self.Zones == nil {
		self.Zones = modules.NewZoneManager(self.regionId, self.projectId, self.signer)
	}

	if self.Vpcs == nil {
		self.Vpcs = modules.NewVpcManager(self.regionId, self.projectId, self.signer)
	}

	if self.Eips == nil {
		self.Eips = modules.NewEipManager(self.regionId, self.projectId, self.signer)
	}

	if self.Disks == nil {
		self.Disks = modules.NewDiskManager(self.regionId, self.projectId, self.signer)
	}

	if self.Keypairs == nil {
		self.Keypairs = modules.NewKeypairManager(self.regionId, self.projectId, self.signer)
	}

	if self.SecurityGroupRules == nil {
		self.SecurityGroupRules = modules.NewSecgroupRuleManager(self.regionId, self.projectId, self.signer)
	}

	if self.SecurityGroups == nil {
		self.SecurityGroups = modules.NewSecurityGroupManager(self.regionId, self.projectId, self.signer)
	}

	if self.Subnets == nil {
		self.Subnets = modules.NewSubnetManager(self.regionId, self.projectId, self.signer)
	}

	if self.Interface == nil {
		self.Interface = modules.NewInterfaceManager(self.regionId, self.projectId, self.signer)
	}

	if self.Bandwidths == nil {
		self.Bandwidths = modules.NewBandwidthManager(self.regionId, self.projectId, self.signer)
	}

	if self.Port == nil {
		self.Port = modules.NewPortManager(self.regionId, self.signer)
	}
}

// todo: init from envrioment
func NewClient() (*Client, error) {
	return nil, nil
}

func NewClientWithAccessKey(regionId, projectId, accessKey, secretKey string) (*Client, error) {
	c := &Client{}
	err := c.InitWithAccessKey(regionId, projectId, accessKey, secretKey)
	return c, err
}
