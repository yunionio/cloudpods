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

package zstack

import (
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"
)

type SNetworkService struct {
	IPsec          []string `json:"IPsec"`
	VRouterRoute   []string `json:"VRouterRoute"`
	VipQos         []string `json:"VipQos"`
	DNS            []string `json:"DNS"`
	SNAT           []string `json:"SNAT"`
	LoadBalancer   []string `json:"LoadBalancer"`
	Userdata       []string `json:"Userdata"`
	SecurityGroup  []string `json:"SecurityGroup"`
	Eip            []string `json:"Eip"`
	DHCP           []string `json:"DHCP"`
	CentralizedDNS []string `json:"CentralizedDNS"`
	HostRoute      []string `json:"HostRoute"`
	PortForwarding []string `json:"PortForwarding"`
}

type SNetworkServiceProvider struct {
	ZStackBasic
	ZStackTime
	Type                   string   `json:"type"`
	NetworkServiceTypes    []string `json:"networkServiceTypes"`
	AttachedL2NetworkUUIDs []string `json:"attachedL2NetworkUuids"`
}

type SNetworkServiceRef struct {
	L3NetworkUUID              string `json:"l3NetworkUuid"`
	NetworkServiceProviderUUID string `json:"networkServiceProviderUuid"`
	NetworkServiceType         string `json:"networkServiceType"`
}

func (region *SRegion) GetNetworkServices() (*SNetworkService, error) {
	service := &SNetworkService{}
	resp, err := region.client.get("network-services/types", "", "")
	if err != nil {
		return nil, err
	}
	return service, resp.Unmarshal(service, "types")
}

func (region *SRegion) GetNetworkServiceProviders(Type string) ([]SNetworkServiceProvider, error) {
	providers := []SNetworkServiceProvider{}
	params := url.Values{}
	if len(Type) > 0 {
		params.Add("q", "type="+Type)
	}
	return providers, region.client.listAll("network-services/providers", params, &providers)
}

func (region *SRegion) GetNetworkServiceRef(l3Id string, Type string) ([]SNetworkServiceRef, error) {
	refs := []SNetworkServiceRef{}
	params := url.Values{}
	if len(l3Id) > 0 {
		params.Add("q", "l3NetworkUuid="+l3Id)
	}
	if len(Type) > 0 {
		params.Add("q", "networkServiceType="+Type)
	}
	return refs, region.client.listAll("l3-networks/network-services/refs", params, &refs)
}

func (region *SRegion) AttachServiceForl3Network(l3Id string, services []string) error {
	networkServices := map[string][]string{}
	refs, err := region.GetNetworkServiceRef(l3Id, "")
	if err != nil {
		return err
	}
	currentServices := []string{}
	for i := 0; i < len(refs); i++ {
		currentServices = append(currentServices, refs[i].NetworkServiceType)
	}
	for _, service := range services {
		networkServiceProviders, err := region.GetNetworkServiceProviders(service)
		if err != nil || len(networkServiceProviders) == 0 {
			msg := fmt.Sprintf("failed to find network services %s error: %v", service, err)
			log.Errorln(msg)
			return fmt.Errorf(msg)
		}
		attachServices := []string{}
		for i := 0; i < len(networkServiceProviders[0].NetworkServiceTypes); i++ {
			if !utils.IsInStringArray(networkServiceProviders[0].NetworkServiceTypes[i], currentServices) {
				attachServices = append(attachServices, networkServiceProviders[0].NetworkServiceTypes[i])
			}
		}
		if len(attachServices) > 0 {
			networkServices[networkServiceProviders[0].UUID] = attachServices
		}
	}

	if len(networkServices) == 0 {
		return nil
	}
	params := map[string]interface{}{
		"params": map[string]interface{}{
			"networkServices": networkServices,
		},
	}
	resource := fmt.Sprintf("l3-networks/%s/network-services", l3Id)
	_, err = region.client.post(resource, jsonutils.Marshal(params))
	if err != nil {
		log.Errorf("failed to attach network services %s to l3network %s error: %v", services, l3Id, err)
	}
	return err
}

func (region *SRegion) RemoveNetworkService(l3Id string, service string) error {
	return region.client.delete("l3-networks", fmt.Sprintf("%s/network-services?networkServices=%s", l3Id, service), "")
}
