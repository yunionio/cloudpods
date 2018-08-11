package modules

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type UsageManager struct {
	ResourceManager
}

func (this *UsageManager) GetGeneralUsage(session *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	url := "/usages"
	if params != nil {
		range_type, _ := params.GetString("range_type")
		range_id, _ := params.GetString("range_id")
		if len(range_type) > 0 && len(range_id) > 0 {
			url = fmt.Sprintf("%s/%s/%s", url, range_type, range_id)
		}
		dict := params.(*jsonutils.JSONDict)
		dict.Remove("range_type")
		dict.Remove("range_id")
		qs := dict.QueryString()
		if len(qs) > 0 {
			url = fmt.Sprintf("%s?%s", url, qs)
		}
	}
	return this._get(session, url, this.Keyword)
}

var (
	Usages UsageManager
)

func init() {
	Usages = UsageManager{NewComputeManager("usage", "usages",
		[]string{},
		[]string{})}

	registerCompute(&Usages)
}
