package qcloud

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type TInternetChargeType string

const (
	InternetChargeByTraffic   = TInternetChargeType("PayByTraffic")
	InternetChargeByBandwidth = TInternetChargeType("PayByBandwidth")
)

const (
	EIP_STATUS_ASSOCIATING   = "Associating"
	EIP_STATUS_UNASSOCIATING = "Unassociating"
	EIP_STATUS_INUSE         = "InUse"
	EIP_STATUS_AVAILABLE     = "Available"

	EIP_OPERATION_LOCK_FINANCIAL = "financial"
	EIP_OPERATION_LOCK_SECURITY  = "security"

	EIP_INSTANCE_TYPE_ECS   = "EcsInstance" // （默认值）：VPC类型的ECS实例
	EIP_INTANNCE_TYPE_SLB   = "SlbInstance" // ：VPC类型的SLB实例
	EIP_INSTANCE_TYPE_NAT   = "Nat"         // ：NAT网关
	EIP_INSTANCE_TYPE_HAVIP = "HaVip"       // ：HAVIP
)

type SEipAddress struct {
	region *SRegion

	AddressId             string    //	EIP的ID，是EIP的唯一标识。
	AddressName           string    //	EIP名称。
	AddressStatus         string    //	EIP状态。
	AddressIp             string    //	外网IP地址
	InstanceId            string    //	绑定的资源实例ID。可能是一个CVM，NAT。
	CreatedTime           time.Time //	创建时间。按照ISO8601标准表示，并且使用UTC时间。格式为：YYYY-MM-DDThh:mm:ssZ。
	NetworkInterfaceId    string    //	绑定的弹性网卡ID
	PrivateAddressIp      string    //	绑定的资源内网ip
	IsArrears             bool      //	资源隔离状态。true表示eip处于隔离状态，false表示资源处于未隔离装填
	IsBlocked             bool      //	资源封堵状态。true表示eip处于封堵状态，false表示eip处于未封堵状态
	IsEipDirectConnection bool      //	eip是否支持直通模式。true表示eip支持直通模式，false表示资源不支持直通模式
	AddressType           string    //	eip资源类型，包括"CalcIP","WanIP","EIP","AnycastEIP"。其中"CalcIP"表示设备ip，“WanIP”表示普通公网ip，“EIP”表示弹性公网ip，“AnycastEip”表示加速EIP
	CascadeRelease        bool      //	eip是否在解绑后自动释放。true表示eip将会在解绑后自动释放，false表示eip在解绑后不会自动释放
}

func (self *SEipAddress) GetId() string {
	return self.AddressId
}

func (self *SEipAddress) GetName() string {
	return self.AddressName
}

func (self *SEipAddress) GetGlobalId() string {
	return self.AddressId
}

func (self *SEipAddress) GetStatus() string {
	switch self.AddressStatus {
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
	new, err := self.region.GetEip(self.AddressId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SEipAddress) IsEmulated() bool {
	if self.AddressId == self.InstanceId {
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
	return self.AddressIp
}

func (self *SEipAddress) GetMode() string {
	if self.InstanceId == self.AddressId {
		return models.EIP_MODE_INSTANCE_PUBLICIP
	}
	return models.EIP_MODE_STANDALONE_EIP
}

func (self *SEipAddress) GetAssociationType() string {
	switch self.AddressType {
	case "EIP", "AnycastEIP", "WanIP":
		return "server"
	case "CalcIP":
		return "server"
	default:
		log.Fatalf("unsupported type: %s", self.AddressType)
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
	return self.region.DeallocateEIP(self.AddressId)
}

func (self *SEipAddress) GetBandwidth() int {
	return 100
	//return self.Bandwidth
}

func (self *SEipAddress) GetInternetChargeType() string {
	// switch self.InternetChargeType {
	// case string(InternetChargeByTraffic):
	// 	return models.EIP_CHARGE_TYPE_BY_TRAFFIC
	// case string(InternetChargeByBandwidth):
	// 	return models.EIP_CHARGE_TYPE_BY_BANDWIDTH
	// default:
	// 	return models.EIP_CHARGE_TYPE_BY_TRAFFIC
	// }
	return "unkonw"
}

func (self *SEipAddress) Associate(instanceId string) error {
	err := self.region.AssociateEip(self.AddressId, instanceId)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, models.EIP_STATUS_READY, 10*time.Second, 180*time.Second)
}

func (self *SEipAddress) Dissociate() error {
	err := self.region.DissociateEip(self.AddressId)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, models.EIP_STATUS_READY, 10*time.Second, 180*time.Second)
}

func (self *SEipAddress) ChangeBandwidth(bw int) error {
	return self.region.UpdateEipBandwidth(self.AddressId, bw)
}

func (region *SRegion) GetEips(eipId string, offset int, limit int) ([]SEipAddress, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}

	params := make(map[string]string)
	params["Limit"] = fmt.Sprintf("%d", limit)
	params["Offset"] = fmt.Sprintf("%d", offset)

	if len(eipId) > 0 {
		params["AddressIds.0"] = eipId
	}

	body, err := region.vpcRequest("DescribeAddresses", params)
	if err != nil {
		log.Errorf("DescribeEipAddresses fail %s", err)
		return nil, 0, err
	}

	eips := make([]SEipAddress, 0)
	err = body.Unmarshal(&eips, "AddressSet")
	if err != nil {
		log.Errorf("Unmarshal EipAddress details fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Int("TotalCount")
	for i := 0; i < len(eips); i++ {
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

func (region *SRegion) AllocateEIP(name string, bwMbps int, chargeType TInternetChargeType) (*SEipAddress, error) {
	params := make(map[string]string)
	params["Region"] = region.Region
	addRessSet := []string{}
	body, err := region.vpcRequest("AllocateAddresses", params)
	if err != nil {
		return nil, err
	}
	if err := body.Unmarshal(&addRessSet, "AddressSet"); err == nil && len(addRessSet) > 0 {
		params["AddressId"] = addRessSet[0]
		params["AddressName"] = name
		if _, err := region.vpcRequest("ModifyAddressAttribute", params); err != nil {
			return nil, err
		} else if err := region.UpdateEipBandwidth(addRessSet[0], bwMbps); err != nil {
			return nil, err
		}
		return region.GetEip(addRessSet[0])
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) CreateEIP(name string, bwMbps int, chargeType string) (cloudprovider.ICloudEIP, error) {
	var ctype TInternetChargeType
	switch chargeType {
	case models.EIP_CHARGE_TYPE_BY_TRAFFIC:
		ctype = InternetChargeByTraffic
	case models.EIP_CHARGE_TYPE_BY_BANDWIDTH:
		ctype = InternetChargeByBandwidth
	}
	return region.AllocateEIP(name, bwMbps, ctype)
}

func (region *SRegion) DeallocateEIP(eipId string) error {
	params := make(map[string]string)
	params["Region"] = region.Region
	params["AddressIds.0"] = eipId

	_, err := region.vpcRequest("ReleaseAddresses", params)
	if err != nil {
		log.Errorf("ReleaseAddresses fail %s", err)
	}
	return err
}

func (region *SRegion) AssociateEip(eipId string, instanceId string) error {
	params := make(map[string]string)
	params["AllocationId"] = eipId
	params["InstanceId"] = instanceId

	_, err := region.vpcRequest("AssociateEipAddress", params)
	if err != nil {
		log.Errorf("AssociateEipAddress fail %s", err)
	}
	return err
}

func (region *SRegion) DissociateEip(eipId string) error {
	params := make(map[string]string)
	params["Region"] = region.Region
	params["AddressId"] = eipId

	_, err := region.vpcRequest("DisassociateAddress", params)
	if err != nil {
		log.Errorf("UnassociateEipAddress fail %s", err)
	}
	return err
}

func (region *SRegion) UpdateEipBandwidth(eipId string, bw int) error {
	// params := make(map[string]string)
	// params["Region"] = region.Region
	// params["AddressIds.0"] = eipId
	// params["InternetMaxBandwidthOut"] = fmt.Sprintf("%d", bw)

	// _, err := region.vpcRequest("ModifyAddressesBandwidth", params)
	// return err
	// 腾讯云这个接口目前有问题
	return nil
}
