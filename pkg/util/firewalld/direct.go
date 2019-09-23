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
