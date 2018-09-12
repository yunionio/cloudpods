package aliyun

import (
	"time"
	"yunion.io/x/log"
	"fmt"

	"yunion.io/x/pkg/utils"
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type TInternetChargeType string

const (
	InternetChargeByTraffic = TInternetChargeType("PayByTraffic")
	InternetChargeByBandwidth = TInternetChargeType("PayByBandwidth")
)

const (
	EIP_STATUS_ASSOCIATING = "Associating"
	EIP_STATUS_UNASSOCIATING = "Unassociating"
	EIP_STATUS_INUSE = "InUse"
	EIP_STATUS_AVAILABLE = "Available"

	EIP_OPERATION_LOCK_FINANCIAL = "financial"
	EIP_OPERATION_LOCK_SECURITY  = "security"


	EIP_INSTANCE_TYPE_ECS = "EcsInstance" // （默认值）：VPC类型的ECS实例
	EIP_INTANNCE_TYPE_SLB = "SlbInstance" // ：VPC类型的SLB实例
	EIP_INSTANCE_TYPE_NAT = "Nat" // ：NAT网关
	EIP_INSTANCE_TYPE_HAVIP = "HaVip" // ：HAVIP
)

type SEipAddress struct {
	region *SRegion

	AllocationId       string

	InternetChargeType string

	IpAddress          string
	Status string

	InstanceType string
	InstanceId string
	Bandwidth int /* Mbps */

	AllocationTime time.Time

	OperationLocks string
}

func (self *SEipAddress) GetId() string {
	return self.AllocationId
}

func (self *SEipAddress) GetName() string {
	return self.IpAddress
}

func (self *SEipAddress) GetGlobalId() string {
	return self.AllocationId
}

func (self *SEipAddress) GetStatus() string {
	switch self.Status {
	case EIP_STATUS_AVAILABLE, EIP_STATUS_INUSE:
		return models.EIP_STATUS_READY
	case EIP_STATUS_ASSOCIATING:
		return models.EIP_STATUS_ASSOCIATE
	case EIP_STATUS_UNASSOCIATING:
		return models.EIP_STATUS_DISSOCIATE
	default:
		return models.EIP_STATUS_UNKNOWN
	}
}

func (self *SEipAddress) Refresh() error {
	if self.IsEmulated() {
		return nil
	}
	new, err := self.region.GetEip(self.AllocationId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SEipAddress) IsEmulated() bool {
	if self.AllocationId == self.InstanceId {
		// fixed Public IP
		return true
	} else {
		return false
	}
}

func (self *SEipAddress) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SEipAddress) GetIpAddr() string {
	return self.IpAddress
}

func (self *SEipAddress) GetMode() string {
	if self.InstanceId == self.AllocationId {
		return models.EIP_MODE_INSTANCE_PUBLICIP
	} else {
		return models.EIP_MODE_STANDALONE_EIP
	}
}

func (self *SEipAddress) GetAssociationType() string {
	switch self.InstanceType {
	case EIP_INSTANCE_TYPE_ECS:
		return "server"
	default:
		log.Fatalf("unsupported type: %s", self.InstanceType)
		return "unsupported"
	}
}

func (self *SEipAddress) GetAssociationExternalId() string {
	return self.InstanceId
}

func (self *SEipAddress) GetManagerId() string {
	return self.region.client.providerId
}

func (self *SEipAddress) Delete() error {
	return self.region.DeallocateEIP(self.AllocationId)
}

func (self *SEipAddress) GetBandwidth() int {
	return self.Bandwidth
}

func (self *SEipAddress) GetInternetChargeType() string {
	switch self.InternetChargeType {
	case string(InternetChargeByTraffic):
		return models.EIP_CHARGE_TYPE_BY_TRAFFIC
	case string(InternetChargeByBandwidth):
		return models.EIP_CHARGE_TYPE_BY_BANDWIDTH
	default:
		return models.EIP_CHARGE_TYPE_BY_TRAFFIC
	}
}

func (self *SEipAddress) Associate(instanceId string) error {
	err := self.region.AssociateEip(self.AllocationId, instanceId)
	if err != nil {
		return err
	}
	err = cloudprovider.WaitStatus(self, models.EIP_STATUS_READY, 10*time.Second, 180*time.Second)
	return err
}

func (self *SEipAddress) Dissociate() error {
	err := self.region.DissociateEip(self.AllocationId, self.InstanceId)
	if err != nil {
		return err
	}
	err = cloudprovider.WaitStatus(self, models.EIP_STATUS_READY, 10*time.Second, 180*time.Second)
	return err
}

func (self *SEipAddress) ChangeBandwidth(bw int) error {
	return self.region.UpdateEipBandwidth(self.AllocationId, bw)
}

func (region *SRegion) GetEips(eipId string, offset int, limit int) ([]SEipAddress, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}

	params := make(map[string]string)
	params["RegionId"] = region.RegionId
	params["PageSize"] = fmt.Sprintf("%d", limit)
	params["PageNumber"] = fmt.Sprintf("%d", (offset/limit)+1)

	if len(eipId) > 0 {
		params["AllocationId"] = eipId
	}

	body, err := region.ecsRequest("DescribeEipAddresses", params)
	if err != nil {
		log.Errorf("DescribeEipAddresses fail %s", err)
		return nil, 0, err
	}

	// log.Errorf("%s", body)

	eips := make([]SEipAddress, 0)
	err = body.Unmarshal(&eips, "EipAddresses", "EipAddress")
	if err != nil {
		log.Errorf("Unmarshal EipAddress details fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Int("TotalCount")
	for i := 0; i < len(eips); i += 1 {
		eips[i].region = region
	}
	return eips, int(total), nil
}

func (region *SRegion) GetEip(eipId string) (*SEipAddress, error) {
	eips, total, err := region.GetEips(eipId, 0, 1)
	if err != nil {
		return nil, err
	}
	if total != 1 {
		return nil, cloudprovider.ErrNotFound
	}
	return &eips[0], nil
}

func (region *SRegion) AllocateEIP(bwMbps int, chargeType TInternetChargeType) (*SEipAddress, error) {
	params := make(map[string]string)

	params["Bandwidth"] = fmt.Sprintf("%d", bwMbps)
	params["InternetChargeType"] = string(chargeType)
	params["InstanceChargeType"] = "PostPaid"
	params["ClientToken"] = utils.GenRequestId(20)

	body, err := region.ecsRequest("AllocateEipAddress", params)
	if err != nil {
		log.Errorf("AllocateEipAddress fail %s", err)
		return nil, err
	}

	eipId, err := body.GetString("AllocationId")
	if err != nil {
		log.Errorf("fail to get AllocationId after EIP allocation??? %s", err)
		return nil, err
	}

	return region.GetEip(eipId)
}

func (region *SRegion) CreateEIP(bwMbps int, chargeType string) (cloudprovider.ICloudEIP, error) {
	var ctype TInternetChargeType
	switch chargeType {
	case models.EIP_CHARGE_TYPE_BY_TRAFFIC:
		ctype = InternetChargeByTraffic
	case models.EIP_CHARGE_TYPE_BY_BANDWIDTH:
		ctype = InternetChargeByBandwidth
	}
	eip, err := region.AllocateEIP(bwMbps, ctype)
	return eip, err
}

func (region *SRegion) DeallocateEIP(eipId string) (error) {
	params := make(map[string]string)
	params["AllocationId"] = eipId

	_, err := region.ecsRequest("ReleaseEipAddress", params)
	if err != nil {
		log.Errorf("ReleaseEipAddress fail %s", err)
	}
	return err
}

func (region *SRegion) AssociateEip(eipId string, instanceId string) (error) {
	params := make(map[string]string)
	params["AllocationId"] = eipId
	params["InstanceId"] = instanceId

	_, err := region.ecsRequest("AssociateEipAddress", params)
	if err != nil {
		log.Errorf("AssociateEipAddress fail %s", err)
	}
	return err
}

func (region *SRegion) DissociateEip(eipId string, instanceId string) (error) {
	params := make(map[string]string)
	params["AllocationId"] = eipId
	params["InstanceId"] = instanceId

	_, err := region.ecsRequest("UnassociateEipAddress", params)
	if err != nil {
		log.Errorf("UnassociateEipAddress fail %s", err)
	}
	return err
}

func (region *SRegion) UpdateEipBandwidth(eipId string, bw int) error {
	params := make(map[string]string)
	params["AllocationId"] = eipId
	params["Bandwidth"] = fmt.Sprintf("%d", bw)

	_, err := region.ecsRequest("ModifyEipAddressAttribute", params)
	if err != nil {
		log.Errorf("ModifyEipAddressAttribute fail %s", err)
	}
	return err
}