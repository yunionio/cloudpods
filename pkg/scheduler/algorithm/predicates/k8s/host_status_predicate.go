package k8s

import (
	"fmt"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates/guest"
	"yunion.io/x/onecloud/pkg/scheduler/cache/candidate"
)

type HostStatusPredicate struct{}

func (p *HostStatusPredicate) Clone() IPredicate {
	return &HostStatusPredicate{}
}

func (p *HostStatusPredicate) Name() string {
	return "host_status"
}

func (p *HostStatusPredicate) PreExecute(cli *kubernetes.Clientset, pod *v1.Pod, node *v1.Node, host *candidate.HostDesc) bool {
	return true
}

func (p *HostStatusPredicate) Execute(cli *kubernetes.Clientset, pod *v1.Pod, node *v1.Node, host *candidate.HostDesc) (bool, error) {
	if host.Status != guest.ExpectedStatus {
		return false, fmt.Errorf("Host status is %s", host.Status)
	}

	if !host.Enabled {
		return false, fmt.Errorf("Host is disabled")
	}
	return true, nil
}
