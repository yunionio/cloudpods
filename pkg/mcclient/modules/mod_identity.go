package modules

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
)

type SIdentityUsageManager struct {
	ResourceManager
}

func (this *SIdentityUsageManager) GetUsage(session *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	url := "/usages"
	if params != nil {
		query := params.QueryString()
		if len(query) > 0 {
			url = fmt.Sprintf("%s?%s", url, query)
		}
	}
	return this._get(session, url, "usage")
}

var (
	IdentityUsages SIdentityUsageManager
	IdentityLogs   ResourceManager
)

func init() {
	IdentityUsages = SIdentityUsageManager{NewIdentityV3Manager("usage", "usages",
		[]string{},
		[]string{})}

	IdentityLogs = NewIdentityV3Manager("event", "events",
		[]string{"id", "ops_time", "obj_id", "obj_type", "obj_name", "user", "user_id", "tenant", "tenant_id", "owner_tenant_id", "action", "notes"},
		[]string{})
}
