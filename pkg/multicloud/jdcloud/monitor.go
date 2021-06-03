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
