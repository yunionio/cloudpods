package modules

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func (this *ResourceManager) filterSingleResult(session *mcclient.ClientSession, result jsonutils.JSONObject, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if this.enableFilter && this.readFilter != nil {
		return this.readFilter(session, result, query)
	}
	return result, nil
}

func (this *ResourceManager) filterListResults(session *mcclient.ClientSession, results *ListResult, query jsonutils.JSONObject) (*ListResult, error) {
	if this.enableFilter && this.readFilter != nil {
		for i := 0; i < len(results.Data); i += 1 {
			val, err := this.readFilter(session, results.Data[i], query)
			if err == nil {
				results.Data[i] = val
			} else {
				log.Warningf("readFilter fail for %s: %s", results.Data[i], err)
			}
		}
	}
	return results, nil
}
