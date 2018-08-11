package api

import (
	"strconv"

	"github.com/bitly/go-simplejson"

	"yunion.io/x/log"
)

// CandidateListArgs is a struct just for parsing candidate
// resource list parameters.
type CandidateListArgs struct {
	Type      string
	Zone      string
	Pool      string
	Limit     int64
	Offset    int64
	Avaliable bool
}

type ResultResource struct {
	Free     float64 `json:"free"`
	Reserved float64 `json:"reserverd"`
	Total    float64 `json:"total"`
}

func NewResultResourceString(free, reserverd, total string) (*ResultResource, error) {
	f, err := strconv.ParseFloat(free, 64)
	r, err := strconv.ParseFloat(reserverd, 64)
	t, err := strconv.ParseFloat(total, 64)
	if err != nil {
		return nil, err
	}
	return NewResultResource(f, r, t), nil
}

func NewResultResource(f, r, t float64) *ResultResource {
	return &ResultResource{
		Free:     f,
		Reserved: r,
		Total:    t,
	}
}

func NewResultResourceInt64(f, r, t int64) *ResultResource {
	free := float64(f)
	reserverd := float64(r)
	total := float64(t)
	return NewResultResource(free, reserverd, total)
}

type CandidateListResultItem struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Cpu          ResultResource `json:"cpu"`
	Mem          ResultResource `json:"mem"`
	Storage      ResultResource `json:"storage"`
	Status       string         `json:"status"`
	HostStatus   string         `json:"host_status"`
	EnableStatus string         `json:"enable_status"`
	HostType     string         `json:"host_type"`
}

type CandidateListResult struct {
	Data   []CandidateListResultItem `json:"data"`
	Total  int64                     `json:"total"`
	Limit  int64                     `json:"limit"`
	Offset int64                     `json:"offset"`
}

const (
	DefaultCandidateListArgsLimit = 20
)

// NewCandidateListArgs provides a function that
// will parse candidate's list args from a json data.
func NewCandidateListArgs(sjson *simplejson.Json) (*CandidateListArgs, error) {
	args := &CandidateListArgs{
		Limit: DefaultCandidateListArgsLimit,
	}
	if argsType, ok := sjson.CheckGet("type"); ok {
		args.Type = argsType.MustString()
	} else {
		args.Type = "all"
	}

	if zone, ok := sjson.CheckGet("zone"); ok {
		args.Zone = zone.MustString()
	}

	if pool, ok := sjson.CheckGet("pool"); ok {
		args.Pool = pool.MustString()
	}

	if limit, ok := sjson.CheckGet("limit"); ok {
		limitv, err := limit.Int64()
		if err != nil {
			limitv, err = strconv.ParseInt(limit.MustString(), 10, 64)
			if err != nil {
				log.Errorln(err)
			}
		}
		args.Limit = limitv
	}

	if offset, ok := sjson.CheckGet("offset"); ok {
		args.Offset = offset.MustInt64()
	}

	if avaliable, ok := sjson.CheckGet("avaliable"); ok {
		args.Avaliable = avaliable.MustBool()
	}

	return args, nil
}

// CandidateDetailArgs is a struct just for parsing candidate
// resource parameters.
type CandidateDetailArgs struct {
	ID   string
	Type string
}

type CandidateDetailResult struct {
	Candidate interface{} `json:"candidate"`
}

// NewCandidateDetailArgs provides a function that
// will parse candidate's args from a json data.
func NewCandidateDetailArgs(sjson *simplejson.Json, id string) (*CandidateDetailArgs, error) {
	args := new(CandidateDetailArgs)
	args.ID = id

	if argsType, ok := sjson.CheckGet("type"); ok {
		args.Type = argsType.MustString()
	} else {
		args.Type = HostTypeHost
	}

	return args, nil
}
