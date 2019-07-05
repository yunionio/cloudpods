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

package secrules

import (
	"bytes"
	"sort"
)

type SecurityRuleSet []SecurityRule

func (srs SecurityRuleSet) Len() int {
	return len(srs)
}

func (srs SecurityRuleSet) Swap(i, j int) {
	srs[i], srs[j] = srs[j], srs[i]
}

func (srs SecurityRuleSet) Less(i, j int) bool {
	if srs[i].Priority > srs[j].Priority {
		return true
	} else if srs[i].Priority == srs[j].Priority {
		return srs[i].String() < srs[j].String()
	}
	return false
}

func (srs SecurityRuleSet) stringList() []string {
	r := make([]string, len(srs))
	for i := range srs {
		r = append(r, srs[i].String())
	}
	return r
}

func (srs SecurityRuleSet) String() string {
	buf := bytes.Buffer{}
	for i := range srs {
		buf.WriteString(srs[i].String())
		buf.WriteString(";")
	}
	return buf.String()
}

func (srs SecurityRuleSet) equals(srs1 SecurityRuleSet) bool {
	if len(srs) != len(srs1) {
		return false
	}
	for i := range srs {
		if !srs[i].equals(&srs1[i]) {
			return false
		}
	}
	return true
}

// convert to pure allow list
//
// requirements on srs
//
//  - ordered by priority
//  - same direction
//
func (srs SecurityRuleSet) AllowList() SecurityRuleSet {
	r := SecurityRuleSet{}
	wq := make(SecurityRuleSet, len(srs))
	copy(wq, srs)

	for len(wq) > 0 {
		sr := wq[0]
		if sr.Action == SecurityRuleAllow {
			r = append(r, sr)
			wq = wq[1:]
			continue
		}
		wq = wq.cutOutFirst()
	}
	r = r.collapse()
	return r
}

func (srs SecurityRuleSet) cutOutFirst() SecurityRuleSet {
	r := SecurityRuleSet{}
	if len(srs) == 0 {
		return r
	}
	sr := &srs[0]
	srs_ := srs[1:]

	for _, sr_ := range srs_ {
		if sr.Action == sr_.Action {
			r = append(r, sr_)
			continue
		}
		cut := sr_.cutOut(sr)
		r = append(r, cut...)
	}
	return r
}

// collapse result of AllowList
//
//  - same direction
//  - same action
//
//  As they share the same action, priority's influence on order of rules can be ignored
//
func (srs SecurityRuleSet) collapse() SecurityRuleSet {
	srs1 := make(SecurityRuleSet, len(srs))
	copy(srs1, srs)
	for i := range srs1 {
		sr := &srs1[i]
		if len(sr.Ports) > 0 {
			sort.Slice(sr.Ports, func(i, j int) bool {
				return sr.Ports[i] < sr.Ports[j]
			})
		}
	}
	sort.Slice(srs1, func(i, j int) bool {
		sr0 := &srs1[i]
		sr1 := &srs1[j]
		if sr0.Protocol != sr1.Protocol {
			return sr0.Protocol < sr1.Protocol
		}
		net0 := sr0.IPNet.String()
		net1 := sr1.IPNet.String()
		if net0 != net1 {
			return net0 < net1
		}
		if sr0.PortStart > 0 && sr0.PortEnd > 0 {
			if sr1.PortStart > 0 && sr1.PortEnd > 0 {
				return sr0.PortStart < sr1.PortStart
			}
			// port range comes first
			return true
		} else if len(sr0.Ports) > 0 {
			if sr1.PortStart > 0 && sr1.PortEnd > 0 {
				return false
			} else if len(sr1.Ports) > 0 {
				sr0l := len(sr0.Ports)
				sr1l := len(sr1.Ports)
				for i := 0; i < sr0l && i < sr1l; i++ {
					if sr0.Ports[i] != sr1.Ports[i] {
						return sr0.Ports[i] < sr1.Ports[i]
					}
				}
				return sr0l < sr1l
			}
		}
		return sr0.Priority < sr1.Priority
	})
	// merge ports
	for i := len(srs1) - 1; i > 0; i-- {
		sr0 := &srs1[i-1]
		sr1 := &srs1[i]
		if sr0.Protocol != sr1.Protocol {
			continue
		}
		if !sr0.netEquals(sr1) {
			continue
		}
		if len(sr0.Ports) > 0 && len(sr1.Ports) > 0 {
			ps := newPortsFromInts(sr0.Ports...)
			ps = append(ps, newPortsFromInts(sr1.Ports...)...)
			ps = ps.dedup()
			sr0.Ports = ps.IntSlice()
			srs1 = append(srs1[:i], srs1[i+1:]...)
		} else if sr0.PortStart > 0 && sr1.PortStart > 0 && sr0.PortEnd > 0 && sr1.PortEnd > 0 {
			if sr0.PortEnd == sr1.PortStart-1 {
				sr0.PortEnd = sr1.PortEnd
				srs1 = append(srs1[:i], srs1[i+1:]...)
			} else if sr0.PortStart-1 == sr1.PortEnd {
				sr0.PortStart = sr1.PortStart
				srs1 = append(srs1[:i], srs1[i+1:]...)
			} else if sr0.PortStart == sr1.PortStart && sr0.PortEnd == sr1.PortEnd {
				srs1 = append(srs1[:i], srs1[i+1:]...)
			}
			// save that contains, intersects
		}
	}
	return srs1
}
