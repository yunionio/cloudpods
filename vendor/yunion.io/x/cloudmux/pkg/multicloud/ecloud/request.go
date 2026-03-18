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
	"io"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SJSONRequest struct {
	SBaseRequest
	Data jsonutils.JSONObject
}

func NewJSONRequest(data jsonutils.JSONObject) *SJSONRequest {
	jr := &SJSONRequest{
		Data:         data,
		SBaseRequest: *NewBaseRequest(),
	}
	headers := jr.GetHeaders()
	headers["Content-Type"] = "application/json"
	return jr
}

func newBaseJSONRequest(regionId string, endpoint string, port string, serverPath string, query map[string]string, data jsonutils.JSONObject) SJSONRequest {
	req := *NewJSONRequest(data)
	req.SBaseRequest.Endpoint = endpoint
	req.SBaseRequest.Port = port
	req.SBaseRequest.RegionId = regionId
	req.SBaseRequest.ServerPath = serverPath
	if data != nil {
		req.SBaseRequest.Content = []byte(data.String())
	}
	mergeMap(req.GetQueryParams(), query)
	return req
}

func (jr *SJSONRequest) GetBodyReader() io.Reader {
	if jr.Data == nil {
		return nil
	}
	return strings.NewReader(jr.Data.String())
}

// SOpenApiInstanceRequest 用于调用新的 OpenAPI 云主机实例列表接口：
// 走区域 endpoint (api-xxx.cmecloud.cn:8443)，路径为 /api/openapi-instance/v4/list/describe-instances
type SOpenApiInstanceRequest struct {
	SJSONRequest
	RegionId string
}

// SOpenApiInstanceActionRequest 用于 OpenAPI Instance 区域接口（与实例列表一致走 api-*.cmecloud.cn:8443）。
// 可用于除 list/describe-instances 之外的其他 instance openapi path。
type SOpenApiInstanceActionRequest struct {
	SJSONRequest
	RegionId   string
	ServerPath string
}

func NewOpenApiInstanceRequest(regionId string, data jsonutils.JSONObject) *SOpenApiInstanceRequest {
	host := openApiRegionHost(regionId)
	r := SOpenApiInstanceRequest{RegionId: regionId}
	r.SJSONRequest = newBaseJSONRequest(regionId, host, "8443", "/api/openapi-instance/v4/list/describe-instances", nil, data)
	return &r
}

func (rr *SOpenApiInstanceRequest) GetPort() string {
	return "8443"
}

func (rr *SOpenApiInstanceRequest) GetEndpoint() string {
	return openApiRegionHost(rr.RegionId)
}

func (rr *SOpenApiInstanceRequest) GetHeaders() map[string]string {
	h := rr.SJSONRequest.GetHeaders()
	h["Region-Id"] = rr.RegionId
	// OpenAPI 示例中带有 User-Agent，但不是必须字段，这里不强制设置
	return h
}

func NewOpenApiInstanceActionRequest(regionId string, serverPath string, data jsonutils.JSONObject) *SOpenApiInstanceActionRequest {
	host := openApiRegionHost(regionId)
	r := SOpenApiInstanceActionRequest{RegionId: regionId, ServerPath: serverPath}
	r.SJSONRequest = newBaseJSONRequest(regionId, host, "8443", serverPath, nil, data)
	return &r
}

func (rr *SOpenApiInstanceActionRequest) GetPort() string       { return "8443" }
func (rr *SOpenApiInstanceActionRequest) GetEndpoint() string   { return openApiRegionHost(rr.RegionId) }
func (rr *SOpenApiInstanceActionRequest) GetServerPath() string { return rr.ServerPath }
func (rr *SOpenApiInstanceActionRequest) GetHeaders() map[string]string {
	h := rr.SJSONRequest.GetHeaders()
	if len(rr.RegionId) > 0 {
		h["Region-Id"] = rr.RegionId
	}
	return h
}

func (r *SOpenApiInstanceActionRequest) Base() *SBaseRequest { return &r.SJSONRequest.SBaseRequest }

// openApiRegionHostFallback 与官方 SDK 不一致的 regionId -> 主机名（省/区名与机房城市名不同时需单独映射），先查表再走拼接。
var openApiRegionHostFallback = map[string]string{
	"cn-beijing-1":   "api-beijing-2.cmecloud.cn",   // 华北北京3
	"cn-jiangsu-1":   "api-wuxi-1.cmecloud.cn",      // 华东苏州
	"cn-guangdong-1": "api-dongguan-1.cmecloud.cn",  // 华南广州3
	"cn-sichuan-1":   "api-yaan-1.cmecloud.cn",      // 西南成都
	"cn-henan-1":     "api-zhengzhou-1.cmecloud.cn", // 华中郑州
	"cn-hunan-1":     "api-zhuzhou-1.cmecloud.cn",   // 华中长沙2
	"cn-shandong-1":  "api-jinan-1.cmecloud.cn",     // 华东济南
	"cn-shaanxi-1":   "api-xian-1.cmecloud.cn",      // 西北西安（陕 vs 山）
	"cn-shangxi-1":   "api-shanxi-1.cmecloud.cn",    // 山西太原
	"cn-zhejiang-1":  "api-ningbo-1.cmecloud.cn",    // 华东杭州
	"cn-yunnan-1":    "api-yunnan-2.cmecloud.cn",    // 云南昆明2
	"cn-neimenggu-1": "api-huhehaote-1.cmecloud.cn", // 华北呼和浩特
	"cn-guzhou-1":    "api-guiyang-1.cmecloud.cn",   // 西南贵阳（贵州->贵阳）
	"cn-hubei-2":     "api-wuhan-1.cmecloud.cn",     // 湖北武汉
}

// openApiRegionHost 由 regionId 得到区域 API 主机：先查 fallback（与官方 SDK 一致的特殊项），再按 api-{suffix}.cmecloud.cn 拼接，避免 API 返回新 region 拿不到 endpoint。
func openApiRegionHost(regionId string) string {
	if host := openApiRegionHostFallback[regionId]; host != "" {
		return host
	}
	suffix := strings.TrimPrefix(regionId, "cn-")
	if suffix == "" {
		suffix = regionId
	}
	return "api-" + suffix + ".cmecloud.cn"
}

// openApiVpcConsoleHost 用于 OpenAPI VPC：ecloudsdkvpc 使用 console-*.cmecloud.cn 且会正确转发 POST body，api-* 网关可能不转 body 导致“参数为空”。
var openApiVpcConsoleHostFallback = map[string]string{
	"cn-beijing-1":   "console-beijing-2.cmecloud.cn",
	"cn-jiangsu-1":   "console-wuxi-1.cmecloud.cn",
	"cn-guangdong-1": "console-dongguan-1.cmecloud.cn",
	"cn-sichuan-1":   "console-yaan-1.cmecloud.cn",
	"cn-henan-1":     "console-zhengzhou-1.cmecloud.cn",
	"cn-hunan-1":     "console-zhuzhou-1.cmecloud.cn",
	"cn-shandong-1":  "console-jinan-1.cmecloud.cn",
	"cn-shaanxi-1":   "console-xian-1.cmecloud.cn",
	"cn-shanghai-1":  "console-shanghai-1.cmecloud.cn",
	"cn-chongqing-1": "console-chongqing-1.cmecloud.cn",
	"cn-zhejiang-1":  "console-ningbo-1.cmecloud.cn",
	"cn-tianjin-1":   "console-tianjin-1.cmecloud.cn",
	"cn-jilin-1":     "console-jilin-1.cmecloud.cn",
	"cn-hubei-2":     "console-hubei-1.cmecloud.cn",
	"cn-jiangxi-1":   "console-jiangxi-1.cmecloud.cn",
	"cn-gansu-1":     "console-gansu-1.cmecloud.cn",
	"cn-shangxi-1":   "console-shanxi-1.cmecloud.cn",
	"cn-liaoning-1":  "console-liaoning-1.cmecloud.cn",
	"cn-yunnan-1":    "console-yunnan-2.cmecloud.cn",
	"cn-hebei-1":     "console-hebei-1.cmecloud.cn",
	"cn-fujian-1":    "console-fujian-1.cmecloud.cn",
	"cn-guangxi-1":   "console-guangxi-1.cmecloud.cn",
	"cn-anhui-1":     "console-anhui-1.cmecloud.cn",
	"cn-neimenggu-1": "console-huhehaote-1.cmecloud.cn",
	"cn-guzhou-1":    "console-guiyang-1.cmecloud.cn",
	"cn-hainan-1":    "console-hainan-1.cmecloud.cn",
	"cn-xinjiang-1":  "console-xinjiang-1.cmecloud.cn",
}

func openApiVpcConsoleHost(regionId string) string {
	if host := openApiVpcConsoleHostFallback[regionId]; host != "" {
		return host
	}
	suffix := strings.TrimPrefix(regionId, "cn-")
	if suffix == "" {
		suffix = regionId
	}
	return "console-" + suffix + ".cmecloud.cn"
}

// SOpenApiRegionRequest 用于调用新的 OpenAPI 区域列表接口：
// 需走区域 endpoint (api-xxx.cmecloud.cn:8443)，与 yunion.io/x/ecloud 成功样例一致。
type SOpenApiRegionRequest struct {
	SJSONRequest
	RegionId string
}

func NewOpenApiRegionRequest(regionId string, data jsonutils.JSONObject) *SOpenApiRegionRequest {
	host := openApiRegionHost(regionId)
	r := SOpenApiRegionRequest{
		SJSONRequest: *NewJSONRequest(data),
		RegionId:     regionId,
	}
	r.SBaseRequest.Port = "8443"
	r.SBaseRequest.Endpoint = host
	r.SBaseRequest.RegionId = regionId
	r.ServerPath = "/api/openapi-instance/v4/region/describe-regions"
	return &r
}

func (rr *SOpenApiRegionRequest) GetPort() string {
	return "8443"
}

func (rr *SOpenApiRegionRequest) GetEndpoint() string {
	return openApiRegionHost(rr.RegionId)
}

func (rr *SOpenApiRegionRequest) GetHeaders() map[string]string {
	h := rr.SJSONRequest.GetHeaders()
	if len(rr.RegionId) > 0 {
		h["Region-Id"] = rr.RegionId
	}
	return h
}

// SOpenApiZoneRequest 用于 OpenAPI 可用区列表：GET 区域 endpoint 上的 describe-zones，与 describe-regions 同机房子网。
type SOpenApiZoneRequest struct {
	SJSONRequest
	RegionId string
}

func NewOpenApiZoneRequest(regionId string, data jsonutils.JSONObject) *SOpenApiZoneRequest {
	host := openApiRegionHost(regionId)
	r := SOpenApiZoneRequest{
		SJSONRequest: *NewJSONRequest(data),
		RegionId:     regionId,
	}
	r.SBaseRequest.Port = "8443"
	r.SBaseRequest.Endpoint = host
	r.SBaseRequest.RegionId = regionId
	r.ServerPath = "/api/openapi-instance/v4/region/describe-zones"
	return &r
}

func (rr *SOpenApiZoneRequest) GetPort() string     { return "8443" }
func (rr *SOpenApiZoneRequest) GetEndpoint() string { return openApiRegionHost(rr.RegionId) }
func (rr *SOpenApiZoneRequest) GetHeaders() map[string]string {
	h := rr.SJSONRequest.GetHeaders()
	if len(rr.RegionId) > 0 {
		h["Region-Id"] = rr.RegionId
	}
	return h
}

func (r *SOpenApiZoneRequest) Base() *SBaseRequest {
	return &r.SJSONRequest.SBaseRequest
}

// SOpenApiRequest 通用 OpenAPI 请求，用于 ecloud.10086.cn 上任意 path（GET/POST）。
// 新接口迁移时可直接使用，无需再新增专用 Request 类型。
type SOpenApiRequest struct {
	SJSONRequest
	RegionId   string
	ServerPath string
}

func NewOpenApiRequest(regionId string, serverPath string, data jsonutils.JSONObject) *SOpenApiRequest {
	r := SOpenApiRequest{RegionId: regionId, ServerPath: serverPath}
	r.SJSONRequest = newBaseJSONRequest(regionId, "ecloud.10086.cn", "", serverPath, nil, data)
	return &r
}

func (rr *SOpenApiRequest) GetPort() string   { return "" }
func (rr *SOpenApiRequest) GetEndpoint() string {
	return "ecloud.10086.cn"
}
func (rr *SOpenApiRequest) GetServerPath() string { return rr.ServerPath }
func (rr *SOpenApiRequest) GetRegionId() string   { return rr.RegionId }
func (rr *SOpenApiRequest) GetHeaders() map[string]string {
	h := rr.SJSONRequest.GetHeaders()
	if len(rr.RegionId) > 0 {
		h["Region-Id"] = rr.RegionId
	}
	return h
}

// regionIdToPoolId 创建 VPC 时 body.region 需为资源池 ID（与 ecloudsdkvpc initRegions 一致）
var regionIdToPoolId = map[string]string{
	"cn-beijing-1":   "CIDC-RP-29", // 北京2
	"cn-jiangsu-1":   "CIDC-RP-25", // 无锡
	"cn-guangdong-1": "CIDC-RP-26", // 东莞
	"cn-sichuan-1":   "CIDC-RP-27", // 雅安
	"cn-henan-1":     "CIDC-RP-28", // 郑州
	"cn-hunan-1":     "CIDC-RP-30", // 株洲
	"cn-shandong-1":  "CIDC-RP-31", // 济南
	"cn-shaanxi-1":   "CIDC-RP-32", // 西安
	"cn-shanghai-1":  "CIDC-RP-33",
	"cn-chongqing-1": "CIDC-RP-34",
	"cn-zhejiang-1":  "CIDC-RP-35", // 宁波
	"cn-tianjin-1":   "CIDC-RP-36",
	"cn-jilin-1":     "CIDC-RP-37",
	"cn-hubei-2":     "CIDC-RP-38", // 湖北
	"cn-jiangxi-1":   "CIDC-RP-39",
	"cn-gansu-1":     "CIDC-RP-40",
	"cn-shangxi-1":   "CIDC-RP-41", // 山西
	"cn-liaoning-1":  "CIDC-RP-42",
	"cn-yunnan-1":    "CIDC-RP-43",
	"cn-hebei-1":     "CIDC-RP-44",
	"cn-fujian-1":    "CIDC-RP-45",
	"cn-guangxi-1":   "CIDC-RP-46",
	"cn-anhui-1":     "CIDC-RP-47",
	"cn-neimenggu-1": "CIDC-RP-48", // 呼和浩特
	"cn-guzhou-1":    "CIDC-RP-49", // 贵阳
	"cn-hainan-1":    "CIDC-RP-53",
	"cn-xinjiang-1":  "CIDC-RP-54",
}

// SOpenApiVpcRequest 用于 OpenAPI VPC 接口：与 ecloudsdkvpc 一致使用 console-*.cmecloud.cn:8443，否则 api-* 网关可能不转发 POST body 导致“可用区region不能为空”。
type SOpenApiVpcRequest struct {
	SJSONRequest
	RegionId   string
	ServerPath string
}

func (rr *SOpenApiVpcRequest) GetHeaders() map[string]string {
	h := rr.SJSONRequest.GetHeaders()
	if len(rr.RegionId) > 0 {
		h["Region-Id"] = rr.RegionId
	}
	return h
}

func newOpenApiConsoleRequest(regionId string, serverPath string, query map[string]string, data jsonutils.JSONObject) (host string, req SJSONRequest) {
	host = openApiVpcConsoleHost(regionId)
	req = newBaseJSONRequest(regionId, host, "8443", serverPath, query, data)
	return host, req
}

func NewOpenApiVpcRequest(regionId string, serverPath string, query map[string]string, data jsonutils.JSONObject) *SOpenApiVpcRequest {
	_, req := newOpenApiConsoleRequest(regionId, serverPath, query, data)
	r := SOpenApiVpcRequest{
		SJSONRequest: req,
		RegionId:     regionId,
		ServerPath:   serverPath,
	}
	return &r
}

func (rr *SOpenApiVpcRequest) GetPort() string     { return "8443" }
func (rr *SOpenApiVpcRequest) GetEndpoint() string { return openApiVpcConsoleHost(rr.RegionId) }
func (rr *SOpenApiVpcRequest) Base() *SBaseRequest { return &rr.SJSONRequest.SBaseRequest }

// SOpenApiEbsRequest 用于 EBS/磁盘/快照 OpenAPI：与 ecloudsdkebs 一致使用 console-*.cmecloud.cn:8443。
type SOpenApiEbsRequest struct {
	SJSONRequest
	RegionId   string
	ServerPath string
}

func NewOpenApiEbsRequest(regionId string, serverPath string, query map[string]string, data jsonutils.JSONObject) *SOpenApiEbsRequest {
	_, req := newOpenApiConsoleRequest(regionId, serverPath, query, data)
	r := SOpenApiEbsRequest{
		SJSONRequest: req,
		RegionId:     regionId,
		ServerPath:   serverPath,
	}
	return &r
}

func (rr *SOpenApiEbsRequest) GetPort() string     { return "8443" }
func (rr *SOpenApiEbsRequest) GetEndpoint() string { return openApiVpcConsoleHost(rr.RegionId) }
func (rr *SOpenApiEbsRequest) Base() *SBaseRequest { return &rr.SJSONRequest.SBaseRequest }
func (rr *SOpenApiEbsRequest) GetHeaders() map[string]string {
	h := rr.SJSONRequest.GetHeaders()
	if len(rr.RegionId) > 0 {
		h["Region-Id"] = rr.RegionId
	}
	return h
}

// SOpenApiMopcRequest 用于 MOPC 开放接口（如账户余额查询）。ecloudsdkmopc 中 CIDC-CORE-00 对应 ecloud.10086.cn，/api/openapi-mop/ 仅在该网关可路由，console-* 会报 M02C002 gateway cannot route。
type SOpenApiMopcRequest struct {
	SJSONRequest
	RegionId   string
	ServerPath string
}

const mopcBalanceQueryPath = "/api/openapi-mop/openapi"

func NewOpenApiMopcBalanceRequest(regionId string, userId string) *SOpenApiMopcRequest {
	body := jsonutils.Marshal(map[string]interface{}{
		"balanceQueryPOSTBody":   map[string]string{"userId": userId},
		"balanceQueryPOSTHeader": map[string]string{},
	})
	r := SOpenApiMopcRequest{RegionId: regionId, ServerPath: mopcBalanceQueryPath}
	r.SJSONRequest = newBaseJSONRequest(regionId, "ecloud.10086.cn", "", mopcBalanceQueryPath, nil, body)
	r.SBaseRequest.QueryParams["method"] = "SYAN_UNHT_balancequeryOpen"
	r.SBaseRequest.QueryParams["format"] = "json"
	r.SBaseRequest.QueryParams["status"] = "1"
	return &r
}

func (rr *SOpenApiMopcRequest) GetPort() string     { return "" }
func (rr *SOpenApiMopcRequest) GetEndpoint() string { return "ecloud.10086.cn" }
func (rr *SOpenApiMopcRequest) Base() *SBaseRequest { return &rr.SJSONRequest.SBaseRequest }
func (rr *SOpenApiMopcRequest) GetHeaders() map[string]string {
	h := rr.SJSONRequest.GetHeaders()
	if len(rr.RegionId) > 0 {
		h["Region-Id"] = rr.RegionId
	}
	return h
}

func mergeMap(m1, m2 map[string]string) {
	if m2 == nil {
		return
	}
	for k, v := range m2 {
		m1[k] = v
	}
}

type SBaseRequest struct {
	Method         string
	Endpoint       string
	ServerPath     string
	Port           string
	RegionId       string
	ReadTimeout    time.Duration
	ConnectTimeout time.Duration
	isInsecure     *bool

	QueryParams map[string]string
	Headers     map[string]string
	Content     []byte
}

func NewBaseRequest() *SBaseRequest {
	return &SBaseRequest{
		QueryParams: map[string]string{},
		Headers:     map[string]string{},
	}
}

func (br *SBaseRequest) GetMethod() string {
	return br.Method
}

func (br *SBaseRequest) SetMethod(method string) {
	br.Method = method
}

func (br *SBaseRequest) GetEndpoint() string {
	return br.Endpoint
}

func (br *SBaseRequest) GetServerPath() string {
	return br.ServerPath
}

func (br *SBaseRequest) GetPort() string {
	return br.Port
}

func (br *SBaseRequest) GetRegionId() string {
	return br.RegionId
}

func (br *SBaseRequest) GetHeaders() map[string]string {
	return br.Headers
}

func (br *SBaseRequest) GetQueryParams() map[string]string {
	return br.QueryParams
}

func (br *SBaseRequest) GetBodyReader() io.Reader {
	if len(br.Content) > 0 {
		return bytes.NewReader(br.Content)
	}
	return nil
}

func (br *SBaseRequest) GetVersion() string {
	return "2016-12-05"
}

func (br *SBaseRequest) GetTimestamp() string {
	sh, _ := time.LoadLocation("Asia/Shanghai")
	return time.Now().In(sh).Format("2006-01-02T15:04:05Z")
}

func (br *SBaseRequest) GetReadTimeout() time.Duration {
	return br.ReadTimeout
}

func (br *SBaseRequest) GetConnectTimeout() time.Duration {
	return br.ConnectTimeout
}

func (br *SBaseRequest) GetHTTPSInsecure() bool {
	if br.isInsecure == nil {
		return false
	}
	return *br.isInsecure
}

func (br *SBaseRequest) SetHTTPSInsecure(isInsecure bool) {
	br.isInsecure = &isInsecure
}

func (br *SBaseRequest) GetUserAgent() map[string]string {
	return nil
}

// Base 返回底层的 SBaseRequest 指针，便于客户端统一处理。
func (jr *SJSONRequest) Base() *SBaseRequest {
	return &jr.SBaseRequest
}

func (r *SOpenApiInstanceRequest) Base() *SBaseRequest {
	return &r.SJSONRequest.SBaseRequest
}

func (r *SOpenApiRegionRequest) Base() *SBaseRequest {
	return &r.SJSONRequest.SBaseRequest
}

func (r *SOpenApiRequest) Base() *SBaseRequest {
	return &r.SJSONRequest.SBaseRequest
}

func (br *SBaseRequest) ForMateResponseBody(jrbody jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if jrbody == nil || !jrbody.Contains("state") {
		return nil, ErrMissKey{
			Key: "state",
			Jo:  jrbody,
		}
	}
	state, _ := jrbody.GetString("state")
	switch state {
	case "OK":
		if !jrbody.Contains("body") {
			// 部分接口（如 DELETE）仅返回 state:OK，无 body
			return jsonutils.NewDict(), nil
		}
		body, _ := jrbody.Get("body")
		return body, nil
	default:
		if jrbody.Contains("errorMessage") {
			msg, _ := jrbody.GetString("errorMessage")
			if strings.Contains(msg, "Invalid parameter AccessKey") {
				return nil, errors.Wrapf(cloudprovider.ErrInvalidAccessKey, "%s", msg)
			}
			return nil, &httputils.JSONClientError{Code: 400, Details: msg}
		}
		return nil, &httputils.JSONClientError{Code: 400, Details: jrbody.String()}
	}
}
