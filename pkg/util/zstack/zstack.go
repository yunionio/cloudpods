package zstack

import (
	"context"
	"crypto/sha512"
	"fmt"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

const (
	CLOUD_PROVIDER_ZSTACK = api.CLOUD_PROVIDER_ZSTACK
	ZSTACK_DEFAULT_REGION = "ZStack"
	ZSTACK_API_VERSION    = "v1"
)

type SZStackClient struct {
	providerID   string
	providerName string
	username     string
	password     string
	authURL      string

	sessionID string

	iregions []cloudprovider.ICloudRegion

	Debug bool
}

func NewZStackClient(providerID string, providerName string, authURL string, username string, password string, isDebug bool) (*SZStackClient, error) {
	cli := &SZStackClient{
		providerID:   providerID,
		providerName: providerName,
		authURL:      authURL,
		username:     username,
		password:     password,
		Debug:        isDebug,
	}
	if err := cli.connect(); err != nil {
		return nil, err
	}
	cli.iregions = []cloudprovider.ICloudRegion{&SRegion{client: cli, Name: ZSTACK_DEFAULT_REGION}}
	return cli, nil
}

// func (cli *SZStackClient) fetchRegions() error {
// 	if err := cli.connect(); err != nil {
// 		return err
// 	}

// 	regions := []SRegion{}
// 	if err := cli.list("/zstack/v1/zones", &regions); err != nil {
// 		return err
// 	}
// 	for i := 0; i < len(regions); i++ {
// 		regions[i].client = cli
// 		cli.iregions = append(cli.iregions, &regions[i])
// 	}
// 	return nil
// }

func (cli *SZStackClient) getRequestURL(resource string, params []string) string {
	return cli.authURL + fmt.Sprintf("/zstack/%s/%s", ZSTACK_API_VERSION, resource) + "?" + strings.Join(params, "&")
}

func (cli *SZStackClient) connect() error {
	client := httputils.GetDefaultClient()
	header := http.Header{}
	header.Add("Content-Type", "application/json")
	authURL := cli.authURL + "/zstack/v1/accounts/login"
	body := jsonutils.Marshal(map[string]interface{}{
		"logInByAccount": map[string]string{
			"accountName": cli.username,
			"password":    fmt.Sprintf("%x", sha512.Sum512([]byte(cli.password))),
		},
	})
	_, resp, err := httputils.JSONRequest(client, context.Background(), "PUT", authURL, header, body, cli.Debug)
	if err != nil {
		return err
	}
	cli.sessionID, err = resp.GetString("inventory", "uuid")
	return err
}

func (cli *SZStackClient) listAll(resource string, params []string, retVal interface{}) error {
	result := []jsonutils.JSONObject{}
	start, limit := 0, 50
	for {
		resp, err := cli._list(resource, start, limit, params)
		if err != nil {
			return err
		}
		objs, err := resp.GetArray("inventories")
		if err != nil {
			return err
		}
		result = append(result, objs...)
		if start+limit > len(result) {
			inventories := jsonutils.Marshal(map[string][]jsonutils.JSONObject{"inventories": result})
			return inventories.Unmarshal(retVal, "inventories")
		}
		start += limit
	}
}

func (cli *SZStackClient) _list(resource string, start int, limit int, params []string) (jsonutils.JSONObject, error) {
	client := httputils.GetDefaultClient()
	header := http.Header{}
	header.Add("Content-Type", "application/json")
	header.Add("Authorization", "OAuth "+cli.sessionID)
	if params == nil {
		params = []string{}
	}
	params = append(params, "replyWithCount=true")
	params = append(params, fmt.Sprintf("start=%d", start))
	if limit == 0 {
		limit = 50
	}
	params = append(params, fmt.Sprintf("limit=%d", limit))
	requestURL := cli.getRequestURL(resource, params)
	_, resp, err := httputils.JSONRequest(client, context.Background(), "GET", requestURL, header, nil, cli.Debug)
	return resp, err
}

func (cli *SZStackClient) list(baseURL string, start int, limit int, params []string, retVal interface{}) error {
	resp, err := cli._list(baseURL, start, limit, params)
	if err != nil {
		return err
	}
	return resp.Unmarshal(retVal, "inventories")
}

func (cli *SZStackClient) GetRegion(regionId string) *SRegion {
	for i := 0; i < len(cli.iregions); i++ {
		if cli.iregions[i].GetId() == regionId {
			return cli.iregions[i].(*SRegion)
		}
	}
	return nil
}

func (cli *SZStackClient) GetRegions() []SRegion {
	regions := make([]SRegion, len(cli.iregions))
	for i := 0; i < len(regions); i++ {
		region := cli.iregions[i].(*SRegion)
		regions[i] = *region
	}
	return regions
}
