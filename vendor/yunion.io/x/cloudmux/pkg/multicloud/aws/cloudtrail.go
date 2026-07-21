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

package aws

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SEventResource struct {
	// The name of the resource referenced by the event returned. These are user-created
	// names whose values will depend on the environment. For example, the resource
	// name might be "auto-scaling-test-group" for an Auto Scaling Group or "i-1234567"
	// for an EC2 Instance.
	ResourceName string `type:"string"`

	// The type of a resource referenced by the event returned. When the resource
	// type cannot be determined, null is returned. Some examples of resource types
	// are: Instance for EC2, Trail for CloudTrail, DBInstance for RDS, and AccessKey
	// for IAM. To learn more about how to look up and filter events by the resource
	// types supported for a service, see Filtering CloudTrail Events (https://docs.aws.amazon.com/awscloudtrail/latest/userguide/view-cloudtrail-events-console.html#filtering-cloudtrail-events).
	ResourceType string `type:"string"`
}

type SEvent struct {
	// The AWS access key ID that was used to sign the request. If the request was
	// made with temporary security credentials, this is the access key ID of the
	// temporary credentials.
	AccessKeyId string `type:"string"`

	// A JSON string that contains a representation of the event returned.
	CloudTrailEvent string `type:"string"`

	// The CloudTrail ID of the event returned.
	EventId string `type:"string"`

	// The name of the event returned.
	EventName string `type:"string"`

	// The AWS service that the request was made to.
	EventSource string `type:"string"`

	// The date and time of the event returned.
	EventTime time.Time `type:"timestamp"`

	// Information about whether the event is a write event or a read event.
	ReadOnly string `type:"string"`

	// A list of resources referenced by the event returned.
	Resources []SEventResource `type:"list"`

	// A user name or role name of the requester that called the API in the event
	// returned.
	Username string `type:"string"`
}

type sCloudTrailEvent struct {
	AccessKeyId     string
	CloudTrailEvent string
	EventId         string
	EventName       string
	EventSource     string
	EventTime       float64
	ReadOnly        string
	Resources       []SEventResource
	Username        string
}

func (self *SEvent) GetName() string {
	return self.EventName
}

func (self *SEvent) GetService() string {
	return self.EventSource
}

func (self *SEvent) GetAction() string {
	return self.EventName
}

func (self *SEvent) GetResourceType() string {
	return self.EventSource
}

func (self *SEvent) GetRequestId() string {
	return self.EventId
}

func (self *SEvent) GetRequest() jsonutils.JSONObject {
	obj, _ := jsonutils.Parse([]byte(self.CloudTrailEvent))
	return obj
}

func (self *SEvent) GetAccount() string {
	return fmt.Sprintf("%s(%s)", self.AccessKeyId, self.Username)
}

func (self *SEvent) IsSuccess() bool {
	return !strings.Contains(self.CloudTrailEvent, "errorMessage")
}

func (self *SEvent) GetCreatedAt() time.Time {
	return self.EventTime
}

func cloudTrailUnixTime(t time.Time) float64 {
	return float64(t.UnixNano()) / float64(time.Second)
}

func (self *SRegion) LookupEvents(start, end time.Time, withReadEvent bool) ([]SEvent, error) {
	params := map[string]interface{}{}
	if !start.IsZero() {
		params["StartTime"] = cloudTrailUnixTime(start)
	}
	if !end.IsZero() {
		params["EndTime"] = cloudTrailUnixTime(end)
	}
	if !withReadEvent {
		params["LookupAttributes"] = []map[string]string{
			{
				"AttributeKey":   "ReadOnly",
				"AttributeValue": "false",
			},
		}
	}

	events := []SEvent{}
	for {
		ret := struct {
			Events    []sCloudTrailEvent
			NextToken string
		}{}
		var err error
		for {
			err = self.cloudtrailRequest("LookupEvents", params, &ret)
			if err != nil {
				if strings.Contains(err.Error(), "ThrottlingException") {
					log.Warningf("LookupEvents ThrottlingException, try after 3 seconds")
					time.Sleep(time.Second * 3)
					continue
				}
				return nil, errors.Wrapf(err, "LookupEvents(%s, %s)", start, end)
			}
			break
		}
		for _, evt := range ret.Events {
			event := SEvent{
				AccessKeyId:     evt.AccessKeyId,
				CloudTrailEvent: evt.CloudTrailEvent,
				EventId:         evt.EventId,
				EventName:       evt.EventName,
				EventSource:     evt.EventSource,
				EventTime:       time.Unix(0, int64(evt.EventTime*float64(time.Second))),
				ReadOnly:        evt.ReadOnly,
				Resources:       evt.Resources,
				Username:        evt.Username,
			}
			if strings.Contains(event.CloudTrailEvent, "awsRegion") && !strings.Contains(event.CloudTrailEvent, fmt.Sprintf(`"awsRegion":"%s"`, self.RegionId)) {
				continue
			}
			events = append(events, event)
		}
		if len(ret.NextToken) == 0 {
			break
		}
		params["NextToken"] = ret.NextToken
	}
	return events, nil
}

func (self *SRegion) GetICloudEvents(start time.Time, end time.Time, withReadEvent bool) ([]cloudprovider.ICloudEvent, error) {
	events, err := self.LookupEvents(start, end, withReadEvent)
	if err != nil {
		return nil, errors.Wrapf(err, "LookupEvents(%s, %s)", start, end)
	}
	ret := []cloudprovider.ICloudEvent{}
	for i := range events {
		ret = append(ret, &events[i])
	}
	return ret, nil
}
