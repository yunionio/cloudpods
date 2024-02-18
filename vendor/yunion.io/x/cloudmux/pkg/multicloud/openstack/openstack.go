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

package openstack

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/version"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/cloudmux/pkg/multicloud/openstack/oscli"
)

const (
	CLOUD_PROVIDER_OPENSTACK = api.CLOUD_PROVIDER_OPENSTACK
	OPENSTACK_DEFAULT_REGION = "RegionOne"

	OPENSTACK_SERVICE_COMPUTE      = "compute"
	OPENSTACK_SERVICE_NETWORK      = "network"
	OPENSTACK_SERVICE_IDENTITY     = "identity"
	OPENSTACK_SERVICE_VOLUMEV3     = "volumev3"
	OPENSTACK_SERVICE_VOLUMEV2     = "volumev2"
	OPENSTACK_SERVICE_VOLUME       = "volume"
	OPENSTACK_SERVICE_IMAGE        = "image"
	OPENSTACK_SERVICE_LOADBALANCER = "load-balancer"

	ErrNoEndpoint = errors.Error("no valid endpoint")
)

type OpenstackClientConfig struct {
	cpcfg cloudprovider.ProviderConfig

	authURL       string
	username      string
	password      string
	project       string
	projectDomain string

	domainName   string
	endpointType string

	debug bool
}

func NewOpenstackClientConfig(authURL, username, password, project, projectDomain string) *OpenstackClientConfig {
	cfg := &OpenstackClientConfig{
		authURL:       authURL,
		username:      username,
		password:      password,
		project:       project,
		projectDomain: projectDomain,
	}
	return cfg
}

func (cfg *OpenstackClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *OpenstackClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *OpenstackClientConfig) DomainName(domainName string) *OpenstackClientConfig {
	cfg.domainName = domainName
	return cfg
}

func (cfg *OpenstackClientConfig) EndpointType(endpointType string) *OpenstackClientConfig {
	cfg.endpointType = endpointType
	return cfg
}

func (cfg *OpenstackClientConfig) Debug(debug bool) *OpenstackClientConfig {
	cfg.debug = debug
	return cfg
}

type SOpenStackClient struct {
	*OpenstackClientConfig

	tokenCredential oscli.TokenCredential
	iregions        []cloudprovider.ICloudRegion

	defaultRegionName string

	projects []SProject
}

func NewOpenStackClient(cfg *OpenstackClientConfig) (*SOpenStackClient, error) {
	cli := &SOpenStackClient{
		OpenstackClientConfig: cfg,
	}
	err := cli.fetchToken()
	if err != nil {
		return nil, err
	}
	return cli, cli.fetchRegions()
}

func (cli *SOpenStackClient) getDefaultRegionName() string {
	return cli.defaultRegionName
}

func (cli *SOpenStackClient) getProjectToken(projectId, projectName string) (oscli.TokenCredential, error) {
	client := cli.getDefaultClient()
	tokenCredential, err := client.Authenticate(cli.username, cli.password, cli.domainName, projectName, cli.projectDomain)
	if err != nil {
		e, ok := err.(*httputils.JSONClientError)
		if ok {
			// 避免有泄漏密码的风险
			e.Request.Body = nil
			return nil, errors.Wrap(e, "Authenticate")
		}
		return nil, errors.Wrap(err, "Authenticate")
	}
	return tokenCredential, nil
}

func (cli *SOpenStackClient) GetCloudRegionExternalIdPrefix() string {
	return fmt.Sprintf("%s/%s/", CLOUD_PROVIDER_OPENSTACK, cli.cpcfg.Id)
}

func (cli *SOpenStackClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccount := cloudprovider.SSubAccount{
		Account: fmt.Sprintf("%s/%s", cli.project, cli.username),
		Name:    cli.cpcfg.Name,
		Id:      cli.tokenCredential.GetProjectDomainId(),

		HealthStatus: api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}
	if len(cli.domainName) > 0 {
		subAccount.Account = fmt.Sprintf("%s/%s", subAccount.Account, cli.domainName)
	}
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (cli *SOpenStackClient) fetchRegions() error {
	regions := cli.tokenCredential.GetRegions()
	cli.iregions = make([]cloudprovider.ICloudRegion, len(regions))
	for i := 0; i < len(regions); i++ {
		region := SRegion{client: cli, Name: regions[i]}
		cli.iregions[i] = &region
		cli.defaultRegionName = regions[0]
	}
	return nil
}

type OpenstackError struct {
	httputils.JSONClientError
}

func (ce *OpenstackError) ParseErrorFromJsonResponse(statusCode int, status string, body jsonutils.JSONObject) error {
	if body != nil {
		body.Unmarshal(ce)
	}
	if ce.Code == 0 {
		ce.Code = statusCode
	}
	if len(ce.Details) == 0 && body != nil {
		ce.Details = body.String()
	}
	if len(ce.Class) == 0 {
		ce.Class = http.StatusText(statusCode)
	}
	if statusCode == 404 {
		return errors.Wrap(cloudprovider.ErrNotFound, ce.Error())
	}
	return ce
}

type sApiVersion struct {
	MinVersion string
	Version    string
	Id         string
	Status     string
}

type sApiVersions struct {
	Versions []sApiVersion
	Version  sApiVersion
}

func (v *sApiVersions) GetMaxVersion() string {
	if v.Version.Status == "CURRENT" && len(v.Version.Id) > 0 {
		return strings.TrimPrefix(v.Version.Id, "v")
	}
	maxVersion := v.Version.Version
	for _, _version := range v.Versions {
		if version.GT(_version.Version, maxVersion) {
			maxVersion = _version.Version
		}
		if v.Version.Status == "CURRENT" && len(v.Version.Id) > 0 {
			return strings.TrimPrefix(v.Version.Id, "v")
		}
	}
	return maxVersion
}

func (cli *SOpenStackClient) getApiVerion(token oscli.TokenCredential, url string, debug bool) (string, error) {
	client := httputils.NewJsonClient(cli.getDefaultClient().HttpClient())
	req := httputils.NewJsonRequest(httputils.THttpMethod("GET"), strings.TrimSuffix(url, token.GetTenantId()), nil)
	header := http.Header{}
	header.Set("X-Auth-Token", token.GetTokenString())
	req.SetHeader(header)
	oe := &OpenstackError{}
	_, resp, err := client.Send(context.Background(), req, oe, debug)
	if err != nil {
		return "", errors.Wrap(err, "get api version")
	}
	versions := &sApiVersions{}
	resp.Unmarshal(&versions)
	return versions.GetMaxVersion(), nil
}

func (cli *SOpenStackClient) GetMaxVersion(region, service string) (string, error) {
	serviceUrl, err := cli.tokenCredential.GetServiceURL(service, region, "", cli.endpointType)
	if err != nil {
		return "", errors.Wrapf(err, "GetServiceURL(%s, %s, %s)", service, region, cli.endpointType)
	}
	header := http.Header{}
	header.Set("X-Auth-Token", cli.tokenCredential.GetTokenString())
	return cli.getApiVerion(cli.tokenCredential, serviceUrl, cli.debug)
}

func (cli *SOpenStackClient) jsonReuest(token oscli.TokenCredential, service, region, endpointType string, method httputils.THttpMethod, resource string, query url.Values, body interface{}, debug bool) (jsonutils.JSONObject, error) {
	serviceUrl, err := token.GetServiceURL(service, region, "", endpointType)
	if err != nil {
		return nil, errors.Wrapf(err, "GetServiceURL(%s, %s, %s)", service, region, endpointType)
	}
	header := http.Header{}
	header.Set("X-Auth-Token", token.GetTokenString())
	apiVersion := ""
	switch service {
	case OPENSTACK_SERVICE_IMAGE, OPENSTACK_SERVICE_IDENTITY:
	case OPENSTACK_SERVICE_COMPUTE:
		apiVersion = "2.1"
		// https://bugs.launchpad.net/horizon/+bug/1493205
		if strings.HasPrefix(resource, "/os-keypairs") {
			apiVersion = "2.2"
		}
	default:
		apiVersion, err = cli.getApiVerion(token, serviceUrl, debug)
		if err != nil {
			log.Errorf("get service %s api version error: %v", service, err)
		}
	}
	if len(apiVersion) > 0 {
		switch service {
		case OPENSTACK_SERVICE_COMPUTE:
			header.Set("X-Openstack-Nova-API-Version", apiVersion)
		case OPENSTACK_SERVICE_IMAGE:
			header.Set("X-Openstack-Glance-API-Version", apiVersion)
		case OPENSTACK_SERVICE_VOLUME, OPENSTACK_SERVICE_VOLUMEV2, OPENSTACK_SERVICE_VOLUMEV3:
			header.Set("Openstack-API-Version", fmt.Sprintf("volume %s", apiVersion))
		case OPENSTACK_SERVICE_NETWORK:
			header.Set("X-Openstack-Neutron-API-Version", apiVersion)
		case OPENSTACK_SERVICE_IDENTITY:
			header.Set("X-Openstack-Identity-API-Version", apiVersion)
		}
	}

	if service == OPENSTACK_SERVICE_IDENTITY {
		if strings.HasSuffix(serviceUrl, "/v3/") {
			serviceUrl = strings.TrimSuffix(serviceUrl, "/v3/")
		} else if strings.HasSuffix(serviceUrl, "/v3") {
			serviceUrl = strings.TrimSuffix(serviceUrl, "/v3")
		}
	}

	requestUrl := resource
	if !strings.HasPrefix(resource, serviceUrl) {
		requestUrl = fmt.Sprintf("%s/%s", strings.TrimSuffix(serviceUrl, "/"), strings.TrimPrefix(resource, "/"))
	}

	if query != nil && len(query) > 0 {
		requestUrl = fmt.Sprintf("%s?%s", requestUrl, query.Encode())
	}

	return cli._jsonRequest(method, requestUrl, header, body, debug)
}

func (cli *SOpenStackClient) _jsonRequest(method httputils.THttpMethod, url string, header http.Header, params interface{}, debug bool) (jsonutils.JSONObject, error) {
	client := httputils.NewJsonClient(cli.getDefaultClient().HttpClient())
	req := httputils.NewJsonRequest(method, url, params)
	req.SetHeader(header)
	oe := &OpenstackError{}
	_, resp, err := client.Send(context.Background(), req, oe, debug)
	return resp, err
}

func (cli *SOpenStackClient) ecsRequest(region string, method httputils.THttpMethod, resource string, query url.Values, body interface{}) (jsonutils.JSONObject, error) {
	token := cli.tokenCredential
	if method == httputils.POST && query != nil && len(query.Get("project_id")) > 0 {
		projectId := query.Get("project_id")
		var err error
		token, err = cli.getProjectTokenCredential(projectId)
		if err != nil {
			return nil, errors.Wrapf(err, "getProjectTokenCredential(%s)", projectId)
		}
	}
	return cli.jsonReuest(token, OPENSTACK_SERVICE_COMPUTE, region, cli.endpointType, method, resource, query, body, cli.debug)
}

func (cli *SOpenStackClient) ecsCreate(projectId, region, resource string, body interface{}) (jsonutils.JSONObject, error) {
	token := cli.tokenCredential
	if len(projectId) > 0 {
		var err error
		token, err = cli.getProjectTokenCredential(projectId)
		if err != nil {
			return nil, errors.Wrapf(err, "getProjectTokenCredential(%s)", projectId)
		}
	}
	return cli.jsonReuest(token, OPENSTACK_SERVICE_COMPUTE, region, cli.endpointType, httputils.POST, resource, nil, body, cli.debug)
}

func (cli *SOpenStackClient) ecsDo(projectId, region, resource string, body interface{}) (jsonutils.JSONObject, error) {
	token := cli.tokenCredential
	if len(projectId) > 0 {
		var err error
		token, err = cli.getProjectTokenCredential(projectId)
		if err != nil {
			return nil, errors.Wrapf(err, "getProjectTokenCredential(%s)", projectId)
		}
	}
	return cli.jsonReuest(token, OPENSTACK_SERVICE_COMPUTE, region, cli.endpointType, httputils.POST, resource, nil, body, cli.debug)
}

func (cli *SOpenStackClient) iamRequest(region string, method httputils.THttpMethod, resource string, query url.Values, body interface{}) (jsonutils.JSONObject, error) {
	return cli.jsonReuest(cli.tokenCredential, OPENSTACK_SERVICE_IDENTITY, region, cli.endpointType, method, resource, query, body, cli.debug)
}

func (cli *SOpenStackClient) vpcRequest(region string, method httputils.THttpMethod, resource string, query url.Values, body interface{}) (jsonutils.JSONObject, error) {
	return cli.jsonReuest(cli.tokenCredential, OPENSTACK_SERVICE_NETWORK, region, cli.endpointType, method, resource, query, body, cli.debug)
}

func (cli *SOpenStackClient) imageRequest(region string, method httputils.THttpMethod, resource string, query url.Values, body interface{}) (jsonutils.JSONObject, error) {
	return cli.jsonReuest(cli.tokenCredential, OPENSTACK_SERVICE_IMAGE, region, cli.endpointType, method, resource, query, body, cli.debug)
}

func (cli *SOpenStackClient) bsRequest(region string, method httputils.THttpMethod, resource string, query url.Values, body interface{}) (jsonutils.JSONObject, error) {
	for _, service := range []string{OPENSTACK_SERVICE_VOLUMEV3, OPENSTACK_SERVICE_VOLUMEV2, OPENSTACK_SERVICE_VOLUME} {
		_, err := cli.tokenCredential.GetServiceURL(service, region, "", cli.endpointType)
		if err == nil {
			return cli.jsonReuest(cli.tokenCredential, service, region, cli.endpointType, method, resource, query, body, cli.debug)
		}
	}
	return nil, errors.Wrap(ErrNoEndpoint, "cinder service")
}

func (cli *SOpenStackClient) bsCreate(projectId, region, resource string, body interface{}) (jsonutils.JSONObject, error) {
	token := cli.tokenCredential
	if len(projectId) > 0 {
		var err error
		token, err = cli.getProjectTokenCredential(projectId)
		if err != nil {
			return nil, errors.Wrapf(err, "getProjectTokenCredential(%s)", projectId)
		}
	}
	for _, service := range []string{OPENSTACK_SERVICE_VOLUMEV3, OPENSTACK_SERVICE_VOLUMEV2, OPENSTACK_SERVICE_VOLUME} {
		_, err := token.GetServiceURL(service, region, "", cli.endpointType)
		if err == nil {
			return cli.jsonReuest(token, service, region, cli.endpointType, httputils.POST, resource, nil, body, cli.debug)
		}
	}
	return nil, errors.Wrap(ErrNoEndpoint, "cinder service")
}

func (cli *SOpenStackClient) imageUpload(region, url string, size int64, body io.Reader, callback func(progress float32)) (*http.Response, error) {
	header := http.Header{}
	header.Set("Content-Type", "application/octet-stream")
	session := cli.getDefaultSession(region)
	reader := multicloud.NewProgress(size, 99, body, callback)
	return session.RawRequest(OPENSTACK_SERVICE_IMAGE, "", httputils.PUT, url, header, reader)
}

func (cli *SOpenStackClient) lbRequest(region string, method httputils.THttpMethod, resource string, query url.Values, body interface{}) (jsonutils.JSONObject, error) {
	return cli.jsonReuest(cli.tokenCredential, OPENSTACK_SERVICE_LOADBALANCER, region, cli.endpointType, method, resource, query, body, cli.debug)
}

func (cli *SOpenStackClient) fetchToken() error {
	if cli.tokenCredential != nil {
		return nil
	}
	var err error
	cli.tokenCredential, err = cli.getDefaultToken()
	if err != nil {
		return err
	}
	return cli.checkEndpointType()
}

func (cli *SOpenStackClient) checkEndpointType() error {
	for _, regionName := range cli.tokenCredential.GetRegions() {
		_, err := cli.tokenCredential.GetServiceURL(OPENSTACK_SERVICE_COMPUTE, regionName, "", cli.endpointType)
		if err == nil {
			return nil
		}
		for _, endpointType := range []string{"internal", "admin", "public"} {
			_, err = cli.tokenCredential.GetServiceURL(OPENSTACK_SERVICE_COMPUTE, regionName, "", endpointType)
			if err == nil {
				cli.endpointType = endpointType
				return nil
			}
		}
	}
	return errors.Errorf("failed to find right endpoint type for compute service")
}

func (cli *SOpenStackClient) getDefaultSession(regionName string) *oscli.ClientSession {
	if len(regionName) == 0 {
		regionName = cli.getDefaultRegionName()
	}
	client := cli.getDefaultClient()
	return client.NewSession(context.Background(), regionName, "", cli.endpointType, cli.tokenCredential)
}

func (cli *SOpenStackClient) getDefaultClient() *oscli.Client {
	client := oscli.NewClient(cli.authURL, 5, cli.debug, true)
	client.SetHttpTransportProxyFunc(cli.cpcfg.ProxyFunc)
	_client := client.GetClient()
	ts, _ := _client.Transport.(*http.Transport)
	_client.Transport = cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response) error, error) {
		if cli.cpcfg.ReadOnly {
			if req.Method == "GET" || req.Method == "HEAD" {
				return nil, nil
			}
			// 认证
			if req.Method == "POST" && strings.HasSuffix(req.URL.Path, "auth/tokens") {
				return nil, nil
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return nil, nil
	})

	return client
}

func (cli *SOpenStackClient) getDefaultToken() (oscli.TokenCredential, error) {
	client := cli.getDefaultClient()
	token, err := client.Authenticate(cli.username, cli.password, cli.domainName, cli.project, cli.projectDomain)
	if err != nil {
		if e, ok := err.(*httputils.JSONClientError); ok {
			if e.Class == "Unauthorized" {
				return nil, errors.Wrapf(cloudprovider.ErrInvalidAccessKey, err.Error())
			}
		}
		return nil, errors.Wrap(err, "Authenticate")
	}
	return token, nil
}

func (cli *SOpenStackClient) getProjectTokenCredential(projectId string) (oscli.TokenCredential, error) {
	project, err := cli.GetProject(projectId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetProject(%s)", projectId)
	}
	return cli.getProjectToken(project.Id, project.Name)
}

func (cli *SOpenStackClient) GetRegion(regionId string) *SRegion {
	for i := 0; i < len(cli.iregions); i++ {
		if cli.iregions[i].GetId() == regionId {
			return cli.iregions[i].(*SRegion)
		}
	}
	return nil
}

func (cli *SOpenStackClient) GetIRegions() []cloudprovider.ICloudRegion {
	return cli.iregions
}

func (cli *SOpenStackClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(cli.iregions); i++ {
		if cli.iregions[i].GetGlobalId() == id {
			return cli.iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (cli *SOpenStackClient) GetRegions() []SRegion {
	regions := make([]SRegion, len(cli.iregions))
	for i := 0; i < len(regions); i++ {
		region := cli.iregions[i].(*SRegion)
		regions[i] = *region
	}
	return regions
}

func (cli *SOpenStackClient) fetchProjects() error {
	var err error
	cli.projects, err = cli.GetProjects()
	if err != nil {
		return errors.Wrap(err, "GetProjects")
	}
	return nil
}

func (cli *SOpenStackClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	err := cli.fetchProjects()
	if err != nil {
		return nil, errors.Wrap(err, "fetchProjects")
	}
	iprojects := []cloudprovider.ICloudProject{}
	for i := 0; i < len(cli.projects); i++ {
		cli.projects[i].client = cli
		iprojects = append(iprojects, &cli.projects[i])
	}
	return iprojects, nil
}

func (cli *SOpenStackClient) GetProject(id string) (*SProject, error) {
	err := cli.fetchProjects()
	if err != nil {
		return nil, errors.Wrap(err, "fetchProjects")
	}
	for i := 0; i < len(cli.projects); i++ {
		if cli.projects[i].Id == id {
			return &cli.projects[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (cli *SOpenStackClient) CreateIProject(name string) (cloudprovider.ICloudProject, error) {
	return cli.CreateProject(name, "")
}

func (cli *SOpenStackClient) CreateProject(name, desc string) (*SProject, error) {
	params := map[string]interface{}{
		"project": map[string]interface{}{
			"name":        name,
			"domain_id":   cli.tokenCredential.GetProjectDomainId(),
			"enabled":     true,
			"description": desc,
		},
	}
	resp, err := cli.iamRequest(cli.getDefaultRegionName(), httputils.POST, "/v3/projects", nil, params)
	if err != nil {
		return nil, errors.Wrap(err, "iamRequest")
	}
	project := SProject{client: cli}
	err = resp.Unmarshal(&project, "project")
	if err != nil {
		return nil, errors.Wrap(err, "result.Unmarshal")
	}
	err = cli.AssignRoleToUserOnProject(cli.tokenCredential.GetUserId(), project.Id, "admin")
	if err != nil {
		return nil, errors.Wrap(err, "AssignRoleToUserOnProject")
	}
	return &project, nil
}

func (self *SOpenStackClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_PROJECT,
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_NETWORK,
		cloudprovider.CLOUD_CAPABILITY_SECURITY_GROUP,
		cloudprovider.CLOUD_CAPABILITY_EIP,
		cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		cloudprovider.CLOUD_CAPABILITY_QUOTA + cloudprovider.READ_ONLY_SUFFIX,
		// cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		// cloudprovider.CLOUD_CAPABILITY_RDS,
		// cloudprovider.CLOUD_CAPABILITY_CACHE,
		// cloudprovider.CLOUD_CAPABILITY_EVENT,
	}
	return caps
}
