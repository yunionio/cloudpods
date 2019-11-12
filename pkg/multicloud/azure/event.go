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

package azure

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SAuthorization struct {
	Action string
	Scope  string
}

type SClaims struct {
	Aud      string
	Iss      string
	Iat      string
	Nbf      string
	Exp      string
	Aio      string
	Appid    string
	Appidacr string
	Uti      string
	Ver      string
}

type SLocalized struct {
	Value          string
	LocalizedValue string
}

type SEvent struct {
	region *SRegion

	Authorization        SAuthorization
	Channels             string
	Claims               SClaims
	CorrelationId        string
	Description          string
	EventDataId          string
	EventName            SLocalized
	Category             SLocalized
	Level                string
	ResourceGroupName    string
	ResourceProviderName SLocalized
	ResourceId           string
	ResourceType         SLocalized
	OperationId          string
	OperationName        SLocalized
	Properties           string
	Status               SLocalized
	SubStatus            SLocalized
	Caller               string
	EventTimestamp       time.Time
	SubmissionTimestamp  time.Time
	SubscriptionId       string
	TenantId             string
	ID                   string
	Name                 string
}

func (event *SEvent) GetName() string {
	return event.ResourceId
}

func (event *SEvent) GetService() string {
	return event.ResourceProviderName.Value
}

func (event *SEvent) GetAction() string {
	return event.OperationName.Value
}

func (event *SEvent) GetResourceType() string {
	return event.ResourceType.Value
}

func (event *SEvent) GetRequestId() string {
	return event.CorrelationId
}

func (event *SEvent) GetRequest() jsonutils.JSONObject {
	return jsonutils.Marshal(event)
}

func (event *SEvent) GetAccount() string {
	return event.Claims.Appid
}

func (event *SEvent) IsSuccess() bool {
	return event.Status.Value != "Failed"
}

func (event *SEvent) GetCreatedAt() time.Time {
	return event.EventTimestamp
}
func (region *SRegion) GetICloudEvents(start time.Time, end time.Time, withReadEvent bool) ([]cloudprovider.ICloudEvent, error) {
	events, err := region.GetEvents(start, end)
	if err != nil {
		return nil, err
	}
	iEvents := []cloudprovider.ICloudEvent{}
	for i := range events {
		read := false
		for _, k := range []string{"read", "listKeys"} {
			if strings.Contains(events[i].Authorization.Action, k) {
				read = true
				break
			}
		}
		if withReadEvent || !read {
			iEvents = append(iEvents, &events[i])
		}
	}
	return iEvents, nil
}

func (region *SRegion) GetEvents(start time.Time, end time.Time) ([]SEvent, error) {
	events := []SEvent{}
	params := url.Values{}
	if start.IsZero() {
		start = time.Now().AddDate(0, 0, -7)
	}
	if end.IsZero() {
		end = time.Now()
	}
	params.Set("$filter", fmt.Sprintf("eventTimestamp ge '%s' and eventTimestamp le '%s' and eventChannels eq 'Admin, Operation' and levels eq 'Critical,Error,Warning,Informational'", start.Format("2006-01-02T15:04:05Z"), end.Format("2006-01-02T15:04:05Z")))
	nextLink := fmt.Sprintf("microsoft.insights/eventtypes/management/values?%s", params.Encode())
	var err error
	for {
		_events := []SEvent{}
		nextLink, err = region.client.ListAllWithNextToken(nextLink, &_events)
		if err != nil {
			return nil, err
		}
		events = append(events, _events...)
		if len(nextLink) > 0 {
			nextLink = nextLink[strings.Index(nextLink, "microsoft.insights"):]
		}
		if len(nextLink) == 0 || len(_events) == 0 {
			break
		}
	}
	return events, nil
}
