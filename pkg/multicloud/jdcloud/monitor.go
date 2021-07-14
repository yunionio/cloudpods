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

package jdcloud

import (
	jc_apis "github.com/jdcloud-api/jdcloud-sdk-go/services/monitor/apis"
	client "github.com/jdcloud-api/jdcloud-sdk-go/services/monitor/client"
)

type DescribeMetricDataRequest struct {
	*jc_apis.DescribeMetricDataRequest
}

type DescribeMetricDataResponse struct {
	*jc_apis.DescribeMetricDataResponse
}

func (r *SRegion) GetMetricsData(req *DescribeMetricDataRequest) (*DescribeMetricDataResponse, error) {
	if len(req.RegionId) == 0 {
		req.RegionId = r.GetId()
	}
	monitorClient := client.NewMonitorClient(r.Credential)
	monitorClient.Logger = Logger{}
	response, err := monitorClient.DescribeMetricData(req.DescribeMetricDataRequest)
	if err != nil {
		return nil, err
	}
	return &DescribeMetricDataResponse{response}, err
}

func NewDescribeMetricDataRequestWithAllParams(
	regionId string,
	metric string,
	startTime *string,
	endTime *string,
	timeInterval *string,
	serviceCode *string,
	resourceId string,
) *DescribeMetricDataRequest {
	var aggrType, downSampleType = "avg", "avg"
	request := new(DescribeMetricDataRequest)
	request.DescribeMetricDataRequest = jc_apis.NewDescribeMetricDataRequestWithAllParams(regionId, metric, &aggrType, &downSampleType, startTime, endTime,
		timeInterval, nil, nil, nil, serviceCode, nil, resourceId)
	return request
}
