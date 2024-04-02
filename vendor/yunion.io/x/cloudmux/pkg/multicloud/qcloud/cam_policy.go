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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SPolicy struct {
	client *SQcloudClient

	PolicyId       int64
	PolicyName     string
	AddTime        time.Time
	CreateMode     string
	PolicyType     string
	Description    string
	PolicyDocument string
}

func (self *SPolicy) GetGlobalId() string {
	return fmt.Sprintf("%d", self.PolicyId)
}

func (self *SPolicy) GetName() string {
	return self.PolicyName
}

func (self *SPolicy) GetPolicyType() api.TPolicyType {
	if self.PolicyType == "User" {
		return api.PolicyTypeCustom
	}
	return api.PolicyTypeSystem
}

func (self *SPolicy) UpdateDocument(document *jsonutils.JSONDict) error {
	return self.client.UpdatePolicy(int(self.PolicyId), document.String(), "")
}

func (self *SPolicy) GetDocument() (*jsonutils.JSONDict, error) {
	if len(self.PolicyDocument) == 0 {
		err := self.Refresh()
		if err != nil {
			return nil, errors.Wrapf(err, "Refesh")
		}
	}
	jsonObj, err := jsonutils.Parse([]byte(self.PolicyDocument))
	if err != nil {
		return nil, errors.Wrapf(err, "jsonutils.Parse")
	}
	return jsonObj.(*jsonutils.JSONDict), nil
}

func (self *SPolicy) Delete() error {
	return self.client.DeletePolicy([]int{int(self.PolicyId)})
}

func (self *SPolicy) Refresh() error {
	p, err := self.client.GetPolicy(self.GetGlobalId())
	if err != nil {
		return errors.Wrapf(err, "GetPolicy(%s)", self.GetGlobalId())
	}
	// p.PolicyId一般为0,若jsonutils.Update会有问题
	self.Description = p.Description
	self.PolicyDocument = p.PolicyDocument
	return nil
}

func (self *SPolicy) GetDescription() string {
	if len(self.Description) == 0 {
		err := self.Refresh()
		if err != nil {
			return ""
		}
	}
	return self.Description
}

func (self *SQcloudClient) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	ret := []cloudprovider.ICloudpolicy{}
	offset := 1
	for {
		part, total, err := self.ListPolicies("", "", offset, 50)
		if err != nil {
			return nil, errors.Wrap(err, "ListPolicies")
		}
		for i := range part {
			part[i].client = self
			ret = append(ret, &part[i])
		}
		if len(ret) >= total {
			break
		}
		offset += 1
	}
	return ret, nil
}

func (self *SQcloudClient) ListAttachedUserPolicies(uin string, offset int, limit int) ([]SPolicy, int, error) {
	if offset < 1 {
		offset = 1
	}
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	params := map[string]string{
		"TargetUin": uin,
		"Page":      fmt.Sprintf("%d", offset),
		"Rp":        fmt.Sprintf("%d", limit),
	}
	resp, err := self.camRequest("ListAttachedUserPolicies", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "camRequest.ListAttachedUserPolicies")
	}
	policies := []SPolicy{}
	err = resp.Unmarshal(&policies, "List")
	if err != nil {
		return nil, 0, errors.Wrap(err, "resp.Unmarshal")
	}
	total, _ := resp.Float("TotalNum")
	return policies, int(total), nil
}

func (self *SQcloudClient) AttachUserPolicy(uin, policyId string) error {
	params := map[string]string{
		"AttachUin": uin,
		"PolicyId":  policyId,
	}
	_, err := self.camRequest("AttachUserPolicy", params)
	return err
}

func (self *SQcloudClient) DetachUserPolicy(uin, policyId string) error {
	params := map[string]string{
		"DetachUin": uin,
		"PolicyId":  policyId,
	}
	_, err := self.camRequest("DetachUserPolicy", params)
	return err
}

// https://cloud.tencent.com/document/api/598/34570
func (self *SQcloudClient) ListPolicies(keyword, scope string, offset int, limit int) ([]SPolicy, int, error) {
	if offset < 1 {
		offset = 1
	}
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	params := map[string]string{
		"Page": fmt.Sprintf("%d", offset),
		"Rp":   fmt.Sprintf("%d", limit),
	}
	if len(scope) > 0 {
		params["Scope"] = scope
	}
	if len(keyword) > 0 {
		params["Keyword"] = keyword
	}
	resp, err := self.camRequest("ListPolicies", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "camRequest.ListPolicies")
	}
	policies := []SPolicy{}
	err = resp.Unmarshal(&policies, "List")
	if err != nil {
		return nil, 0, errors.Wrap(err, "resp.Unmarshal")
	}
	total, _ := resp.Float("TotalNum")
	return policies, int(total), nil
}

func (self *SQcloudClient) GetPolicy(policyId string) (*SPolicy, error) {
	params := map[string]string{
		"PolicyId": policyId,
	}
	resp, err := self.camRequest("GetPolicy", params)
	if err != nil {
		return nil, errors.Wrap(err, "GetPolicy")
	}
	policy := SPolicy{}
	err = resp.Unmarshal(&policy)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return &policy, nil
}

func (self *SQcloudClient) DeletePolicy(policyIds []int) error {
	params := map[string]string{}
	for i, policyId := range policyIds {
		params[fmt.Sprintf("PolicyId.%d", i)] = fmt.Sprintf("%d", policyId)
	}
	_, err := self.camRequest("DeletePolicy", params)
	return err
}

func (self *SQcloudClient) GetPolicyByName(name string) (*SPolicy, error) {
	policies := []SPolicy{}
	offset := 1
	for {
		part, total, err := self.ListPolicies(name, "", offset, 50)
		if err != nil {
			return nil, errors.Wrapf(err, "ListPolicies")
		}
		policies = append(policies, part...)
		if len(policies) >= total {
			break
		}
		offset += 1
	}
	for i := range policies {
		if policies[i].PolicyName == name {
			policies[i].client = self
			return &policies[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "GetPolicyByName(%s)", name)
}

func (self *SQcloudClient) CreatePolicy(name, document, desc string) (*SPolicy, error) {
	params := map[string]string{
		"PolicyName":     name,
		"PolicyDocument": document,
		"Description":    desc,
	}
	_, err := self.camRequest("CreatePolicy", params)
	if err != nil {
		return nil, errors.Wrapf(err, "CreatePolicy")
	}
	return self.GetPolicyByName(name)
}

func (self *SQcloudClient) CreateICloudpolicy(opts *cloudprovider.SCloudpolicyCreateOptions) (cloudprovider.ICloudpolicy, error) {
	policy, err := self.CreatePolicy(opts.Name, opts.Document.String(), opts.Desc)
	if err != nil {
		return nil, errors.Wrapf(err, "CreatePolicy")
	}
	return policy, nil
}

func (self *SQcloudClient) UpdatePolicy(policyId int, document, desc string) error {
	params := map[string]string{
		"PolicyId": fmt.Sprintf("%d", policyId),
	}
	if len(document) > 0 {
		params["PolicyDocument"] = document
	}
	if len(desc) > 0 {
		params["Description"] = desc
	}
	_, err := self.camRequest("UpdatePolicy", params)
	return err
}
