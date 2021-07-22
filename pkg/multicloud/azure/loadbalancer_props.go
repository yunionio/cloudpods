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

type SLoadbalancerProperties struct {
	Sku                                 Sku                                 `json:"sku"`
	ProvisioningState                   string                              `json:"provisioningState"`
	OperationalState                    string                              `json:"operationalState"`
	ResourceGUID                        string                              `json:"resourceGuid"`
	GatewayIPConfigurations             []GatewayIPConfiguration            `json:"gatewayIPConfigurations"`
	SSLCertificates                     []SSLCertificate                    `json:"sslCertificates"`
	FrontendIPConfigurations            []FrontendIPConfiguration           `json:"frontendIPConfigurations"`
	FrontendPorts                       []FrontendPort                      `json:"frontendPorts"`
	BackendAddressPools                 []BackendAddressPool                `json:"backendAddressPools"`
	BackendHTTPSettingsCollection       []BackendHTTPSettingsCollection     `json:"backendHttpSettingsCollection"`
	LoadBalancingRules                  []LoadBalancingRule                 `json:"loadBalancingRules"`
	Probes                              []Probe                             `json:"probes"`
	InboundNatRules                     []InboundNatRule                    `json:"inboundNatRules"`
	HTTPListeners                       []HTTPListener                      `json:"httpListeners"`
	URLPathMaps                         []URLPathMap                        `json:"urlPathMaps"`
	RequestRoutingRules                 []RequestRoutingRule                `json:"requestRoutingRules"`
	RedirectConfigurations              []RedirectConfiguration             `json:"redirectConfigurations"`
	WebApplicationFirewallConfiguration WebApplicationFirewallConfiguration `json:"webApplicationFirewallConfiguration"`
	EnableHttp2                         bool                                `json:"enableHttp2"`
}

type BackendAddressPool struct {
	Name       string                       `json:"name"`
	ID         string                       `json:"id"`
	Etag       string                       `json:"etag"`
	Properties BackendAddressPoolProperties `json:"properties"`
	Type       string                       `json:"type"`
}

type BackendAddressPoolProperties struct {
	ProvisioningState       string                   `json:"provisioningState"`
	LoadBalancingRules      []Subnet                 `json:"loadBalancingRules"`
	BackendIPConfigurations []BackendIPConfiguration `json:"backendIPConfigurations"`
	BackendAddresses        []BackendAddress         `json:"backendAddresses"`
	RequestRoutingRules     []BackendIPConfiguration `json:"requestRoutingRules"`
	URLPathMaps             []BackendIPConfiguration `json:"urlPathMaps"`
	PathRules               []BackendIPConfiguration `json:"pathRules"`
}

type BackendIPConfiguration struct {
	ID string `json:"id"`
}

type RedirectConfiguration struct {
	Name       string                          `json:"name"`
	ID         string                          `json:"id"`
	Etag       string                          `json:"etag"`
	Properties RedirectConfigurationProperties `json:"properties"`
	Type       string                          `json:"type"`
}

type RedirectConfigurationProperties struct {
	ProvisioningState  string                   `json:"provisioningState"`
	RedirectType       string                   `json:"redirectType"`
	TargetUrl          string                   `json:"targetUrl"`
	TargetListener     PublicIPAddressElement   `json:"targetListener"`
	IncludePath        bool                     `json:"includePath"`
	IncludeQueryString bool                     `json:"includeQueryString"`
	URLPathMaps        []PublicIPAddressElement `json:"urlPathMaps,omitempty"`
	PathRules          []PublicIPAddressElement `json:"pathRules,omitempty"`
}

type WebApplicationFirewallConfiguration struct {
	Enabled            bool          `json:"enabled"`
	FirewallMode       string        `json:"firewallMode"`
	RuleSetType        string        `json:"ruleSetType"`
	RuleSetVersion     string        `json:"ruleSetVersion"`
	DisabledRuleGroups []interface{} `json:"disabledRuleGroups"`
}

type URLPathMap struct {
	Name       string               `json:"name"`
	ID         string               `json:"id"`
	Etag       string               `json:"etag"`
	Properties URLPathMapProperties `json:"properties"`
	Type       string               `json:"type"`
}

type URLPathMapProperties struct {
	ProvisioningState            string                   `json:"provisioningState"`
	DefaultBackendAddressPool    *PublicIPAddressElement  `json:"defaultBackendAddressPool,omitempty"`
	DefaultBackendHTTPSettings   *PublicIPAddressElement  `json:"defaultBackendHttpSettings,omitempty"`
	PathRules                    []PathRule               `json:"pathRules"`
	RequestRoutingRules          []PublicIPAddressElement `json:"requestRoutingRules"`
	DefaultRedirectConfiguration *PublicIPAddressElement  `json:"defaultRedirectConfiguration,omitempty"`
}

type PublicIPAddressElement struct {
	ID string `json:"id"`
}

type PathRule struct {
	Name       string             `json:"name"`
	ID         string             `json:"id"`
	Etag       string             `json:"etag"`
	Properties PathRuleProperties `json:"properties"`
	Type       string             `json:"type"`
}

type PathRuleProperties struct {
	ProvisioningState     string                  `json:"provisioningState"`
	Paths                 []string                `json:"paths"`
	BackendAddressPool    *PublicIPAddressElement `json:"backendAddressPool,omitempty"`
	BackendHTTPSettings   *PublicIPAddressElement `json:"backendHttpSettings,omitempty"`
	RedirectConfiguration *PublicIPAddressElement `json:"redirectConfiguration,omitempty"`
}

type BackendAddress struct {
	IPAddress string `json:"ipAddress"`
}

type BackendHTTPSettingsCollection struct {
	Name       string                                  `json:"name"`
	ID         string                                  `json:"id"`
	Properties BackendHTTPSettingsCollectionProperties `json:"properties"`
}

type BackendHTTPSettingsCollectionProperties struct {
	Port                int                `json:"port"`
	Protocol            string             `json:"protocol"`
	Probe               *ProbeId           `json:"probe"`
	ConnectionDraining  ConnectionDraining `json:"connectionDraining"`
	CookieBasedAffinity string             `json:"cookieBasedAffinity"`
	AffinityCookieName  string             `json:"affinityCookieName"`
	HostName            string             `json:"hostName"`
	RequestTimeout      int                `json:"requestTimeout"`
}

type ConnectionDraining struct {
	Enabled           bool   `json:"enabled"`
	DrainTimeoutInSEC string `json:"drainTimeoutInSec"`
}

type ProbeId struct {
	ID string `json:"id"`
}

type FrontendPort struct {
	Name       string                 `json:"name"`
	ID         string                 `json:"id"`
	Properties FrontendPortProperties `json:"properties"`
}

type FrontendPortProperties struct {
	Port int `json:"port"`
}

type GatewayIPConfiguration struct {
	Name       string                           `json:"name"`
	ID         string                           `json:"id"`
	Properties GatewayIPConfigurationProperties `json:"properties"`
}

type GatewayIPConfigurationProperties struct {
	Subnet PublicIPAddress `json:"subnet"`
}

type RequestRoutingRule struct {
	Name       string                       `json:"name"`
	ID         string                       `json:"id"`
	Properties RequestRoutingRuleProperties `json:"properties"`
}

type RequestRoutingRuleProperties struct {
	RuleType            string          `json:"ruleType"`
	HTTPListener        PublicIPAddress `json:"httpListener"`
	BackendAddressPool  PublicIPAddress `json:"backendAddressPool"`
	BackendHTTPSettings PublicIPAddress `json:"backendHttpSettings"`
}

type SSLCertificate struct {
	Name       string                   `json:"name"`
	ID         string                   `json:"id"`
	Properties SSLCertificateProperties `json:"properties"`
}

type SSLCertificateProperties struct {
	ProvisioningState string `json:"provisioningState"`
	PublicCertData    string `json:"public_cert_data"`
	Data              string `json:"data"`
	Password          string `json:"password"`
}

type Sku struct {
	Name     string `json:"name"`
	Tier     string `json:"tier"`
	Capacity string `json:"capacity"`
}

type Subnet struct {
	ID string `json:"id"`
}

type FrontendIPConfiguration struct {
	Name       string                            `json:"name"`
	ID         string                            `json:"id"`
	Etag       string                            `json:"etag"`
	Type       string                            `json:"type"`
	Zones      []string                          `json:"zones"`
	Properties FrontendIPConfigurationProperties `json:"properties"`
}

type FrontendIPConfigurationProperties struct {
	ProvisioningState         string           `json:"provisioningState"`
	PrivateIPAddress          string           `json:"privateIPAddress"`
	PublicIPAddress           *PublicIPAddress `json:"publicIPAddress,omitempty"`
	PrivateIPAllocationMethod string           `json:"privateIPAllocationMethod"`
	PrivateIPAddressVersion   string           `json:"privateIPAddressVersion"`
	Subnet                    Subnet           `json:"subnet"`
	LoadBalancingRules        []Subnet         `json:"loadBalancingRules"`
	InboundNatRules           []Subnet         `json:"inboundNatRules"`
}

type InboundNatRule struct {
	Name       string                   `json:"name"`
	ID         string                   `json:"id"`
	Etag       string                   `json:"etag"`
	Type       string                   `json:"type"`
	Properties InboundNatRuleProperties `json:"properties"`
}

type InboundNatRuleProperties struct {
	ProvisioningState       string `json:"provisioningState"`
	FrontendIPConfiguration Subnet `json:"frontendIPConfiguration"`
	FrontendPort            int64  `json:"frontendPort"`
	BackendPort             int64  `json:"backendPort"`
	EnableFloatingIP        bool   `json:"enableFloatingIP"`
	IdleTimeoutInMinutes    int64  `json:"idleTimeoutInMinutes"`
	Protocol                string `json:"protocol"`
	EnableTCPReset          bool   `json:"enableTcpReset"`
}

type SLoadbalancerSku struct {
	Name string `json:"name"`
}

type Probe struct {
	lb *SLoadbalancer

	Name       string          `json:"name"`
	ID         string          `json:"id"`
	Etag       string          `json:"etag"`
	Type       string          `json:"type"`
	Properties ProbeProperties `json:"properties"`
}

type ProbeProperties struct {
	// tcp/udp/http/https
	ProvisioningState string `json:"provisioningState"`
	Protocol          string `json:"protocol"`
	Port              int    `json:"port"`
	RequestPath       string `json:"requestPath"`
	IntervalInSeconds int    `json:"intervalInSeconds"`
	NumberOfProbes    int    `json:"numberOfProbes"`
	// http/ https only
	Host                                string   `json:"host"`
	Path                                string   `json:"path"`
	Interval                            int      `json:"interval"`
	Timeout                             int      `json:"timeout"`
	UnhealthyThreshold                  int      `json:"unhealthyThreshold"`
	PickHostNameFromBackendHTTPSettings bool     `json:"pickHostNameFromBackendHttpSettings"`
	MinServers                          int      `json:"minServers"`
	Match                               Match    `json:"match"`
	LoadBalancingRules                  []Subnet `json:"loadBalancingRules"`
}

type Match struct {
	Body        string   `json:"body"`
	StatusCodes []string `json:"statusCodes"`
}

type LoadBalancingRule struct {
	Name       string                      `json:"name"`
	ID         string                      `json:"id"`
	Etag       string                      `json:"etag"`
	Type       string                      `json:"type"`
	Properties LoadBalancingRuleProperties `json:"properties"`
}

type LoadBalancingRuleProperties struct {
	ProvisioningState       string `json:"provisioningState"`
	FrontendIPConfiguration Subnet `json:"frontendIPConfiguration"`
	FrontendPort            int    `json:"frontendPort"`
	BackendPort             int    `json:"backendPort"`
	EnableFloatingIP        bool   `json:"enableFloatingIP"`
	IdleTimeoutInMinutes    int    `json:"idleTimeoutInMinutes"`
	Protocol                string `json:"protocol"`
	EnableTCPReset          bool   `json:"enableTcpReset"`
	LoadDistribution        string `json:"loadDistribution"`
	BackendAddressPool      Subnet `json:"backendAddressPool"`
	Probe                   Subnet `json:"probe"`
}

type HTTPListener struct {
	Name       string                 `json:"name"`
	ID         string                 `json:"id"`
	Properties HTTPListenerProperties `json:"properties"`
}

type HTTPListenerProperties struct {
	FrontendIPConfiguration     PublicIPAddress `json:"frontendIPConfiguration"`
	FrontendPort                PublicIPAddress `json:"frontendPort"`
	HostName                    string          `json:"hostName"`
	Protocol                    string          `json:"protocol"`
	SSLCertificate              PublicIPAddress `json:"sslCertificate"`
	RequestRoutingRules         []Subnet        `json:"requestRoutingRules"`
	RequireServerNameIndication bool            `json:"requireServerNameIndication"`
}

type ContentProperties struct {
	ProvisioningState           string                            `json:"provisioningState"`
	ResourceGUID                string                            `json:"resourceGuid"`
	IPConfigurations            []NetworkInterfaceIPConfiguration `json:"ipConfigurations"`
	DNSSettings                 DNSSettings                       `json:"dnsSettings"`
	MACAddress                  string                            `json:"macAddress"`
	EnableAcceleratedNetworking bool                              `json:"enableAcceleratedNetworking"`
	EnableIPForwarding          bool                              `json:"enableIPForwarding"`
	NetworkSecurityGroup        NetworkSecurityGroup              `json:"networkSecurityGroup"`
	Primary                     bool                              `json:"primary"`
	VirtualMachine              NetworkSecurityGroup              `json:"virtualMachine"`
	HostedWorkloads             []interface{}                     `json:"hostedWorkloads"`
	TapConfigurations           []interface{}                     `json:"tapConfigurations"`
}

type DNSSettings struct {
	DNSServers               []interface{} `json:"dnsServers"`
	AppliedDNSServers        []interface{} `json:"appliedDnsServers"`
	InternalDomainNameSuffix string        `json:"internalDomainNameSuffix"`
}

type NetworkInterfaceIPConfiguration struct {
	Name       string                    `json:"name"`
	ID         string                    `json:"id"`
	Etag       string                    `json:"etag"`
	Type       string                    `json:"type"`
	Properties IPConfigurationProperties `json:"properties"`
}

type IPConfigurationProperties struct {
	ProvisioningState               string                 `json:"provisioningState"`
	PrivateIPAddress                string                 `json:"privateIPAddress"`
	PrivateIPAllocationMethod       string                 `json:"privateIPAllocationMethod"`
	PublicIPAddress                 NetworkSecurityGroup   `json:"publicIPAddress"`
	Subnet                          NetworkSecurityGroup   `json:"subnet"`
	Primary                         bool                   `json:"primary"`
	PrivateIPAddressVersion         string                 `json:"privateIPAddressVersion"`
	LoadBalancerBackendAddressPools []NetworkSecurityGroup `json:"loadBalancerBackendAddressPools"`
	LoadBalancerInboundNatRules     []NetworkSecurityGroup `json:"loadBalancerInboundNatRules"`
}

type NetworkSecurityGroup struct {
	ID string `json:"id"`
}
