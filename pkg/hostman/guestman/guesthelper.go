package guestman

import "yunion.io/x/jsonutils"

type SBaseParms struct {
	Sid  string
	Body jsonutils.JSONObject
}

type SGuestDeploy struct {
	SGuestBaseParms
	IsInit bool
}
