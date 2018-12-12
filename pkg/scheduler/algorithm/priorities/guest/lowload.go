package guest

import (
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/priorities"
	"yunion.io/x/onecloud/pkg/scheduler/core"
	"yunion.io/x/onecloud/pkg/scheduler/core/score"
)

type LowLoadPriority struct {
	priorities.BasePriority
}

func (p *LowLoadPriority) Name() string {
	return "host_lowload"
}

func (p *LowLoadPriority) Clone() core.Priority {
	return &LowLoadPriority{}
}

func (p *LowLoadPriority) Map(u *core.Unit, c core.Candidater) (core.HostPriority, error) {
	h := priorities.NewPriorityHelper(p, u, c)

	hc, err := p.HostCandidate(c)
	if err != nil {
		return core.HostPriority{}, err
	}

	cpuCommitRate := float64(hc.RunningCPUCount) / float64(hc.TotalCPUCount)
	memCommitRate := float64(hc.RunningMemSize) / float64(hc.TotalMemSize)
	if cpuCommitRate < 0.5 && memCommitRate < 0.5 {
		score := 10 * (1 - cpuCommitRate - memCommitRate)
		h.SetScore(int(score))
	}
	return h.GetResult()
}

func (p *LowLoadPriority) ScoreIntervals() score.Intervals {
	return score.NewIntervals(0, 1, 5)
}
