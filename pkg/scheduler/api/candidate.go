// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"encoding/json"
	"io"
	"strconv"

	simplejson "github.com/bitly/go-simplejson"
)

// CandidateListArgs is a struct just for parsing candidate
// resource list parameters.
type CandidateListArgs struct {
	Type   string `json:"type"`
	Region string `json:"region"`
	Zone   string `json:"zone"`
	//Pool      string
	Limit     int64 `json:"limit"`
	Offset    int64 `json:"offset"`
	Avaliable bool  `json:"available"`
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
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Cpu          ResultResource         `json:"cpu"`
	Mem          ResultResource         `json:"mem"`
	Storage      ResultResource         `json:"storage"`
	Status       string                 `json:"status"`
	HostStatus   string                 `json:"host_status"`
	EnableStatus string                 `json:"enable_status"`
	HostType     string                 `json:"host_type"`
	PendingUsage map[string]interface{} `json:"pending_usage"`
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
func NewCandidateListArgs(r io.Reader) (*CandidateListArgs, error) {
	args := CandidateListArgs{}
	err := json.NewDecoder(r).Decode(&args)
	if err != nil {
		return nil, err
	}
	if args.Limit == 0 {
		args.Limit = DefaultCandidateListArgsLimit
	}
	if args.Type == "" {
		args.Type = "all"
	}

	return &args, nil
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
