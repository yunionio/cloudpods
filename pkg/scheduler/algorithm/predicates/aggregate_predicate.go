package predicates

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/scheduler/algorithm/plugin"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
	"yunion.io/x/onecloud/pkg/scheduler/core/score"
	"yunion.io/x/onecloud/pkg/scheduler/db/models"
	"yunion.io/x/onecloud/pkg/util/conditionparser"
)

// NOTE:    Aggregate  Description
// require: Must be scheduled to the specified host
// prefer: Priority to the specified host
// avoid: Try to avoid scheduling to the specified host
// exclude: Do not allow scheduling on the specified host

// AggregatePredicate is designed to quickly filter unavailable
// hosts and improve scheduling efficiency by tabbing whether
// the host is available.
type AggregatePredicate struct {
	BasePredicate
	plugin.BasePlugin
	AggregateHosts    hostsAggregatesMap
	RequireAggregates []api.Aggregate
	ExcludeAggregates []api.Aggregate
	AvoidAggregates   []api.Aggregate
	PreferAggregates  []api.Aggregate
	AggregateMap      map[string]api.Aggregate
}

type hostAggregates []*models.Aggregate

type hostsAggregatesMap map[string]hostAggregates

func (p *AggregatePredicate) Name() string {
	return "host_aggregate"
}

func (p *AggregatePredicate) Clone() core.FitPredicate {
	return &AggregatePredicate{
		AggregateMap: make(map[string]api.Aggregate, 0),
	}
}

func getHostAndServerSchedDesc(u *core.Unit, c core.Candidater) *jsonutils.JSONDict {
	ret := jsonutils.NewDict()
	hostSchedDesc := c.GetSchedDesc()
	srvSchedDesc := jsonutils.Marshal(u.SchedData())
	ret.Add(hostSchedDesc, "host")
	ret.Add(srvSchedDesc, "server")
	return ret
}

func getHostDynamicSchedtags(u *core.Unit, c core.Candidater) ([]*models.Aggregate, error) {
	schedDesc := getHostAndServerSchedDesc(u, c)

	dynamicTags, err := models.FetchEnabledDynamicschedtags()
	if err != nil {
		return nil, err
	}
	aggs := []*models.Aggregate{}
	for _, tag := range dynamicTags {
		matched, err := conditionparser.Eval(tag.Condition, schedDesc)
		if err != nil {
			log.Errorf("Condition parse eval: condition: %q, desc: %s, error: %v", tag.Condition, schedDesc, err)
			continue
		}
		if !matched {
			continue
		}
		aggregate, err := tag.FetchSchedTag()
		if err != nil {
			log.Errorf("Get dynamic schedtag %q error: %v", tag.SchedtagId, err)
			continue
		}
		aggs = append(aggs, aggregate)
	}
	return aggs, nil
}

func mergeHostSchedtags(c core.Candidater, staticTags, dynamicTags []*models.Aggregate) []*models.Aggregate {
	isIn := func(tags []*models.Aggregate, dt *models.Aggregate) bool {
		for _, t := range tags {
			if t.ID == dt.ID {
				return true
			}
		}
		return false
	}
	ret := []*models.Aggregate{}
	ret = append(ret, staticTags...)
	for _, dt := range dynamicTags {
		if !isIn(staticTags, dt) {
			ret = append(ret, dt)
			log.Debugf("Append dynamic schedtag %s to host %q", dt, c.IndexKey())
		}
	}
	return ret
}

func hostsAggregatesInfo(u *core.Unit, cs []core.Candidater) (hostsAggregatesMap, []*models.Aggregate) {
	ret := make(map[string]hostAggregates, 0)
	allAggs := make([]*models.Aggregate, 0)
	for _, c := range cs {
		hostAggs := c.GetHostAggregates()
		dynamicMatchedAggs, err := getHostDynamicSchedtags(u, c)
		if err != nil {
			log.Errorf("Get host %q dynamic schedtag error: %v", c.IndexKey(), err)
		} else {
			hostAggs = mergeHostSchedtags(c, hostAggs, dynamicMatchedAggs)
		}
		ret[c.IndexKey()] = hostAggs
	}
	if len(cs) > 0 {
		allAggs = cs[0].GetAggregates()
	}
	return ret, allAggs
}

func (p *AggregatePredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	data := u.SchedData()

	if len(data.Candidates) > 0 {
		return false, nil
	}

	hsMap, allAggs := hostsAggregatesInfo(u, cs)
	p.AggregateHosts = hsMap
	appendedAggIds := make(map[string]int, len(data.Aggregates))

	for _, aggregate := range data.Aggregates {
		switch aggregate.Strategy {
		case api.AggregateStrategyRequire:
			p.RequireAggregates = append(p.RequireAggregates, aggregate)
		case api.AggregateStrategyExclude:
			p.ExcludeAggregates = append(p.ExcludeAggregates, aggregate)

		case api.AggregateStrategyPrefer:
			p.PreferAggregates = append(p.PreferAggregates, aggregate)

		case api.AggregateStrategyAvoid:
			p.AvoidAggregates = append(p.AvoidAggregates, aggregate)
		}

		p.AggregateMap[aggregate.Idx] = aggregate
		appendedAggIds[aggregate.Idx] = 1
	}

	for _, aggregate := range allAggs {
		_, nameOk := appendedAggIds[aggregate.Name]
		_, idOk := appendedAggIds[aggregate.ID]
		if !(nameOk || idOk) {
			agg := api.Aggregate{Idx: aggregate.ID, Strategy: aggregate.DefaultStrategy}
			switch agg.Strategy {
			case api.AggregateStrategyRequire:
				p.RequireAggregates = append(p.RequireAggregates, agg)
			case api.AggregateStrategyExclude:
				p.ExcludeAggregates = append(p.ExcludeAggregates, agg)

			case api.AggregateStrategyPrefer:
				p.PreferAggregates = append(p.PreferAggregates, agg)

			case api.AggregateStrategyAvoid:
				p.AvoidAggregates = append(p.AvoidAggregates, agg)
			}
		}
	}

	u.AppendSelectPlugin(p)
	return true, nil
}

func getHostAggregateCount(inAggs []api.Aggregate, hAggs []*models.Aggregate, strategy string) (countMap map[string]int) {
	countMap = make(map[string]int)

	in := func(hAgg *models.Aggregate, inAggs []api.Aggregate) bool {
		for _, agg := range inAggs {
			if agg.Idx == hAgg.ID || agg.Idx == hAgg.Name {
				return true
			}
		}
		return false
	}

	for _, hAgg := range hAggs {
		if in(hAgg, inAggs) {
			countMap[fmt.Sprintf("%s:%s:%s", hAgg.ID, hAgg.Name, strategy)]++
		}
	}
	return
}

func (p *AggregatePredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := NewPredicateHelper(p, u, c)

	if errMsg := p.exec(h); len(errMsg) > 0 {
		h.Exclude(errMsg)
	}

	return h.GetResult()
}

func (p *AggregatePredicate) exec(h *PredicateHelper) string {
	ahs := p.AggregateHosts
	candidateID := h.Candidate.IndexKey()

	log.V(10).Debugf(">>>> ExcludeAggregates: %#v, RequireAggregates: %#v, AvoidAggregates: %#v, PreferAggregates: %#v, candidateID: %v", p.ExcludeAggregates, p.RequireAggregates, p.AvoidAggregates, p.PreferAggregates, candidateID)

	if len(p.ExcludeAggregates) > 0 {
		inExclude := func(a *models.Aggregate) bool {
			for _, agg := range p.ExcludeAggregates {
				if agg.Idx == a.ID || agg.Idx == a.Name {
					return true
				}
			}

			return false
		}

		if ah, ok := ahs[candidateID]; ok {
			for _, a := range ah {
				if inExclude(a) {
					return fmt.Sprintf("exclude by aggregate: '%s:%s'", a.Name, a.ID)
				}
			}
		}
	}

	if len(p.RequireAggregates) > 0 {
		var as []*models.Aggregate = nil
		if ah, ok := ahs[candidateID]; ok {
			as = ah
		}

		inRequire := func(agg api.Aggregate) bool {
			for _, a := range as {
				if a.ID == agg.Idx || a.Name == agg.Idx {
					return true
				}
			}

			return false
		}

		for _, agg := range p.RequireAggregates {
			if !inRequire(agg) {
				return fmt.Sprintf("need aggregate: '%s'", agg.Idx)
			}
		}
	}

	return ""
}

func (p *AggregatePredicate) OnPriorityEnd(u *core.Unit, c core.Candidater) {
	hostAggs, ok := p.AggregateHosts[c.IndexKey()]
	if !ok {
		return
	}

	avoidCountMap := getHostAggregateCount(p.AvoidAggregates, hostAggs, api.AggregateStrategyAvoid)
	preferCountMap := getHostAggregateCount(p.PreferAggregates, hostAggs, api.AggregateStrategyPrefer)

	setScore := func(aggCountMap map[string]int, postiveScore bool) {
		stepScore := core.PriorityStep
		if !postiveScore {
			stepScore = -stepScore
		}
		for n, count := range aggCountMap {
			u.SetFrontScore(
				c.IndexKey(),
				score.NewScore(score.TScore(count*stepScore), n),
			)
		}
	}

	setScore(preferCountMap, true)
	setScore(avoidCountMap, false)
}
