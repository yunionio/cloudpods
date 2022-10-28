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

package aliyun

import (
	"strings"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	EVENT_REGION_HANGZHOU = "cn-hangzhou"
)

type SAttributes struct {
	CreationDate     time.Time
	MfaAuthenticated bool
}

type SSessionContext struct {
	Attributes SAttributes
}

type SUserIdentity struct {
	AccessKeyId    string
	AccountId      string
	PrincipalId    string
	SessionContext SSessionContext
	Type           string
	UserName       string
}

type SEvent struct {
	region              *SRegion
	AdditionalEventData map[string]string
	ApiVersion          string
	EventId             string
	EventName           string
	EventSource         string
	EventTime           time.Time
	EventType           string
	EventVersion        string
	RequestId           string
	RequestParameters   map[string]string
	ServiceName         string
	SourceIpAddress     string
	UserAgent           string
	UserIdentity        SUserIdentity
	ResponseElements    map[string]string
	IsGlobal            bool
}

func (event *SEvent) GetCreatedAt() time.Time {
	return event.EventTime
}

func (event *SEvent) GetName() string {
	return event.EventName
}

func (event *SEvent) GetAction() string {
	return event.ServiceName
}

func (event *SEvent) GetResourceType() string {
	return ""
}

func (event *SEvent) GetRequestId() string {
	return event.RequestId
}

func (event *SEvent) GetRequest() jsonutils.JSONObject {
	return jsonutils.Marshal(event)
}

func (event *SEvent) GetAccount() string {
	if account, ok := event.AdditionalEventData["loginAccount"]; ok {
		return account
	}
	if len(event.UserIdentity.AccessKeyId) > 0 {
		return event.UserIdentity.AccessKeyId
	}
	return event.UserIdentity.UserName
}

func (event *SEvent) GetService() string {
	return event.ServiceName
}

func (event *SEvent) IsSuccess() bool {
	return true
}

func (region *SRegion) GetICloudEvents(start time.Time, end time.Time, withReadEvent bool) ([]cloudprovider.ICloudEvent, error) {
	var (
		events  []SEvent
		err     error
		token   string
		_events []SEvent
		iEvents []cloudprovider.ICloudEvent
		eventRW string
	)

	eventRW = "Write"
	if withReadEvent {
		eventRW = ""
	}

	for {
		_events, token, err = region.GetEvents(start, end, token, eventRW, "")
		if err != nil {
			return nil, errors.Wrap(err, "region.GetEvents")
		}
		events = append(events, _events...)
		if len(token) == 0 || len(_events) == 0 {
			break
		}
	}

	for i := range events {
		if events[i].IsGlobal && region.RegionId != ALIYUN_DEFAULT_REGION {
			continue
		}
		if !withReadEvent && strings.HasPrefix(events[i].EventName, "Query") { // QueryInstanceBill QueryBill
			continue
		}
		iEvents = append(iEvents, &events[i])
	}
	return iEvents, nil
}

func (region *SRegion) GetEvents(start time.Time, end time.Time, token string, eventRW string, requestId string) ([]SEvent, string, error) {
	params := map[string]string{
		"RegionId": region.RegionId,
	}
	if !start.IsZero() {
		params["StartTime"] = start.Format("2006-01-02T15:04:05Z")
	}
	if !end.IsZero() {
		params["EndTime"] = end.Format("2006-01-02T15:04:05Z")
	}
	if utils.IsInStringArray(eventRW, []string{"Read", "Write"}) {
		params["LookupAttribute.1.Key"] = "EventRW"
		params["LookupAttribute.1.Value"] = eventRW
	}
	if len(token) > 0 {
		params["NextToken"] = token
	}
	if len(requestId) > 0 {
		params["Request"] = requestId
	}
	resp, err := region.trialRequest("LookupEvents", params)
	if err != nil {
		if strings.Contains(err.Error(), "no such host") { // cn-wulanchabu not support
			return []SEvent{}, "", nil
		}
		return nil, "", err
	}

	events := []SEvent{}
	err = resp.Unmarshal(&events, "Events")
	if err != nil {
		return nil, "", err
	}
	nextToken, _ := resp.GetString("NextToken")
	return events, nextToken, nil
}
