package predicates

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/util/conditionparser"
)

type ISchedtagPredicate interface {
	GetExcludeTags() []api.Schedtag
	GetRequireTags() []api.Schedtag
	GetAvoidTags() []api.Schedtag
	GetPreferTags() []api.Schedtag
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
	requireTags  []api.Schedtag
	execludeTags []api.Schedtag
	preferTags   []api.Schedtag
	avoidTags    []api.Schedtag
	checker      *SchedtagChecker
}

func NewSchedtagPredicate(reqTags []api.Schedtag, allTags []models.SSchedtag) *SchedtagPredicate {
	p := new(SchedtagPredicate)
	requireTags, execludeTags, preferTags, avoidTags := GetRequestSchedtags(reqTags, allTags)
	p.requireTags = requireTags
	p.execludeTags = execludeTags
	p.preferTags = preferTags
	p.avoidTags = avoidTags
	p.checker = new(SchedtagChecker)
	return p
}

func (p *SchedtagPredicate) GetExcludeTags() []api.Schedtag {
	return p.execludeTags
}

func (p *SchedtagPredicate) GetRequireTags() []api.Schedtag {
	return p.requireTags
}

func (p *SchedtagPredicate) GetAvoidTags() []api.Schedtag {
	return p.avoidTags
}

func (p *SchedtagPredicate) GetPreferTags() []api.Schedtag {
	return p.preferTags
}

func (p *SchedtagPredicate) Check(candidate ISchedtagCandidate) error {
	return p.checker.Check(p, candidate)
}

func GetSchedtagCount(inTags []api.Schedtag, objTags []models.SSchedtag, strategy string) (countMap map[string]int) {
	countMap = make(map[string]int)

	in := func(objTag models.SSchedtag, inTags []api.Schedtag) bool {
		for _, tag := range inTags {
			if tag.Idx == objTag.Id || tag.Idx == objTag.Name {
				return true
			}
		}
		return false
	}

	for _, objTag := range objTags {
		if in(objTag, inTags) {
			countMap[fmt.Sprintf("%s:%s:%s", objTag.Id, objTag.Name, strategy)]++
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

func GetRequestSchedtags(reqTags []api.Schedtag, allTags []models.SSchedtag) (requireTags, execludeTags, preferTags, avoidTags []api.Schedtag) {
	requireTags = make([]api.Schedtag, 0)
	execludeTags = make([]api.Schedtag, 0)
	preferTags = make([]api.Schedtag, 0)
	avoidTags = make([]api.Schedtag, 0)

	appendedTagIds := make(map[string]int)

	appendTagByStrategy := func(tag api.Schedtag) {
		switch tag.Strategy {
		case models.STRATEGY_REQUIRE:
			requireTags = append(requireTags, tag)
		case models.STRATEGY_EXCLUDE:
			execludeTags = append(execludeTags, tag)
		case models.STRATEGY_PREFER:
			preferTags = append(preferTags, tag)
		case models.STRATEGY_AVOID:
			avoidTags = append(avoidTags, tag)
		}
	}

	for _, tag := range reqTags {
		appendTagByStrategy(tag)

		appendedTagIds[tag.Idx] = 1
	}

	for _, tag := range allTags {
		_, nameOk := appendedTagIds[tag.Name]
		_, idOk := appendedTagIds[tag.Id]

		if !(nameOk || idOk) {
			apiTag := api.Schedtag{Idx: tag.Id, Strategy: tag.DefaultStrategy}
			appendTagByStrategy(apiTag)
		}
	}

	return
}

type SchedtagChecker struct{}

type apiTags []api.Schedtag

func (t apiTags) contains(objTag models.SSchedtag) bool {
	for _, tag := range t {
		if tag.Idx == objTag.Id || tag.Idx == objTag.Name {
			return true
		}
	}
	return false
}

type objTags []models.SSchedtag

func (t objTags) contains(atag api.Schedtag) bool {
	for _, tag := range t {
		if tag.Id == atag.Idx || tag.Name == atag.Idx {
			return true
		}
	}
	return false
}

func (c *SchedtagChecker) contains(tags []api.Schedtag, objTag models.SSchedtag) bool {
	for _, tag := range tags {
		if tag.Idx == objTag.Id || tag.Idx == objTag.Name {
			return true
		}
	}
	return false
}

func (c *SchedtagChecker) HasIntersection(tags []api.Schedtag, objTags []models.SSchedtag) (bool, *models.SSchedtag) {
	var atags apiTags = tags
	for _, objTag := range objTags {
		if atags.contains(objTag) {
			return true, &objTag
		}
	}
	return false, nil
}

func (c *SchedtagChecker) Contains(objectTags []models.SSchedtag, tags []api.Schedtag) (bool, *api.Schedtag) {
	var otags objTags = objectTags
	for _, tag := range tags {
		if !otags.contains(tag) {
			return false, &tag
		}
	}
	return true, nil
}

func (p *SchedtagChecker) getDynamicSchedtags(schedDesc *jsonutils.JSONDict) ([]models.SSchedtag, error) {
	if schedDesc == nil {
		return []models.SSchedtag{}, nil
	}
	dynamicTags := models.DynamicschedtagManager.GetAllEnabledDynamicSchedtags()

	tags := []models.SSchedtag{}
	for _, tag := range dynamicTags {
		matched, err := conditionparser.Eval(tag.Condition, schedDesc)
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
			log.Debugf("Append dynamic schedtag %s to %s %q", dt, candiate.ResourceType(), candiate.IndexKey())
		}
	}
	return ret
}

func (c *SchedtagChecker) GetCandidateSchedtags(candidate ISchedtagCandidate) ([]models.SSchedtag, error) {
	staticTags := candidate.GetSchedtags()
	dynamicTags, err := c.getDynamicSchedtags(candidate.GetDynamicSchedDesc())
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
	if len(execludeTags) > 0 {
		if ok, tag := c.HasIntersection(execludeTags, candidateTags); ok {
			return fmt.Errorf("Execlude by schedtag: '%s:%s'", tag.Name, tag.Id)
		}
	}

	requireTags := p.GetRequireTags()
	if len(requireTags) > 0 {
		if ok, tag := c.Contains(candidateTags, requireTags); !ok {
			return fmt.Errorf("Need schedtag: '%s'", tag.Idx)
		}
	}

	return nil
}
