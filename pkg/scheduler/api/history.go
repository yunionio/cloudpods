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
	"strconv"

	simplejson "github.com/bitly/go-simplejson"
)

type HistoryArgs struct {
	Offset int64
	Limit  int64
	All    bool
}

type HistoryItem struct {
	Time         string   `json:"time"`
	Consuming    string   `json:"consuming"`
	SessionID    string   `json:"session_id"`
	Count        string   `json:"count"`
	Tenants      []string `json:"tenants"`
	Status       string   `json:"status"`
	Guests       []string `json:"guests"`
	IsSuggestion bool     `json:"is_suggestion"`
}

type HistoryResult struct {
	Items  []*HistoryItem `json:"data"`
	Total  int64          `json:"total"`
	Offset int64          `json:"offset"`
	Limit  int64          `json:"limit"`
}

func toInt64(j *simplejson.Json) int64 {
	if s, err := j.String(); err == nil {
		if r, err0 := strconv.Atoi(s); err0 == nil {
			return int64(r)
		}
	}

	return j.MustInt64()
}

func NewHistoryArgs(sjson *simplejson.Json) (*HistoryArgs, error) {
	args := new(HistoryArgs)

	if offset, ok := sjson.CheckGet("offset"); ok {
		args.Offset = toInt64(offset)
	}

	if limit, ok := sjson.CheckGet("limit"); ok {
		args.Limit = toInt64(limit)
	}

	if all, ok := sjson.CheckGet("all"); ok {
		args.All = all.MustBool()
	}

	return args, nil
}

type HistoryDetailArgs struct {
	ID  string
	Raw bool
	Log bool
}

type HistoryTask struct {
	Type      string     `json:"type"`
	Status    string     `json:"status"`
	Data      *SchedInfo `json:"data"`
	Time      string     `json:"time"`
	Consuming string     `json:"consuming"`
	//Result    []SchedResultItem `json:"result"`
	Result      interface{} `json:"result"`
	Error       string      `json:"error"`
	Logs        []string    `json:"logs"`
	CapacityMap interface{} `json:"capacity_map"`
}

type HistoryDetail struct {
	Time      string        `json:"time"`
	Consuming string        `json:"consuming"`
	SessionID string        `json:"session_id"`
	Tasks     []HistoryTask `json:"tasks"`
	Input     string        `json:"input"`
	Output    string        `json:"output"`
	Error     string        `json:"error"`
}

type HistoryDetailResult struct {
	Detail *HistoryDetail `json:"history"`
}

func NewHistoryDetailArgs(sjson *simplejson.Json, id string) (*HistoryDetailArgs, error) {
	args := new(HistoryDetailArgs)
	args.ID = id

	if raw, ok := sjson.CheckGet("raw"); ok {
		args.Raw = raw.MustBool()
	}

	if log, ok := sjson.CheckGet("log"); ok {
		args.Log = log.MustBool()
	}

	return args, nil
}
