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

package handler

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

type Request struct {
	ctx context.Context
	w   http.ResponseWriter
	r   *http.Request
	err error

	session *mcclient.ClientSession
	params  map[string]string
	query   jsonutils.JSONObject
	body    jsonutils.JSONObject
	mod1    modulebase.Manager
	mod2    modulebase.Manager
	mod3    modulebase.Manager
}

func newRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) *Request {
	params := appctx.AppContextParams(ctx)
	token := AppContextToken(ctx)
	session := auth.GetSession(ctx, token, FetchRegion(r), params[APIVer])
	query, err := jsonutils.ParseQueryString(r.URL.RawQuery)
	req := &Request{
		ctx:     ctx,
		w:       w,
		r:       r,
		session: session,
		params:  params,
		query:   query,
	}
	if err != nil {
		req.err = errors.Errorf("Parse query string %s failed: %v", r.URL.RawQuery, err)
		return req
	}
	var body jsonutils.JSONObject = nil
	if utils.IsInStringArray(r.Method, []string{PUT, POST, DELETE, PATCH}) {
		body, err = appsrv.FetchJSON(r)
		if err != nil {
			req.err = errors.Errorf("failed to decode JSON request body: %v", err)
			return req
		}
		req.body = body
	}
	return req
}

func (req *Request) Error() error {
	return req.err
}

func (req *Request) Session() *mcclient.ClientSession {
	return req.session
}

func (req *Request) Query() jsonutils.JSONObject {
	return req.query
}

func (req *Request) Body() jsonutils.JSONObject {
	return req.body
}

func (req *Request) Params() map[string]string {
	return req.params
}

func (req *Request) ResName() string {
	return req.params[ResName]
}

func (req *Request) ResID() string {
	return req.params[ResID]
}

func (req *Request) ResName2() string {
	return req.params[ResName2]
}

func (req *Request) ResID2() string {
	return req.params[ResID2]
}

func (req *Request) ResName3() string {
	return req.params[ResName3]
}

func (req *Request) ResID3() string {
	return req.params[ResID3]
}

func (req *Request) Action() string {
	return req.params[Action]
}

func (req *Request) Spec() string {
	return req.params[Spec]
}

func (req *Request) Mod1() modulebase.Manager {
	return req.mod1
}

func (req *Request) Mod2() modulebase.Manager {
	return req.mod2
}

func (req *Request) Mod3() modulebase.Manager {
	return req.mod3
}

func (req Request) findMod(resKey string) (modulebase.Manager, error) {
	resName := req.params[resKey]
	module, err := modulebase.GetModule(req.session, resName)
	if err != nil {
		return nil, errors.Errorf("found module by %s: %v", resName, err)
	}
	if module == nil {
		return nil, httperrors.NewNotFoundError("resource %s module not exists", resName)
	}
	return module, nil
}

func (req *Request) WithMod1() *Request {
	if req.err != nil {
		return req
	}
	mod, err := req.findMod(ResName)
	if err != nil {
		req.err = err
		return req
	}
	req.mod1 = mod
	return req
}

func (req *Request) WithMod2() *Request {
	if req.err != nil {
		return req
	}
	mod, err := req.findMod(ResName2)
	if err != nil {
		req.err = err
		return req
	}
	req.mod2 = mod
	return req
}

func (req *Request) WithMod3() *Request {
	if req.err != nil {
		return req
	}
	mod, err := req.findMod(ResName3)
	if err != nil {
		req.err = err
		return req
	}
	req.mod3 = mod
	return req
}
