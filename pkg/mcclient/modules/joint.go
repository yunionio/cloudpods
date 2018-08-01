package modules

import (
	"fmt"
	"net/url"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/mcclient"
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
	return this._get(s, path, this.Keyword)
}

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

func (this *JointResourceManager) ListDescendent(s *mcclient.ClientSession, mid string, params jsonutils.JSONObject) (*ListResult, error) {
	path := fmt.Sprintf("/%s/%s/%s", this.Master.KeyString(), url.PathEscape(mid), this.Slave.KeyString())
	if params != nil {
		qs := params.QueryString()
		if len(qs) > 0 {
			path = fmt.Sprintf("%s?%s", path, qs)
		}
	}
	return this._list(s, path, this.KeywordPlural)
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
	return this._list(s, path, this.KeywordPlural)
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
	return this._post(s, path, this.params2Body(params), this.Keyword)
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
	return this._delete(s, path, nil, this.Keyword)
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
	return this._put(s, path, this.params2Body(params), this.Keyword)
}

func (this *JointResourceManager) Patch(s *mcclient.ClientSession, mid, sid string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/%s/%s/%s", this.Master.KeyString(), url.PathEscape(mid), this.Slave.KeyString(), url.PathEscape(sid))
	return this._patch(s, path, this.params2Body(params), this.Keyword)
}
