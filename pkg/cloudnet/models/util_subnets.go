package models

import (
	"strings"

	"yunion.io/x/pkg/util/netutils"
)

type Subnets []*netutils.IPV4Prefix

func (nets Subnets) StrList() []string {
	r := make([]string, 0, len(nets))
	for _, p := range nets {
		r = append(r, p.String())
	}
	return r
}

func (nets Subnets) String() string {
	r := nets.StrList()
	return strings.Join(r, ",")
}

func (nets Subnets) ContainsAny(nets1 Subnets) bool {
	contains, _ := nets.ContainsAnyEx(nets1)
	return contains
}

func (nets Subnets) ContainsAnyEx(nets1 Subnets) (bool, *netutils.IPV4Prefix) {
	for _, p0 := range nets {
		for _, p1 := range nets1 {
			if p0.Equals(p1) {
				return true, p0
			}
		}
	}
	return false, nil
}
