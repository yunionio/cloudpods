// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package modulebase

import (
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/pkg/mcclient"
)

type JointResourceManager struct {
	ResourceManager
	Master Manager
	Slave  Manager
}

var _ JointManager = (*JointResourceManager)(nil)

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

func (this *JointResourceManager) ListDescendent(s *mcclient.ClientSession, mid string, params jsonutils.JSONObject) (*printutils.ListResult, error) {
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

func (this *JointResourceManager) ListDescendent2(s *mcclient.ClientSession, sid string, params jsonutils.JSONObject) (*printutils.ListResult, error) {
	return this.ListAscendent(s, sid, params)
}

func (this *JointResourceManager) ListAscendent(s *mcclient.ClientSession, mid string, params jsonutils.JSONObject) (*printutils.ListResult, error) {
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
	result, err := this._post(s, path, this.params2Body(s, params, this.Keyword), this.Keyword)
	if err != nil {
		return nil, err
	}
	return this.filterSingleResult(s, result, nil)
}

func (this *JointResourceManager) BatchAttach(s *mcclient.ClientSession, mid string, sids []string, params jsonutils.JSONObject) []printutils.SubmitResult {
	return BatchDo(sids, func(sid string) (jsonutils.JSONObject, error) {
		return this.Attach(s, mid, sid, params)
	})
}

func (this *JointResourceManager) BatchAttach2(s *mcclient.ClientSession, mid string, sids []string, params jsonutils.JSONObject) []printutils.SubmitResult {
	return BatchDo(sids, func(sid string) (jsonutils.JSONObject, error) {
		return this.Attach(s, sid, mid, params)
	})
}

func (this *JointResourceManager) Detach(s *mcclient.ClientSession, mid, sid string, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/%s/%s/%s", this.Master.KeyString(), url.PathEscape(mid), this.Slave.KeyString(), url.PathEscape(sid))
	if query != nil {
		qs := query.QueryString()
		if len(qs) > 0 {
			path = fmt.Sprintf("%s?%s", path, qs)
		}
	}
	result, err := this._delete(s, path, nil, this.Keyword)
	if err != nil {
		return nil, err
	}
	return this.filterSingleResult(s, result, nil)
}

func (this *JointResourceManager) BatchDetach(s *mcclient.ClientSession, mid string, sids []string) []printutils.SubmitResult {
	return BatchDo(sids, func(sid string) (jsonutils.JSONObject, error) {
		return this.Detach(s, mid, sid, nil)
	})
}

func (this *JointResourceManager) BatchDetach2(s *mcclient.ClientSession, mid string, sids []string) []printutils.SubmitResult {
	return BatchDo(sids, func(sid string) (jsonutils.JSONObject, error) {
		return this.Detach(s, sid, mid, nil)
	})
}

func (this *JointResourceManager) Update(s *mcclient.ClientSession, mid, sid string, query jsonutils.JSONObject, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/%s/%s/%s", this.Master.KeyString(), url.PathEscape(mid), this.Slave.KeyString(), url.PathEscape(sid))
	if query != nil {
		queryStr := query.QueryString()
		if len(queryStr) > 0 {
			path = fmt.Sprintf("%s?%s", path, queryStr)
		}
	}
	result, err := this._put(s, path, this.params2Body(s, params, this.Keyword), this.Keyword)
	if err != nil {
		return nil, err
	}
	return this.filterSingleResult(s, result, nil)
}

func (this *JointResourceManager) Patch(s *mcclient.ClientSession, mid, sid string, query jsonutils.JSONObject, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/%s/%s/%s", this.Master.KeyString(), url.PathEscape(mid), this.Slave.KeyString(), url.PathEscape(sid))
	if query != nil {
		queryStr := query.QueryString()
		if len(queryStr) > 0 {
			path = fmt.Sprintf("%s?%s", path, queryStr)
		}
	}
	result, err := this._patch(s, path, this.params2Body(s, params, this.Keyword), this.Keyword)
	if err != nil {
		return nil, err
	}
	return this.filterSingleResult(s, result, nil)
}

func (this *JointResourceManager) BatchUpdate(s *mcclient.ClientSession, mid string, sids []string, query jsonutils.JSONObject, params jsonutils.JSONObject) []printutils.SubmitResult {
	return BatchDo(sids, func(sid string) (jsonutils.JSONObject, error) {
		return this.Update(s, mid, sid, query, params)
	})
}

func (this *JointResourceManager) BatchPatch(s *mcclient.ClientSession, mid string, sids []string, query jsonutils.JSONObject, params jsonutils.JSONObject) []printutils.SubmitResult {
	return BatchDo(sids, func(sid string) (jsonutils.JSONObject, error) {
		return this.Patch(s, mid, sid, query, params)
	})
}
