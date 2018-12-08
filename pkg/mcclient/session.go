package mcclient

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/pkg/utils"
)

const (
	TASK_ID         = "X-Task-Id"
	TASK_NOTIFY_URL = "X-Task-Notify-Url"
	AUTH_TOKEN      = "X-Auth-Token"
	REGION_VERSION  = "X-Region-Version"

	DEFAULT_API_VERSION = "v1"
	V2_API_VERSION      = "v2"
)

var (
	MutilVersionService = []string{"compute"}
	ApiVersionByModule  = true
)

func DisableApiVersionByModule() {
	ApiVersionByModule = false
}

func EnableApiVersionByModule() {
	ApiVersionByModule = true
}

type ClientSession struct {
	client        *Client
	region        string
	zone          string
	endpointType  string
	token         TokenCredential
	Header        http.Header /// headers for this session
	notifyChannel chan string

	defaultApiVersion string
}

func populateHeader(self *http.Header, update http.Header) {
	for k, v := range update {
		for _, vv := range v {
			// self.Add(k, vv)
			self.Set(k, vv)
		}
	}
}

func GetTokenHeaders(userCred TokenCredential) http.Header {
	headers := http.Header{}
	headers.Set(AUTH_TOKEN, userCred.GetTokenString())
	headers.Set(REGION_VERSION, V2_API_VERSION)
	return headers
}

func SplitVersionedURL(url string) (string, string) {
	endidx := len(url) - 1
	for ; endidx >= 0 && url[endidx] == '/'; endidx-- {
	}
	lastslash := strings.LastIndexByte(url[0:endidx+1], '/')
	if lastslash >= 0 {
		if strings.EqualFold(url[lastslash+1:endidx+1], "latest") {
			return url[0:lastslash], ""
		}
		match, err := regexp.MatchString(`^v\d+\.?\d*`, url[lastslash+1:endidx+1])
		if err == nil && match {
			return url[0:lastslash], url[lastslash+1 : endidx+1]
		}
	}
	return url[0 : endidx+1], ""
}

/* func stripURLVersion(url string) string {
	base, _ := SplitVersionedURL(url)
	log.Debugf("stripURLVersion %s => %s", url, base)
	return base
}*/

func (this *ClientSession) GetClient() *Client {
	return this.client
}

func (this *ClientSession) getServiceName(service, apiVersion string) string {
	if utils.IsInStringArray(service, MutilVersionService) && len(apiVersion) > 0 && apiVersion != DEFAULT_API_VERSION {
		service = fmt.Sprintf("%s_%s", service, apiVersion)
	}
	return service
}

func (this *ClientSession) getApiVersion(moduleApiVersion string) string {
	if moduleApiVersion != "" && ApiVersionByModule {
		return moduleApiVersion
	}
	return this.defaultApiVersion
}

func (this *ClientSession) GetServiceURL(service, endpointType string) (string, error) {
	return this.GetServiceVersionURL(service, endpointType, this.getApiVersion(""))
}

func (this *ClientSession) GetServiceVersionURL(service, endpointType, apiVersion string) (string, error) {
	if len(this.endpointType) > 0 {
		// session specific endpoint type should override the input endpointType, which is supplied by manager
		endpointType = this.endpointType
	}
	service = this.getServiceName(service, apiVersion)
	url, err := this.token.GetServiceURL(service, this.region, this.zone, endpointType)
	if err != nil {
		url, err = this.client.serviceCatalog.GetServiceURL(service, this.region, this.zone, endpointType)
	}
	return url, err
}

func (this *ClientSession) GetServiceURLs(service, endpointType string) ([]string, error) {
	return this.GetServiceVersionURLs(service, endpointType, this.getApiVersion(""))
}

func (this *ClientSession) GetServiceVersionURLs(service, endpointType, apiVersion string) ([]string, error) {
	if len(this.endpointType) > 0 {
		// session specific endpoint type should override the input endpointType, which is supplied by manager
		endpointType = this.endpointType
	}
	service = this.getServiceName(service, apiVersion)
	urls, err := this.token.GetServiceURLs(service, this.region, this.zone, endpointType)
	if err != nil {
		urls, err = this.client.serviceCatalog.GetServiceURLs(service, this.region, this.zone, endpointType)
	}
	return urls, err
}

func (this *ClientSession) getBaseUrl(service, endpointType, apiVersion string) (string, error) {
	if len(service) > 0 {
		if strings.HasPrefix(service, "http://") || strings.HasPrefix(service, "https://") {
			return service, nil
		} else {
			return this.GetServiceVersionURL(service, endpointType, this.getApiVersion(apiVersion))
		}
	} else {
		return "", fmt.Errorf("Empty service type or baseURL")
	}
}

func (this *ClientSession) RawVersionRequest(
	service, endpointType, method, url string,
	headers http.Header, body io.Reader,
	apiVersion string,
) (*http.Response, error) {
	baseurl, err := this.getBaseUrl(service, endpointType, apiVersion)
	if err != nil {
		return nil, err
	}
	tmpHeader := http.Header{}
	if headers != nil {
		populateHeader(&tmpHeader, headers)
	}
	populateHeader(&tmpHeader, this.Header)
	return this.client.rawRequest(baseurl,
		this.token.GetTokenString(),
		method, url, tmpHeader, body)
}

func (this *ClientSession) RawRequest(service, endpointType, method, url string, headers http.Header, body io.Reader) (*http.Response, error) {
	return this.RawVersionRequest(service, endpointType, method, url, headers, body, "")
}

func (this *ClientSession) JSONVersionRequest(
	service, endpointType, method, url string,
	headers http.Header, body jsonutils.JSONObject,
	apiVersion string,
) (http.Header, jsonutils.JSONObject, error) {
	baseUrl, err := this.getBaseUrl(service, endpointType, apiVersion)
	if err != nil {
		return headers, nil, err
	}
	tmpHeader := http.Header{}
	if headers != nil {
		populateHeader(&tmpHeader, headers)
	}
	populateHeader(&tmpHeader, this.Header)
	return this.client.jsonRequest(baseUrl,
		this.token.GetTokenString(),
		method, url, tmpHeader, body)
}

func (this *ClientSession) JSONRequest(service, endpointType, method, url string, headers http.Header, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	return this.JSONVersionRequest(service, endpointType, method, url, headers, body, "")
}

func (this *ClientSession) ParseJSONResponse(resp *http.Response, err error) (http.Header, jsonutils.JSONObject, error) {
	return httputils.ParseJSONResponse(resp, err, this.client.debug)
}

func (this *ClientSession) HasSystemAdminPrivelege() bool {
	return this.token.HasSystemAdminPrivelege()
}

func (this *ClientSession) GetRegion() string {
	return this.region
}

func (this *ClientSession) GetUserId() string {
	return this.token.GetUserId()
}

func (this *ClientSession) GetTenantId() string {
	return this.token.GetTenantId()
}

func (this *ClientSession) GetTenantName() string {
	return this.token.GetTenantName()
}

func (this *ClientSession) SetTaskNotifyUrl(url string) {
	this.Header.Add(TASK_NOTIFY_URL, url)
}

func (this *ClientSession) RemoveTaskNotifyUrl() {
	this.Header.Del(TASK_NOTIFY_URL)
}

func (this *ClientSession) PrepareTask() {
	// start a random htttp server
	this.notifyChannel = make(chan string)

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	port := 55000 + r1.Intn(1000)
	ip := utils.GetOutboundIP()
	addr := fmt.Sprintf("%s:%d", ip.String(), port)
	url := fmt.Sprintf("http://%s", addr)
	this.SetTaskNotifyUrl(url)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "ok")
		body, err := ioutil.ReadAll(r.Body)
		var msg string
		if err != nil {
			msg = fmt.Sprintf("Read request data error: %s", err)
		} else {
			msg = string(body)
		}
		this.notifyChannel <- msg
	})

	go func() {
		fmt.Println("List on address: ", url)
		if err := http.ListenAndServe(addr, nil); err != nil {
			fmt.Printf("Task notify server error: %s\n", err)
		}
	}()
}

func (this *ClientSession) WaitTaskNotify() {
	if this.notifyChannel != nil {
		msg := <-this.notifyChannel
		fmt.Println("---------------Task complete -------------")
		fmt.Println(msg)
	}
}

func (this *ClientSession) GetApiVersion() string {
	apiVersion := this.getApiVersion("")
	if len(apiVersion) == 0 {
		return DEFAULT_API_VERSION
	}
	return apiVersion
}

func (this *ClientSession) ToJson() jsonutils.JSONObject {
	params := jsonutils.NewDict()
	simpleToken := SimplifyToken(this.token)
	tokenJson := jsonutils.Marshal(simpleToken)
	params.Update(tokenJson)
	params.Add(jsonutils.NewString(this.GetApiVersion()), "api_version")
	if len(this.endpointType) > 0 {
		params.Add(jsonutils.NewString(this.endpointType), "endpoint_type")
	}
	if len(this.region) > 0 {
		params.Add(jsonutils.NewString(this.region), "region")
	}
	if len(this.zone) > 0 {
		params.Add(jsonutils.NewString(this.zone), "zone")
	}
	return params
}

func (cs *ClientSession) GetToken() TokenCredential {
	return cs.token
}
