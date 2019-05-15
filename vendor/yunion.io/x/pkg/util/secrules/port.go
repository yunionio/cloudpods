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
	"sort"
)

type ports []uint16

func newPortsFromInts(ints ...int) ports {
	ps := make(ports, len(ints))
	for i := range ints {
		// panic on invalid value
		ps[i] = uint16(ints[i])
	}
	return ps
}

func (ps ports) IntSlice() []int {
	ints := make([]int, len(ps))
	for i := range ps {
		ints[i] = int(ps[i])
	}
	return ints
}

func (ps ports) Len() int {
	return len(ps)
}

func (ps ports) Swap(i, j int) {
	ps[i], ps[j] = ps[j], ps[i]
}

func (ps ports) Less(i, j int) bool {
	return ps[i] < ps[j]
}

func (ps ports) contains(p uint16) bool {
	for _, p0 := range ps {
		if p0 == p {
			return true
		}
	}
	return false
}

func (ps ports) containsPorts(ps1 ports) bool {
	for _, p := range ps1 {
		if !ps.contains(p) {
			return false
		}
	}
	return true
}

func (ps ports) dedup() ports {
	ps1 := make(ports, len(ps))
	copy(ps1, ps)
	sort.Sort(ps1)
	for i := len(ps1) - 1; i > 0; i-- {
		if ps1[i] == ps1[i-1] {
			ps1 = append(ps1[:i], ps1[i+1:]...)
		}
	}
	return ps1
}

func (ps ports) sameAs(ps1 ports) bool {
	if len(ps) != len(ps1) {
		return false
	}
	for i := range ps {
		if ps[i] != ps1[i] {
			return false
		}
	}
	return true
}

func (ps ports) substractPortRange(pr *portRange) (left, subs ports) {
	left = ports{}
	subs = ports{}
	for _, p := range ps {
		if pr.contains(p) {
			subs = append(subs, p)
		} else {
			left = append(left, p)
		}
	}
	return
}

func (ps ports) substractPorts(ps1 ports) (left, subs ports) {
	left = ports{}
	subs = ports{}
	for _, p0 := range ps {
		if ps1.contains(p0) {
			subs = append(subs, p0)
		} else {
			left = append(left, p0)
		}
	}
	return
}

type portRange struct {
	start uint16
	end   uint16
}

func newPortRange(s, e uint16) *portRange {
	// panic on s > e
	return &portRange{
		start: s,
		end:   e,
	}
}

func (pr *portRange) equals(pr1 *portRange) bool {
	return pr.start == pr1.start && pr.end == pr1.end
}

func (pr *portRange) contains(p uint16) bool {
	return p >= pr.start && p <= pr.end
}

func (pr *portRange) containsRange(pr1 *portRange) bool {
	return pr.start <= pr1.start && pr.end >= pr1.end
}

func (pr *portRange) count() uint16 {
	return pr.end - pr.start + 1
}

func (pr *portRange) substractPortRange(pr1 *portRange) (lefts []*portRange, sub *portRange) {
	// no intersection, no substract
	if pr.end < pr1.start || pr.start > pr1.end {
		l := *pr
		lefts = []*portRange{&l}
		return
	}

	// pr contains pr1
	if pr.containsRange(pr1) {
		nns := [][2]int32{
			[2]int32{int32(pr.start), int32(pr1.start) - 1},
			[2]int32{int32(pr1.end) + 1, int32(pr.end)},
		}
		lefts = []*portRange{}
		for _, nn := range nns {
			if nn[0] <= nn[1] {
				lefts = append(lefts, &portRange{
					start: uint16(nn[0]),
					end:   uint16(nn[1]),
				})
			}
		}
		s := *pr1
		sub = &s
		return
	}

	// pr contained by pr1
	if pr1.containsRange(pr) {
		s := *pr
		sub = &s
		return
	}

	// intersect, pr on the left
	if pr.start < pr1.start && pr.end >= pr1.start {
		lefts = []*portRange{&portRange{start: pr.start, end: pr1.start - 1}}
		sub = &portRange{pr1.start, pr.end}
		return
	}

	// intersect, pr on the right
	if pr.start <= pr1.end && pr.end > pr1.end {
		lefts = []*portRange{&portRange{pr1.end + 1, pr.end}}
		sub = &portRange{pr.start, pr1.end}
		return
	}

	// no intersection
	return
}

func (pr *portRange) substractPorts(ps1 ports) (lefts []*portRange, subs ports) {
	// no duplicate
	// then ordered
	ps2 := make(ports, len(ps1))
	copy(ps2, ps1)
	sort.Sort(ps2)

	lefts = []*portRange{}
	subs = ports{}
	s := pr.start
	for _, p := range ps2 {
		if pr.contains(p) {
			if p > s {
				lefts = append(lefts, &portRange{
					start: s,
					end:   p - 1,
				})
			}
			subs = append(subs, p)
			s = p + 1
		}
	}
	if s != 0 && s <= pr.end {
		lefts = append(lefts, &portRange{
			start: s,
			end:   pr.end,
		})
	}
	return
}
