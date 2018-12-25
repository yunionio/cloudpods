package guestman

import "yunion.io/x/jsonutils"

type SGuestDeploy struct {
	Sid    string
	Body   jsonutils.JSONObject
	IsInit bool
}

type SGuestSync struct {
	Sid  string
	Body jsonutils.JSONObject
}
