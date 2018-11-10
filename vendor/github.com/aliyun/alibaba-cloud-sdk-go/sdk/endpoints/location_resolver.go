/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package endpoints

import (
	"encoding/json"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"time"
)

const (
	EndpointCacheExpireTime = 3600 //Seconds
)

var lastClearTimePerProduct map[string]int64 = make(map[string]int64)
var endpointCache map[string]string = make(map[string]string)

type LocationResolver struct {
}

func (resolver *LocationResolver) TryResolve(param *ResolveParam) (endpoint string, support bool, err error) {
	if len(param.LocationProduct) <= 0 {
		support = false
		return
	}

	//get from cache
	cacheKey := param.Product + "#" + param.RegionId
	if endpointCache != nil && len(endpointCache[cacheKey]) > 0 && !CheckCacheIsExpire(cacheKey) {
		endpoint = endpointCache[cacheKey]
		support = true
		return
	}

	//get from remote
	getEndpointRequest := requests.NewCommonRequest()
	getEndpointRequest.Product = "Location"
	getEndpointRequest.Version = "2015-06-12"
	getEndpointRequest.ApiName = "DescribeEndpoints"
	getEndpointRequest.Domain = "location.aliyuncs.com"
	getEndpointRequest.Method = "GET"

	getEndpointRequest.QueryParams["Id"] = param.RegionId
	getEndpointRequest.QueryParams["ServiceCode"] = param.LocationProduct
	if len(param.LocationEndpoint) > 0 {
		getEndpointRequest.QueryParams["Type"] = param.LocationEndpoint
	} else {
		getEndpointRequest.QueryParams["Type"] = "openAPI"
	}

	response, err := param.CommonApi(getEndpointRequest)
	//{"Endpoints":{"Endpoint":[{"Protocols":{"Protocols":["HTTP","HTTPS"]},"Type":"openAPI","Namespace":"26842","Id":"cn-hangzhou","SerivceCode":"apigateway","Endpoint":"apigateway.cn-hangzhou.aliyuncs.com"}]},"RequestId":"3287538B-19A0-4550-9995-143C5EDBD955","Success":true}
	var getEndpointResponse GetEndpointResponse
	if !response.IsSuccess() {
		support = false
		return
	}

	json.Unmarshal([]byte(response.GetHttpContentString()), &getEndpointResponse)
	if !getEndpointResponse.Success || getEndpointResponse.Endpoints == nil {
		support = false
		return
	}
	if len(getEndpointResponse.Endpoints.Endpoint) <= 0 {
		support = false
		return
	}
	if len(getEndpointResponse.Endpoints.Endpoint[0].Endpoint) > 0 {
		endpoint = getEndpointResponse.Endpoints.Endpoint[0].Endpoint
		endpointCache[cacheKey] = endpoint
		lastClearTimePerProduct[cacheKey] = time.Now().Unix()
		support = true
		return
	}

	support = false
	return
}

func CheckCacheIsExpire(cacheKey string) bool {
	lastClearTime := lastClearTimePerProduct[cacheKey]
	if lastClearTime <= 0 {
		lastClearTime = time.Now().Unix()
		lastClearTimePerProduct[cacheKey] = lastClearTime
	}

	now := time.Now().Unix()
	elapsedTime := now - lastClearTime
	if elapsedTime > EndpointCacheExpireTime {
		return true
	}

	return false
}

type GetEndpointResponse struct {
	Endpoints *EndpointsObj
	RequestId string
	Success   bool
}

type EndpointsObj struct {
	Endpoint []EndpointObj
}

type EndpointObj struct {
	Protocols   map[string]string
	Type        string
	Namespace   string
	Id          string
	SerivceCode string
	Endpoint    string
}
