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

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/proxy"
)

type InfluxdbProxyHandler struct {
	prefix      string
	serviceName string
}

func NewInfluxdbProxyHandler(prefix string) *InfluxdbProxyHandler {
	return NewProxyHandlerWithService(prefix, apis.SERVICE_TYPE_INFLUXDB)
}

func NewProxyHandlerWithService(prefix string, serviceName string) *InfluxdbProxyHandler {
	return &InfluxdbProxyHandler{
		prefix:      prefix,
		serviceName: serviceName,
	}
}

func requestManipulator(ctx context.Context, r *http.Request) (*http.Request, error) {
	token, _, _ := fetchAuthInfo(ctx, r)
	if token != nil {
		r.Header.Set("X-Auth-Token", token.GetTokenString())
	}
	return r, nil
}

func (h *InfluxdbProxyHandler) Bind(app *appsrv.Application) {
	app.AddReverseProxyHandler(h.prefix, fetchReverseEndpoint(h.serviceName), requestManipulator)
}

func getEndpointSchemeHost(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	nu := &url.URL{
		Host:   u.Host,
		Scheme: u.Scheme,
	}
	return nu.String(), nil
}

func fetchReverseEndpoint(serviceName string) *proxy.SEndpointFactory {
	f := func(ctx context.Context, r *http.Request) (string, error) {
		endpointType := "internalURL"
		session := auth.GetAdminSession(ctx, FetchRegion(r))
		ep, err := session.GetServiceURL(serviceName, endpointType)
		if err != nil {
			return "", err
		}
		return getEndpointSchemeHost(ep)
	}
	return proxy.NewEndpointFactory(f, serviceName)
}
