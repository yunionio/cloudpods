package predicates

import (
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type DomainPredicate struct {
	BasePredicate
}

func (p *DomainPredicate) Name() string {
	return "host_domain"
}

func (p *DomainPredicate) Clone() core.FitPredicate {
	return &DomainPredicate{}
}

func (p *DomainPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := NewPredicateHelper(p, u, c)
	getter := c.Getter()
	cloudprovider := getter.Cloudprovider()
	if cloudprovider == nil || getter.IsPublic() {
		return h.GetResult()
	}
	domainId := getter.DomainId()
	if domainId != u.SchedInfo.Domain {
		h.Exclude2("domain_belong", domainId, u.SchedInfo.Domain)
	}
	return h.GetResult()
}
