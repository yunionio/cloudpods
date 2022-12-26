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
	"net/url"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"

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

func removeLeadingSlash(p string) string {
	for len(p) > 0 && p[0] == '/' {
		p = p[1:]
	}
	return p
}

func (h *SBackendServiceProxyHandler) requestManipulator(ctx context.Context, r *http.Request) (*http.Request, error) {
	// remove leading prefixes /api/s/<service>
	path := r.URL.Path[len("/api/s/"):]
	path = removeLeadingSlash(path)
	slashPos := strings.Index(path, "/")
	if slashPos <= 0 {
		return r, httperrors.NewBadRequestError("invalid request URL %s", r.URL.Path)
	}
	path = path[slashPos:]
	if strings.HasPrefix(path, "/r/") {
		path = path[len("/r/"):]
		path = removeLeadingSlash(path)
		slashPos := strings.Index(path, "/")
		if slashPos <= 0 {
			return r, httperrors.NewBadRequestError("invalid request URL %s", r.URL.Path)
		}
		path = path[slashPos:]
		if strings.HasPrefix(path, "/z/") {
			path = path[len("/z/"):]
			path = removeLeadingSlash(path)
			slashPos := strings.Index(path, "/")
			if slashPos <= 0 {
				return r, httperrors.NewBadRequestError("invalid request URL %s", r.URL.Path)
			}
			path = path[slashPos:]
		}
	}
	log.Debugf("Path: %s => %s", r.URL.Path, path)
	r.URL = &url.URL{
		Path:     path,
		RawQuery: r.URL.RawQuery,
		Fragment: r.URL.Fragment,
	}
	return r, nil
}

func (h *SBackendServiceProxyHandler) Bind(app *appsrv.Application) {
	app.AddReverseProxyHandlerWithCallbackConfig(
		h.prefix, h.fetchReverseEndpoint(), h.requestManipulator,
		func(method string, hi *appsrv.SHandlerInfo) *appsrv.SHandlerInfo {
			if method == "OPTIONS" {
				return nil
			}
			if method == "GET" || method == "PUT" || method == "POST" {
				hi.SetProcessTimeout(6 * time.Hour)
			}
			var worker *appsrv.SWorkerManager
			if method == "GET" || method == "HEAD" {
				worker = appsrv.NewWorkerManager("apigateway-backend-api-read", 8, appsrv.DEFAULT_BACKLOG, false)
			} else {
				worker = appsrv.NewWorkerManager("apigateway-backend-api-write", 4, appsrv.DEFAULT_BACKLOG, false)
			}
			hi.SetWorkerManager(worker)
			return hi
		},
	)
}

func (h *SBackendServiceProxyHandler) fetchReverseEndpoint() *proxy.SEndpointFactory {
	f := func(ctx context.Context, r *http.Request) (string, error) {
		params := appctx.AppContextParams(ctx)
		serviceName := params["<service>"]
		if len(serviceName) == 0 {
			return "", httperrors.NewBadRequestError("no service")
		}
		path := r.URL.Path
		serviceSeg := fmt.Sprintf("/api/s/%s/", serviceName)
		pos := strings.Index(path, serviceSeg)
		if pos < 0 {
			return "", httperrors.NewBadRequestError("malformed URL, expect service")
		}
		path = path[pos+len(serviceSeg):]
		path = removeLeadingSlash(path)
		region := FetchRegion(r)
		zone := ""
		if strings.HasPrefix(path, "r/") {
			path = path[len("r/"):]
			path = removeLeadingSlash(path)
			slashPos := strings.Index(path, "/")
			if slashPos <= 0 {
				return "", httperrors.NewBadRequestError("malformed URL, expect region")
			}
			region = path[:slashPos]
			path = path[slashPos+1:]
			if strings.HasPrefix(path, "z/") {
				path = path[len("z/"):]
				path = removeLeadingSlash(path)
				slashPos := strings.Index(path, "/")
				if slashPos <= 0 {
					return "", httperrors.NewBadRequestError("malformed URL, expect zone")
				}
				zone = path[:slashPos]
			}
		}
		endpointType := "internalURL"
		session := auth.GetAdminSession(ctx, region)
		if len(zone) > 0 {
			session.SetZone(zone)
		}
		ep, err := session.GetServiceURL(serviceName, endpointType)
		if err != nil {
			return "", httperrors.NewBadRequestError("invalid service %s: %s", serviceName, err)
		}
		return getEndpointSchemeHost(ep)
	}
	return proxy.NewEndpointFactory(f, "backendService")
}
