package stats

import "yunion.io/x/onecloud/pkg/util/pod/cadvisor"

type ContainerStatsProvider interface {
	ListPodStats() ([]PodStats, error)
	ListPodStatsAndUpdateCPUNanoCoreUsage() ([]PodStats, error)
	ListPodCPUAndMemoryStats() ([]PodStats, error)
	ImageFsStats() (FsStats, error)
	ImageFsDevice() (string, error)
}

type StatsProvider struct {
	cadvisor cadvisor.Interface
	ContainerStatsProvider
}

func NewCRIStatsProvider(
	cadvisor cadvisor.Interface,
) *StatsProvider {
	return nil
}

func newStatsProvider(
	cadvisor cadvisor.Interface,
) *StatsProvider {
	return &StatsProvider{
		cadvisor: cadvisor,
	}
}
