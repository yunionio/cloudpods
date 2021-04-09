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

package cloudprovider

import (
	"fmt"
	"sort"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
)

type SAccessGroup struct {
	Name           string
	NetworkType    string
	FileSystemType string
	Desc           string
}

type TRWAccessType string
type TUserAccessType string

const (
	RWAccessTypeRW = TRWAccessType("RW")
	RWAccessTypeR  = TRWAccessType("R")

	UserAccessTypeNoRootSquash = TUserAccessType("no_root_squash")
	UserAccessTypeRootSquash   = TUserAccessType("root_squash")
	UserAccessTypeAllSquash    = TUserAccessType("all_squash")
)

func GetAccessGroupRuleInfo(group ICloudAccessGroup) (AccessGroupRuleInfo, error) {
	ret := AccessGroupRuleInfo{
		MinPriority:             group.GetMinPriority(),
		MaxPriority:             group.GetMaxPriority(),
		SupportedUserAccessType: group.GetSupporedUserAccessTypes(),
	}
	var err error
	ret.Rules, err = group.GetRules()
	if err != nil {
		return ret, errors.Wrapf(err, "GetRules")
	}
	return ret, nil
}

type AccessGroupRule struct {
	Id             string
	ExternalId     string
	Priority       int
	RWAccessType   TRWAccessType
	UserAccessType TUserAccessType
	Source         string
}

func (self AccessGroupRule) String() string {
	return fmt.Sprintf("%s-%s-%s", self.RWAccessType, self.UserAccessType, self.Source)
}

type AccessGroupRuleSet []AccessGroupRule

func (srs AccessGroupRuleSet) Len() int {
	return len(srs)
}

func (srs AccessGroupRuleSet) Swap(i, j int) {
	srs[i], srs[j] = srs[j], srs[i]
}

func (srs AccessGroupRuleSet) Less(i, j int) bool {
	return srs[i].Priority < srs[j].Priority || (srs[i].Priority == srs[j].Priority && srs[i].String() < srs[j].String())
}

type AccessGroupRuleInfo struct {
	MaxPriority             int
	MinPriority             int
	SupportedUserAccessType []TUserAccessType
	Rules                   AccessGroupRuleSet
}

func (self *AccessGroupRuleInfo) Sort() {
	if self.MinPriority < self.MaxPriority {
		sort.Sort(self.Rules)
		return
	}
	sort.Sort(sort.Reverse(self.Rules))
}

func CompareAccessGroupRules(src, dest AccessGroupRuleInfo, debug bool) (common, added, removed AccessGroupRuleSet) {
	src.Sort()
	dest.Sort()
	var addPriority = func(init int, min, max int) int {
		inc := 1
		if max < min {
			max, min, inc = min, max, -1
		}
		if init >= max || init <= min {
			return init
		}
		return init + inc
	}
	i, j, priority := 0, 0, (dest.MinPriority-1+dest.MaxPriority)/2
	for i < len(src.Rules) || j < len(dest.Rules) {
		if i < len(src.Rules) && j < len(dest.Rules) {
			destRuleStr := dest.Rules[j].String()
			srcRuleStr := src.Rules[i].String()
			if debug {
				log.Debugf("compare src %s priority(%d) %s -> dest %s priority(%d) %s\n",
					src.Rules[i].ExternalId, src.Rules[i].Priority, src.Rules[i].String(),
					dest.Rules[j].ExternalId, dest.Rules[j].Priority, dest.Rules[j].String())
			}
			cmp := strings.Compare(destRuleStr, srcRuleStr)
			if cmp == 0 {
				dest.Rules[j].Id = src.Rules[i].Id
				common = append(common, dest.Rules[j])
				priority = dest.Rules[j].Priority
				i++
				j++
			} else if cmp < 0 {
				removed = append(removed, dest.Rules[j])
				j++
			} else {
				if isIn, _ := utils.InArray(src.Rules[i].UserAccessType, dest.SupportedUserAccessType); isIn {
					priority = addPriority(priority, dest.MinPriority, dest.MaxPriority)
					src.Rules[i].Priority = priority
					added = append(added, src.Rules[i])
				}
				i++
			}
		} else if i >= len(src.Rules) {
			removed = append(removed, dest.Rules[j])
			j++
		} else if j >= len(dest.Rules) {
			if isIn, _ := utils.InArray(src.Rules[i].UserAccessType, dest.SupportedUserAccessType); isIn {
				priority = addPriority(priority, dest.MinPriority, dest.MaxPriority)
				src.Rules[i].Priority = priority
				added = append(added, src.Rules[i])
			}
			i++
		}
	}
	return common, added, removed
}
