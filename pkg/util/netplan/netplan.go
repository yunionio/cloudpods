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

package netplan

import "yunion.io/x/jsonutils"

// Configuration examples reference from https://netplan.io/examples/
// manpage: http://manpages.ubuntu.com/manpages/cosmic/man5/netplan.5.html
type Configuration struct {
	Network *Network `json:"network"`
}

func NewConfiguration(network *Network) *Configuration {
	return &Configuration{
		Network: network,
	}
}

func (c *Configuration) YAMLString() string {
	return toYAMLString(c)
}

type NetworkRenderer string

const (
	VERSION2                                = 2
	NetworkRendererNetworkd NetworkRenderer = "networkd"
)

type Network struct {
	Version   uint                       `json:"version"`
	Renderer  NetworkRenderer            `json:"renderer"`
	Ethernets map[string]*EthernetConfig `json:"ethernets"`
	Bonds     map[string]*Bond           `json:"bonds"`
}

type EthernetConfigMatch struct {
	MacAddress string `json:"macaddress"`
}

func NewEthernetConfigMatchMac(macAddr string) *EthernetConfigMatch {
	return &EthernetConfigMatch{
		MacAddress: macAddr,
	}
}

type EthernetConfig struct {
	DHCP4       bool                 `json:"dhcp4,omitfalse"`
	DHCP6       bool                 `json:"dhcp6,omitfalse"`
	Addresses   []string             `json:"addresses"`
	Match       *EthernetConfigMatch `json:"match"`
	MacAddress  string               `json:"macaddress"`
	Gateway4    string               `json:"gateway4"`
	Gateway6    string               `json:"gateway6"`
	Routes      []*Route             `json:"routes"`
	Nameservers *Nameservers         `json:"nameservers"`
	Mtu         int16                `json:"mtu,omitzero"`
}

type Route struct {
	To     string `json:"to"`
	Via    string `json:"via"`
	Metric uint   `json:"metric"`
	// OnLink bool `json:"on-link"`
}

type Nameservers struct {
	Search    []string `json:"search"`
	Addresses []string `json:"addresses"`
}

type Bond struct {
	EthernetConfig
	Interfaces []string        `json:"interfaces"`
	Parameters IBondModeParams `json:"parameters"`
}

func toYAMLString(obj interface{}) string {
	return jsonutils.Marshal(obj).YAMLString()
}

func (b *Bond) YAMLString() string {
	return toYAMLString(b)
}

type BondMode string

const (
	// ref: https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/7/html/networking_guide/overview-of-bonding-modes-and-the-required-settings-on-the-switch
	// mode0
	bondModeBalanceRR = "balance-rr"
	// mode1
	bondModeActiveBackup = "active-backup"
	// mode4
	bondMode8023AD = "802.3ad"
)

type IBondModeParams interface {
	GetMode() string
}

type BondModeBaseParams struct {
	Mode               string `json:"mode"`
	MiiMonitorInterval int    `json:"mii-monitor-interval,omitzero"`
	GratuitiousArp     int    `json:"gratuitiousi-arp,omitzero"`
}

func (c BondModeBaseParams) GetMode() string {
	return c.Mode
}

func (c *BondModeBaseParams) SetMiiMonitorInterval(i int) {
	c.MiiMonitorInterval = i
}

func (c *BondModeBaseParams) SetGratutiousArp(g int) {
	c.GratuitiousArp = g
}

type BondModeActiveBackupParams struct {
	*BondModeBaseParams

	Primary string `json:"primary"`
}

func NewBondMode0Params() *BondModeBaseParams {
	return &BondModeBaseParams{
		Mode: bondModeBalanceRR,
	}
}

func NewBondModeActiveBackupParams(primary string) *BondModeActiveBackupParams {
	return &BondModeActiveBackupParams{
		BondModeBaseParams: &BondModeBaseParams{
			Mode: bondModeActiveBackup,
		},
		Primary: primary,
	}
}

type BondMode4Params struct {
	*BondModeBaseParams
}

func NewBondMode4Params() *BondMode4Params {
	return &BondMode4Params{
		BondModeBaseParams: &BondModeBaseParams{
			Mode: bondMode8023AD,
		},
	}
}

func NewNetwork() *Network {
	return &Network{
		Version:   VERSION2,
		Renderer:  NetworkRendererNetworkd,
		Ethernets: make(map[string]*EthernetConfig),
		Bonds:     make(map[string]*Bond),
	}
}

func (n *Network) AddEthernet(name string, ether *EthernetConfig) *Network {
	n.Ethernets[name] = ether
	return n
}

func (n *Network) AddBond(name string, bond *Bond) *Network {
	n.Bonds[name] = bond
	return n
}

func (n *Network) YAMLString() string {
	return toYAMLString(n)
}

func NewDHCP4EthernetConfig() *EthernetConfig {
	return &EthernetConfig{
		DHCP4: true,
	}
}

func (c *EthernetConfig) EnableDHCP6() {
	c.DHCP6 = true
}

func NewStaticEthernetConfig(
	addr string,
	addr6 string,
	gateway string,
	gateway6 string,
	search []string,
	nameservers []string,
	routes []*Route,
) *EthernetConfig {
	addrs := []string{
		addr,
	}
	if len(addr6) > 0 {
		addrs = append(addrs, addr6)
	}
	return &EthernetConfig{
		DHCP4:     false,
		Addresses: addrs,
		Gateway4:  gateway,
		Gateway6:  gateway6,
		Routes:    routes,
		Nameservers: &Nameservers{
			Search:    search,
			Addresses: nameservers,
		},
	}
}

func (c *EthernetConfig) YAMLString() string {
	return toYAMLString(c)
}

func newBondModeByParams(conf *EthernetConfig, interfaces []string, params IBondModeParams) *Bond {
	return &Bond{
		EthernetConfig: *conf,
		Interfaces:     interfaces,
		Parameters:     params,
	}
}

func NewBondMode0(conf *EthernetConfig, interfaces []string) *Bond {
	params := NewBondMode0Params()
	params.SetMiiMonitorInterval(100)
	return newBondModeByParams(conf, interfaces, params)
}

func NewBondMode1(conf *EthernetConfig, interfaces []string) *Bond {
	params := NewBondModeActiveBackupParams(interfaces[0])
	params.SetMiiMonitorInterval(100)
	return newBondModeByParams(conf, interfaces, params)
}
func NewBondMode4(conf *EthernetConfig, interfaces []string) *Bond {
	params := NewBondMode4Params()
	// TODO: figure out what follows options related to netplan config
	// miimon: 1
	// lacp_rate: 1
	// xmit_hash_policy: 1
	params.SetMiiMonitorInterval(100)
	return newBondModeByParams(conf, interfaces, params)
}
