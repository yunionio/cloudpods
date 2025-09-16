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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEthernetConfig(t *testing.T) {
	c := NewDHCP4EthernetConfig()

	assert := assert.New(t)
	assert.YAMLEq("dhcp4: true", c.YAMLString())

	c.EnableDHCP6()
	assert.YAMLEq("dhcp4: true\ndhcp6: true", c.YAMLString())
}

func TestNewBondMode4(t *testing.T) {
	c := NewBondMode4(
		&EthernetConfig{
			DHCP4:     false,
			DHCP6:     false,
			Gateway4:  "192.168.1.1",
			Gateway6:  "fd:3ffe:3200::1",
			Addresses: []string{"192.168.1.252/24", "fd:3ffe:3200::2/64"},
			Nameservers: &Nameservers{
				Search:    []string{"local"},
				Addresses: []string{"8.8.8.8", "8.8.4.4"},
			},
		},
		[]string{"enp2s0", "enp3s0"},
	)

	yamlStr := `
addresses:
- 192.168.1.252/24
- fd:3ffe:3200::2/64
gateway4: 192.168.1.1
gateway6: fd:3ffe:3200::1
interfaces:
- enp2s0
- enp3s0
nameservers:
  addresses:
  - 8.8.8.8
  - 8.8.4.4
  search:
  - local
parameters:
  mii-monitor-interval: 100
  mode: "802.3ad"
`

	assert := assert.New(t)
	assert.YAMLEq(yamlStr, c.YAMLString())
}

func TestNewNetwork(t *testing.T) {
	n := NewNetwork()
	n.AddEthernet("eth0", NewDHCP4EthernetConfig())
	n.AddEthernet("eth1", NewStaticEthernetConfig(
		"10.10.10.2/24",
		"",
		"10.10.10.1",
		"",
		[]string{"mydomain", "otherdomain"},
		[]string{"114.114.114.114"},
		nil,
	))

	c := NewConfiguration(n)

	yamlStr := `
network:
  ethernets:
    eth0:
      dhcp4: true
    eth1:
      addresses:
      - 10.10.10.2/24
      gateway4: 10.10.10.1
      nameservers:
        addresses:
        - 114.114.114.114
        search: ["mydomain", "otherdomain"]
  renderer: networkd
  version: 2
`

	assert := assert.New(t)
	assert.YAMLEq(yamlStr, c.YAMLString())
}
