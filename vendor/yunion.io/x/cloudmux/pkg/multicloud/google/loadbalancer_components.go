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

package google

import (
	"fmt"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SUrlMap struct {
	SResourceBase

	CreationTimestamp string        `json:"creationTimestamp"`
	HostRules         []HostRule    `json:"hostRules"`
	PathMatchers      []PathMatcher `json:"pathMatchers"`
	DefaultService    string        `json:"defaultService"`
	Fingerprint       string        `json:"fingerprint"`
	Region            string        `json:"region"`
	Kind              string        `json:"kind"`
}

type SForwardingRule struct {
	SResourceBase

	CreationTimestamp   string   `json:"creationTimestamp"`
	Description         string   `json:"description"`
	Region              string   `json:"region"`
	IPAddress           string   `json:"IPAddress"`
	IPProtocol          string   `json:"IPProtocol"`
	PortRange           string   `json:"portRange"`
	Target              string   `json:"target"`
	LoadBalancingScheme string   `json:"loadBalancingScheme"`
	Subnetwork          string   `json:"subnetwork"`
	Network             string   `json:"network"`
	NetworkTier         string   `json:"networkTier"`
	LabelFingerprint    string   `json:"labelFingerprint"`
	Fingerprint         string   `json:"fingerprint"`
	Kind                string   `json:"kind"`
	Ports               []string `json:"ports"`
	BackendService      string   `json:"backendService"`
}

// type STargetProxy struct {
// }
type SBackendServices struct {
	SResourceBase

	CreationTimestamp    string             `json:"creationTimestamp"`
	Description          string             `json:"description"`
	Backends             []Backend          `json:"backends"`
	HealthChecks         []string           `json:"healthChecks"`
	TimeoutSEC           int64              `json:"timeoutSec"`
	Port                 int64              `json:"port"`
	Protocol             string             `json:"protocol"`
	Fingerprint          string             `json:"fingerprint"`
	PortName             string             `json:"portName"`
	SessionAffinity      string             `json:"sessionAffinity"`
	AffinityCookieTTLSEC int64              `json:"affinityCookieTtlSec"`
	Region               string             `json:"region"`
	LoadBalancingScheme  string             `json:"loadBalancingScheme"`
	ConnectionDraining   ConnectionDraining `json:"connectionDraining"`
	LocalityLBPolicy     string             `json:"localityLbPolicy"`
	ConsistentHash       ConsistentHash     `json:"consistentHash"`
	Kind                 string             `json:"kind"`
	EnableCDN            bool               `json:"enableCDN"`
	Network              string             `json:"network"`
}

//
//type STargetPool struct {
//}

type STargetHttpProxy struct {
	SResourceBase

	CreationTimestamp string `json:"creationTimestamp"`
	Description       string `json:"description"`
	URLMap            string `json:"urlMap"`
	Region            string `json:"region"`
	ProxyBind         bool   `json:"proxyBind"`
	Fingerprint       string `json:"fingerprint"`
	Kind              string `json:"kind"`
}

type STargetHttpsProxy struct {
	SResourceBase

	CreationTimestamp   string   `json:"creationTimestamp"`
	Description         string   `json:"description"`
	URLMap              string   `json:"urlMap"`
	SSLCertificates     []string `json:"sslCertificates"`
	QuicOverride        string   `json:"quicOverride"`
	SSLPolicy           string   `json:"sslPolicy"`
	Region              string   `json:"region"`
	ProxyBind           bool     `json:"proxyBind"`
	ServerTLSPolicy     string   `json:"serverTlsPolicy"`
	AuthorizationPolicy string   `json:"authorizationPolicy"`
	Fingerprint         string   `json:"fingerprint"`
	Kind                string   `json:"kind"`
}

type STargetTcpProxy struct {
	SResourceBase

	Kind              string `json:"kind"`
	CreationTimestamp string `json:"creationTimestamp"`
	Description       string `json:"description"`
	Service           string `json:"service"`
	Region            string `json:"region"`
	ProxyBind         bool   `json:"proxyBind"`
}

type SInstanceGroup struct {
	SResourceBase
	region    *SRegion
	instances []SInstanceGroupInstance

	CreationTimestamp string      `json:"creationTimestamp"`
	Description       string      `json:"description"`
	NamedPorts        []NamedPort `json:"namedPorts"`
	Network           string      `json:"network"`
	Fingerprint       string      `json:"fingerprint"`
	Zone              string      `json:"zone"`
	Size              int64       `json:"size"`
	Region            string      `json:"region"`
	Subnetwork        string      `json:"subnetwork"`
	Kind              string      `json:"kind"`
}

type SInstanceGroupInstance struct {
	instanceGroup *SInstanceGroup
	Instance      string      `json:"instance"`
	Status        string      `json:"status"`
	NamedPorts    []NamedPort `json:"namedPorts"`
}

type NamedPort struct {
	Name string `json:"name"`
	Port int64  `json:"port"`
}

type ConsistentHash struct {
	HTTPCookie      HTTPCookie `json:"httpCookie"`
	MinimumRingSize string     `json:"minimumRingSize"`
}

type HTTPCookie struct {
	Name string `json:"name"`
	Path string `json:"path"`
	TTL  TTL    `json:"ttl"`
}

type TTL struct {
	Seconds string `json:"seconds"`
	Nanos   int64  `json:"nanos"`
}

type RouteAction struct {
	WeightedBackendServices []WeightedBackendService `json:"weightedBackendServices"`
	URLRewrite              URLRewrite               `json:"urlRewrite"`
	Timeout                 MaxStreamDuration        `json:"timeout"`
	RetryPolicy             RetryPolicy              `json:"retryPolicy"`
	RequestMirrorPolicy     RequestMirrorPolicy      `json:"requestMirrorPolicy"`
	CorsPolicy              CorsPolicy               `json:"corsPolicy"`
	FaultInjectionPolicy    FaultInjectionPolicy     `json:"faultInjectionPolicy"`
	MaxStreamDuration       MaxStreamDuration        `json:"maxStreamDuration"`
}

type CorsPolicy struct {
	AllowOrigins       []string `json:"allowOrigins"`
	AllowOriginRegexes []string `json:"allowOriginRegexes"`
	AllowMethods       []string `json:"allowMethods"`
	AllowHeaders       []string `json:"allowHeaders"`
	ExposeHeaders      []string `json:"exposeHeaders"`
	MaxAge             int64    `json:"maxAge"`
	AllowCredentials   bool     `json:"allowCredentials"`
	Disabled           bool     `json:"disabled"`
}

type FaultInjectionPolicy struct {
	Delay Delay `json:"delay"`
	Abort Abort `json:"abort"`
}

type Abort struct {
	HTTPStatus int64   `json:"httpStatus"`
	Percentage float64 `json:"percentage"`
}

type Delay struct {
	FixedDelay MaxStreamDuration `json:"fixedDelay"`
	Percentage float64           `json:"percentage"`
}

type MaxStreamDuration struct {
	Seconds string `json:"seconds"`
	Nanos   int64  `json:"nanos"`
}

type RequestMirrorPolicy struct {
	BackendService string `json:"backendService"`
}

type RetryPolicy struct {
	RetryConditions []string          `json:"retryConditions"`
	NumRetries      int64             `json:"numRetries"`
	PerTryTimeout   MaxStreamDuration `json:"perTryTimeout"`
}

type URLRewrite struct {
	PathPrefixRewrite string `json:"pathPrefixRewrite"`
	HostRewrite       string `json:"hostRewrite"`
}

type WeightedBackendService struct {
	BackendService string       `json:"backendService"`
	Weight         int64        `json:"weight"`
	HeaderAction   HeaderAction `json:"headerAction"`
}

type HeaderAction struct {
	RequestHeadersToRemove  []string       `json:"requestHeadersToRemove"`
	RequestHeadersToAdd     []HeadersToAdd `json:"requestHeadersToAdd"`
	ResponseHeadersToRemove []string       `json:"responseHeadersToRemove"`
	ResponseHeadersToAdd    []HeadersToAdd `json:"responseHeadersToAdd"`
}

type HeadersToAdd struct {
	HeaderName  string `json:"headerName"`
	HeaderValue string `json:"headerValue"`
	Replace     bool   `json:"replace"`
}

type URLRedirect struct {
	HostRedirect         string `json:"hostRedirect"`
	PathRedirect         string `json:"pathRedirect"`
	PrefixRedirect       string `json:"prefixRedirect"`
	RedirectResponseCode string `json:"redirectResponseCode"`
	HTTPSRedirect        bool   `json:"httpsRedirect"`
	StripQuery           bool   `json:"stripQuery"`
}

type HostRule struct {
	Description string   `json:"description"`
	Hosts       []string `json:"hosts"`
	PathMatcher string   `json:"pathMatcher"`
}

type PathMatcher struct {
	Name               string       `json:"name"`
	Description        string       `json:"description"`
	DefaultService     string       `json:"defaultService"`
	DefaultRouteAction RouteAction  `json:"defaultRouteAction"`
	DefaultURLRedirect URLRedirect  `json:"defaultUrlRedirect"`
	PathRules          []PathRule   `json:"pathRules"`
	RouteRules         []RouteRule  `json:"routeRules"`
	HeaderAction       HeaderAction `json:"headerAction"`
}

type PathRule struct {
	Service     string      `json:"service"`
	RouteAction RouteAction `json:"routeAction"`
	URLRedirect URLRedirect `json:"urlRedirect"`
	Paths       []string    `json:"paths"`
}

type RouteRule struct {
	Priority     int64        `json:"priority"`
	Description  string       `json:"description"`
	MatchRules   []MatchRule  `json:"matchRules"`
	Service      string       `json:"service"`
	RouteAction  RouteAction  `json:"routeAction"`
	URLRedirect  URLRedirect  `json:"urlRedirect"`
	HeaderAction HeaderAction `json:"headerAction"`
}

type MatchRule struct {
	PrefixMatch           string                `json:"prefixMatch"`
	FullPathMatch         string                `json:"fullPathMatch"`
	RegexMatch            string                `json:"regexMatch"`
	IgnoreCase            bool                  `json:"ignoreCase"`
	HeaderMatches         []HeaderMatch         `json:"headerMatches"`
	QueryParameterMatches []QueryParameterMatch `json:"queryParameterMatches"`
	MetadataFilters       []MetadataFilter      `json:"metadataFilters"`
}

type HeaderMatch struct {
	HeaderName   string     `json:"headerName"`
	ExactMatch   string     `json:"exactMatch"`
	RegexMatch   string     `json:"regexMatch"`
	RangeMatch   RangeMatch `json:"rangeMatch"`
	PresentMatch bool       `json:"presentMatch"`
	PrefixMatch  string     `json:"prefixMatch"`
	SuffixMatch  string     `json:"suffixMatch"`
	InvertMatch  bool       `json:"invertMatch"`
}

type RangeMatch struct {
	RangeStart string `json:"rangeStart"`
	RangeEnd   string `json:"rangeEnd"`
}

type MetadataFilter struct {
	FilterMatchCriteria string   `json:"filterMatchCriteria"`
	FilterLabels        []Header `json:"filterLabels"`
}

type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type QueryParameterMatch struct {
	Name         string `json:"name"`
	PresentMatch bool   `json:"presentMatch"`
	ExactMatch   string `json:"exactMatch"`
	RegexMatch   string `json:"regexMatch"`
}

type Test struct {
	Description                  string   `json:"description"`
	Host                         string   `json:"host"`
	Path                         string   `json:"path"`
	Headers                      []Header `json:"headers"`
	Service                      string   `json:"service"`
	ExpectedOutputURL            string   `json:"expectedOutputUrl"`
	ExpectedRedirectResponseCode int64    `json:"expectedRedirectResponseCode"`
}

type Backend struct {
	Group         string `json:"group"`
	BalancingMode string `json:"balancingMode"`
	Failover      bool   `json:"failover"`
}

type ConnectionDraining struct {
	DrainingTimeoutSEC int64 `json:"drainingTimeoutSec"`
}

type HealthChecks struct {
	SResourceBase

	CreationTimestamp  string          `json:"creationTimestamp"`
	Description        string          `json:"description"`
	CheckIntervalSEC   int64           `json:"checkIntervalSec"`
	TimeoutSEC         int64           `json:"timeoutSec"`
	UnhealthyThreshold int64           `json:"unhealthyThreshold"`
	HealthyThreshold   int64           `json:"healthyThreshold"`
	Type               string          `json:"type"`
	HTTPSHealthCheck   HTTPHealthCheck `json:"httpsHealthCheck"`
	Region             string          `json:"region"`
	Kind               string          `json:"kind"`
	Http2HealthCheck   HTTPHealthCheck `json:"http2HealthCheck"`
	TCPHealthCheck     TCPHealthCheck  `json:"tcpHealthCheck"`
	SSLHealthCheck     SSLHealthCheck  `json:"sslHealthCheck"`
	HTTPHealthCheck    HTTPHealthCheck `json:"httpHealthCheck"`
}

type HTTPHealthCheck struct {
	Port        int64  `json:"port"`
	Host        string `json:"host"`
	RequestPath string `json:"requestPath"`
	ProxyHeader string `json:"proxyHeader"`
	Response    string `json:"response"`
}

type SSLHealthCheck struct {
	Port        int64  `json:"port"`
	Request     string `json:"request"`
	Response    string `json:"response"`
	ProxyHeader string `json:"proxyHeader"`
}

type TCPHealthCheck struct {
	Port        int64  `json:"port"`
	ProxyHeader string `json:"proxyHeader"`
}

type LogConfig struct {
	Enable bool `json:"enable"`
}

func (self *SInstanceGroup) GetInstances() ([]SInstanceGroupInstance, error) {
	if self.instances != nil {
		return self.instances, nil
	}

	ret := make([]SInstanceGroupInstance, 0)
	resourceId := strings.Replace(self.GetGlobalId(), fmt.Sprintf("projects/%s/", self.region.GetProjectId()), "", -1)
	err := self.region.listAll("POST", resourceId+"/listInstances", nil, &ret)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return nil, nil
		}

		return nil, errors.Wrap(err, "ListAll")
	}

	for i := range ret {
		ret[i].instanceGroup = self
	}

	self.instances = ret
	return ret, nil
}

func (self *SRegion) getLoadbalancerComponents(resource string, filter string, result interface{}) error {
	url := fmt.Sprintf("regions/%s/%s", self.Name, resource)
	params := map[string]string{}
	if len(filter) > 0 {
		params["filter"] = filter
	}

	err := self.ListAll(url, params, result)
	if err != nil {
		return errors.Wrap(err, "ListAll")
	}

	return nil
}

func (self *SRegion) getInstanceGroups(zoneId, resource string, filter string, result interface{}) error {
	url := fmt.Sprintf("zones/%s/%s", zoneId, resource)
	params := map[string]string{}
	if len(filter) > 0 {
		params["filter"] = filter
	}

	err := self.ListAll(url, params, result)
	if err != nil {
		return errors.Wrap(err, "ListAll")
	}

	return nil
}

/*
As mentioned by Patrick W, there is no direct entity 'load balancer', its just a collection of components.
The list seen in the UI that appears to be the load balancer is actually the url-map component,
which can be seen via the API with:
gcloud compute url-maps list
*/
func (self *SRegion) GetRegionalUrlMaps(filter string) ([]SUrlMap, error) {
	ret := make([]SUrlMap, 0)
	err := self.getLoadbalancerComponents("urlMaps", filter, &ret)
	return ret, err
}

func (self *SRegion) GetRegionalBackendServices(filter string) ([]SBackendServices, error) {
	ret := make([]SBackendServices, 0)
	err := self.getLoadbalancerComponents("backendServices", filter, &ret)
	return ret, err
}

func (self *SRegion) GetRegionalForwardingRule(filter string) ([]SForwardingRule, error) {
	ret := make([]SForwardingRule, 0)
	err := self.getLoadbalancerComponents("forwardingRules", filter, &ret)
	return ret, err
}

func (self *SRegion) GetRegionalTargetHttpProxies(filter string) ([]STargetHttpProxy, error) {
	ret := make([]STargetHttpProxy, 0)
	err := self.getLoadbalancerComponents("targetHttpProxies", filter, &ret)
	return ret, err
}

func (self *SRegion) GetRegionalTargetHttpsProxies(filter string) ([]STargetHttpsProxy, error) {
	ret := make([]STargetHttpsProxy, 0)
	err := self.getLoadbalancerComponents("targetHttpsProxies", filter, &ret)
	return ret, err
}

func (self *SRegion) GetRegionalInstanceGroups(filter string) ([]SInstanceGroup, error) {
	ret := make([]SInstanceGroup, 0)
	err := self.getLoadbalancerComponents("instanceGroups", filter, &ret)
	for i := range ret {
		ret[i].region = self
	}
	return ret, err
}

func (self *SRegion) GetRegionalHealthChecks(filter string) ([]HealthChecks, error) {
	ret := make([]HealthChecks, 0)
	err := self.getLoadbalancerComponents("healthChecks", filter, &ret)
	return ret, err
}

func (self *SRegion) GetGlobalHealthChecks(filter string) ([]HealthChecks, error) {
	ret := make([]HealthChecks, 0)
	params := map[string]string{}
	if len(filter) > 0 {
		params["filter"] = filter
	}

	err := self.ListAll("global/healthChecks", params, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "ListAll")
	}

	return ret, err
}

func (self *SRegion) GetRegionalSslCertificates(filter string) ([]SLoadbalancerCertificate, error) {
	ret := make([]SLoadbalancerCertificate, 0)
	err := self.getLoadbalancerComponents("sslCertificates", filter, &ret)
	for i := range ret {
		ret[i].region = self
	}
	return ret, err
}

func (self *SLoadbalancer) GetTargetHttpsProxies() ([]STargetHttpsProxy, error) {
	ret := make([]STargetHttpsProxy, 0)
	filter := fmt.Sprintf("urlMap eq %s", self.GetId())
	err := self.region.getLoadbalancerComponents("targetHttpsProxies", filter, &ret)
	return ret, err
}

func (self *SLoadbalancer) GetTargetHttpProxies() ([]STargetHttpProxy, error) {
	ret := make([]STargetHttpProxy, 0)
	filter := fmt.Sprintf("urlMap eq %s", self.GetId())
	err := self.region.getLoadbalancerComponents("targetHttpProxies", filter, &ret)
	return ret, err
}

// ForwardingRule 目标是: target proxy or backend service
// http&https 是由target proxy 转发到后端服务
func (self *SLoadbalancer) GetForwardingRules() ([]SForwardingRule, error) {
	if self.forwardRules != nil {
		return self.forwardRules, nil
	}

	hps, err := self.GetTargetHttpProxies()
	if err != nil {
		return nil, errors.Wrap(err, "GetTargetHttpProxies")
	}

	hsps, err := self.GetTargetHttpsProxies()
	if err != nil {
		return nil, errors.Wrap(err, "GetTargetHttpsProxies")
	}

	targets := make([]string, 0)
	for i := range hps {
		targets = append(targets, fmt.Sprintf(`(target="%s")`, hps[i].GetId()))
	}

	for i := range hsps {
		targets = append(targets, fmt.Sprintf(`(target="%s")`, hsps[i].GetId()))
	}

	if strings.Contains(self.GetId(), "/backendServices/") {
		targets = append(targets, fmt.Sprintf(`(backendService="%s")`, self.GetId()))
	}

	if len(targets) == 0 {
		return []SForwardingRule{}, nil
	}

	filter := strings.Join(targets, " OR ")
	ret := make([]SForwardingRule, 0)
	err = self.region.getLoadbalancerComponents("forwardingRules", filter, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "GetForwardingRules")
	}

	if len(ret) > 0 {
		self.forwardRules = ret
	}
	return ret, nil
}

func (self *SLoadbalancer) GetBackendServices() ([]SBackendServices, error) {
	if self.isHttpLb && self.urlMap != nil {
		ret := make([]SBackendServices, 0)
		ids := []string{self.urlMap.DefaultService}
		for i := range self.urlMap.PathMatchers {
			ps := self.urlMap.PathMatchers[i]
			if len(ps.DefaultService) > 0 && !utils.IsInStringArray(ps.DefaultService, ids) {
				ids = append(ids, ps.DefaultService)
			}

			for j := range ps.PathRules {
				if len(ps.PathRules[j].Service) > 0 && !utils.IsInStringArray(ps.PathRules[j].Service, ids) {
					ids = append(ids, ps.PathRules[j].Service)
				}
			}
		}

		filters := []string{}
		for i := range ids {
			filters = append(filters, fmt.Sprintf(`(selfLink="%s")`, ids[i]))
		}

		if len(filters) == 0 {
			return []SBackendServices{}, nil
		}
		err := self.region.getLoadbalancerComponents("backendServices", strings.Join(filters, " OR "), &ret)
		self.backendServices = ret
		return ret, err
	}

	return self.backendServices, nil
}

func (self *SLoadbalancer) GetInstanceGroupsMap() (map[string]SInstanceGroup, error) {
	igs, err := self.GetInstanceGroups()
	if err != nil {
		return nil, errors.Wrap(err, "GetInstanceGroups")
	}

	ret := make(map[string]SInstanceGroup, 0)
	for i := range igs {
		ig := igs[i]
		ig.region = self.region
		ret[ig.SelfLink] = ig
	}

	return ret, nil
}

func (self *SLoadbalancer) GetInstanceGroups() ([]SInstanceGroup, error) {
	if self.instanceGroups != nil {
		return self.instanceGroups, nil
	}

	if self.backendServices == nil {
		bss, err := self.GetBackendServices()
		if err != nil {
			return nil, errors.Wrap(err, "GetBackendServices")
		}
		self.backendServices = bss
	}

	bgs := []string{}
	for i := range self.backendServices {
		_bgs := self.backendServices[i].Backends
		for j := range _bgs {
			if !utils.IsInStringArray(_bgs[j].Group, bgs) {
				bgs = append(bgs, _bgs[j].Group)
			}
		}
	}

	if len(bgs) == 0 {
		return []SInstanceGroup{}, nil
	}

	regionFilters := []string{}
	zonesFilter := map[string][]string{}
	for i := range bgs {
		if !strings.Contains(bgs[i], "/zones/") {
			regionFilters = append(regionFilters, fmt.Sprintf(`(selfLink="%s")`, bgs[i]))
		} else {
			ig := bgs[i]
			index := strings.Index(ig, "/zones/")
			zoneId := strings.Split(ig[index:], "/")[2]
			if fs, ok := zonesFilter[zoneId]; ok {
				f := fmt.Sprintf(`(selfLink="%s")`, ig)
				if !utils.IsInStringArray(f, fs) {
					zonesFilter[zoneId] = append(fs, f)
				}
			} else {
				zonesFilter[zoneId] = []string{fmt.Sprintf(`(selfLink="%s")`, ig)}
			}
		}
	}

	igs := make([]SInstanceGroup, 0)
	// regional instance groups
	if len(regionFilters) > 0 {
		_igs, err := self.region.GetRegionalInstanceGroups(strings.Join(regionFilters, " OR "))
		if err != nil {
			return nil, errors.Wrap(err, "GetRegionalInstanceGroups")
		}

		igs = append(igs, _igs...)
	}

	for z, fs := range zonesFilter {
		_igs := make([]SInstanceGroup, 0)
		err := self.region.getInstanceGroups(z, "instanceGroups", strings.Join(fs, " OR "), &_igs)
		if err != nil {
			return nil, errors.Wrap(err, "getInstanceGroups")
		}

		igs = append(igs, _igs...)
	}

	self.instanceGroups = igs
	return igs, nil
}

func (self *SLoadbalancer) GetHealthChecks() ([]HealthChecks, error) {
	if self.healthChecks != nil {
		return self.healthChecks, nil
	}

	hcs, err := self.region.GetRegionalHealthChecks("")
	if err != nil {
		return nil, errors.Wrap(err, "GetRegionalHealthChecks")
	}

	ghcs, err := self.region.GetGlobalHealthChecks("")
	if err != nil {
		return nil, errors.Wrap(err, "GetGlobalHealthChecks")
	}

	self.healthChecks = append(self.healthChecks, ghcs...)
	self.healthChecks = append(self.healthChecks, hcs...)
	return self.healthChecks, err
}

func (self *SLoadbalancer) GetHealthCheckMaps() (map[string]HealthChecks, error) {
	hcs, err := self.GetHealthChecks()
	if err != nil {
		return nil, errors.Wrap(err, "GetHealthChecks")
	}

	ret := map[string]HealthChecks{}
	for i := range hcs {
		ret[hcs[i].SelfLink] = hcs[i]
	}
	return ret, err
}
