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

package proxy

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/i18n"
)

type EndpointGenerator func(context.Context, *http.Request) (string, error)
type RequestManipulator func(ctx context.Context, r *http.Request) (*http.Request, error)

type SEndpointFactory struct {
	generator   EndpointGenerator
	serviceName string
}

func NewEndpointFactory(f EndpointGenerator, serviceName string) *SEndpointFactory {
	return &SEndpointFactory{
		generator:   f,
		serviceName: serviceName,
	}
}

type SReverseProxy struct {
	*SEndpointFactory
	manipulator RequestManipulator
}

func NewHTTPReverseProxy(ef *SEndpointFactory, m RequestManipulator) *SReverseProxy {
	return &SReverseProxy{
		SEndpointFactory: ef,
		manipulator:      m,
	}
}

func (p *SReverseProxy) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	ctx = i18n.WithRequestLang(ctx, r)
	endpoint, err := p.generator(ctx, r)
	if err != nil {
		httperrors.InternalServerError(ctx, w, "%v", err)
		return
	}
	remoteUrl, err := url.Parse(endpoint)
	if err != nil {
		httperrors.InternalServerError(ctx, w, "failed parsing url %q: %v", endpoint, err)
		return
	}
	log.Debugf("Forwarding to servie: %q, url: %q", p.serviceName, remoteUrl.String())
	proxy := httputil.NewSingleHostReverseProxy(remoteUrl)
	proxy.Transport = &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
	}
	r, err = p.manipulator(ctx, r)
	if err != nil {
		httperrors.InternalServerError(ctx, w, "%v", err)
		return
	}
	proxy.ServeHTTP(w, r)
}
