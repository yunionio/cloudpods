package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/yunionio/log"
	"github.com/yunionio/pkg/httperrors"
)

type EndpointGenerator func(context.Context, http.ResponseWriter, *http.Request) (string, error)

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
}

func NewHTTPReverseProxy(ef *SEndpointFactory) *SReverseProxy {
	return &SReverseProxy{
		SEndpointFactory: ef,
	}
}

func (p *SReverseProxy) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	endpoint, err := p.generator(ctx, w, r)
	if err != nil {
		httperrors.InternalServerError(w, err.Error())
		return
	}
	remoteUrl, err := url.Parse(endpoint)
	if err != nil {
		httperrors.InternalServerError(w, fmt.Sprintf("Parse remote url %q: %v", endpoint, err))
		return
	}
	log.Debugf("Forwarding to servie: %q, url: %q", p.serviceName, remoteUrl.String())
	proxy := httputil.NewSingleHostReverseProxy(remoteUrl)
	r.Header.Del("Cookie")
	r.Header.Del("X-Auth-Token")
	proxy.ServeHTTP(w, r)
	return
}
