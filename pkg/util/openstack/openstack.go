package openstack

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/version"
)

const (
	CLOUD_PROVIDER_OPENSTACK = models.CLOUD_PROVIDER_OPENSTACK
	OPENSTACK_DEFAULT_REGION = "RegionOne"
)

type SOpenStackClient struct {
	providerID      string
	providerName    string
	authURL         string
	username        string
	password        string
	project         string
	endpointType    string
	client          *mcclient.Client
	tokenCredential mcclient.TokenCredential
	iregions        []cloudprovider.ICloudRegion

	Debug bool
}

func NewOpenStackClient(providerID string, providerName string, authURL string, username string, password string, project string, endpointType string, isDebug bool) (*SOpenStackClient, error) {
	cli := &SOpenStackClient{
		providerID:   providerID,
		providerName: providerName,
		authURL:      authURL,
		username:     username,
		password:     password,
		project:      project,
		endpointType: endpointType,
		Debug:        isDebug,
	}
	return cli, cli.fetchRegions()
}

func (cli *SOpenStackClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccount := cloudprovider.SSubAccount{
		Account: fmt.Sprintf("%s/%s", cli.project, cli.username),
		Name:    cli.providerName,
	}
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (cli *SOpenStackClient) fetchRegions() error {
	if err := cli.connect(); err != nil {
		return err
	}
	regions := cli.tokenCredential.GetRegions()
	cli.iregions = make([]cloudprovider.ICloudRegion, len(regions))
	for i := 0; i < len(regions); i++ {
		region := SRegion{client: cli, Name: regions[i]}
		cli.iregions[i] = &region
	}
	return nil
}

func (cli *SOpenStackClient) Request(region, service, method string, url string, microversion string, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	header := http.Header{}
	if len(microversion) > 0 {
		header.Set("X-Openstack-Nova-API-Version", microversion)
	}
	ctx := context.Background()
	session := cli.client.NewSession(ctx, region, "", cli.endpointType, cli.tokenCredential, "")
	header, resp, err := session.JSONRequest(service, "", httputils.THttpMethod(method), url, header, body)
	if err != nil && body != nil {
		uri, _ := session.GetServiceURL(service, "")
		log.Errorf("microversion %s url: %s, params: %s", microversion, uri+url, body.PrettyString())
	}
	return header, resp, err
}

func (cli *SOpenStackClient) RawRequest(region, service, method string, url string, microversion string, body jsonutils.JSONObject) (*http.Response, error) {
	header := http.Header{}
	if len(microversion) > 0 {
		header.Set("X-Openstack-Nova-API-Version", microversion)
	}
	ctx := context.Background()
	session := cli.client.NewSession(ctx, region, "", cli.endpointType, cli.tokenCredential, "")
	data := strings.NewReader("")
	if body != nil {
		data = strings.NewReader(body.String())
	}
	return session.RawRequest(service, "", httputils.THttpMethod(method), url, header, data)
}

func (cli *SOpenStackClient) StreamRequest(region, service, method string, url string, microversion string, body io.Reader) (*http.Response, error) {
	header := http.Header{}
	if len(microversion) > 0 {
		header.Set("X-Openstack-Nova-API-Version", microversion)
	}
	header.Set("Content-Type", "application/octet-stream")
	ctx := context.Background()
	session := cli.client.NewSession(ctx, region, "", cli.endpointType, cli.tokenCredential, "")
	return session.RawRequest(service, "", httputils.THttpMethod(method), url, header, body)
}

func (cli *SOpenStackClient) getVersion(region string, service string) (string, string, error) {
	ctx := context.Background()
	session := cli.client.NewSession(ctx, region, "", cli.endpointType, cli.tokenCredential, "")
	uri, err := session.GetServiceURL(service, cli.endpointType)
	if err != nil {
		return "", "", err
	}
	url := uri
	telnetID := cli.tokenCredential.GetTenantId()
	if strings.Index(uri, telnetID) > 0 {
		url = uri[0:strings.Index(uri, telnetID)]
	}
	_, resp, err := session.JSONRequest(url, "", "GET", "/", nil, nil)
	if err != nil {
		return "", "", err
	}
	minVersion, _ := resp.GetString("version", "min_version")
	maxVersion, _ := resp.GetString("version", "version")
	if resp.Contains("versions") {
		minVersion, maxVersion = "1000.0", ""
		versions, _ := resp.GetArray("versions")
		for _, _version := range versions {
			if _minVersion, _ := _version.GetString("min_version"); len(_minVersion) > 0 {
				if version.LT(_minVersion, minVersion) {
					minVersion = _minVersion
				}
			}
			if _maxVersion, _ := _version.GetString("version"); len(_maxVersion) > 0 {
				if version.GT(_maxVersion, maxVersion) {
					maxVersion = _maxVersion
				}
			}
		}
		if minVersion == "1000.0" {
			minVersion, maxVersion = "", ""
		}
	}
	return minVersion, maxVersion, nil
}

func (cli *SOpenStackClient) connect() error {
	cli.client = mcclient.NewClient(cli.authURL, 5, cli.Debug, false, "", "")
	tokenCredential, err := cli.client.Authenticate(cli.username, cli.password, "", cli.project)
	if err != nil {
		return err
	}
	cli.tokenCredential = tokenCredential
	return nil
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
