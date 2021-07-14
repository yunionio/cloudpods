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

	"github.com/aws/aws-sdk-go/service/organizations"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

/*
 * {"arn":"arn:aws:organizations::285906155448:account/o-vgh74bqhdw/285906155448","email":"swordqiu@gmail.com","id":"285906155448","joined_method":"INVITED","joined_timestamp":"2021-02-09T03:55:27.724000Z","name":"qiu jian","status":"ACTIVE"}
 */
type SAccount struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Arn    string `json:"arn"`
	Email  string `json:"email"`
	Status string `json:"status"`

	JoinedMethod    string    `json:"joined_method"`
	JoinedTimestamp time.Time `json:"joined_timestamp"`

	IsMaster bool `json:"is_master"`
}

/*
 * {
 *   Arn: "arn:aws:organizations::031871565791:policy/o-gn75phg8ge/service_control_policy/p-4l9recev",
 *   AwsManaged: false,
 *   Description: "Create Preventive SCP Guardrails",
 *   Id: "p-4l9recev",
 *   Name: "SCP-PREVENTIVE-GUARDRAILS",
 *   Type: "SERVICE_CONTROL_POLICY"
 * }
 */
type SOrgPolicy struct {
	Arn         string `json:"arn"`
	AwsManaged  bool   `json:"aws_managed"`
	Description string `json:"description"`
	Id          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
}

const (
	SERVICE_CONTROL_POLICY    = "SERVICE_CONTROL_POLICY"
	TAG_POLICY                = "TAG_POLICY"
	BACKUP_POLICY             = "BACKUP_POLICY"
	AISERVICES_OPT_OUT_POLICY = "AISERVICES_OPT_OUT_POLICY"
)

func (r *SRegion) ListPolicies(filter string) ([]SOrgPolicy, error) {
	orgCli, err := r.getOrganizationClient()
	if err != nil {
		return nil, errors.Wrap(err, "GetOrganizationClient")
	}
	var nextToken *string
	policies := make([]SOrgPolicy, 0)

	for {
		input := organizations.ListPoliciesInput{}
		input.SetFilter(filter)
		if nextToken != nil {
			input.SetNextToken(*nextToken)
		}
		parts, err := orgCli.ListPolicies(&input)
		if err != nil {
			return nil, errors.Wrap(err, "ListPolicies")
		}
		for _, pPtr := range parts.Policies {
			p := SOrgPolicy{
				Arn:         *pPtr.Arn,
				AwsManaged:  *pPtr.AwsManaged,
				Description: *pPtr.Description,
				Id:          *pPtr.Id,
				Name:        *pPtr.Name,
				Type:        *pPtr.Type,
			}
			policies = append(policies, p)
		}
		if parts.NextToken == nil || len(*parts.NextToken) == 0 {
			break
		} else {
			nextToken = parts.NextToken
		}
	}
	return policies, nil
}

func (r *SRegion) ListPoliciesForTarget(filter string, targetId string) ([]SOrgPolicy, error) {
	orgCli, err := r.getOrganizationClient()
	if err != nil {
		return nil, errors.Wrap(err, "GetOrganizationClient")
	}
	var nextToken *string
	policies := make([]SOrgPolicy, 0)

	for {
		input := organizations.ListPoliciesForTargetInput{}
		input.SetFilter(filter)
		input.SetTargetId(targetId)
		if nextToken != nil {
			input.SetNextToken(*nextToken)
		}
		parts, err := orgCli.ListPoliciesForTarget(&input)
		if err != nil {
			return nil, errors.Wrap(err, "ListPoliciesForTarget")
		}
		for _, pPtr := range parts.Policies {
			p := SOrgPolicy{
				Arn:         *pPtr.Arn,
				AwsManaged:  *pPtr.AwsManaged,
				Description: *pPtr.Description,
				Id:          *pPtr.Id,
				Name:        *pPtr.Name,
				Type:        *pPtr.Type,
			}
			policies = append(policies, p)
		}
		if parts.NextToken == nil || len(*parts.NextToken) == 0 {
			break
		} else {
			nextToken = parts.NextToken
		}
	}
	return policies, nil
}

func (r *SRegion) DescribeOrgPolicy(pId string) (jsonutils.JSONObject, error) {
	orgCli, err := r.getOrganizationClient()
	if err != nil {
		return nil, errors.Wrap(err, "GetOrganizationClient")
	}
	input := organizations.DescribePolicyInput{}
	input.SetPolicyId(pId)
	output, err := orgCli.DescribePolicy(&input)
	if err != nil {
		return nil, errors.Wrap(err, "DescribePolicy")
	}
	content, err := jsonutils.ParseString(*output.Policy.Content)
	if err != nil {
		return nil, errors.Wrap(err, "ParseJSON")
	}
	return content, nil
}

func (r *SRegion) ListAccounts() ([]SAccount, error) {
	orgCli, err := r.getOrganizationClient()
	if err != nil {
		return nil, errors.Wrap(err, "GetOrganizationClient")
	}
	input := organizations.DescribeOrganizationInput{}
	orgOutput, err := orgCli.DescribeOrganization(&input)
	if err != nil {
		log.Errorf("%#v", err)
		return nil, errors.Wrap(err, "DescribeOrganization")
	}

	var nextToken *string
	accounts := make([]SAccount, 0)
	for {
		input := organizations.ListAccountsInput{}
		if nextToken != nil {
			input.NextToken = nextToken
		}
		parts, err := orgCli.ListAccounts(&input)
		if err != nil {
			return nil, errors.Wrap(err, "ListAccounts")
		}
		for _, actPtr := range parts.Accounts {
			account := SAccount{
				ID:              *actPtr.Id,
				Name:            *actPtr.Name,
				Arn:             *actPtr.Arn,
				Email:           *actPtr.Email,
				Status:          *actPtr.Status,
				JoinedMethod:    *actPtr.JoinedMethod,
				JoinedTimestamp: *actPtr.JoinedTimestamp,
			}
			if *orgOutput.Organization.MasterAccountId == *actPtr.Id {
				account.IsMaster = true
			}
			accounts = append(accounts, account)
		}
		if parts.NextToken == nil || len(*parts.NextToken) == 0 {
			break
		} else {
			nextToken = parts.NextToken
		}
	}
	return accounts, nil
}

func (self *SAwsClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	defRegion, err := self.getDefaultRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "getDefaultRegion")
	}
	accounts, err := defRegion.ListAccounts()
	if err != nil {
		// find errors
		if strings.Contains(err.Error(), "AWSOrganizationsNotInUseException") || strings.Contains(err.Error(), "AccessDeniedException") {
			// permission denied, fall back to single account mode
			subAccount := cloudprovider.SSubAccount{}
			subAccount.Name = self.cpcfg.Name
			subAccount.Account = self.accessKey
			subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
			return []cloudprovider.SSubAccount{subAccount}, nil
		} else {
			return nil, errors.Wrap(err, "ListAccounts")
		}
	} else {
		// check if caller is a root caller
		caller, _ := self.GetCallerIdentity()
		isRootAccount := false
		// arn:aws:iam::285906155448:root
		if caller != nil && strings.HasSuffix(caller.Arn, ":root") {
			log.Debugf("root %s", caller.Arn)
			isRootAccount = true
		}
		subAccounts := []cloudprovider.SSubAccount{}
		for _, account := range accounts {
			subAccount := cloudprovider.SSubAccount{}
			if account.Status == "ACTIVE" {
				subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
			} else {
				subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_SUSPENDED
			}
			if account.IsMaster {
				subAccount.Name = self.cpcfg.Name
				subAccount.Account = self.accessKey
			} else {
				if isRootAccount {
					log.Warningf("Cannot access non-master account with root account!!")
					subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NO_PERMISSION
				}
				subAccount.Name = fmt.Sprintf("%s/%s", account.Name, account.ID)
				subAccount.Account = fmt.Sprintf("%s/%s", self.accessKey, account.ID)
			}
			subAccounts = append(subAccounts, subAccount)
		}
		return subAccounts, nil
	}
}

func (r *SRegion) ListParents(childId string) error {
	orgCli, err := r.getOrganizationClient()
	if err != nil {
		return errors.Wrap(err, "GetOrganizationClient")
	}
	input := organizations.ListParentsInput{}
	input.SetChildId(childId)
	parents, err := orgCli.ListParents(&input)
	if err != nil {
		return errors.Wrap(err, "ListParents")
	}
	log.Debugf("%#v", parents)
	return nil
}

func (r *SRegion) DescribeOrganizationalUnit(ouId string) error {
	orgCli, err := r.getOrganizationClient()
	if err != nil {
		return errors.Wrap(err, "GetOrganizationClient")
	}
	input := organizations.DescribeOrganizationalUnitInput{}
	input.SetOrganizationalUnitId(ouId)
	output, err := orgCli.DescribeOrganizationalUnit(&input)
	if err != nil {
		return errors.Wrap(err, "DescribeOrganizationUnit")
	}
	log.Debugf("%#v", output)
	return nil
}
