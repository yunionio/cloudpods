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

package ecloud

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/stringutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	CLOUD_PROVIDER_ECLOUD    = api.CLOUD_PROVIDER_ECLOUD
	CLOUD_PROVIDER_ECLOUD_CN = "移动云"
	CLOUD_PROVIDER_ECLOUD_EN = "Ecloud"
	CLOUD_API_VERSION        = "2016-12-05"

	ECLOUD_DEFAULT_REGION = "cn-beijing-1"
)

type SEcloudClientConfig struct {
	cpcfg     cloudprovider.ProviderConfig
	AccessKey string
	Secret    string
	debug     bool
}

func NewEcloudClientConfig(accessKey, secret string) *SEcloudClientConfig {
	cfg := &SEcloudClientConfig{
		AccessKey: accessKey,
		Secret:    secret,
	}
	return cfg
}

func (cfg *SEcloudClientConfig) SetCloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *SEcloudClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *SEcloudClientConfig) SetDebug(debug bool) *SEcloudClientConfig {
	cfg.debug = debug
	return cfg
}

type SEcloudClient struct {
	*SEcloudClientConfig

	httpClient *http.Client
}

func NewEcloudClient(cfg *SEcloudClientConfig) (*SEcloudClient, error) {
	httpClient := cfg.cpcfg.AdaptiveTimeoutHttpClient()
	cli := &SEcloudClient{
		SEcloudClientConfig: cfg,
		httpClient:          httpClient,
	}
	return cli, nil
}

func (self *SEcloudClient) GetAccessEnv() string {
	return api.CLOUD_ACCESS_ENV_ECLOUD_CHINA
}

func (ec *SEcloudClient) GetRegions() ([]SRegion, error) {
	ctx := context.Background()
	req := NewOpenApiRegionRequest(ECLOUD_DEFAULT_REGION, nil)
	ret := make([]SRegion, 0)
	if err := ec.doList(ctx, req.Base(), &ret); err != nil {
		return nil, err
	}
	for i := range ret {
		ret[i].client = ec
	}
	return ret, nil
}

func (ec *SEcloudClient) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	regions, err := ec.GetRegions()
	if err != nil {
		return nil, err
	}
	iregions := make([]cloudprovider.ICloudRegion, len(regions))
	for i := range iregions {
		iregions[i] = &regions[i]
	}
	return iregions, nil
}

func (ec *SEcloudClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	iregions, err := ec.GetIRegions()
	if err != nil {
		return nil, err
	}
	for i := range iregions {
		if iregions[i].GetGlobalId() == id {
			return iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (ec *SEcloudClient) GetRegionById(id string) (*SRegion, error) {
	iregions, err := ec.GetIRegions()
	if err != nil {
		return nil, err
	}
	for i := range iregions {
		if iregions[i].GetId() == id {
			return iregions[i].(*SRegion), nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (ec *SEcloudClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_COMPUTE + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_NETWORK + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_SECURITY_GROUP + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_EIP + cloudprovider.READ_ONLY_SUFFIX,
	}
	return caps
}

func (ec *SEcloudClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Id = ec.GetAccountId()
	subAccount.Name = ec.cpcfg.Name
	subAccount.Account = ec.AccessKey
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (ec *SEcloudClient) GetAccountId() string {
	return ec.AccessKey
}

// GetBalance 查询账户余额，使用 MOPC 开放接口（与 ecloudsdkmopc BalanceQueryPOST 一致）。
func (ec *SEcloudClient) GetBalance() (*cloudprovider.SBalanceInfo, error) {
	type accountInfo struct {
		AccountId   string `json:"accountId"`
		Balance     string `json:"balance"`
		OweAmount   string `json:"oweAmount"`
		NABalance   string `json:"nABalance"`
		DetailName  string `json:"detailName"`
		DetailValue string `json:"detailValue"`
	}
	type accMegRsp struct {
		AccountInfo []accountInfo `json:"accountInfo"`
		RspCode     string        `json:"rspCode"`
		RspDesc     string        `json:"rspDesc"`
	}
	type resultBody struct {
		RspCode   string    `json:"rspCode"`
		RspDesc   string    `json:"rspDesc"`
		AccMegRsp accMegRsp `json:"accMegRsp"`
	}
	type mopcResp struct {
		RespCode string     `json:"respCode"`
		RespDesc string     `json:"respDesc"`
		Result   resultBody `json:"result"`
	}

	ctx := context.Background()
	regionId := ECLOUD_DEFAULT_REGION
	req := NewOpenApiMopcBalanceRequest(regionId, ec.GetAccountId())
	base := req.Base()
	base.Method = "POST"
	body, err := ec.doRequestRaw(ctx, base)
	if err != nil {
		return nil, err
	}
	resp := mopcResp{}
	if err := body.Unmarshal(&resp); err != nil {
		return nil, errors.Wrap(err, "unmarshal mopc balance response")
	}
	// 顶层 respCode: "0"/"00" 视为成功
	if resp.RespCode != "" && resp.RespCode != "0" && resp.RespCode != "00" {
		return nil, fmt.Errorf("balance query failed: respCode=%s respDesc=%s", resp.RespCode, resp.RespDesc)
	}
	// result.rspCode: "00"/"0000" 视为成功
	if resp.Result.RspCode != "" && resp.Result.RspCode != "00" && resp.Result.RspCode != "0000" {
		return nil, fmt.Errorf("balance result error: rspCode=%s rspDesc=%s", resp.Result.RspCode, resp.Result.RspDesc)
	}
	amount := 0.0
	if len(resp.Result.AccMegRsp.AccountInfo) > 0 {
		balanceStr := resp.Result.AccMegRsp.AccountInfo[0].Balance
		if balanceStr != "" {
			if v, err := strconv.ParseFloat(balanceStr, 64); err == nil {
				amount = v
			}
		}
	}
	return &cloudprovider.SBalanceInfo{
		Currency: "CNY",
		Amount:   amount,
		Status:   "",
	}, nil
}

func (ec *SEcloudClient) GetCloudRegionExternalIdPrefix() string {
	return CLOUD_PROVIDER_ECLOUD
}

// completeSingParams 填充签名相关的公共 query 参数。
func (ec *SEcloudClient) completeSingParams(request *SBaseRequest) (err error) {
	queryParams := request.GetQueryParams()
	// 每次签名前先清理旧的 Signature，避免在同一个 request 上重复签名（如分页循环）时将旧签名参与新的签名计算。
	delete(queryParams, "Signature")
	queryParams["AccessKey"] = ec.AccessKey
	queryParams["Version"] = request.GetVersion()
	queryParams["Timestamp"] = request.GetTimestamp()
	queryParams["SignatureMethod"] = "HmacSHA1"
	queryParams["SignatureVersion"] = "V2.0"
	queryParams["SignatureNonce"] = stringutils.UUID4()
	return
}

// buildStringToSign 生成签名字符串，兼容老版移动云签名规则。
func (ec *SEcloudClient) buildStringToSign(request *SBaseRequest) string {
	signParams := request.GetQueryParams()
	queryString := getUrlFormedMap(signParams)
	queryString = strings.Replace(queryString, "+", "%20", -1)
	queryString = strings.Replace(queryString, "*", "%2A", -1)
	queryString = strings.Replace(queryString, "%7E", "~", -1)
	shaString := sha256.Sum256([]byte(queryString))
	summaryQuery := hex.EncodeToString(shaString[:])
	serverPath := strings.Replace(request.GetServerPath(), "/", "%2F", -1)
	return fmt.Sprintf("%s\n%s\n%s", request.GetMethod(), serverPath, summaryQuery)
}

func signSHA1HMAC(source, secret string) string {
	key := []byte(secret)
	h := hmac.New(sha1.New, key)
	h.Write([]byte(source))
	signedBytes := h.Sum(nil)
	return hex.EncodeToString(signedBytes)
}

// parseBodyToList 统一从 API 返回的 body 中解析列表，兼容 content / regions 或直接为数组，避免各处重复处理。
func parseBodyToList(body jsonutils.JSONObject) (*jsonutils.JSONArray, error) {
	if body == nil {
		return nil, fmt.Errorf("response body is nil")
	}
	if arr, ok := body.(*jsonutils.JSONArray); ok {
		return arr, nil
	}
	if body.Contains("content") {
		content, _ := body.Get("content")
		if arr, ok := content.(*jsonutils.JSONArray); ok {
			return arr, nil
		}
		// content 为 null 或 empty:true 时视为空列表
		if content == nil || (body.Contains("empty") && body.Contains("total")) {
			return jsonutils.NewArray(), nil
		}
	}
	if body.Contains("regions") {
		regions, _ := body.Get("regions")
		if arr, ok := regions.(*jsonutils.JSONArray); ok {
			return arr, nil
		}
	}
	if body.Contains("zones") {
		zones, _ := body.Get("zones")
		if arr, ok := zones.(*jsonutils.JSONArray); ok {
			return arr, nil
		}
	}
	return nil, fmt.Errorf("response body should be array or contain content/regions/zones array, got:\n%s", body)
}

func (ec *SEcloudClient) doGet(ctx context.Context, r *SBaseRequest, result interface{}) error {
	r.SetMethod("GET")
	data, err := ec.request(ctx, r)
	if err != nil {
		return err
	}
	return data.Unmarshal(result)
}

func (ec *SEcloudClient) doPost(ctx context.Context, r *SBaseRequest, result interface{}) error {
	r.SetMethod("POST")
	data, err := ec.request(ctx, r)
	if err != nil {
		return err
	}
	return data.Unmarshal(result)
}

func (ec *SEcloudClient) doList(ctx context.Context, r *SBaseRequest, result interface{}) error {
	r.SetMethod("GET")
	// doList 会自动翻页；为避免修改调用方传入的 request，这里拷贝一份 query 参数用于翻页循环。
	query := map[string]string{}
	for k, v := range r.GetQueryParams() {
		query[k] = v
	}
	origQuery := r.QueryParams
	defer func() { r.QueryParams = origQuery }()
	pageStr, hasPage := query["page"]
	pageSizeStr, hasPageSize := query["pageSize"]

	// 使用 page/pageSize 做简单分页聚合：
	// - 若调用方未显式设置 page/pageSize，则默认 page=1,pageSize=100，并自动翻页，直至返回为空或不足一页。
	page := 1
	if hasPage {
		if v, err := strconv.Atoi(pageStr); err == nil && v > 0 {
			page = v
		}
	}
	pageSize := 100
	if hasPageSize {
		if v, err := strconv.Atoi(pageSizeStr); err == nil && v > 0 {
			pageSize = v
		}
	}

	all := jsonutils.NewArray()
	for {
		query["page"] = strconv.Itoa(page)
		query["pageSize"] = strconv.Itoa(pageSize)
		r.QueryParams = query
		data, err := ec.request(ctx, r)
		if err != nil {
			return err
		}
		arr, err := parseBodyToList(data)
		if err != nil {
			return err
		}
		if arr.Length() == 0 {
			break
		}
		for i := 0; i < arr.Length(); i++ {
			item, _ := arr.GetAt(i)
			all.Add(item)
		}
		if arr.Length() < pageSize {
			break
		}
		page++
	}
	return all.Unmarshal(result)
}

// doPostList 与 doList 类似，但使用 POST 方法，适配新的 OpenAPI 列表接口。
func (ec *SEcloudClient) doPostList(ctx context.Context, r *SBaseRequest, result interface{}) error {
	r.SetMethod("POST")

	// POST 列表接口分页参数可能在：
	// - 最外层：{"page":1,"pageSize":100,...}
	// 这里自动翻页聚合：若未显式设置，则默认 page=1,pageSize=100。
	origContent := r.Content
	defer func() { r.Content = origContent }()
	// doPostList 会自动翻页；为避免修改调用方传入的 request，这里基于 Content/默认值构造并循环写回 r.Content。
	var reqBody jsonutils.JSONObject
	if len(r.Content) > 0 {
		if jb, err := jsonutils.Parse(r.Content); err == nil {
			reqBody = jb
		}
	}
	if reqBody == nil {
		reqBody = jsonutils.NewDict()
	}
	// 注意：这里的 dict 是从 Content parse 出来的独立对象，不会影响调用方原始 JSON 对象。
	dict, ok := reqBody.(*jsonutils.JSONDict)
	if !ok {
		return errors.Errorf("doPostList request body should be JSON object, got: %s", reqBody)
	}

	page := 1
	pageSize := 100

	// 优先读取最外层 page/pageSize
	if v, err := dict.Int("page"); err == nil && v > 0 {
		page = int(v)
	}
	if v, err := dict.Int("pageSize"); err == nil && v > 0 {
		pageSize = int(v)
	}

	// 写回分页参数：统一更新/写入最外层 page/pageSize。
	setPage := func(p, ps int) {
		dict.Set("page", jsonutils.NewInt(int64(p)))
		dict.Set("pageSize", jsonutils.NewInt(int64(ps)))
	}

	// 兜底：确保 pageSize 合法
	if pageSize <= 0 {
		pageSize = 100
	}
	if page <= 0 {
		page = 1
	}

	all := jsonutils.NewArray()
	for {
		setPage(page, pageSize)
		r.Content = []byte(dict.String())

		data, err := ec.request(ctx, r)
		if err != nil {
			return err
		}
		arr, err := parseBodyToList(data)
		if err != nil {
			return err
		}
		if arr.Length() == 0 {
			break
		}
		for i := 0; i < arr.Length(); i++ {
			item, _ := arr.GetAt(i)
			all.Add(item)
		}
		if arr.Length() < pageSize {
			break
		}
		page++
	}
	return all.Unmarshal(result)
}

func (ec *SEcloudClient) request(ctx context.Context, r *SBaseRequest) (jsonutils.JSONObject, error) {
	jrbody, err := ec.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	return r.ForMateResponseBody(jrbody)
}

// doRequestRaw 返回原始响应 body（不经过 ForMateResponseBody），用于 MOPC 等返回格式与 state/body 不同的接口。
func (ec *SEcloudClient) doRequestRaw(ctx context.Context, r *SBaseRequest) (jsonutils.JSONObject, error) {
	return ec.doRequest(ctx, r)
}

func (ec *SEcloudClient) doRequest(ctx context.Context, r *SBaseRequest) (jsonutils.JSONObject, error) {
	// sign
	ec.completeSingParams(r)
	stringToSign := ec.buildStringToSign(r)
	secret := "BC_SIGNATURE&" + ec.Secret
	signature := signSHA1HMAC(stringToSign, secret)
	query := r.GetQueryParams()
	query["Signature"] = signature
	header := r.GetHeaders()
	header["Content-Type"] = "application/json"
	var urlStr string
	port := r.GetPort()
	if len(port) > 0 {
		urlStr = fmt.Sprintf("https://%s:%s%s", r.GetEndpoint(), port, r.GetServerPath())
	} else {
		urlStr = fmt.Sprintf("https://%s%s", r.GetEndpoint(), r.GetServerPath())
	}
	// 注意：URL query 需要与签名参数一致并进行标准转义，避免出现特殊字符解析/签名不一致问题。
	queryString := getUrlFormedMap(r.GetQueryParams())
	if len(queryString) > 0 {
		urlStr = urlStr + "?" + queryString
	}
	resp, err := httputils.Request(
		ec.httpClient,
		ctx,
		httputils.THttpMethod(r.GetMethod()),
		urlStr,
		convertHeader(header),
		r.GetBodyReader(),
		ec.debug,
	)
	defer httputils.CloseResponse(resp)
	if err != nil {
		return nil, err
	}
	rbody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read body of response")
	}
	if ec.debug {
		fmt.Fprintf(os.Stderr, "Response body: %s\n", string(rbody))
	}
	rbody = bytes.TrimSpace(rbody)

	var jrbody jsonutils.JSONObject
	if len(rbody) > 0 && (rbody[0] == '{' || rbody[0] == '[') {
		var err error
		jrbody, err = jsonutils.Parse(rbody)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to parsing json: %s", rbody)
		}
	}
	return jrbody, nil
}

type ErrMissKey struct {
	Key string
	Jo  jsonutils.JSONObject
}

func (mk ErrMissKey) Error() string {
	return fmt.Sprintf("The response body should contain the %q key, but it doesn't. It is:\n%s", mk.Key, mk.Jo)
}

func convertHeader(mh map[string]string) http.Header {
	header := http.Header{}
	for k, v := range mh {
		header.Add(k, v)
	}
	return header
}

func getUrlFormedMap(source map[string]string) (urlEncoded string) {
	urlEncoder := url.Values{}
	for key, value := range source {
		urlEncoder.Add(key, value)
	}
	urlEncoded = urlEncoder.Encode()
	return
}
