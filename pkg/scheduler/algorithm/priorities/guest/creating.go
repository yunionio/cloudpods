package guest

import (
	"github.com/yunionio/onecloud/pkg/scheduler/algorithm/priorities"
	"github.com/yunionio/onecloud/pkg/scheduler/core"
)

type CreatingPriority struct {
	priorities.BasePriority
}

func (p *CreatingPriority) Name() string {
	return "creating"
}

func (p *CreatingPriority) Clone() core.Priority {
	return &CreatingPriority{}
}

func (p *CreatingPriority) Map(u *core.Unit, c core.Candidater) (core.HostPriority, error) {

	h := priorities.NewPriorityHelper(p, u, c)

	hc, err := p.HostCandidate(c)
	if err != nil {
		return core.HostPriority{}, err
	}

	if hc.CreatingGuestCount > 0 {
		score := -int(hc.CreatingGuestCount) * 20
		h.SetScore(score)
	}

	return h.GetResult()
}
