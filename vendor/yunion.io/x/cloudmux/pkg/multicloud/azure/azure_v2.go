package azure

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"
)

const (
	ENV_NAME_CHINA  = "AzureChinaCloud"
	ENV_NAME_GLOBAL = "AzurePublicCloud"

	SERVICE_MANAGEMENT = "management"
	SERVICE_GRAPH      = "graph"
	SERVICE_AAD        = "aad"
)

var azServices = map[string]map[string]string{
	SERVICE_GRAPH: {
		ENV_NAME_GLOBAL: "https://graph.microsoft.com/v1.0",
		ENV_NAME_CHINA:  "https://microsoftgraph.chinacloudapi.cn/v1.0",
	},
	SERVICE_MANAGEMENT: {
		ENV_NAME_GLOBAL: "https://management.azure.com",
		ENV_NAME_CHINA:  "https://management.chinacloudapi.cn",
	},
	SERVICE_AAD: {
		ENV_NAME_GLOBAL: "https://login.microsoftonline.com",
		ENV_NAME_CHINA:  "https://login.chinacloudapi.cn",
	},
}

type Token struct {
	TokenType    string
	ExpiresIn    int64
	ExtExpiresIn int64
	ExpiresOn    int64
	NotBefore    int64
	Resource     string
	AccessToken  string
}

func (t Token) Token() string {
	return fmt.Sprintf("%s %s", t.TokenType, t.AccessToken)
}

func (t Token) isExpire() bool {
	expire := time.Unix(t.NotBefore, 0)
	return expire.Before(time.Now())
}

func (self *SAzureClient) client() *http.Client {
	if self.httpClient != nil {
		return self.httpClient
	}
	httpClient := self.cpcfg.AdaptiveTimeoutHttpClient()
	transport, _ := httpClient.Transport.(*http.Transport)
	httpClient.Transport = cloudprovider.GetCheckTransport(transport, func(req *http.Request) (func(resp *http.Response) error, error) {
		if self.cpcfg.ReadOnly {
			if req.Method == "GET" || (req.Method == "POST" && strings.HasSuffix(req.URL.Path, "oauth2/token")) {
				return nil, nil
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return nil, nil
	})
	self.httpClient = httpClient
	return self.httpClient
}

func (self *SAzureClient) auth(resource string) (string, error) {
	self.tokenLock.Lock()
	defer self.tokenLock.Unlock()

	if token, ok := self.tokenMap[resource]; ok && !token.isExpire() {
		return token.Token(), nil
	}

	data := url.Values{}
	data.Set("client_id", self.clientId)
	data.Set("client_secret", self.clientSecret)
	data.Set("grant_type", "client_credentials")
	data.Set("resource", resource)

	domain := azServices[SERVICE_AAD][self.envName]
	url := fmt.Sprintf("%s/%s/oauth2/token?api-version=1.0", domain, self.tenantId)
	client := self.client()
	resp, err := client.PostForm(url, data)
	if err != nil {
		return "", errors.Wrapf(err, "auth")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrapf(err, "read body")
	}
	obj, err := jsonutils.Parse(body)
	if err != nil {
		return "", errors.Wrapf(err, "parse body %s", string(body))
	}
	if obj.Contains("error") {
		return "", errors.Errorf(string(body))
	}
	token := &Token{}
	err = obj.Unmarshal(token)
	if err != nil {
		return "", errors.Wrapf(err, "unmarshal token")
	}
	self.tokenMap[resource] = token
	return token.Token(), nil
}

func (self *SAzureClient) Do(req *http.Request) (*http.Response, error) {
	resource := fmt.Sprintf("https://%s", req.Host)
	token, err := self.auth(resource)
	if err != nil {
		return nil, errors.Wrapf(err, "auth")
	}
	req.Header.Set("Authorization", token)
	return self.client().Do(req)
}

func (self *SAzureClient) list_v2(resource, apiVersion string, params url.Values) (jsonutils.JSONObject, error) {
	return self._list_v2(SERVICE_MANAGEMENT, resource, apiVersion, params)
}

func (self *SAzureClient) post_v2(resource, apiVersion string, body map[string]interface{}) (jsonutils.JSONObject, error) {
	return self._post_v2(SERVICE_MANAGEMENT, resource, apiVersion, body)
}

func (self *SAzureClient) _list_v2(service string, resource, apiVersion string, params url.Values) (jsonutils.JSONObject, error) {
	return self._request_v2(service, httputils.GET, resource, apiVersion, params, nil)
}

func (self *SAzureClient) _post_v2(service string, resource, apiVersion string, body map[string]interface{}) (jsonutils.JSONObject, error) {
	return self._request_v2(service, httputils.POST, resource, apiVersion, nil, body)
}

func (self *SAzureClient) _request_v2(service string, method httputils.THttpMethod, resource, apiVersion string, params url.Values, body map[string]interface{}) (jsonutils.JSONObject, error) {
	value := []jsonutils.JSONObject{}
	for {
		resp, err := self.__request_v2(service, method, resource, apiVersion, params, body)
		if err != nil {
			return nil, err
		}
		if gotypes.IsNil(resp) {
			return jsonutils.NewDict(), nil
		}
		if !resp.Contains("value") {
			return resp, nil
		}
		part := struct {
			Value         []jsonutils.JSONObject
			NextLink      string
			OdataNextLink string `json:"@odata.nextLink"`
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrapf(err, "resp.Unmarshal")
		}
		value = append(value, part.Value...)
		if len(part.Value) == 0 || (len(part.NextLink) == 0 && len(part.OdataNextLink) == 0) {
			break
		}
		nextLink := part.NextLink
		if len(nextLink) == 0 {
			nextLink = part.OdataNextLink
		}
		link, err := url.Parse(nextLink)
		if err != nil {
			return nil, errors.Wrapf(err, "url.Parse(%s)", nextLink)
		}
		token := ""
		for _, key := range []string{"$skipToken", "$skiptoken"} {
			_token := link.Query().Get(key)
			if len(_token) > 0 {
				token = _token
			}
			params.Del(key)
		}
		params.Set("$skipToken", token)
	}
	return jsonutils.Marshal(map[string]interface{}{"value": value}), nil
}

func (self *SAzureClient) __request_v2(service string, method httputils.THttpMethod, resource, apiVersion string, params url.Values, body map[string]interface{}) (jsonutils.JSONObject, error) {
	if params == nil {
		params = url.Values{}
	}
	if len(apiVersion) > 0 {
		params.Set("api-version", apiVersion)
	}

	domain := azServices[service][self.envName]
	url := fmt.Sprintf("%s/%s", strings.TrimSuffix(domain, "/"), strings.TrimPrefix(resource, "/"))
	if service == SERVICE_MANAGEMENT {
		switch resource {
		case "subscriptions":
		case "locations", "resourcegroups", "resources":
			url = fmt.Sprintf("%s/subscriptions/%s/%s", strings.TrimSuffix(domain, "/"), self._subscriptionId(), resource)
		default:
			if !strings.HasPrefix(resource, "/") {
				url = fmt.Sprintf("%s/subscriptions/%s/providers/%s", strings.TrimSuffix(domain, "/"), self._subscriptionId(), resource)
			}
		}
	}
	if len(params) > 0 {
		filters := []string{}
		if params.Has("$filter") {
			filters = params["$filter"]
			params.Set("$filter", strings.Join(filters, " and "))
		}
		url += fmt.Sprintf("?%s", params.Encode())
	}
	var input jsonutils.JSONObject = nil
	if body != nil {
		input = jsonutils.Marshal(body)
	}
	_, resp, err := httputils.JSONRequest(self, self.ctx, method, url, nil, input, self.debug)
	return resp, err
}

func (self *SAzureClient) delete_v2(resource, apiVersion string, body map[string]interface{}) (jsonutils.JSONObject, error) {
	return self._delete_v2("", resource, apiVersion)
}

func (self *SAzureClient) _delete_v2(service string, resource, apiVersion string) (jsonutils.JSONObject, error) {
	domain := azServices[service][self.envName]
	url := fmt.Sprintf("%s/%s", domain, resource)
	if len(apiVersion) > 0 {
		url += fmt.Sprintf("?api-version=%s", apiVersion)
	}
	_, resp, err := httputils.JSONRequest(self, self.ctx, httputils.DELETE, url, nil, nil, self.debug)
	return resp, err
}

func (self *SAzureClient) patch_v2(resource, apiVersion string, body map[string]interface{}) (jsonutils.JSONObject, error) {
	return self._patch_v2("", resource, apiVersion, body)
}

func (self *SAzureClient) _patch_v2(service string, resource, apiVersion string, body map[string]interface{}) (jsonutils.JSONObject, error) {
	domain := azServices[service][self.envName]
	url := fmt.Sprintf("%s/%s", domain, resource)
	if len(apiVersion) > 0 {
		url += fmt.Sprintf("?api-version=%s", apiVersion)
	}
	_, resp, err := httputils.JSONRequest(self, self.ctx, httputils.PATCH, url, nil, jsonutils.Marshal(body), self.debug)
	return resp, err
}
