package api

import (
	"github.com/bitly/go-simplejson"
)

type CompletedNotifyArgs struct {
	SessionID string
}

type CompletedNotifyResult struct {
}

func NewCompletedNotifyArgs(sjson *simplejson.Json, sessionId string) (*CompletedNotifyArgs, error) {
	args := new(CompletedNotifyArgs)
	args.SessionID = sessionId

	return args, nil
}
