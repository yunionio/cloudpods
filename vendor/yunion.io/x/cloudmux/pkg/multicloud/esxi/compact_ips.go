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

package esxi

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type sCompactIPs map[string][]int

type sCompactIPItem struct {
	prefix string
	host   []int
}

type sCompactIPList []sCompactIPItem

func (cip sCompactIPs) append(ip string) {
	seg := strings.Split(ip, ".")
	prefix := strings.Join(seg[:3], ".")
	host, _ := strconv.Atoi(seg[3])
	cip[prefix] = append(cip[prefix], int(host))
}

func (cip sCompactIPs) items() []sCompactIPItem {
	ret := make([]sCompactIPItem, 0)
	for k := range cip {
		sort.Ints(cip[k])
		ret = append(ret, sCompactIPItem{
			prefix: k,
			host:   cip[k],
		})
	}
	return ret
}

func (item sCompactIPItem) cost() int {
	segs := strings.Split(item.prefix, ".")
	cost := 0
	for _, seg := range segs {
		segInt, _ := strconv.Atoi(seg)
		cost += cost*255 + segInt
	}
	return cost
}

func (item sCompactIPItem) String() string {
	lastHost := -1
	hostStr := strings.Builder{}
	span := 0
	for i := 0; i <= len(item.host); i++ {
		if lastHost < 0 || i == len(item.host) || lastHost+1 < item.host[i] {
			if span > 1 {
				hostStr.WriteString(fmt.Sprintf("-%d", lastHost))
			} else if span == 1 {
				hostStr.WriteString(fmt.Sprintf(",%d", lastHost))
			}
			if i < len(item.host) {
				if i > 0 {
					hostStr.WriteByte(',')
				}
				hostStr.WriteString(fmt.Sprintf("%d", item.host[i]))
				lastHost = item.host[i]
			}
			span = 0 // reset
		} else if lastHost+1 == item.host[i] {
			span++
			lastHost = item.host[i]
		}
	}
	return item.prefix + "." + hostStr.String()
}

func (a sCompactIPList) Len() int           { return len(a) }
func (a sCompactIPList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a sCompactIPList) Less(i, j int) bool { return a[i].cost() < a[j].cost() }

func compactIPs(ips []string) string {
	cip := sCompactIPs{}
	for _, ip := range ips {
		cip.append(ip)
	}
	items := cip.items()
	sort.Sort(sCompactIPList(items))
	lines := make([]string, len(items))
	for i := range items {
		lines[i] = items[i].String()
	}
	return strings.Join(lines, ";")
}

func compactMacs(macs []string) string {
	ret := make([]string, 0)
	sort.Strings(macs)
	for i := range macs {
		if len(ret) == 0 || ret[len(ret)-1] != macs[i] {
			ret = append(ret, macs[i])
		}
	}
	return strings.Join(macs, ",")
}
