package azure

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
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
	SERVICE_STORAGE    = "storage"
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
	SERVICE_STORAGE: {
		ENV_NAME_GLOBAL: "https://%s.blob.core.windows.net",
		ENV_NAME_CHINA:  "https://%s.blob.core.chinacloudapi.cn",
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

func (region *SRegion) list_storage_v2(accessKey, bucket string, container string, params url.Values, retVal interface{}) error {
	return region.client.list_storage_v2(accessKey, bucket, container, params, retVal)
}

func (region *SRegion) put_storage_v2(accessKey, bucket string, container string, header http.Header, params url.Values, body io.Reader, retVal interface{}) error {
	return region.client.put_storage_v2(accessKey, bucket, container, header, params, body, retVal)
}

func (self *SAzureClient) list_storage_v2(accessKey, bucket string, container string, params url.Values, retVal interface{}) error {
	_, _, err := self.__storage_request(accessKey, bucket, container, httputils.GET, nil, params, nil, retVal)
	if err != nil {
		return errors.Wrapf(err, "list_storage_v2")
	}
	return nil
}

func (cli *SAzureClient) delete_storage_v2(accessKey, bucket string, container string, header http.Header, params url.Values) error {
	_, _, err := cli.__storage_request(accessKey, bucket, container, httputils.DELETE, header, params, nil, nil)
	if err != nil {
		return errors.Wrapf(err, "delete_storage_v2")
	}
	return nil
}

func (cli *SAzureClient) put_storage_v2(accessKey, bucket string, container string, header http.Header, params url.Values, body io.Reader, retVal interface{}) error {
	_, _, err := cli.__storage_request(accessKey, bucket, container, httputils.PUT, header, params, body, retVal)
	if err != nil {
		return errors.Wrapf(err, "put_storage_v2")
	}
	return nil
}

func (cli *SAzureClient) put_header_storage_v2(accessKey, bucket string, container string, header http.Header, params url.Values) (http.Header, error) {
	header, _, err := cli.__storage_request(accessKey, bucket, container, httputils.PUT, header, params, nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "put_header_storage_v2")
	}
	return header, nil
}

func (region *SRegion) get_header_storage_v2(accessKey, bucket string, container string, params url.Values) (http.Header, error) {
	header, _, err := region.client.__storage_request(accessKey, bucket, container, httputils.GET, nil, params, nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "get_header_storage_v2")
	}
	return header, nil
}

func (region *SRegion) get_body_storage_v2(accessKey, bucket string, container string, header http.Header, params url.Values) (io.Reader, error) {
	_, body, err := region.client.__storage_request(accessKey, bucket, container, httputils.GET, header, params, nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "get_body_storage_v2")
	}
	return body, nil
}

func (region *SRegion) header_storage_v2(accessKey, bucket string, container string, params url.Values) (http.Header, error) {
	header, _, err := region.client.__storage_request(accessKey, bucket, container, httputils.HEAD, nil, params, nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "header_storage_v2")
	}
	return header, nil
}

func computeHmac256(message string, accountKey string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(accountKey)
	if err != nil {
		return "", errors.Wrapf(err, "base64.StdEncoding.DecodeString(%s)", accountKey)
	}
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

func buildCanonicalizedHeader(headers http.Header) string {
	cm := make(map[string]string)

	for k := range headers {
		headerName := strings.TrimSpace(strings.ToLower(k))
		if strings.HasPrefix(headerName, "x-ms-") {
			cm[headerName] = headers.Get(k)
		}
	}

	if len(cm) == 0 {
		return ""
	}

	keys := []string{}
	for key := range cm {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	ch := bytes.NewBufferString("")

	for _, key := range keys {
		ch.WriteString(key)
		ch.WriteRune(':')
		ch.WriteString(cm[key])
		ch.WriteRune('\n')
	}

	return strings.TrimSuffix(ch.String(), "\n")
}

func buildCanonicalizedString(verb httputils.THttpMethod, headers http.Header, canonicalizedResource string) string {
	contentLength := headers.Get("Content-Length")
	if contentLength == "0" {
		contentLength = ""
	}
	date := ""

	return strings.Join([]string{
		strings.ToUpper(string(verb)),
		headers.Get("Content-Encoding"),
		headers.Get("Content-Language"),
		contentLength,
		headers.Get("Content-MD5"),
		headers.Get("Content-Type"),
		date,
		headers.Get("If-Modified-Since"),
		headers.Get("If-Match"),
		headers.Get("If-None-Match"),
		headers.Get("If-Unmodified-Since"),
		headers.Get("Range"),
		buildCanonicalizedHeader(headers),
		canonicalizedResource,
	}, "\n")
}

func buildCanonicalizedResource(bucket, uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("url.Parse: %v", err)
	}

	cr := bytes.NewBufferString("")
	cr.WriteString("/")
	cr.WriteString(bucket)

	if len(u.Path) > 0 {
		// Any portion of the CanonicalizedResource string that is derived from
		// the resource's URI should be encoded exactly as it is in the URI.
		// -- https://msdn.microsoft.com/en-gb/library/azure/dd179428.aspx
		cr.WriteString(u.EscapedPath())
	}

	params, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return "", fmt.Errorf("url.ParseQuery: %v", err)
	}

	// See https://github.com/Azure/azure-storage-net/blob/master/Lib/Common/Core/Util/AuthenticationUtility.cs#L277

	if len(params) > 0 {
		cr.WriteString("\n")

		keys := []string{}
		for key := range params {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		completeParams := []string{}
		for _, key := range keys {
			if len(params[key]) > 1 {
				sort.Strings(params[key])
			}

			completeParams = append(completeParams, fmt.Sprintf("%s:%s", key, strings.Join(params[key], ",")))
		}
		cr.WriteString(strings.Join(completeParams, "\n"))
	}

	return cr.String(), nil
}

func (self *SAzureClient) __storage_request(accessKey, bucket string, container string, method httputils.THttpMethod, header http.Header, params url.Values, body io.Reader, retVal interface{}) (http.Header, io.Reader, error) {
	if params == nil {
		params = url.Values{}
	}

	domain := fmt.Sprintf(azServices[SERVICE_STORAGE][self.envName], bucket)
	url := fmt.Sprintf("%s/%s", strings.TrimSuffix(domain, "/"), container)
	if len(params) > 0 {
		url += fmt.Sprintf("?%s", params.Encode())
	}

	utcTime := time.Now().UTC().Format(http.TimeFormat)

	if gotypes.IsNil(header) {
		header = http.Header{}
	}

	header.Set("x-ms-date", utcTime)
	header.Set("x-ms-version", "2018-03-28")

	canRes, err := buildCanonicalizedResource(bucket, url)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "buildCanonicalizedResource")
	}

	canString := buildCanonicalizedString(method, header, canRes)

	// 4. 计算 Shared Key 签名
	signature, err := computeHmac256(canString, accessKey)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "computeHmac256")
	}

	header.Set("Authorization", fmt.Sprintf("SharedKey %s:%s", bucket, signature))

	resp, err := httputils.Request(self.client(), self.ctx, method, url, header, body, self.debug)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Request")
	}

	if resp.StatusCode >= 400 {
		defer httputils.CloseResponse(resp)
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "ReadAll")
		}
		return nil, nil, errors.Errorf("resp: %d url: %s header: %v, data: %s", resp.StatusCode, url, header, string(data))
	}
	if gotypes.IsNil(retVal) {
		return resp.Header, resp.Body, nil
	}
	defer httputils.CloseResponse(resp)

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "ReadAll")
	}
	err = xml.Unmarshal(data, retVal)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "xml.Unmarshal")
	}
	return resp.Header, resp.Body, nil
}

func (self *SAzureClient) _request_v2(service string, method httputils.THttpMethod, resource, apiVersion string, params url.Values, body map[string]interface{}) (jsonutils.JSONObject, error) {
	value := []jsonutils.JSONObject{}
	if gotypes.IsNil(params) {
		params = url.Values{}
	}
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
