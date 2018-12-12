package core

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/scheduler/core/score"
	"yunion.io/x/onecloud/pkg/scheduler/db/models"
)

const (
	PriorityStep int = 1
)

type FailedCandidate struct {
	Stage     string
	Candidate Candidater
	Reasons   []PredicateFailureReason
}

type FailedCandidates struct {
	Candidates []FailedCandidate
}

type SelectPlugin interface {
	Name() string
	OnPriorityEnd(*Unit, Candidater)
	OnSelectEnd(u *Unit, c Candidater, count int64)
}

type Kind int

const (
	KindFree Kind = iota
	KindRaw
	KindReserved
)

// Candidater replace host Candidate resource info
type Candidater interface {
	// IndexKey return candidate cache item's ident, usually host ID
	IndexKey() string
	// Get return candidate cache item's value by key
	Get(key string) interface{}
	// XGet return candidate cache item's value by key and kind
	XGet(key string, kind Kind) interface{}
	Type() int

	GetAggregates() []*models.Aggregate
	GetHostAggregates() []*models.Aggregate

	GetSchedDesc() *jsonutils.JSONDict
	GetGuestCount() int64
	GetResourceType() string
}

// HostPriority represents the priority of scheduling to particular host, higher priority is better.
type HostPriority struct {
	// Name of the host
	Host string
	// Score associated with the host
	Score Score
	// Resource wraps Candidate host info
	Candidate Candidater
}

type HostPriorityList []HostPriority

func (h HostPriorityList) Len() int {
	return len(h)
}

func (h HostPriorityList) Less(i, j int) bool {
	if score.Equal(h[i].Score.ScoreBucket, h[j].Score.ScoreBucket) {
		return h[i].Host < h[j].Host
	}
	return score.Less(h[i].Score.ScoreBucket, h[j].Score.ScoreBucket)
}

func (h HostPriorityList) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

type FitPredicate interface {
	// Get filter's name
	Name() string
	Clone() FitPredicate
	PreExecute(*Unit, []Candidater) (bool, error)
	Execute(*Unit, Candidater) (bool, []PredicateFailureReason, error)
}

type PredicateFailureReason interface {
	GetReason() string
}

type PriorityPreFunction func(*Unit, []Candidater) (bool, []PredicateFailureReason, error)

// PriorityMapFunction is a function that computes per-resource results for a given resource.
type PriorityMapFunction func(*Unit, Candidater) (HostPriority, error)

// PriorityReduceFunction is a function that aggregated per-resource results and computes
// final scores for all hosts.
type PriorityReduceFunction func(*Unit, []Candidater, HostPriorityList) error

type PriorityConfig struct {
	Name   string
	Pre    PriorityPreFunction
	Map    PriorityMapFunction
	Reduce PriorityReduceFunction
	Weight int
}

type Priority interface {
	Name() string
	Clone() Priority
	Map(*Unit, Candidater) (HostPriority, error)
	Reduce(*Unit, []Candidater, HostPriorityList) error
	PreExecute(*Unit, []Candidater) (bool, []PredicateFailureReason, error)

	// Score intervals
	ScoreIntervals() score.Intervals
}
