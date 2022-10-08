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

package predicates

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/data_manager/schedtag"
	"yunion.io/x/onecloud/pkg/util/conditionparser"
)

type ISchedtagPredicate interface {
	GetExcludeTags() []computeapi.SchedtagConfig
	GetRequireTags() []computeapi.SchedtagConfig
	GetAvoidTags() []computeapi.SchedtagConfig
	GetPreferTags() []computeapi.SchedtagConfig
}

type ISchedtagCandidate interface {
	GetId() string
	IndexKey() string
	ResourceType() string
	// GetSchedtags return schedtags bind to this candidate
	GetSchedtags() []schedtag.ISchedtag
	// GetDynamicSchedDesc return schedule description used by dynamic schedtags condition eval
	GetDynamicSchedDesc() *jsonutils.JSONDict
}

type SchedtagPredicate struct {
	requireTags  []computeapi.SchedtagConfig
	execludeTags []computeapi.SchedtagConfig
	preferTags   []computeapi.SchedtagConfig
	avoidTags    []computeapi.SchedtagConfig
	checker      *SchedtagChecker
}

func NewSchedtagPredicate(reqTags []*computeapi.SchedtagConfig, allTags []schedtag.ISchedtag) *SchedtagPredicate {
	p := new(SchedtagPredicate)
	requireTags, execludeTags, preferTags, avoidTags := GetRequestSchedtags(reqTags, allTags)
	p.requireTags = requireTags
	p.execludeTags = execludeTags
	p.preferTags = preferTags
	p.avoidTags = avoidTags
	p.checker = &SchedtagChecker{}
	return p
}

func (p *SchedtagPredicate) GetExcludeTags() []computeapi.SchedtagConfig {
	return p.execludeTags
}

func (p *SchedtagPredicate) GetRequireTags() []computeapi.SchedtagConfig {
	return p.requireTags
}

func (p *SchedtagPredicate) GetAvoidTags() []computeapi.SchedtagConfig {
	return p.avoidTags
}

func (p *SchedtagPredicate) GetPreferTags() []computeapi.SchedtagConfig {
	return p.preferTags
}

func (p *SchedtagPredicate) Check(candidate ISchedtagCandidate) error {
	return p.checker.Check(p, candidate)
}

func GetSchedtagCount(inTags []computeapi.SchedtagConfig, objTags []schedtag.ISchedtag, strategy string) (countMap map[string]int) {
	countMap = make(map[string]int)

	in := func(objTag schedtag.ISchedtag, inTags []computeapi.SchedtagConfig) (bool, int) {
		for _, tag := range inTags {
			if tag.Id == objTag.GetId() || tag.Id == objTag.GetName() {
				return true, tag.Weight
			}
		}
		return false, 0
	}

	for _, objTag := range objTags {
		if ok, weight := in(objTag, inTags); ok {
			key := fmt.Sprintf("%s:%s:%s", objTag.GetId(), objTag.GetName(), strategy)
			score, ok := countMap[key]
			if ok {
				score += weight
			} else {
				score = weight
			}
			countMap[key] = score
		}
	}
	return
}

func GetRequestSchedtags(reqTags []*computeapi.SchedtagConfig, allTags []schedtag.ISchedtag) (requireTags, execludeTags, preferTags, avoidTags []computeapi.SchedtagConfig) {
	requireTags = make([]computeapi.SchedtagConfig, 0)
	execludeTags = make([]computeapi.SchedtagConfig, 0)
	preferTags = make([]computeapi.SchedtagConfig, 0)
	avoidTags = make([]computeapi.SchedtagConfig, 0)

	appendedTagIds := make(map[string]int)

	for _, reqTag := range reqTags {
		for _, dbTag := range allTags {
			if reqTag.Id == dbTag.GetId() || reqTag.Id == dbTag.GetName() {
				if reqTag.Strategy == "" {
					reqTag.Strategy = dbTag.GetDefaultStrategy()
				}
			}
		}
	}

	appendTagByStrategy := func(tag *computeapi.SchedtagConfig, defaultWeight int) {
		if tag.Weight <= 0 {
			tag.Weight = defaultWeight
		}
		switch tag.Strategy {
		case models.STRATEGY_REQUIRE:
			requireTags = append(requireTags, *tag)
		case models.STRATEGY_EXCLUDE:
			execludeTags = append(execludeTags, *tag)
		case models.STRATEGY_PREFER:
			preferTags = append(preferTags, *tag)
		case models.STRATEGY_AVOID:
			avoidTags = append(avoidTags, *tag)
		}
	}

	for _, tag := range reqTags {
		appendTagByStrategy(tag, 10)

		appendedTagIds[tag.Id] = 1
	}

	for _, tag := range allTags {
		_, nameOk := appendedTagIds[tag.GetName()]
		_, idOk := appendedTagIds[tag.GetId()]

		if !(nameOk || idOk) {
			apiTag := &computeapi.SchedtagConfig{Id: tag.GetId(), Strategy: tag.GetDefaultStrategy()}
			appendTagByStrategy(apiTag, 1)
		}
	}

	return
}

type SchedtagChecker struct {
}

type apiTags []computeapi.SchedtagConfig

func (t apiTags) contains(objTag schedtag.ISchedtag) bool {
	for _, tag := range t {
		if tag.Id == objTag.GetId() || tag.Id == objTag.GetName() {
			return true
		}
	}
	return false
}

type objTags []schedtag.ISchedtag

func (t objTags) contains(atag computeapi.SchedtagConfig) bool {
	for _, tag := range t {
		if tag.GetId() == atag.Id || tag.GetName() == atag.Id {
			return true
		}
	}
	return false
}

func (c *SchedtagChecker) contains(tags []computeapi.SchedtagConfig, objTag models.SSchedtag) bool {
	for _, tag := range tags {
		if tag.Id == objTag.Id || tag.Id == objTag.Name {
			return true
		}
	}
	return false
}

func (c *SchedtagChecker) HasIntersection(tags []computeapi.SchedtagConfig, objTags []schedtag.ISchedtag) (bool, schedtag.ISchedtag) {
	var atags apiTags = tags
	for _, objTag := range objTags {
		if atags.contains(objTag) {
			return true, objTag
		}
	}
	return false, nil
}

func (c *SchedtagChecker) Contains(objectTags []schedtag.ISchedtag, tags []computeapi.SchedtagConfig) (bool, *computeapi.SchedtagConfig) {
	var otags objTags = objectTags
	for _, tag := range tags {
		if !otags.contains(tag) {
			return false, &tag
		}
	}
	return true, nil
}

func (p *SchedtagChecker) getDynamicSchedtags(resType string, schedDesc *jsonutils.JSONDict) ([]schedtag.ISchedtag, error) {
	if schedDesc == nil {
		return []schedtag.ISchedtag{}, nil
	}
	dynamicTags, err := schedtag.GetEnabledDynamicSchedtagsByResource(resType)
	if err != nil {
		return nil, errors.Wrapf(err, "GetEnabledDynamicSchedtagsByResource %q", resType)
	}

	tags := []schedtag.ISchedtag{}
	for _, tag := range dynamicTags {
		matched, err := conditionparser.EvalBool(tag.GetCondition(), schedDesc)
		if err != nil {
			log.Errorf("Condition parse eval: tag: %q, condition: %q, desc: %s, error: %v", tag.GetName(), tag.GetCondition(), schedDesc, err)
			continue
		}
		if !matched {
			continue
		}
		objTag := tag.GetSchedtag()
		if objTag != nil {
			tags = append(tags, objTag)
		}
	}
	return tags, nil
}

func (c *SchedtagChecker) mergeSchedtags(candiate ISchedtagCandidate, staticTags, dynamicTags []schedtag.ISchedtag) []schedtag.ISchedtag {
	isIn := func(tags []schedtag.ISchedtag, dt schedtag.ISchedtag) bool {
		for _, t := range tags {
			if t.GetId() == dt.GetId() {
				return true
			}
		}
		return false
	}
	ret := []schedtag.ISchedtag{}
	ret = append(ret, staticTags...)
	for _, dt := range dynamicTags {
		if !isIn(staticTags, dt) {
			ret = append(ret, dt)
			log.Debugf("Append dynamic schedtag %#v to %s %q", dt, candiate.ResourceType(), candiate.IndexKey())
		}
	}
	return ret
}

func (c *SchedtagChecker) GetCandidateSchedtags(candidate ISchedtagCandidate) ([]schedtag.ISchedtag, error) {
	// staticTags := candidate.GetSchedtags()
	staticTags := schedtag.GetCandidateSchedtags(candidate.ResourceType(), candidate.GetId())
	dynamicTags, err := c.getDynamicSchedtags(candidate.ResourceType(), candidate.GetDynamicSchedDesc())
	if err != nil {
		return nil, err
	}
	return c.mergeSchedtags(candidate, staticTags, dynamicTags), nil
}

func (c *SchedtagChecker) Check(p ISchedtagPredicate, candidate ISchedtagCandidate) error {
	candidateTags, err := c.GetCandidateSchedtags(candidate)
	if err != nil {
		return err
	}

	execludeTags := p.GetExcludeTags()
	requireTags := p.GetRequireTags()

	log.V(10).Debugf("[SchedtagChecker] check candidate: %s requireTags: %#v, execludeTags: %#v, candidateTags: %#v", candidate.IndexKey(), requireTags, execludeTags, candidateTags)

	candiInfo := fmt.Sprintf("%s:%s", candidate.ResourceType(), candidate.IndexKey())
	if len(execludeTags) > 0 {
		if ok, tag := c.HasIntersection(execludeTags, candidateTags); ok {
			return fmt.Errorf("schedtag %q exclude %s", tag.GetName(), candiInfo)
		}
	}

	if len(requireTags) > 0 {
		if ok, tag := c.Contains(candidateTags, requireTags); !ok {
			return fmt.Errorf("%s need schedtag: %q", candiInfo, tag.Id)
		}
	}

	return nil
}

func GetInputSchedtagByType(tags []*computeapi.SchedtagConfig, types ...string) []*computeapi.SchedtagConfig {
	ret := make([]*computeapi.SchedtagConfig, 0)
	for _, tag := range tags {
		if utils.IsInStringArray(tag.ResourceType, types) {
			ret = append(ret, tag)
		}
	}
	return ret
}
