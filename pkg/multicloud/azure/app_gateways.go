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

package azure

import (
	"net/url"
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SGatewayipconfiguration struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	Etag       string `json:"etag"`
	Properties struct {
		Provisioningstate string `json:"provisioningState"`
		Subnet            struct {
			ID string `json:"id"`
		} `json:"subnet"`
	} `json:"properties"`
	Type string `json:"type"`
}

type SFrontendipconfiguration struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	Etag       string `json:"etag"`
	Type       string `json:"type"`
	Properties struct {
		Provisioningstate         string `json:"provisioningState"`
		Privateipallocationmethod string `json:"privateIPAllocationMethod"`
		PublicIPAddress           struct {
			ID string
		}
		PrivateIPAddress string
		Subnet           struct {
			ID string `json:"id"`
		} `json:"subnet"`
		Httplisteners []struct {
			ID string `json:"id"`
		} `json:"httpListeners"`
	} `json:"properties"`
}

type SFrontendport struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	Etag       string `json:"etag"`
	Properties struct {
		Provisioningstate string `json:"provisioningState"`
		Port              int    `json:"port"`
		Httplisteners     []struct {
			ID string `json:"id"`
		} `json:"httpListeners"`
	} `json:"properties"`
	Type string `json:"type"`
}

type SBackendaddresspool struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	Etag       string `json:"etag"`
	Properties struct {
		Provisioningstate   string        `json:"provisioningState"`
		Backendaddresses    []interface{} `json:"backendAddresses"`
		Requestroutingrules []struct {
			ID string `json:"id"`
		} `json:"requestRoutingRules"`
	} `json:"properties"`
	Type string `json:"type"`
}

type SBackendhttpsettingscollection struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	Etag       string `json:"etag"`
	Properties struct {
		Provisioningstate              string `json:"provisioningState"`
		Port                           int    `json:"port"`
		Protocol                       string `json:"protocol"`
		Cookiebasedaffinity            string `json:"cookieBasedAffinity"`
		Pickhostnamefrombackendaddress bool   `json:"pickHostNameFromBackendAddress"`
		Requesttimeout                 int    `json:"requestTimeout"`
		Requestroutingrules            []struct {
			ID string `json:"id"`
		} `json:"requestRoutingRules"`
	} `json:"properties"`
	Type string `json:"type"`
}

type SHttplistener struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	Etag       string `json:"etag"`
	Properties struct {
		Provisioningstate       string `json:"provisioningState"`
		Frontendipconfiguration struct {
			ID string `json:"id"`
		} `json:"frontendIPConfiguration"`
		Frontendport struct {
			ID string `json:"id"`
		} `json:"frontendPort"`
		Protocol                    string `json:"protocol"`
		Requireservernameindication bool   `json:"requireServerNameIndication"`
		Requestroutingrules         []struct {
			ID string `json:"id"`
		} `json:"requestRoutingRules"`
	} `json:"properties"`
	Type string `json:"type"`
}

type SRequestroutingrule struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	Etag       string `json:"etag"`
	Properties struct {
		Provisioningstate string `json:"provisioningState"`
		Ruletype          string `json:"ruleType"`
		Httplistener      struct {
			ID string `json:"id"`
		} `json:"httpListener"`
		Backendaddresspool struct {
			ID string `json:"id"`
		} `json:"backendAddressPool"`
		Backendhttpsettings struct {
			ID string `json:"id"`
		} `json:"backendHttpSettings"`
	} `json:"properties"`
	Type string `json:"type"`
}

type SApplicationGatewayProperties struct {
	Provisioningstate string `json:"provisioningState"`
	Resourceguid      string `json:"resourceGuid"`
	Sku               struct {
		Name     string `json:"name"`
		Tier     string `json:"tier"`
		Capacity string `json:"capacity"`
	} `json:"sku"`
	Operationalstate                    string                           `json:"operationalState"`
	Gatewayipconfigurations             []SGatewayipconfiguration        `json:"gatewayIPConfigurations"`
	Sslcertificates                     []interface{}                    `json:"sslCertificates"`
	Authenticationcertificates          []interface{}                    `json:"authenticationCertificates"`
	Frontendipconfigurations            []SFrontendipconfiguration       `json:"frontendIPConfigurations"`
	Frontendports                       []SFrontendport                  `json:"frontendPorts"`
	Backendaddresspools                 []SBackendaddresspool            `json:"backendAddressPools"`
	Backendhttpsettingscollection       []SBackendhttpsettingscollection `json:"backendHttpSettingsCollection"`
	Httplisteners                       []SHttplistener                  `json:"httpListeners"`
	Urlpathmaps                         []interface{}                    `json:"urlPathMaps"`
	Requestroutingrules                 []SRequestroutingrule            `json:"requestRoutingRules"`
	Probes                              []interface{}                    `json:"probes"`
	Redirectconfigurations              []interface{}                    `json:"redirectConfigurations"`
	Webapplicationfirewallconfiguration struct {
		Enabled            bool          `json:"enabled"`
		Firewallmode       string        `json:"firewallMode"`
		Rulesettype        string        `json:"ruleSetType"`
		Rulesetversion     string        `json:"ruleSetVersion"`
		Disabledrulegroups []interface{} `json:"disabledRuleGroups"`
		Requestbodycheck   bool          `json:"requestBodyCheck"`
	} `json:"webApplicationFirewallConfiguration"`
	Enablehttp2 bool `json:"enableHttp2"`
}

type SApplicationGateway struct {
	region *SRegion
	multicloud.SResourceBase
	multicloud.AzureTags

	Name       string                        `json:"name"`
	ID         string                        `json:"id"`
	Etag       string                        `json:"etag"`
	Type       string                        `json:"type"`
	Location   string                        `json:"location"`
	Properties SApplicationGatewayProperties `json:"properties"`
}

func (self *SApplicationGateway) GetName() string {
	return self.Name
}

func (self *SApplicationGateway) GetId() string {
	return self.ID
}

func (self *SApplicationGateway) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SApplicationGateway) GetStatus() string {
	switch self.Properties.Provisioningstate {
	case "Deleting":
		return api.APP_GATEWAY_STATUS_DELETING
	case "Failed":
		return api.APP_GATEWAY_STATUS_CREATE_FAILED
	case "Succeeded":
		return api.APP_GATEWAY_STATUS_AVAILABLE
	case "Updating":
		return api.APP_GATEWAY_STATUS_UPDATING
	}
	return api.APP_GATEWAY_STATUS_AVAILABLE
}

func (self *SApplicationGateway) GetInstanceType() string {
	return self.Properties.Sku.Name
}

func (self *SApplicationGateway) GetBackends() ([]cloudprovider.SAppGatewayBackend, error) {
	ret := []cloudprovider.SAppGatewayBackend{}
	for _, conf := range self.Properties.Backendaddresspools {
		backend := cloudprovider.SAppGatewayBackend{
			Name:         conf.Name,
			RoutingRules: []cloudprovider.SAppGatewayRoutingRule{},
		}
		for _, r := range conf.Properties.Requestroutingrules {
			rule := cloudprovider.SAppGatewayRoutingRule{}
			for _, _rule := range self.Properties.Requestroutingrules {
				if r.ID == _rule.ID {
					rule.Name = _rule.Name
					rule.Type = _rule.Properties.Ruletype
					backend.RoutingRules = append(backend.RoutingRules, rule)
					break
				}
			}
		}
		ret = append(ret, backend)
	}
	return ret, nil
}

func (self *SApplicationGateway) GetFrontends() ([]cloudprovider.SAppGatewayFrontend, error) {
	ret := []cloudprovider.SAppGatewayFrontend{}
	for _, conf := range self.Properties.Frontendipconfigurations {
		front := cloudprovider.SAppGatewayFrontend{
			Name:         conf.Name,
			HttpListener: []cloudprovider.SAppGatewayHttpListener{},
		}
		for _, l := range conf.Properties.Httplisteners {
			listener := cloudprovider.SAppGatewayHttpListener{}
			for _, p := range self.Properties.Httplisteners {
				if strings.ToLower(p.ID) == strings.ToLower(l.ID) {
					listener.Name = p.Name
					listener.Protocol = p.Properties.Protocol
					break
				}
			}
			for _, p := range self.Properties.Frontendports {
				for _, _p := range self.Properties.Httplisteners {
					if strings.ToLower(_p.ID) == strings.ToLower(l.ID) {
						listener.Port = p.Properties.Port
						break
					}
				}
				if listener.Port > 0 {
					break
				}
			}
			if len(listener.Name) > 0 {
				front.HttpListener = append(front.HttpListener, listener)
			}
		}
		if len(conf.Properties.PrivateIPAddress) > 0 {
			front.IpAddr = conf.Properties.PrivateIPAddress
			front.Type = "Vpc"
		} else if len(conf.Properties.PublicIPAddress.ID) > 0 {
			eip, err := self.region.GetEip(conf.Properties.PublicIPAddress.ID)
			if err != nil {
				continue
			}
			front.IpAddr = eip.GetIpAddr()
			front.Type = "Eip"
		}
		ret = append(ret, front)
	}
	return ret, nil
}

func (self *SRegion) ListAppGateways() ([]SApplicationGateway, error) {
	apps := []SApplicationGateway{}
	err := self.list("Microsoft.Network/applicationGateways", url.Values{}, &apps)
	if err != nil {
		return nil, errors.Wrapf(err, "list")
	}
	return apps, nil
}

func (self *SRegion) GetApplicationGateway(id string) (*SApplicationGateway, error) {
	ret := &SApplicationGateway{region: self}
	return ret, self.get(id, url.Values{}, ret)
}

func (self *SRegion) GetICloudApplicationGateways() ([]cloudprovider.ICloudApplicationGateway, error) {
	apps, err := self.ListAppGateways()
	if err != nil {
		return nil, errors.Wrapf(err, "ListAppGateways")
	}
	ret := []cloudprovider.ICloudApplicationGateway{}
	for i := range apps {
		apps[i].region = self
		ret = append(ret, &apps[i])
	}
	return ret, nil
}

func (self *SRegion) GetICloudApplicationGatewayById(id string) (cloudprovider.ICloudApplicationGateway, error) {
	return self.GetApplicationGateway(id)
}
