package openstack

import (
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
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
	client          *mcclient.Client
	tokenCredential mcclient.TokenCredential
	iregions        []cloudprovider.ICloudRegion
}

func NewOpenStackClient(providerID string, providerName string, authURL string, username string, password string, project string) (*SOpenStackClient, error) {
	cli := &SOpenStackClient{providerID: providerID, providerName: providerName,
		authURL: authURL, username: username, password: password, project: project}

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

func (cli *SOpenStackClient) Get(region string, url string, microversion string, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	if body == nil {
		body = jsonutils.NewDict()
	}
	header := http.Header{}
	if len(microversion) > 0 {
		header.Set("X-Openstack-Nova-API-Version", microversion)
	}
	session := cli.client.NewSession(region, "", "internal", cli.tokenCredential, "")
	return session.JSONRequest("compute", "", "GET", url, header, body)
}

func (cli *SOpenStackClient) getComputeVersion(region string, service string) (string, string, error) {
	session := cli.client.NewSession(region, "", "internal", cli.tokenCredential, "")
	_, resp, err := session.JSONRequest(service, "", "GET", "/", nil, nil)
	if err != nil {
		return "", "", err
	}
	minVersion, _ := resp.GetString("version", "min_version")
	maxVersion, _ := resp.GetString("version", "version")
	return minVersion, maxVersion, nil
}

func (cli *SOpenStackClient) connect() error {
	cli.client = mcclient.NewClient(cli.authURL, 5, false, false, "", "")
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

func (cli *SOpenStackClient) GetRegions() []SRegion {
	regions := make([]SRegion, len(cli.iregions))
	for i := 0; i < len(regions); i++ {
		region := cli.iregions[i].(*SRegion)
		regions[i] = *region
	}
	return regions
}
