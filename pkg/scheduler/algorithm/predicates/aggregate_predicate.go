package predicates

import (
	"yunion.io/x/jsonutils"

	computemodels "yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/plugin"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
	"yunion.io/x/onecloud/pkg/scheduler/core/score"
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

	SchedtagPredicate *SchedtagPredicate
}

func (p *AggregatePredicate) Name() string {
	return "host_aggregate"
}

func (p *AggregatePredicate) Clone() core.FitPredicate {
	return &AggregatePredicate{}
}

func (p *AggregatePredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	data := u.SchedData()

	if len(data.Candidates) > 0 {
		return false, nil
	}

	allAggs, err := GetAllSchedtags(computemodels.HostManager.KeywordPlural())
	if err != nil {
		return false, err
	}
	p.SchedtagPredicate = NewSchedtagPredicate(data.Schedtags, allAggs)

	u.AppendSelectPlugin(p)
	return true, nil
}

func (p *AggregatePredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := NewPredicateHelper(p, u, c)

	if errMsg := p.exec(h); len(errMsg) > 0 {
		h.Exclude(errMsg)
	}

	return h.GetResult()
}

type schedtagCandidateW struct {
	core.Candidater
	schedData *api.SchedInfo
}

func (w schedtagCandidateW) GetDynamicSchedDesc() *jsonutils.JSONDict {
	ret := jsonutils.NewDict()
	hostSchedDesc := w.GetSchedDesc()
	srvSchedDesc := w.schedData.ToConditionInput()
	ret.Add(hostSchedDesc, computemodels.HostManager.Keyword())
	ret.Add(srvSchedDesc, computemodels.GuestManager.Keyword())
	return ret
}

func (w schedtagCandidateW) GetSchedtags() []computemodels.SSchedtag {
	return w.Getter().HostSchedtags()
}

func (w schedtagCandidateW) ResourceType() string {
	return computemodels.HostManager.KeywordPlural()
}

func (p *AggregatePredicate) exec(h *PredicateHelper) string {
	if err := p.SchedtagPredicate.Check(
		schedtagCandidateW{
			Candidater: h.Candidate,
			schedData:  h.Unit.SchedData(),
		},
	); err != nil {
		return err.Error()
	}

	return ""
}

func SetCandidateScoreBySchedtag(u *core.Unit, c core.Candidater, aggCountMap map[string]int, postiveScore bool) {
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

func (p *AggregatePredicate) OnPriorityEnd(u *core.Unit, c core.Candidater) {
	hostAggs := c.Getter().HostSchedtags()

	avoidCountMap := GetSchedtagCount(p.SchedtagPredicate.GetAvoidTags(), hostAggs, api.AggregateStrategyAvoid)
	preferCountMap := GetSchedtagCount(p.SchedtagPredicate.GetPreferTags(), hostAggs, api.AggregateStrategyPrefer)

	setScore := SetCandidateScoreBySchedtag

	setScore(u, c, preferCountMap, true)
	setScore(u, c, avoidCountMap, false)
}
