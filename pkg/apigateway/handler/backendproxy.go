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
	"net/url"
	"strings"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/proxy"
)

type SBackendServiceProxyHandler struct {
	prefix string
}

func NewBackendServiceProxyHandler(prefix string) *SBackendServiceProxyHandler {
	return &SBackendServiceProxyHandler{
		prefix: prefix,
	}
}

func (h *SBackendServiceProxyHandler) requestManipulator(ctx context.Context, r *http.Request) (*http.Request, error) {
	// remove leading prefixes /api/s/<service>
	path := r.URL.Path[len("/api/s/"):]
	slashPos := strings.Index(path, "/")
	if slashPos <= 0 {
		return r, httperrors.NewBadRequestError("invalid request URL %s", r.URL.Path)
	}
	path = path[slashPos:]
	log.Debugf("Path: %s => %s", r.URL.Path, path)
	r.URL = &url.URL{
		Path:     path,
		RawQuery: r.URL.RawQuery,
		Fragment: r.URL.Fragment,
	}
	return r, nil
}

func (h *SBackendServiceProxyHandler) Bind(app *appsrv.Application) {
	app.AddReverseProxyHandler(h.prefix, h.fetchReverseEndpoint(), h.requestManipulator)
}

func (h *SBackendServiceProxyHandler) fetchReverseEndpoint() *proxy.SEndpointFactory {
	f := func(ctx context.Context, r *http.Request) (string, error) {
		params := appctx.AppContextParams(ctx)
		serviceName := params["<service>"]
		if len(serviceName) == 0 {
			return "", httperrors.NewBadRequestError("no service")
		}
		endpointType := "internalURL"
		session := auth.GetAdminSession(ctx, FetchRegion(r), "")
		ep, err := session.GetServiceURL(serviceName, endpointType)
		if err != nil {
			return "", httperrors.NewBadRequestError("invalid service %s: %s", serviceName, err)
		}
		return getEndpointSchemeHost(ep)
	}
	return proxy.NewEndpointFactory(f, "backendService")
}
