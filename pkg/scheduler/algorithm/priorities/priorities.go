package priorities

import (
	"yunion.io/x/onecloud/pkg/scheduler/algorithm"
	"yunion.io/x/onecloud/pkg/scheduler/cache/candidate"
	"yunion.io/x/onecloud/pkg/scheduler/core"
	"yunion.io/x/onecloud/pkg/scheduler/core/score"
)

// PriorityHelper is a struct that as a base interface for all priorities.
type PriorityHelper struct {
	priority  core.Priority
	unit      *core.Unit
	Candidate core.Candidater
	score     score.SScore
	err       error
}

func NewPriorityHelper(p core.Priority, u *core.Unit, c core.Candidater) *PriorityHelper {
	return &PriorityHelper{
		priority:  p,
		unit:      u,
		Candidate: c,
	}
}

func (h *PriorityHelper) setIntervalScore(val int) score.SScore {
	h.score = score.NewScore(
		h.priority.ScoreIntervals().ToScore(int64(val)),
		h.priority.Name())
	return h.score
}

func (h *PriorityHelper) setRawScore(val int) score.SScore {
	h.score = score.NewScore(
		score.TScore(val),
		h.priority.Name())
	return h.score
}

func (h *PriorityHelper) SetScore(val int) {
	h.setIntervalScore(val)
	h.unit.SetScore(h.Candidate.IndexKey(), h.score)
}

func (h *PriorityHelper) SetFrontScore(val int) {
	h.setIntervalScore(val)
	h.unit.SetFrontScore(h.Candidate.IndexKey(), h.score)
}

func (h *PriorityHelper) SetRawScore(val int) {
	h.setRawScore(val)
	h.unit.SetScore(h.Candidate.IndexKey(), h.score)
}

func (h *PriorityHelper) SetFrontRawScore(val int) {
	h.setRawScore(val)
	h.unit.SetFrontScore(h.Candidate.IndexKey(), h.score)
}

func (h *PriorityHelper) SetError(err error) {
	h.err = err
}

func (h *PriorityHelper) GetResult() (core.HostPriority, error) {
	return core.HostPriority{
		Host:      h.Candidate.IndexKey(),
		Candidate: h.Candidate,
	}, h.err
}

// BasePriority is a default struct for priority that all the priorities
// will contain it and implement its PreExecute(),Map(),Reduce() and
// Name() methods.
type BasePriority struct {
}

func (b *BasePriority) PreExecute(u *core.Unit, cs []core.Candidater) (bool, []core.PredicateFailureReason, error) {
	return true, nil, nil
}

func (b *BasePriority) Map(u *core.Unit, c core.Candidater) (core.HostPriority, error) {
	return core.HostPriority{}, nil
}

func (b *BasePriority) Reduce(u *core.Unit, cs []core.Candidater, result core.HostPriorityList) error {
	return nil
}

func (b *BasePriority) Name() string {
	return "base_priorites_should_not_be_called"
}

func (b *BasePriority) HostCandidate(c core.Candidater) (*candidate.HostDesc, error) {
	return algorithm.ToHostCandidate(c)
}

func (b *BasePriority) ScoreIntervals() score.Intervals {
	return score.NewIntervals(0, 1, 2)
}
