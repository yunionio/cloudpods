package api

import (
	"github.com/bitly/go-simplejson"
)

type CleanupArgs struct {
	ResType string
}

type CleanupResult struct {
}

func NewCleanupArgs(sjson *simplejson.Json) (*CleanupArgs, error) {
	args := new(CleanupArgs)
	if resType, ok := sjson.CheckGet("res_type"); ok {
		args.ResType = resType.MustString()
	}

	return args, nil
}
