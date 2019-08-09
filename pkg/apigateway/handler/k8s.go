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
	"fmt"
	"net/http"
	"path"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

type K8sResourceHandler struct {
	prefix string
}

type k8sResourceEnv struct {
	session   *mcclient.ClientSession
	cluster   string
	namespace string
	kind      string
	name      string
}

func NewK8sResourceHandler(prefix string) *K8sResourceHandler {
	return &K8sResourceHandler{
		prefix: prefix,
	}
}

func (h *K8sResourceHandler) Bind(app *appsrv.Application) {
	app.AddHandler(GET, h.instancePrefix(""), FetchAuthToken(h.Get))
	app.AddHandler(GET, h.instancePrefix("yaml"), FetchAuthToken(h.GetYAML))
	app.AddHandler(PUT, h.instancePrefix(""), FetchAuthToken(h.Put))
	app.AddHandler(DELETE, h.instancePrefix(""), FetchAuthToken(h.Delete))
}

func (h *K8sResourceHandler) instancePrefix(segs ...string) string {
	url := fmt.Sprintf("%s/<kind>/<name>", h.prefix)
	if len(segs) == 0 {
		return url
	}
	newSegs := []string{url}
	newSegs = append(newSegs, segs...)
	return path.Join(newSegs...)
}

func (h *K8sResourceHandler) fetchEnv(ctx context.Context, req *http.Request) (*k8sResourceEnv, error) {
	pathParams := appctx.AppContextParams(ctx)
	kind := pathParams["<kind>"]
	resName := pathParams["<name>"]
	params, err := jsonutils.ParseQueryString(req.URL.RawQuery)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Parse query: %v", err)
	}
	namespace, _ := params.GetString("namespace")
	cluster, _ := params.GetString("cluster")
	token := AppContextToken(ctx)
	s := auth.GetSession(ctx, token, FetchRegion(req), "")
	return &k8sResourceEnv{
		session:   s,
		cluster:   cluster,
		namespace: namespace,
		kind:      kind,
		name:      resName,
	}, nil
}

func (h *K8sResourceHandler) Get(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	env, err := h.fetchEnv(ctx, req)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	detail, err := k8s.RawResource.Get(env.session, env.kind, env.namespace, env.name, env.cluster)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, detail)
}

func (h *K8sResourceHandler) GetYAML(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	env, err := h.fetchEnv(ctx, req)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	detail, err := k8s.RawResource.GetYAML(env.session, env.kind, env.namespace, env.name, env.cluster)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.Send(w, string(detail))
}

func (h *K8sResourceHandler) Put(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	env, err := h.fetchEnv(ctx, req)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	data, err := appsrv.FetchJSON(req)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	err = k8s.RawResource.Put(env.session, env.kind, env.namespace, env.name, data, env.cluster)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *K8sResourceHandler) Delete(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	env, err := h.fetchEnv(ctx, req)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	if err := k8s.RawResource.Delete(env.session, env.kind, env.namespace, env.name, env.cluster); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
