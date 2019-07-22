package huawei

import (
	"time"

	"github.com/pkg/errors"
	"yunion.io/x/log"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423020.html
type SElasticcache struct {
	multicloud.SElasticcacheBase

	region *SRegion

	Name                  string   `json:"name"`
	Engine                string   `json:"engine"`
	CapacityGB            int      `json:"capacity"`
	IP                    string   `json:"ip"`
	DomainName            string   `json:"domainName"`
	Port                  int      `json:"port"`
	Status                string   `json:"status"`
	Libos                 bool     `json:"libos"`
	Description           string   `json:"description"`
	Task                  string   `json:"task"`
	MaxMemoryMB           int      `json:"max_memory"`
	UsedMemoryMB          int      `json:"used_memory"`
	InstanceID            string   `json:"instance_id"`
	ResourceSpecCode      string   `json:"resource_spec_code"`
	EngineVersion         string   `json:"engine_version"`
	InternalVersion       string   `json:"internal_version"`
	ChargingMode          int      `json:"charging_mode"`
	CapacityMinor         string   `json:"capacity_minor"`
	VpcID                 string   `json:"vpc_id"`
	VpcName               string   `json:"vpc_name"`
	TaskStatus            string   `json:"task_status"`
	CreatedAt             string   `json:"created_at"`
	ErrorCode             string   `json:"error_code"`
	UserID                string   `json:"user_id"`
	UserName              string   `json:"user_name"`
	MaintainBegin         string   `json:"maintain_begin"`
	MaintainEnd           string   `json:"maintain_end"`
	NoPasswordAccess      string   `json:"no_password_access"`
	AccessUser            string   `json:"access_user"`
	EnablePublicip        bool     `json:"enable_publicip"`
	PublicipID            string   `json:"publicip_id"`
	PublicipAddress       string   `json:"publicip_address"`
	EnableSSL             bool     `json:"enable_ssl"`
	ServiceUpgrade        bool     `json:"service_upgrade"`
	ServiceTaskID         string   `json:"service_task_id"`
	IsFree                string   `json:"is_free"`
	EnterpriseProjectID   string   `json:"enterprise_project_id"`
	AvailableZones        []string `json:"available_zones"`
	SubnetID              string   `json:"subnet_id"`
	SecurityGroupID       string   `json:"security_group_id"`
	BackendAddrs          []string `json:"backend_addrs"`
	ProductID             string   `json:"product_id"`
	SecurityGroupName     string   `json:"security_group_name"`
	SubnetName            string   `json:"subnet_name"`
	OrderID               string   `json:"order_id"`
	SubnetCIDR            string   `json:"subnet_cidr"`
	InstanceBackupPolicy  string   `json:"instance_backup_policy"`
	EnterpriseProjectName string   `json:"enterprise_project_name"`
}

func (self *SElasticcache) GetId() string {
	return self.InstanceID
}

func (self *SElasticcache) GetName() string {
	return self.Name
}

func (self *SElasticcache) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticcache) GetStatus() string {
	return self.Status
}

func (self *SElasticcache) GetBillingType() string {
	// charging_mode “0”：按需计费    “1”：按包年包月计费
	if self.ChargingMode == 1 {
		return billing_api.BILLING_TYPE_PREPAID
	} else {
		return billing_api.BILLING_TYPE_POSTPAID
	}
}

func (self *SElasticcache) GetCreatedAt() time.Time {
	var createtime time.Time
	if len(self.CreatedAt) > 0 {
		createtime, _ = time.Parse("2006-01-02T15:04:05.000Z", self.CreatedAt)
	}

	return createtime
}

func (self *SElasticcache) GetExpiredAt() time.Time {
	var expiredTime time.Time
	if self.ChargingMode == 1 {
		res, err := self.region.GetOrderResourceDetail(self.GetId())
		if err != nil {
			log.Debugln(err)
		}

		expiredTime = res.ExpireTime
	}

	return expiredTime
}

func (self *SElasticcache) GetInstanceType() string {
	// todo: ??
	return self.ResourceSpecCode
}

func (self *SElasticcache) GetCapacityMB() int {
	return self.CapacityGB * 1024
}

func (self *SElasticcache) GetArchType() string {
	/*
	   资源规格标识。

	   dcs.single_node：表示实例类型为单机
	   dcs.master_standby：表示实例类型为主备
	   dcs.cluster：表示实例类型为集群
	*/

	return ""
}

func (self *SElasticcache) GetNodeType() string {
	return ""
}

func (self *SElasticcache) GetEngine() string {
	return self.Engine
}

func (self *SElasticcache) GetEngineVersion() string {
	return self.EngineVersion
}

func (self *SElasticcache) GetVpcId() string {
	return self.VpcID
}

func (self *SElasticcache) GetZoneId() string {
	if len(self.AvailableZones) > 0 {
		return self.AvailableZones[0]
	}

	return ""
}

func (self *SElasticcache) GetNetworkType() string {
	return api.LB_NETWORK_TYPE_VPC
}

func (self *SElasticcache) GetNetworkId() string {
	return self.SubnetID
}

func (self *SElasticcache) GetPrivateDNS() string {
	return self.DomainName
}

func (self *SElasticcache) GetPrivateIpAddr() string {
	return self.IP
}

func (self *SElasticcache) GetPrivateConnectPort() int {
	return self.Port
}

func (self *SElasticcache) GetPublicDNS() string {
	return self.PublicipAddress
}

func (self *SElasticcache) GetPublicIpAddr() string {
	return self.PublicipAddress
}

func (self *SElasticcache) GetPublicConnectPort() int {
	return self.Port
}

func (self *SElasticcache) GetMaintainStartTime() string {
	return self.MaintainBegin
}

func (self *SElasticcache) GetMaintainEndTime() string {
	return self.MaintainEnd
}

func (self *SElasticcache) GetICloudElasticcacheAccounts() ([]cloudprovider.ICloudElasticcacheAccount, error) {
	iaccounts := []cloudprovider.ICloudElasticcacheAccount{}
	if len(self.AccessUser) > 0 {
		iaccount := &SElasticcacheAccount{cacheDB: self}
		iaccounts = append(iaccounts, iaccount)
	}

	return iaccounts, nil
}

func (self *SElasticcache) GetICloudElasticcacheAcls() ([]cloudprovider.ICloudElasticcacheAcl, error) {
	// 华为云使用安全组做访问控制。目前未支持
	return []cloudprovider.ICloudElasticcacheAcl{}, nil
}

func (self *SElasticcache) GetICloudElasticcacheBackups() ([]cloudprovider.ICloudElasticcacheBackup, error) {
	start := self.GetCreatedAt().Format("20060102150405")
	end := time.Now().Format("20060102150405")
	backups, err := self.region.GetElasticCacheBackups(self.GetId(), start, end)
	if err != nil {
		return nil, err
	}

	ibackups := make([]cloudprovider.ICloudElasticcacheBackup, len(backups))
	for i := range backups {
		backups[i].cacheDB = self
		ibackups[i] = &backups[i]
	}

	return ibackups, nil
}

func (self *SElasticcache) GetICloudElasticcacheParameters() ([]cloudprovider.ICloudElasticcacheParameter, error) {
	parameters, err := self.region.GetElasticCacheParameters(self.GetId())
	if err != nil {
		return nil, err
	}

	iparameters := make([]cloudprovider.ICloudElasticcacheParameter, len(parameters))
	for i := range parameters {
		parameters[i].cacheDB = self
		iparameters[i] = &parameters[i]
	}

	return iparameters, nil
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423035.html
func (self *SRegion) GetElasticCacheBackups(instanceId, startTime, endTime string) ([]SElasticcacheBackup, error) {
	params := make(map[string]string)
	params["instance_id"] = instanceId
	params["beginTime"] = startTime
	params["endTime"] = endTime

	backups := make([]SElasticcacheBackup, 0)
	err := doListAll(self.ecsClient.Elasticcache.ListBackups, params, &backups)
	if err != nil {
		return nil, err
	}

	return backups, nil
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423027.html
func (self *SRegion) GetElasticCacheParameters(instanceId string) ([]SElasticcacheParameter, error) {
	params := make(map[string]string)
	params["instance_id"] = instanceId

	parameters := make([]SElasticcacheParameter, 0)
	err := doListAll(self.ecsClient.Elasticcache.ListParameters, params, &parameters)
	if err != nil {
		return nil, err
	}

	return parameters, nil
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423044.html
func (self *SRegion) GetElasticCaches() ([]SElasticcache, error) {
	params := make(map[string]string)
	caches := make([]SElasticcache, 0)
	err := doListAll(self.ecsClient.Elasticcache.List, params, &caches)
	if err != nil {
		return nil, errors.Wrap(err, "region.GetElasticCaches")
	}

	for i := range caches {
		cache, err := self.GetElasticCache(caches[i].GetId())
		if err != nil {
			return nil, err
		} else {
			caches[i] = *cache
		}

		caches[i].region = self
	}

	return caches, nil
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423020.html
func (self *SRegion) GetElasticCache(instanceId string) (*SElasticcache, error) {
	cache := SElasticcache{}
	err := DoGet(self.ecsClient.Elasticcache.Get, instanceId, nil, &cache)
	if err != nil {
		return nil, errors.Wrapf(err, "region.GetElasticCache %s", instanceId)
	}

	cache.region = self
	return &cache, nil
}
