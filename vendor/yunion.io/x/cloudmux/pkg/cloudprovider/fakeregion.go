// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cloudprovider

import "yunion.io/x/pkg/errors"

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

func (region *SFakeOnPremiseRegion) GetCloudEnv() string {
	return ""
}

func (region *SFakeOnPremiseRegion) Refresh() error {
	return nil
}

func (region *SFakeOnPremiseRegion) IsEmulated() bool {
	return true
}

func (region *SFakeOnPremiseRegion) GetSysTags() map[string]string {
	return nil
}

func (region *SFakeOnPremiseRegion) GetTags() (map[string]string, error) {
	return nil, errors.Wrap(ErrNotImplemented, "GetTags")
}

func (region *SFakeOnPremiseRegion) SetTags(tags map[string]string, replace bool) error {
	return ErrNotImplemented
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

func (region *SFakeOnPremiseRegion) CreateIVpc(opts *VpcCreateOptions) (ICloudVpc, error) {
	return nil, ErrNotSupported
}

func (region *SFakeOnPremiseRegion) CreateEIP(eip *SEip) (ICloudEIP, error) {
	return nil, ErrNotSupported
}

func (region *SFakeOnPremiseRegion) GetISecurityGroupById(id string) (ICloudSecurityGroup, error) {
	return nil, ErrNotSupported
}

func (region *SFakeOnPremiseRegion) GetISecurityGroupByName(opts *SecurityGroupFilterOptions) (ICloudSecurityGroup, error) {
	return nil, ErrNotSupported
}

func (region *SFakeOnPremiseRegion) CreateISecurityGroup(conf *SecurityGroupCreateInput) (ICloudSecurityGroup, error) {
	return nil, ErrNotSupported
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

func (region *SFakeOnPremiseRegion) CreateILoadBalancer(loadbalancer *SLoadbalancerCreateOptions) (ICloudLoadbalancer, error) {
	return nil, ErrNotSupported
}

func (region *SFakeOnPremiseRegion) CreateILoadBalancerAcl(acl *SLoadbalancerAccessControlList) (ICloudLoadbalancerAcl, error) {
	return nil, ErrNotSupported
}
