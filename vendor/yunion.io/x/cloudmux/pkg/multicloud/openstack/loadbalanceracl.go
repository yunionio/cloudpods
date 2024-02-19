// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package openstack

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/apis"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type AclEntrys struct {
	AclEntry []AclEntry
}

type AclEntry struct {
	AclEntryComment string
	AclEntryIP      string
}

type SLoadbalancerAcl struct {
	multicloud.SResourceBase
	OpenStackTags
	listener *SLoadbalancerListener
}

func (acl *SLoadbalancerAcl) GetName() string {
	return acl.listener.Name + "AllowedCidrs"
}

func (acl *SLoadbalancerAcl) GetId() string {
	return acl.listener.ID
}

func (acl *SLoadbalancerAcl) GetGlobalId() string {
	return acl.listener.ID
}

func (acl *SLoadbalancerAcl) GetStatus() string {
	return apis.STATUS_AVAILABLE
}

func (acl *SLoadbalancerAcl) Refresh() error {
	return acl.listener.Refresh()
}

func (acl *SLoadbalancerAcl) GetAclEntries() []cloudprovider.SLoadbalancerAccessControlListEntry {
	aclEntrys := []cloudprovider.SLoadbalancerAccessControlListEntry{}
	for i := 0; i < len(acl.listener.AllowedCidrs); i++ {
		aclEntry := cloudprovider.SLoadbalancerAccessControlListEntry{}
		aclEntry.CIDR = acl.listener.AllowedCidrs[i]
		aclEntry.Comment = "AllowedCidr"
		aclEntrys = append(aclEntrys, aclEntry)
	}
	return aclEntrys
}

func (region *SRegion) UpdateLoadbalancerListenerAllowedCidrs(listenerId string, cidrs []string) error {
	params := jsonutils.NewDict()
	listenerParam := jsonutils.NewDict()
	listenerParam.Add(jsonutils.NewStringArray(cidrs), "allowed_cidrs")
	params.Add(listenerParam, "listener")
	_, err := region.lbUpdate(fmt.Sprintf("/v2/lbaas/listeners/%s", listenerId), params)
	if err != nil {
		return errors.Wrapf(err, `region.lbUpdate(/v2/lbaas/listeners/%s, params)`, listenerId)
	}
	return nil
}

func (acl *SLoadbalancerAcl) Delete() error {
	// ensure listener status
	err := waitLbResStatus(acl.listener, 10*time.Second, 8*time.Minute)
	if err != nil {
		return errors.Wrap(err, `waitLbResStatus(acl.listener, 10*time.Second, 8*time.Minute)`)
	}
	err = acl.listener.region.UpdateLoadbalancerListenerAllowedCidrs(acl.listener.ID, []string{})
	if err != nil {
		return errors.Wrap(err, `acl.listener.region.UpdateLoadbalancerListenerAllowedCidrs(acl.listener.ID, []string{})`)
	}
	err = waitLbResStatus(acl.listener, 10*time.Second, 8*time.Minute)
	if err != nil {
		return errors.Wrap(err, `waitLbResStatus(acl.listener, 10*time.Second, 8*time.Minute)`)
	}
	return nil
}

func (region *SRegion) GetLoadbalancerAclDetail(aclId string) (*SLoadbalancerAcl, error) {
	listener, err := region.GetLoadbalancerListenerbyId(aclId)
	if err != nil {
		return nil, errors.Wrapf(err, "region.GetLoadbalancerListenerbyId(%s)", aclId)
	}
	acl := SLoadbalancerAcl{}
	acl.listener = listener
	return &acl, nil
}

func (region *SRegion) GetLoadBalancerAcls() ([]SLoadbalancerAcl, error) {
	listeners, err := region.GetLoadbalancerListeners()
	if err != nil {
		return nil, errors.Wrap(err, "region.GetLoadbalancerListeners()")
	}
	acls := []SLoadbalancerAcl{}
	for i := 0; i < len(listeners); i++ {
		if len(listeners[i].AllowedCidrs) < 1 {
			continue
		}
		acl := new(SLoadbalancerAcl)
		acl.listener = &listeners[i]
		acls = append(acls, *acl)

	}
	return acls, nil
}

func (region *SRegion) CreateLoadBalancerAcl(acl *cloudprovider.SLoadbalancerAccessControlList) (*SLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (acl *SLoadbalancerAcl) Sync(_acl *cloudprovider.SLoadbalancerAccessControlList) error {
	// ensure listener status
	err := waitLbResStatus(acl.listener, 10*time.Second, 8*time.Minute)
	if err != nil {
		return errors.Wrap(err, "waitLbResStatus(acl.listener, 10*time.Second, 8*time.Minute)")
	}

	cidrs := []string{}
	for i := 0; i < len(_acl.Entrys); i++ {
		cidrs = append(cidrs, _acl.Entrys[i].CIDR)
	}
	err = acl.listener.region.UpdateLoadbalancerListenerAllowedCidrs(acl.listener.ID, cidrs)
	if err != nil {
		return errors.Wrapf(err, "UpdateLoadbalancerListenerAllowedCidrs(%s, cidrs)", acl.listener.ID)
	}
	err = waitLbResStatus(acl.listener, 10*time.Second, 8*time.Minute)
	if err != nil {
		return errors.Wrap(err, "waitLbResStatus(acl.listener, 10*time.Second, 8*time.Minute)")
	}
	return nil
}

func (acl *SLoadbalancerAcl) GetProjectId() string {
	return acl.listener.ProjectID
}
