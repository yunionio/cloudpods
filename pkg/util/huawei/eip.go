package huawei

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type TInternetChargeType string

const (
	InternetChargeByTraffic   = TInternetChargeType("PayByTraffic")
	InternetChargeByBandwidth = TInternetChargeType("PayByBandwidth")
)

type Port struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	AdminStateUp    string `json:"admin_state_up"`
	DNSName         string `json:"dns_name"`
	MACAddress      string `json:"mac_address"`
	NetworkID       string `json:"network_id"`
	TenantID        string `json:"tenant_id"`
	DeviceID        string `json:"device_id"`
	DeviceOwner     string `json:"device_owner"`
	BindingVnicType string `json:"binding:vnic_type"`
}

type Bandwidth struct {
	ID                  string         `json:"id"`
	Name                string         `json:"name"`
	Size                int64          `json:"size"`
	ShareType           string         `json:"share_type"`
	PublicipInfo        []PublicipInfo `json:"publicip_info"`
	TenantID            string         `json:"tenant_id"`
	BandwidthType       string         `json:"bandwidth_type"`
	ChargeMode          string         `json:"charge_mode"`
	BillingInfo         string         `json:"billing_info"`
	EnterpriseProjectID string         `json:"enterprise_project_id"`
}

type PublicipInfo struct {
	PublicipID      string `json:"publicip_id"`
	PublicipAddress string `json:"publicip_address"`
	PublicipType    string `json:"publicip_type"`
	IPVersion       int64  `json:"ip_version"`
}

type Profile struct {
	UserID    string `json:"user_id"`
	ProductID string `json:"product_id"`
	RegionID  string `json:"region_id"`
	OrderID   string `json:"order_id"`
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090598.html
type SEipAddress struct {
	region *SRegion

	ID                  string  `json:"id"`
	Status              string  `json:"status"`
	Profile             Profile `json:"profile"`
	Type                string  `json:"type"`
	PublicIPAddress     string  `json:"public_ip_address"`
	PrivateIPAddress    string  `json:"private_ip_address"`
	TenantID            string  `json:"tenant_id"`
	CreateTime          string  `json:"create_time"`
	BandwidthID         string  `json:"bandwidth_id"`
	BandwidthShareType  string  `json:"bandwidth_share_type"`
	BandwidthSize       int64   `json:"bandwidth_size"`
	BandwidthName       string  `json:"bandwidth_name"`
	EnterpriseProjectID string  `json:"enterprise_project_id"`
	IPVersion           int64   `json:"ip_version"`
	PortId              string  `json:"port_id"`
}

func (self *SEipAddress) GetId() string {
	return self.ID
}

func (self *SEipAddress) GetName() string {
	if len(self.BandwidthName) == 0 {
		return self.BandwidthName
	}

	return self.PublicIPAddress
}

func (self *SEipAddress) GetGlobalId() string {
	return self.ID
}

func (self *SEipAddress) GetStatus() string {
	// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090598.html
	switch self.Status {
	case "ACTIVE", "DOWN", "ELB":
		return models.EIP_STATUS_READY
	case "PENDING_CREATE", "NOTIFYING":
		return models.EIP_STATUS_ALLOCATE
	case "BINDING":
		return models.EIP_STATUS_ALLOCATE
	case "BIND_ERROR":
		return models.EIP_STATUS_ALLOCATE_FAIL
	case "PENDING_DELETE", "NOTIFY_DELETE":
		return models.EIP_STATUS_DEALLOCATE
	default:
		return models.EIP_STATUS_UNKNOWN
	}
}

func (self *SEipAddress) Refresh() error {
	if self.IsEmulated() {
		return nil
	}
	new, err := self.region.GetEip(self.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SEipAddress) IsEmulated() bool {
	return false
}

func (self *SEipAddress) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SEipAddress) GetIpAddr() string {
	return self.PublicIPAddress
}

func (self *SEipAddress) GetMode() string {
	return models.EIP_MODE_STANDALONE_EIP
}

func (self *SEipAddress) GetAssociationType() string {
	return "server"
}

func (self *SEipAddress) GetAssociationExternalId() string {
	// todo： implement me返回关联的实例
	// network/0273a359d61847fc83405926c958c746/ext-floatingips?tenantId=0273a359d61847fc83405926c958c746&limit=2000
	// 只能通过 port id 反查device id.
	if len(self.PortId) > 0 {
		port, err := self.region.GetPort(self.PortId)
		if err != nil {
			return ""
		}

		return port.DeviceID
	}

	return ""
}

func (self *SEipAddress) GetBandwidth() int {
	return int(self.BandwidthSize) // Mb
}

func (self *SEipAddress) GetInternetChargeType() string {
	// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090603.html
	bandwidth, err := self.region.GetEipBandwidth(self.BandwidthID)
	if err != nil {
		return models.EIP_CHARGE_TYPE_BY_TRAFFIC
	}

	if bandwidth.ChargeMode != "traffic" {
		return models.EIP_CHARGE_TYPE_BY_BANDWIDTH
	} else {
		return models.EIP_CHARGE_TYPE_BY_TRAFFIC
	}
}

func (self *SEipAddress) GetManagerId() string {
	return self.region.client.providerId
}

func (self *SEipAddress) Delete() error {
	return self.region.DeallocateEIP(self.ID)
}

func (self *SEipAddress) Associate(instanceId string) error {
	err := self.region.AssociateEip(self.ID, instanceId)
	if err != nil {
		return err
	}
	err = cloudprovider.WaitStatus(self, models.EIP_STATUS_READY, 10*time.Second, 180*time.Second)
	return err
}

func (self *SEipAddress) Dissociate() error {
	// todo : implement me
	err := self.region.DissociateEip(self.ID, "")
	if err != nil {
		return err
	}
	err = cloudprovider.WaitStatus(self, models.EIP_STATUS_READY, 10*time.Second, 180*time.Second)
	return err
}

func (self *SEipAddress) ChangeBandwidth(bw int) error {
	return self.region.UpdateEipBandwidth(self.ID, bw)
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090596.html
func (region *SRegion) AllocateEIP(bwMbps int, chargeType TInternetChargeType) (*SEipAddress, error) {
	// todo: implement me
	return &SEipAddress{}, nil
}

func (self *SRegion) GetEip(eipId string) (*SEipAddress, error) {
	var eip SEipAddress
	err := DoGet(self.ecsClient.Eips.Get, eipId, nil, &eip)
	eip.region = self
	return &eip, err
}

func (self *SRegion) DeallocateEIP(eipId string) error {
	// todo : implement me
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) AssociateEip(eipId string, instanceId string) error {
	// todo : implement me
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) DissociateEip(eipId string, instanceId string) error {
	// todo : implement me
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) UpdateEipBandwidth(eipId string, bw int) error {
	// todo : implement me
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) GetEipBandwidth(bandwidthId string) (Bandwidth, error) {
	bandwidth := Bandwidth{}
	err := DoGet(self.ecsClient.Bandwidths.Get, bandwidthId, nil, &bandwidth)
	return bandwidth, err
}

func (self *SRegion) GetPort(portId string) (Port, error) {
	port := Port{}
	err := DoGet(self.ecsClient.Bandwidths.Get, portId, nil, &port)
	return port, err
}
