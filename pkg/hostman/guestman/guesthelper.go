package guestman

import "yunion.io/x/jsonutils"

type SGuestDeploy struct {
	sid    string
	body   jsonutils.JSONObject
	isInit bool
}
