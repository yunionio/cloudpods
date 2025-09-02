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
	"context"
	"encoding/json"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SElbListenerRules struct {
	Rules      []SElbListenerRule `xml:"Rules>member"`
	NextMarker string
}

type SElbListenerRule struct {
	multicloud.SResourceBase
	multicloud.SLoadbalancerRedirectBase
	AwsTags
	listener *SElbListener
	region   *SRegion

	Priority      string      `xml:"Priority"`
	IsDefaultRule bool        `xml:"IsDefault"`
	Actions       []Action    `xml:"Actions>member"`
	RuleArn       string      `xml:"RuleArn"`
	Conditions    []Condition `xml:"Conditions>member"`
}

type Action struct {
	TargetGroupArn string `xml:"TargetGroupArn"`
	Type           string `xml:"Type"`
}

type Condition struct {
	Field                   string             `xml:"Field"`
	HTTPRequestMethodConfig *Config            `xml:"HttpRequestMethodConfig"`
	Values                  []string           `xml:"Values>member"`
	SourceIPConfig          *Config            `xml:"SourceIpConfig"`
	QueryStringConfig       *QueryStringConfig `xml:"QueryStringConfig"`
	HTTPHeaderConfig        *HTTPHeaderConfig  `xml:"HttpHeaderConfig"`
	PathPatternConfig       *Config            `xml:"PathPatternConfig"`
	HostHeaderConfig        *Config            `xml:"HostHeaderConfig"`
}

type HTTPHeaderConfig struct {
	HTTPHeaderName string   `xml:"HttpHeaderName"`
	Values         []string `xml:"Values>member"`
}

type Config struct {
	Values []string `xml:"Values>member"`
}

type QueryStringConfig struct {
	Values []Query `xml:"Values>member"`
}

type Query struct {
	Key   string `xml:"key"`
	Value string `xml:"value"`
}

func (self *SElbListenerRule) GetId() string {
	return self.RuleArn
}

func (self *SElbListenerRule) GetName() string {
	segs := strings.Split(self.RuleArn, "/")
	return segs[len(segs)-1]
}

func (self *SElbListenerRule) GetGlobalId() string {
	return self.GetId()
}

func (self *SElbListenerRule) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SElbListenerRule) Refresh() error {
	rule, err := self.region.GetElbListenerRule(self.listener.ListenerArn, self.RuleArn)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, rule)
}

func (self *SElbListenerRule) IsDefault() bool {
	return self.IsDefaultRule
}

func (self *SElbListenerRule) IsEmulated() bool {
	return false
}

func (self *SElbListenerRule) GetProjectId() string {
	return ""
}

func (self *SElbListenerRule) GetDomain() string {
	for _, condition := range self.Conditions {
		if condition.Field == "host-header" {
			return strings.Join(condition.Values, ",")
		}
	}

	return ""
}

func (self *SElbListenerRule) GetCondition() string {
	conditon, err := json.Marshal(self.Conditions)
	if err != nil {
		log.Errorf("GetCondition %s", err)
		return ""
	}

	return string(conditon)
}

func (self *SElbListenerRule) GetPath() string {
	for _, condition := range self.Conditions {
		if condition.Field == "path-pattern" {
			return strings.Join(condition.Values, ",")
		}
	}

	return ""
}

func (self *SElbListenerRule) GetBackendGroupId() string {
	for _, action := range self.Actions {
		if action.Type == "forward" {
			return action.TargetGroupArn
		}
	}

	return ""
}

func (self *SElbListenerRule) Delete(ctx context.Context) error {
	return self.region.DeleteElbListenerRule(self.GetId())
}

func (self *SRegion) DeleteElbListenerRule(id string) error {
	return self.elbRequest("DeleteRule", map[string]string{"RuleArn": id}, nil)
}

func (self *SRegion) CreateElbListenerRule(listenerId string, opts *cloudprovider.SLoadbalancerListenerRule) (*SElbListenerRule, error) {
	params := map[string]string{
		"ListenerArn":                     listenerId,
		"Actions.member.1.Type":           "forward",
		"Actions.member.1.TargetGroupArn": opts.BackendGroupId,
		"Priority":                        "1",
	}
	// TODO
	//condtions, err := parseConditions(config.Condition)
	//if err != nil {
	//	return nil, errors.Wrap(err, "parseConditions")
	//}

	//params.SetConditions(condtions)

	ret := &SElbListenerRules{}
	err := self.elbRequest("CreateRule", params, ret)
	if err != nil {
		return nil, err
	}
	for i := range ret.Rules {
		return &ret.Rules[i], nil
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "after created")
}

func (self *SElbListenerRule) GetDescription() string {
	return self.AwsTags.GetDescription()
}
