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

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/conditionparser"
)

type ISchedtagPredicate interface {
	GetExcludeTags() []computeapi.SchedtagConfig
	GetRequireTags() []computeapi.SchedtagConfig
	GetAvoidTags() []computeapi.SchedtagConfig
	GetPreferTags() []computeapi.SchedtagConfig
}

type ISchedtagCandidate interface {
	IndexKey() string
	ResourceType() string
	// GetSchedtags return schedtags bind to this candidate
	GetSchedtags() []models.SSchedtag
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

func NewSchedtagPredicate(reqTags []*computeapi.SchedtagConfig, allTags []models.SSchedtag) *SchedtagPredicate {
	p := new(SchedtagPredicate)
	requireTags, execludeTags, preferTags, avoidTags := GetRequestSchedtags(reqTags, allTags)
	p.requireTags = requireTags
	p.execludeTags = execludeTags
	p.preferTags = preferTags
	p.avoidTags = avoidTags
	p.checker = new(SchedtagChecker)
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

func GetSchedtagCount(inTags []computeapi.SchedtagConfig, objTags []models.SSchedtag, strategy string) (countMap map[string]int) {
	countMap = make(map[string]int)

	in := func(objTag models.SSchedtag, inTags []computeapi.SchedtagConfig) (bool, int) {
		for _, tag := range inTags {
			if tag.Id == objTag.Id || tag.Id == objTag.Name {
				return true, tag.Weight
			}
		}
		return false, 0
	}

	for _, objTag := range objTags {
		if ok, weight := in(objTag, inTags); ok {
			key := fmt.Sprintf("%s:%s:%s", objTag.Id, objTag.Name, strategy)
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

func GetAllSchedtags(resType string) ([]models.SSchedtag, error) {
	tags, err := models.SchedtagManager.GetResourceSchedtags(resType)
	if err != nil {
		return nil, err
	}
	return tags, nil
}

func GetRequestSchedtags(reqTags []*computeapi.SchedtagConfig, allTags []models.SSchedtag) (requireTags, execludeTags, preferTags, avoidTags []computeapi.SchedtagConfig) {
	requireTags = make([]computeapi.SchedtagConfig, 0)
	execludeTags = make([]computeapi.SchedtagConfig, 0)
	preferTags = make([]computeapi.SchedtagConfig, 0)
	avoidTags = make([]computeapi.SchedtagConfig, 0)

	appendedTagIds := make(map[string]int)

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
		_, nameOk := appendedTagIds[tag.Name]
		_, idOk := appendedTagIds[tag.Id]

		if !(nameOk || idOk) {
			apiTag := &computeapi.SchedtagConfig{Id: tag.Id, Strategy: tag.DefaultStrategy}
			appendTagByStrategy(apiTag, 1)
		}
	}

	return
}

type SchedtagChecker struct{}

type apiTags []computeapi.SchedtagConfig

func (t apiTags) contains(objTag models.SSchedtag) bool {
	for _, tag := range t {
		if tag.Id == objTag.Id || tag.Id == objTag.Name {
			return true
		}
	}
	return false
}

type objTags []models.SSchedtag

func (t objTags) contains(atag computeapi.SchedtagConfig) bool {
	for _, tag := range t {
		if tag.Id == atag.Id || tag.Name == atag.Id {
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

func (c *SchedtagChecker) HasIntersection(tags []computeapi.SchedtagConfig, objTags []models.SSchedtag) (bool, *models.SSchedtag) {
	var atags apiTags = tags
	for _, objTag := range objTags {
		if atags.contains(objTag) {
			return true, &objTag
		}
	}
	return false, nil
}

func (c *SchedtagChecker) Contains(objectTags []models.SSchedtag, tags []computeapi.SchedtagConfig) (bool, *computeapi.SchedtagConfig) {
	var otags objTags = objectTags
	for _, tag := range tags {
		if !otags.contains(tag) {
			return false, &tag
		}
	}
	return true, nil
}

func (p *SchedtagChecker) getDynamicSchedtags(resType string, schedDesc *jsonutils.JSONDict) ([]models.SSchedtag, error) {
	if schedDesc == nil {
		return []models.SSchedtag{}, nil
	}
	dynamicTags := models.DynamicschedtagManager.GetEnabledDynamicSchedtagsByResource(resType)

	tags := []models.SSchedtag{}
	for _, tag := range dynamicTags {
		matched, err := conditionparser.EvalBool(tag.Condition, schedDesc)
		if err != nil {
			log.Errorf("Condition parse eval: condition: %q, desc: %s, error: %v", tag.Condition, schedDesc, err)
			continue
		}
		if !matched {
			continue
		}
		objTag := tag.GetSchedtag()
		if objTag != nil {
			tags = append(tags, *objTag)
		}
	}
	return tags, nil
}

func (c *SchedtagChecker) mergeSchedtags(candiate ISchedtagCandidate, staticTags, dynamicTags []models.SSchedtag) []models.SSchedtag {
	isIn := func(tags []models.SSchedtag, dt models.SSchedtag) bool {
		for _, t := range tags {
			if t.Id == dt.Id {
				return true
			}
		}
		return false
	}
	ret := []models.SSchedtag{}
	ret = append(ret, staticTags...)
	for _, dt := range dynamicTags {
		if !isIn(staticTags, dt) {
			ret = append(ret, dt)
			log.Debugf("Append dynamic schedtag %#v to %s %q", dt, candiate.ResourceType(), candiate.IndexKey())
		}
	}
	return ret
}

func (c *SchedtagChecker) GetCandidateSchedtags(candidate ISchedtagCandidate) ([]models.SSchedtag, error) {
	staticTags := candidate.GetSchedtags()
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
			return fmt.Errorf("schedtag %q exclude %s", tag.Name, candiInfo)
		}
	}

	if len(requireTags) > 0 {
		if ok, tag := c.Contains(candidateTags, requireTags); !ok {
			return fmt.Errorf("%s need schedtag: %q", candiInfo, tag.Id)
		}
	}

	return nil
}
