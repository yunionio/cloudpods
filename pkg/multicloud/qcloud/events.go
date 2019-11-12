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

package qcloud

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SEvent struct {
	region          *SRegion
	AccountID       int64
	CloudAuditEvent string
	ErrorCode       int
	EventId         string
	EventName       string
	EventNameCn     string
	EventRegion     string
	EventSource     string
	EventTime       time.Time
	RequestID       string
	ResourceRegion  string
	ResourceTypeCn  string
	Resources       map[string]string
	SecretId        string
	SourceIPAddress string
	Username        string
}

func (event *SEvent) GetName() string {
	if resourceName, ok := event.Resources["ResourceName"]; ok && len(resourceName) > 0 {
		return resourceName
	}
	return event.EventName
}

func (event *SEvent) GetService() string {
	return event.ResourceTypeCn
}

func (event *SEvent) GetAction() string {
	return event.EventName
}

func (event *SEvent) GetResourceType() string {
	if resourceType, ok := event.Resources["ResourceType"]; ok {
		return resourceType
	}
	return ""
}

func (event *SEvent) GetRequest() jsonutils.JSONObject {
	return jsonutils.Marshal(event)
}

func (event *SEvent) GetRequestId() string {
	return event.RequestID
}

func (event *SEvent) GetAccount() string {
	if len(event.SecretId) > 0 {
		return event.SecretId
	}
	return event.Username
}

func (event *SEvent) IsSuccess() bool {
	return event.ErrorCode > 0
}

func (event *SEvent) GetCreatedAt() time.Time {
	return event.EventTime
}

func (region *SRegion) GetICloudEvents(start time.Time, end time.Time, withReadEvent bool) ([]cloudprovider.ICloudEvent, error) {
	events, err := region.GetEvents(start, end)
	if err != nil {
		return nil, err
	}
	iEvents := []cloudprovider.ICloudEvent{}
	for i := range events {
		if withReadEvent || !strings.Contains(events[i].CloudAuditEvent, `"Read"`) {
			iEvents = append(iEvents, &events[i])
		}
	}
	return iEvents, nil
}

func (region *SRegion) GetEvents(start time.Time, end time.Time) ([]SEvent, error) {
	var (
		events    []SEvent
		nextToken string
		err       error
		_events   []SEvent
	)
	for {
		_events, nextToken, err = region.getEvents(start, end, nextToken)
		if err != nil {
			return nil, err
		}
		if len(nextToken) == 0 {
			break
		}
		events = append(events, _events...)
	}
	return events, nil
}

func (region *SRegion) getEvents(start time.Time, end time.Time, nextToken string) ([]SEvent, string, error) {
	params := map[string]string{}
	if start.IsZero() {
		start = time.Now().AddDate(0, 0, -7)
	}
	if end.IsZero() {
		end = time.Now()
	}
	params["StartTime"] = fmt.Sprintf("%d", start.Unix())
	params["EndTime"] = fmt.Sprintf("%d", end.Unix())
	params["MaxResults"] = "50"
	if len(nextToken) > 0 {
		params["NextToken"] = nextToken
	}

	body, err := region.auditRequest("LookUpEvents", params)
	if err != nil {
		return nil, "", err
	}

	events := []SEvent{}

	err = body.Unmarshal(&events, "Events")
	if err != nil {
		return nil, "", errors.Wrap(err, "body.Unmarshal")
	}
	nextToken, _ = body.GetString("NextToken")
	if over, _ := body.Bool("ListOver"); over {
		nextToken = ""
	}
	return events, nextToken, nil
}
