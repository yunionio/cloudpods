package predicates

import (
	"fmt"
	"strings"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/scheduler/core"
)

// BasePredicate is a default struct for all the predicates that will
// include it and implement it's Name() and PreExecute() methods.
type BasePredicate struct{}

func (b *BasePredicate) Name() string {
	return "base_predicate_should_not_be_called"
}

func (b *BasePredicate) PreExecute(unit *core.Unit, candis []core.Candidater) (bool, error) {
	return true, nil
}

type PredicateHelper struct {
	predicate      core.FitPredicate
	predicateFails []core.PredicateFailureReason
	capacity       int64
	Unit           *core.Unit
	Candidate      core.Candidater
}

func (h *PredicateHelper) getResult() (bool, []core.PredicateFailureReason, error) {
	if len(h.predicateFails) > 0 {
		return false, h.predicateFails, nil
	}

	if h.capacity == 0 {
		return false, []core.PredicateFailureReason{}, nil
	}

	return true, nil, nil
}

func (h *PredicateHelper) GetResult() (bool, []core.PredicateFailureReason, error) {
	ok, reasons, err := h.getResult()
	if !ok {
		log.Warningf("[Filter Result] candidate: %q, filter: %q, is_ok: %v, reason: %q, error: %v\n",
			h.Candidate.IndexKey(), h.predicate.Name(), ok, getReasonsString(reasons), err)
	}
	return ok, reasons, err
}

func getReasonsString(reasons []core.PredicateFailureReason) string {
	if len(reasons) == 0 {
		return ""
	}

	ss := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		ss = append(ss, reason.GetReason())
	}
	return strings.Join(ss, ", ")
}

func NewPredicateHelper(pre core.FitPredicate, unit *core.Unit, candi core.Candidater) *PredicateHelper {
	h := &PredicateHelper{
		predicate:      pre,
		capacity:       core.EmptyCapacity,
		predicateFails: []core.PredicateFailureReason{},
		Unit:           unit,
		Candidate:      candi,
	}
	return h
}

func (h *PredicateHelper) GetFailedResult(err error) (bool, []core.PredicateFailureReason, error) {
	return false, nil, err
}

func (h *PredicateHelper) AppendPredicateFail(reason core.PredicateFailureReason) {
	h.predicateFails = append(h.predicateFails, reason)
}

func (h *PredicateHelper) AppendPredicateFailMsg(reason string) {
	h.AppendPredicateFail(NewUnexceptedResourceError(reason))
}

func (h *PredicateHelper) AppendInsufficientResourceError(req, total, free int64) {
	h.AppendPredicateFail(
		NewInsufficientResourceError(h.Candidate.Getter().Name(), req, total, free))
}

// SetCapacity returns the current resource capacity calculated by a filter.
// And 'capacity' default is -1.
func (h *PredicateHelper) SetCapacity(capacity int64) {
	if capacity < 0 {
		capacity = 0
	}

	h.SetCapacityCounter(core.NewNormalCounter(capacity))
}

func (h *PredicateHelper) SetCapacityCounter(counter core.Counter) {
	capacity := counter.GetCount()
	if capacity < core.EmptyCapacity {
		capacity = core.EmptyCapacity
	}

	h.capacity = capacity
	h.Unit.SetCapacity(h.Candidate.IndexKey(), h.predicate.Name(), counter)
}

func (h *PredicateHelper) Exclude(reason string) {
	h.SetCapacity(0)
	h.AppendPredicateFailMsg(reason)
}

func (h *PredicateHelper) Exclude2(predicateName string, current, expected interface{}) {
	h.Exclude(fmt.Sprintf("%s is '%v', expected '%v'", predicateName, current, expected))
}

// UseReserved check whether the unit can use guest reserved resource
func (h *PredicateHelper) UseReserved() bool {
	usable := false
	data := h.Unit.SchedData()
	isoDevs := data.IsolatedDevices
	if len(isoDevs) > 0 {
		usable = true
	}
	return usable
}
