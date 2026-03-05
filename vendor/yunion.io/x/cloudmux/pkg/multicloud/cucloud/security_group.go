package cucloud

import (
	"net/url"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SSecurityGroup struct {
	multicloud.SSecurityGroup
	multicloud.STagBase

	region *SRegion

	ResourceGroupId            string
	ResourceGroupName          string
	BigRegionName              string
	SecurityGroupFlag          string
	CloudBrandCN               string
	ProductName                string
	BigRegionCode              string
	SecurityGroupId            string
	RegionUUID                 string
	CloudType                  string
	CloudId                    string
	ZoneId                     string
	StatusEn                   string
	BigRegionId                string
	NetCardNum                 int
	SecurityGroupRuleNumber    int
	AccountId                  string
	IsDefault                  string
	RegionOssCode              string
	RegionModel                string
	Status                     string
	CloudRegionId              string
	AccountName                string
	TagID                      []interface{}
	RegionName                 string
	SiteName                   string
	Description                string
	CloudBrand                 string
	CloudFlag                  string
	RegionCode                 string
	InstanceId                 string
	SecurityGroupUuid          string
	ZoneName                   string
	ProductNo                  string
	CloudRegionName            string
	SecurityGroupName          string
	SiteCode                   string
	RegionOssName              string
	RemoteSecurityGroupRuleNum int
	RegionVersion              string
	UserID                     string
	CloudRegionCode            string
	CloudRegionNature          string
	InstanceRelationNum        int
	RegionId                   string
	CreateTime                 string
	CloudName                  string
	SiteID                     string
	SecurityGroupRules         []struct {
		SecurityGroupRuleUUID string
		Ethertype             string
		Description           string
		UpdateTime            string
		PortRangeMax          string
		Priority              string
		RemoteIPPrefix        string
		SecurityGroupRuleId   string
		SecurityGroupId       string
		AccountId             string
		Protocol              string
		PortRangeMin          string
		CreateTime            string
		RuleName              string
		Action                string
		Direction             string
	}
}

func (self *SSecurityGroup) GetVpcId() string {
	return ""
}

func (self *SSecurityGroup) GetId() string {
	return self.SecurityGroupId
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.SecurityGroupId
}

func (self *SSecurityGroup) GetDescription() string {
	return self.Description
}

func (self *SSecurityGroup) GetName() string {
	if len(self.SecurityGroupName) > 0 {
		return self.SecurityGroupName
	}
	return self.SecurityGroupId
}

func (self *SSecurityGroup) GetStatus() string {
	return api.SECGROUP_STATUS_READY
}

func (self *SSecurityGroup) Refresh() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SSecurityGroup) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SSecurityGroup) GetRules() ([]cloudprovider.ISecurityGroupRule, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SSecurityGroup) CreateRule(opts *cloudprovider.SecurityGroupRuleCreateOptions) (cloudprovider.ISecurityGroupRule, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetSecurityGroups(id string) ([]SSecurityGroup, error) {
	params := url.Values{}
	params.Set("cloudRegionCode", region.CloudRegionCode)
	if len(id) > 0 {
		params.Set("securityGroupId", id)
	}
	resp, err := region.list("/instance/v1/product/securitygroup", params)
	if err != nil {
		return nil, err
	}
	ret := []SSecurityGroup{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (region *SRegion) GetSecurityGroup(id string) (*SSecurityGroup, error) {
	groups, err := region.GetSecurityGroups(id)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if groups[i].SecurityGroupId == id {
			groups[i].region = region
			return &groups[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}
