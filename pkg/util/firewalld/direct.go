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

package firewalld

import (
	"encoding/xml"
)

type Direct struct {
	Rules []*Rule

	XMLName struct{} `xml:"direct"`
}

func (d *Direct) String() string {
	data, _ := xml.MarshalIndent(d, "", "  ")
	return string(data)
}

type Rule struct {
	// required, smaller the number more front the rule in chain
	Priority int `xml:"priority,attr"`
	// required, netfilter table: "nat", "mangle", etc.
	Table string `xml:"table,attr"`
	// required, ip family: "ipv4", "ipv6", "eb"
	IPv string `xml:"ipv,attr"`
	// required, netfilter chain: "FORWARD", custom chain names
	Chain string `xml:"chain,attr"`

	// match and action command line options for {ip,ip6,eb}tables
	Body string `xml:",chardata"`

	XMLName struct{} `xml:"rule"`
}

func NewIP4Rule(prio int, table, chain, body string) *Rule {
	r := &Rule{
		Priority: prio,
		IPv:      "ipv4",
		Table:    table,
		Chain:    chain,
		Body:     body,
	}
	return r
}

func (r *Rule) String() string {
	data, _ := xml.MarshalIndent(r, "", "  ")
	return string(data)
}

func NewDirect(rules ...*Rule) *Direct {
	d := &Direct{
		Rules: rules,
	}
	return d
}
