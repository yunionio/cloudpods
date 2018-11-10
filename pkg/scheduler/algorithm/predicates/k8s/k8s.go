package k8s

import (
	"fmt"

	"k8s.io/api/core/v1"

	"yunion.io/x/onecloud/pkg/scheduler/cache/candidate"
)

var PredicatesManager *SPredicatesManager

func init() {
	PredicatesManager = newPredicatesManager()

	PredicatesManager.Register(
		&HostStatusPredicate{},
		&NetworkPredicate{},
	)
}

type IPredicate interface {
	Name() string
	Clone() IPredicate
	PreExecute(pod *v1.Pod, node *v1.Node, host *candidate.HostDesc) bool
	Execute(pod *v1.Pod, node *v1.Node, host *candidate.HostDesc) (bool, error)
}

type SPredicatesManager struct {
	predicates []IPredicate
}

func newPredicatesManager() *SPredicatesManager {
	man := &SPredicatesManager{
		predicates: make([]IPredicate, 0),
	}
	return man
}

func (man *SPredicatesManager) Register(pres ...IPredicate) *SPredicatesManager {
	for _, pre := range pres {
		if !man.Has(pre) {
			man.predicates = append(man.predicates, pre)
		}
	}
	return man
}

func (man *SPredicatesManager) Has(newPre IPredicate) bool {
	if len(man.predicates) == 0 {
		return false
	}
	for _, pre := range man.predicates {
		if pre.Name() == newPre.Name() {
			return true
		}
	}
	return false
}

func (man *SPredicatesManager) DoFilter(pod *v1.Pod, node *v1.Node, host *candidate.HostDesc) (bool, error) {
	for _, pre := range man.predicates {
		tmpPre := pre.Clone()
		if !tmpPre.PreExecute(pod, node, host) {
			continue
		}
		fit, err := tmpPre.Execute(pod, node, host)
		if err != nil {
			return false, err
		}
		if !fit {
			return false, fmt.Errorf("Filtered by %s", tmpPre.Name())
		}
	}
	return true, nil
}
