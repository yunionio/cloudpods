package cloudprovider

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/secrules"
)

type SFakeOnPremiseRegion struct {
}

func (region *SFakeOnPremiseRegion) GetId() string {
	return "default"
}

func (region *SFakeOnPremiseRegion) GetName() string {
	return "Default"
}

func (region *SFakeOnPremiseRegion) GetGlobalId() string {
	return "default"
}

func (region *SFakeOnPremiseRegion) GetStatus() string {
	return "available"
}

func (region *SFakeOnPremiseRegion) Refresh() error {
	return nil
}

func (region *SFakeOnPremiseRegion) IsEmulated() bool {
	return true
}

func (region *SFakeOnPremiseRegion) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (region *SFakeOnPremiseRegion) GetGeographicInfo() SGeographicInfo {
	return SGeographicInfo{}
}

func (region *SFakeOnPremiseRegion) GetIZones() ([]ICloudZone, error) {
	return nil, ErrNotSupported
}

func (region *SFakeOnPremiseRegion) GetIZoneById(id string) (ICloudZone, error) {
	return nil, ErrNotSupported
}

func (region *SFakeOnPremiseRegion) GetIVpcById(id string) (ICloudVpc, error) {
	return nil, ErrNotSupported
}

func (region *SFakeOnPremiseRegion) GetIVpcs() ([]ICloudVpc, error) {
	return nil, ErrNotSupported
}

func (region *SFakeOnPremiseRegion) GetIEips() ([]ICloudEIP, error) {
	return nil, ErrNotSupported
}

func (region *SFakeOnPremiseRegion) GetIEipById(id string) (ICloudEIP, error) {
	return nil, ErrNotSupported
}

func (region *SFakeOnPremiseRegion) CreateIVpc(name string, desc string, cidr string) (ICloudVpc, error) {
	return nil, ErrNotSupported
}

func (region *SFakeOnPremiseRegion) CreateEIP(name string, bwMbps int, chargeType string, bgpType string) (ICloudEIP, error) {
	return nil, ErrNotSupported
}

func (region *SFakeOnPremiseRegion) DeleteSecurityGroup(vpcId, secgroupId string) error {
	return ErrNotSupported
}

func (region *SFakeOnPremiseRegion) SyncSecurityGroup(secgroupId string, vpcId string, name string, desc string, rules []secrules.SecurityRule) (string, error) {
	return "", ErrNotSupported
}

func (region *SFakeOnPremiseRegion) GetILoadBalancers() ([]ICloudLoadbalancer, error) {
	return nil, ErrNotSupported
}

func (region *SFakeOnPremiseRegion) GetILoadBalancerById(loadbalancerId string) (ICloudLoadbalancer, error) {
	return nil, ErrNotSupported
}

func (region *SFakeOnPremiseRegion) GetILoadBalancerAclById(aclId string) (ICloudLoadbalancerAcl, error) {
	return nil, ErrNotSupported
}

func (region *SFakeOnPremiseRegion) GetILoadBalancerCertificateById(certId string) (ICloudLoadbalancerCertificate, error) {
	return nil, ErrNotSupported
}

func (region *SFakeOnPremiseRegion) CreateILoadBalancerCertificate(cert *SLoadbalancerCertificate) (ICloudLoadbalancerCertificate, error) {
	return nil, ErrNotImplemented
}

func (region *SFakeOnPremiseRegion) GetILoadBalancerAcls() ([]ICloudLoadbalancerAcl, error) {
	return nil, ErrNotSupported
}

func (region *SFakeOnPremiseRegion) GetILoadBalancerCertificates() ([]ICloudLoadbalancerCertificate, error) {
	return nil, ErrNotSupported
}

func (region *SFakeOnPremiseRegion) CreateILoadBalancer(loadbalancer *SLoadbalancer) (ICloudLoadbalancer, error) {
	return nil, ErrNotSupported
}

func (region *SFakeOnPremiseRegion) CreateILoadBalancerAcl(acl *SLoadbalancerAccessControlList) (ICloudLoadbalancerAcl, error) {
	return nil, ErrNotSupported
}

func (region *SFakeOnPremiseRegion) GetSkus(zoneId string) ([]ICloudSku, error) {
	return nil, ErrNotSupported
}
