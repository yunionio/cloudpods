package zstack

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SEipAddress struct {
	region *SRegion

	ZStackBasic
	VMNicUUID string `json:"vmNicUuid"`
	VipUUID   string `json:"vipUuid"`
	State     string `json:"state"`
	VipIP     string `json:"vipIp"`
	GuestIP   string `json:"guestIp"`
	ZStackTime
}

func (region *SRegion) GetEip(eipId string) (*SEipAddress, error) {
	eips, err := region.GetEips(eipId, "")
	if err != nil {
		return nil, err
	}
	if len(eips) == 1 {
		if eips[0].UUID == eipId {
			return &eips[0], nil
		}
		return nil, cloudprovider.ErrNotFound
	}
	if len(eips) == 0 || len(eipId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (region *SRegion) GetEips(eipId, instanceId string) ([]SEipAddress, error) {
	eips := []SEipAddress{}
	params := []string{}
	if len(eipId) > 0 {
		params = append(params, "q=uuid="+eipId)
	}
	if len(instanceId) > 0 {
		params = append(params, "q=vmNic.vmInstanceUuid="+instanceId)
	}
	err := region.client.listAll("eips", params, &eips)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(eips); i++ {
		eips[i].region = region
	}
	return eips, nil
}

func (eip *SEipAddress) GetId() string {
	return eip.UUID
}

func (eip *SEipAddress) GetName() string {
	return eip.Name
}

func (eip *SEipAddress) GetGlobalId() string {
	return eip.UUID
}

func (eip *SEipAddress) GetStatus() string {
	return api.EIP_STATUS_READY
}

func (eip *SEipAddress) Refresh() error {
	new, err := eip.region.GetEip(eip.UUID)
	if err != nil {
		return err
	}
	return jsonutils.Update(eip, new)
}

func (eip *SEipAddress) IsEmulated() bool {
	return false
}

func (eip *SEipAddress) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (eip *SEipAddress) GetIpAddr() string {
	return eip.VipIP
}

func (eip *SEipAddress) GetMode() string {
	return api.EIP_MODE_STANDALONE_EIP
}

func (eip *SEipAddress) GetAssociationType() string {
	return "server"
}

func (eip *SEipAddress) GetAssociationExternalId() string {
	if len(eip.VMNicUUID) > 0 {
		instances, err := eip.region.GetInstances("", "", eip.VMNicUUID)
		if err != nil && len(instances) == 1 {
			return instances[0].UUID
		}
	}
	return ""
}

func (eip *SEipAddress) GetManagerId() string {
	return eip.region.client.providerID
}

func (eip *SEipAddress) GetBillingType() string {
	return ""
}

func (eip *SEipAddress) GetCreatedAt() time.Time {
	return eip.CreateDate
}

func (eip *SEipAddress) GetExpiredAt() time.Time {
	return time.Time{}
}

func (eip *SEipAddress) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (eip *SEipAddress) GetBandwidth() int {
	return 0
}

func (eip *SEipAddress) GetInternetChargeType() string {
	return api.EIP_CHARGE_TYPE_BY_TRAFFIC
}

func (eip *SEipAddress) Associate(instanceId string) error {
	return cloudprovider.ErrNotImplemented
}

func (eip *SEipAddress) Dissociate() error {
	return cloudprovider.ErrNotImplemented
}

func (eip *SEipAddress) ChangeBandwidth(bw int) error {
	return cloudprovider.ErrNotSupported
}

func (eip *SEipAddress) GetProjectId() string {
	return ""
}
