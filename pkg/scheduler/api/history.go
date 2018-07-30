package api

import (
	"strconv"

	"github.com/bitly/go-simplejson"
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
	Data      *SchedData `json:"data"`
	Time      string     `json:"time"`
	Consuming string     `json:"consuming"`
	//Result    []SchedResultItem `json:"result"`
	Result interface{} `json:"result"`
	Error  string      `json:"error"`
	Logs   []string    `json:"logs"`
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
