package modules

import (
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type JointResourceManager struct {
	ResourceManager
	Master Manager
	Slave  Manager
}

func (this *JointResourceManager) MasterManager() Manager {
	return this.Master
}

func (this *JointResourceManager) SlaveManager() Manager {
	return this.Slave
}

func (this *JointResourceManager) Get(s *mcclient.ClientSession, mid, sid string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/%s/%s/%s", this.Master.KeyString(), url.PathEscape(mid), this.Slave.KeyString(), url.PathEscape(sid))
	if params != nil {
		qs := params.QueryString()
		if len(qs) > 0 {
			path = fmt.Sprintf("%s?%s", path, qs)
		}
	}
	result, err := this._get(s, path, this.Keyword)
	if err != nil {
		return nil, err
	}
	return this.filterSingleResult(s, result, params)
}

/*
func (this *JointResourceManager) List(s *mcclient.ClientSession, params jsonutils.JSONObject) (*ListResult, error) {
	path := fmt.Sprintf("/%s", this.KeyString())
	if params != nil {
		qs := params.QueryString()
		if len(qs) > 0 {
			path = fmt.Sprintf("%s?%s", path, qs)
		}
	}
	return this._list(s, path, this.KeywordPlural)
}
*/

func (this *JointResourceManager) ListDescendent(s *mcclient.ClientSession, mid string, params jsonutils.JSONObject) (*ListResult, error) {
	path := fmt.Sprintf("/%s/%s/%s", this.Master.KeyString(), url.PathEscape(mid), this.Slave.KeyString())
	if params != nil {
		qs := params.QueryString()
		if len(qs) > 0 {
			path = fmt.Sprintf("%s?%s", path, qs)
		}
	}
	results, err := this._list(s, path, this.KeywordPlural)
	if err != nil {
		return nil, err
	}
	return this.filterListResults(s, results, params)
}

func (this *JointResourceManager) ListDescendent2(s *mcclient.ClientSession, sid string, params jsonutils.JSONObject) (*ListResult, error) {
	return this.ListAscendent(s, sid, params)
}

func (this *JointResourceManager) ListAscendent(s *mcclient.ClientSession, mid string, params jsonutils.JSONObject) (*ListResult, error) {
	path := fmt.Sprintf("/%s/%s/%s", this.Slave.KeyString(), url.PathEscape(mid), this.Master.KeyString())
	if params != nil {
		qs := params.QueryString()
		if len(qs) > 0 {
			path = fmt.Sprintf("%s?%s", path, qs)
		}
	}
	results, err := this._list(s, path, this.KeywordPlural)
	if err != nil {
		return nil, err
	}
	return this.filterListResults(s, results, params)
}

/* func (this *JointResourceManager) Exists(s *mcclient.ClientSession, mid, sid string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
    path := fmt.Sprintf("/%s/%s/%s/%s", this.Master.KeyString(), url.PathEscape(mid), this.Slave.KeyString(), url.PathEscape(sid))
    if params != nil {
        qs := params.QueryString()
        if len(qs) > 0 {
            path = fmt.Sprintf("%s?%s", path, qs)
        }
    }
    return this._head(s, path, this.Keyword)
} */

func (this *JointResourceManager) Attach(s *mcclient.ClientSession, mid, sid string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/%s/%s/%s", this.Master.KeyString(), url.PathEscape(mid), this.Slave.KeyString(), url.PathEscape(sid))
	result, err := this._post(s, path, this.params2Body(s, params), this.Keyword)
	if err != nil {
		return nil, err
	}
	return this.filterSingleResult(s, result, nil)
}

func (this *JointResourceManager) BatchAttach(s *mcclient.ClientSession, mid string, sids []string, params jsonutils.JSONObject) []SubmitResult {
	return BatchDo(sids, func(sid string) (jsonutils.JSONObject, error) {
		return this.Attach(s, mid, sid, params)
	})
}

func (this *JointResourceManager) BatchAttach2(s *mcclient.ClientSession, mid string, sids []string, params jsonutils.JSONObject) []SubmitResult {
	return BatchDo(sids, func(sid string) (jsonutils.JSONObject, error) {
		return this.Attach(s, sid, mid, params)
	})
}

func (this *JointResourceManager) Detach(s *mcclient.ClientSession, mid, sid string) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/%s/%s/%s", this.Master.KeyString(), url.PathEscape(mid), this.Slave.KeyString(), url.PathEscape(sid))
	result, err := this._delete(s, path, nil, this.Keyword)
	if err != nil {
		return nil, err
	}
	return this.filterSingleResult(s, result, nil)
}

func (this *JointResourceManager) BatchDetach(s *mcclient.ClientSession, mid string, sids []string) []SubmitResult {
	return BatchDo(sids, func(sid string) (jsonutils.JSONObject, error) {
		return this.Detach(s, mid, sid)
	})
}

func (this *JointResourceManager) BatchDetach2(s *mcclient.ClientSession, mid string, sids []string) []SubmitResult {
	return BatchDo(sids, func(sid string) (jsonutils.JSONObject, error) {
		return this.Detach(s, sid, mid)
	})
}

func (this *JointResourceManager) Update(s *mcclient.ClientSession, mid, sid string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/%s/%s/%s", this.Master.KeyString(), url.PathEscape(mid), this.Slave.KeyString(), url.PathEscape(sid))
	result, err := this._put(s, path, this.params2Body(s, params), this.Keyword)
	if err != nil {
		return nil, err
	}
	return this.filterSingleResult(s, result, nil)
}

func (this *JointResourceManager) Patch(s *mcclient.ClientSession, mid, sid string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.Patch2(s, mid, sid, nil, params)
}

func (this *JointResourceManager) Patch2(s *mcclient.ClientSession, mid, sid string, query, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/%s/%s/%s", this.Master.KeyString(), url.PathEscape(mid), this.Slave.KeyString(), url.PathEscape(sid))
	if query != nil {
		qs := query.QueryString()
		if len(qs) > 0 {
			path = fmt.Sprintf("%s?%s", path, qs)
		}
	}
	result, err := this._patch(s, path, this.params2Body(s, params), this.Keyword)
	if err != nil {
		return nil, err
	}
	return this.filterSingleResult(s, result, nil)
}
