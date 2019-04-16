package ucloud

import (
	"time"

	"yunion.io/x/log"

	"yunion.io/x/jsonutils"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

// https://docs.ucloud.cn/api/unet-api/describe_eip
type SEip struct {
	region *SRegion

	BandwidthMb       int               `json:"Bandwidth"`
	BandwidthType     int               `json:"BandwidthType"`
	ChargeType        string            `json:"ChargeType"`
	CreateTime        int64             `json:"CreateTime"`
	EIPAddr           []EIPAddr         `json:"EIPAddr"`
	EIPID             string            `json:"EIPId"`
	Expire            bool              `json:"Expire"`
	ExpireTime        int64             `json:"ExpireTime"`
	Name              string            `json:"Name"`
	PayMode           string            `json:"PayMode"`
	Remark            string            `json:"Remark"`
	Resource          Resource          `json:"Resource"`
	ShareBandwidthSet ShareBandwidthSet `json:"ShareBandwidthSet"`
	Status            string            `json:"Status"`
	Tag               string            `json:"Tag"`
	Weight            int               `json:"Weight"`
}

func (self *SEip) GetProjectId() string {
	return self.region.client.projectId
}

type EIPAddr struct {
	IP           string `json:"IP"`
	OperatorName string `json:"OperatorName"`
}

type Resource struct {
	ResourceID   string `json:"ResourceID"`
	ResourceName string `json:"ResourceName"`
	ResourceType string `json:"ResourceType"`
	Zone         string `json:"Zone"`
}

type ShareBandwidthSet struct {
	ShareBandwidth     int    `json:"ShareBandwidth"`
	ShareBandwidthID   string `json:"ShareBandwidthId"`
	ShareBandwidthName string `json:"ShareBandwidthName"`
}

func (self *SEip) GetId() string {
	return self.EIPID
}

func (self *SEip) GetName() string {
	if len(self.Name) == 0 {
		return self.GetId()
	}

	return self.Name
}

func (self *SEip) GetGlobalId() string {
	return self.GetId()
}

// 弹性IP的资源绑定状态, 枚举值为: used: 已绑定, free: 未绑定, freeze: 已冻结
func (self *SEip) GetStatus() string {
	switch self.Status {
	case "used":
		return api.EIP_STATUS_ASSOCIATE // ?
	case "free":
		return api.EIP_STATUS_READY
	case "freeze":
		return api.EIP_STATUS_UNKNOWN
	default:
		return api.EIP_STATUS_UNKNOWN
	}
}

func (self *SEip) Refresh() error {
	if self.IsEmulated() {
		return nil
	}
	new, err := self.region.GetEipById(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SEip) IsEmulated() bool {
	return false
}

func (self *SEip) GetMetadata() *jsonutils.JSONDict {
	return nil
}

// 付费方式, 枚举值为: Year, 按年付费; Month, 按月付费; Dynamic, 按小时付费; Trial, 试用. 按小时付费和试用这两种付费模式需要开通权限.
func (self *SEip) GetBillingType() string {
	switch self.ChargeType {
	case "Year", "Month":
		return billing_api.BILLING_TYPE_PREPAID
	default:
		return billing_api.BILLING_TYPE_POSTPAID
	}
}

func (self *SEip) GetExpiredAt() time.Time {
	return time.Unix(self.ExpireTime, 0)
}

func (self *SEip) GetIpAddr() string {
	if len(self.EIPAddr) > 1 {
		log.Warning("GetIpAddr %d eip addr found", len(self.EIPAddr))
	} else if len(self.EIPAddr) == 0 {
		return ""
	}

	return self.EIPAddr[0].IP
}

func (self *SEip) GetMode() string {
	return api.EIP_MODE_STANDALONE_EIP
}

func (self *SEip) GetAssociationType() string {
	return "server"
}

// 已绑定的资源类型, 枚举值为: uhost, 云主机；natgw：NAT网关；ulb：负载均衡器；upm: 物理机; hadoophost: 大数据集群;fortresshost：堡垒机；udockhost：容器；udhost：私有专区主机；vpngw：IPSec VPN；ucdr：云灾备；dbaudit：数据库审计。
func (self *SEip) GetAssociationExternalId() string {
	if self.Resource.ResourceType == "uhost" {
		return self.Resource.ResourceID
	} else if self.Resource.ResourceType != "" {
		log.Warningf("GetAssociationExternalId bind with %s %s.expect uhost", self.Resource.ResourceType, self.Resource.ResourceID)
	}

	return ""
}

func (self *SEip) GetBandwidth() int {
	return self.BandwidthMb
}

// 弹性IP的计费模式, 枚举值为: "Bandwidth", 带宽计费; "Traffic", 流量计费; "ShareBandwidth",共享带宽模式. 默认为 "Bandwidth".
func (self *SEip) GetInternetChargeType() string {
	switch self.PayMode {
	case "Bandwidth":
		return api.EIP_CHARGE_TYPE_BY_BANDWIDTH
	case "Traffic":
		return api.EIP_CHARGE_TYPE_BY_TRAFFIC
	default:
		return api.EIP_CHARGE_TYPE_BY_BANDWIDTH
	}
}

func (self *SEip) GetManagerId() string {
	return self.region.client.providerId
}

func (self *SEip) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SEip) Associate(instanceId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SEip) Dissociate() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SEip) ChangeBandwidth(bw int) error {
	return cloudprovider.ErrNotImplemented
}
