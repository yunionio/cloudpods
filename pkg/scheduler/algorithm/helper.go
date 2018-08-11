package algorithm

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/scheduler/cache/candidate"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

func ToHostCandidate(c core.Candidater) (*candidate.HostDesc, error) {
	d, ok := c.(*candidate.HostDesc)
	if !ok {
		return nil, fmt.Errorf("Can't convert %#v to '*candidate.HostDesc'", c)
	}
	return d, nil
}

func ToBaremetalCandidate(c core.Candidater) (*candidate.BaremetalDesc, error) {
	d, ok := c.(*candidate.BaremetalDesc)
	if !ok {
		return nil, fmt.Errorf("Can't convert %#v to '*candidate.BaremetalDesc'", c)
	}
	return d, nil
}
