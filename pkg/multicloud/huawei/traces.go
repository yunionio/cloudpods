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

package huawei

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SUser struct {
	Domain map[string]string
	Name   string
	Id     string
}

type SEvent struct {
	TraceId      string
	Code         string
	TraceName    string
	ResourceType string
	ApiVersion   string
	SourceIp     string
	TraceType    string
	ServiceType  string
	EventType    string
	ProjectId    string
	Request      string
	Response     string
	TrackerName  string
	TraceStatus  string
	Time         int64
	ResourceId   string
	ResourceName string
	User         SUser
	RecordTime   int64
}

func (event *SEvent) GetName() string {
	if len(event.ResourceId) > 0 {
		return event.ResourceId
	}
	if len(event.ResourceName) > 0 {
		return event.ResourceName
	}
	return event.TraceName
}

func (event *SEvent) GetService() string {
	return event.ServiceType
}

func (event *SEvent) GetAction() string {
	return event.TraceName
}

func (event *SEvent) GetResourceType() string {
	return event.ResourceType
}

func (event *SEvent) GetRequestId() string {
	return event.TraceId
}

func (event *SEvent) GetRequest() jsonutils.JSONObject {
	return jsonutils.Marshal(event)
}

func (event *SEvent) GetAccount() string {
	return event.User.Name
}

func (event *SEvent) IsSuccess() bool {
	return strings.HasPrefix(event.Code, "2")
}

func (event *SEvent) GetCreatedAt() time.Time {
	return time.Unix(event.Time/1000, event.Time%1000)
}

func (self *SRegion) GetICloudEvents(start time.Time, end time.Time, withReadEvent bool) ([]cloudprovider.ICloudEvent, error) {
	if !self.client.isMainProject {
		return nil, cloudprovider.ErrNotSupported
	}
	events, err := self.GetEvents(start, end)
	if err != nil {
		return nil, err
	}
	iEvents := []cloudprovider.ICloudEvent{}
	for i := range events {
		iEvents = append(iEvents, &events[i])
	}
	return iEvents, nil
}

func (self *SRegion) GetEvents(start time.Time, end time.Time) ([]SEvent, error) {
	events := []SEvent{}
	params := map[string]string{}
	if start.IsZero() {
		start = time.Now().AddDate(0, 0, -7)
	}
	if end.IsZero() {
		end = time.Now()
	}
	params["from"] = fmt.Sprintf("%d000", start.Unix())
	params["to"] = fmt.Sprintf("%d000", end.Unix())

	err := doListAllWithMarker(self.ecsClient.Traces.List, params, &events)
	if err != nil {
		return nil, errors.Wrap(err, "doListAllWithMarker")
	}
	return events, nil
}
