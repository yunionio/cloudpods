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

package google

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/log"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SGlobalAddress struct {
	region *SGlobalRegion
	SResourceBase
	multicloud.SEipBase
	GoogleTags

	CreationTimestamp time.Time
	Description       string
	Address           string
	Status            string
	Region            string
	Users             []string
	NetworkTier       string
	AddressType       string
	Kind              string
}

func (addr *SGlobalAddress) GetStatus() string {
	switch addr.Status {
	case "RESERVING":
		return api.EIP_STATUS_ASSOCIATE
	case "RESERVED":
		return api.EIP_STATUS_READY
	case "IN_USE":
		return api.EIP_STATUS_READY
	default:
		log.Errorf("Unknown eip status: %s", addr.Status)
		return api.EIP_STATUS_UNKNOWN
	}
}

func (addr *SGlobalAddress) GetIpAddr() string {
	return addr.Address
}

func (addr *SGlobalAddress) GetMode() string {
	if addr.IsEmulated() {
		return api.EIP_MODE_INSTANCE_PUBLICIP
	}
	return api.EIP_MODE_STANDALONE_EIP
}

func (addr *SGlobalAddress) GetBandwidth() int {
	return 0
}

func (addr *SGlobalAddress) GetInternetChargeType() string {
	return api.EIP_CHARGE_TYPE_BY_TRAFFIC
}

func (addr *SGlobalAddress) Delete() error {
	return addr.region.Delete(addr.SelfLink)
}

func (addr *SGlobalAddress) Associate(conf *cloudprovider.AssociateConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (addr *SGlobalAddress) Dissociate() error {
	return cloudprovider.ErrNotImplemented
}

func (addr *SGlobalAddress) ChangeBandwidth(bw int) error {
	return cloudprovider.ErrNotSupported
}

func (addr *SGlobalAddress) GetProjectId() string {
	return ""
}

func (region *SGlobalRegion) GetEips(address string) ([]SGlobalAddress, error) {
	eips := []SGlobalAddress{}
	params := map[string]string{}
	filters := []string{"addressType=EXTERNAL"}
	if len(address) > 0 {
		filters = append(filters, fmt.Sprintf(`address="%s"`, address))
	}
	params["filter"] = strings.Join(filters, " ADN ")

	resource := "global/addresses"

	err := region.ListAll(resource, params, &eips)
	if err != nil {
		return nil, err
	}

	for i := range eips {
		eips[i].region = region
	}
	return eips, nil
}

func (region *SGlobalRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips, err := region.GetEips("")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudEIP{}
	for i := range eips {
		eips[i].region = region
		ret = append(ret, &eips[i])
	}
	return ret, nil
}

func (region *SGlobalRegion) GetEip(id string) (*SGlobalAddress, error) {
	eip := &SGlobalAddress{region: region}
	return eip, region.Get("addresses", id, eip)
}

func (region *SGlobalRegion) GetIEipById(id string) (cloudprovider.ICloudEIP, error) {
	eip, err := region.GetEip(id)
	if err != nil {
		return nil, err
	}
	return eip, nil
}

func (region *SGlobalRegion) CreateEIP(args *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (addr *SGlobalAddress) GetAssociationExternalId() string {
	associateType := addr.GetAssociationType()
	for _, user := range addr.Users {
		if associateType == api.EIP_ASSOCIATE_TYPE_LOADBALANCER {
			forword := &SForwardingRule{}
			err := addr.region.GetBySelfId(user, forword)
			if err != nil {
				return ""
			}
			proxy := &STargetHttpProxy{}
			err = addr.region.GetBySelfId(forword.Target, proxy)
			if err != nil {
				return ""
			}
			return getGlobalId(proxy.URLMap)
		}
	}
	return ""
}

func (addr *SGlobalAddress) GetAssociationType() string {
	for _, user := range addr.Users {
		if strings.Contains(user, "global/forwardingRules") {
			return api.EIP_ASSOCIATE_TYPE_LOADBALANCER
		}
		return api.EIP_ASSOCIATE_TYPE_SERVER
	}
	return ""
}

func (region *SGlobalRegion) listAll(method string, resource string, params map[string]string, retval interface{}) error {
	return region.client._ecsListAll(method, resource, params, retval)
}

func (region *SGlobalRegion) ListAll(resource string, params map[string]string, retval interface{}) error {
	return region.listAll("GET", resource, params, retval)
}
