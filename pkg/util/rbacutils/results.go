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

package rbacutils

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/util/tagutils"
)

type SPolicyMatch struct {
	Rule        SRbacRule
	DomainTags  tagutils.TTagSetList
	ProjectTags tagutils.TTagSetList
	ObjectTags  tagutils.TTagSetList
}

type SPolicyResult struct {
	Result      TRbacResult
	DomainTags  tagutils.TTagSetList
	ProjectTags tagutils.TTagSetList
	ObjectTags  tagutils.TTagSetList
}

var (
	PolicyDeny = SPolicyResult{
		Result: Deny,
	}
	PolicyAllow = SPolicyResult{
		Result: Allow,
	}
)

type TPolicyMatches []SPolicyMatch

func (matches TPolicyMatches) GetResult() SPolicyResult {
	result := SPolicyResult{
		Result: Deny,
	}
	isWideDomainTag, isWideProjectTag, isWideObjectTag := false, false, false
	for _, match := range matches {
		if match.Rule.Result == Allow {
			result.Result = Allow
			result.DomainTags = result.DomainTags.AppendAll(match.DomainTags)
			result.ProjectTags = result.ProjectTags.AppendAll(match.ProjectTags)
			result.ObjectTags = result.ObjectTags.AppendAll(match.ObjectTags)
			if len(match.DomainTags) == 0 {
				isWideDomainTag = true
			}
			if len(match.ProjectTags) == 0 {
				isWideProjectTag = true
			}
			if len(match.ObjectTags) == 0 {
				isWideObjectTag = true
			}
		}
	}
	if isWideDomainTag {
		result.DomainTags = tagutils.TTagSetList{}
	}
	if isWideProjectTag {
		result.ProjectTags = tagutils.TTagSetList{}
	}
	if isWideObjectTag {
		result.ObjectTags = tagutils.TTagSetList{}
	}
	return result
}

func (result SPolicyResult) String() string {
	return fmt.Sprintf("[%s] domain:%s project:%s object:%s", result.Result, result.DomainTags.String(), result.ProjectTags.String(), result.ObjectTags.String())
}

func (result SPolicyResult) IsEmpty() bool {
	return len(result.ObjectTags) == 0 && len(result.ProjectTags) == 0 && len(result.DomainTags) == 0
}

func (result SPolicyResult) Json() jsonutils.JSONObject {
	ret := jsonutils.NewDict()
	if len(result.ObjectTags) > 0 {
		ret.Add(jsonutils.Marshal(result.ObjectTags), "policy_object_tags")
	}
	if len(result.ProjectTags) > 0 {
		ret.Add(jsonutils.Marshal(result.ProjectTags), "policy_project_tags")
	}
	if len(result.DomainTags) > 0 {
		ret.Add(jsonutils.Marshal(result.DomainTags), "policy_domain_tags")
	}
	return ret
}

func mergeTagList(t1, t2 tagutils.TTagSetList) tagutils.TTagSetList {
	return t1.IntersectList(t2)
}

func (r1 SPolicyResult) Merge(r2 SPolicyResult) SPolicyResult {
	if r1.Result.IsDeny() || r2.Result.IsDeny() {
		return SPolicyResult{Result: Deny}
	}
	r1.ProjectTags = mergeTagList(r1.ProjectTags, r2.ProjectTags)
	r1.DomainTags = mergeTagList(r1.DomainTags, r2.DomainTags)
	r1.ObjectTags = mergeTagList(r1.ObjectTags, r2.ObjectTags)
	return r1
}
