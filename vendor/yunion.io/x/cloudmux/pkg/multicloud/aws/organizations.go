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
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

/*
 * {"arn":"arn:aws:organizations::285906155448:account/o-vgh74bqhdw/285906155448","email":"swordqiu@gmail.com","id":"285906155448","joined_method":"INVITED","joined_timestamp":"2021-02-09T03:55:27.724000Z","name":"qiu jian","status":"ACTIVE"}
 */
type SAccount struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Arn   string `json:"arn"`
	Email string `json:"email"`
	State string `json:"state"`

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
	cfg, err := r.getConfig()
	if err != nil {
		return nil, errors.Wrap(err, "getConfig")
	}
	orgCli := organizations.NewFromConfig(cfg)
	var nextToken *string
	policies := make([]SOrgPolicy, 0)

	for {
		input := organizations.ListPoliciesInput{}
		input.Filter = types.PolicyType(filter)
		if nextToken != nil {
			input.NextToken = nextToken
		}
		parts, err := orgCli.ListPolicies(context.Background(), &input)
		if err != nil {
			return nil, errors.Wrap(err, "ListPolicies")
		}
		for _, pPtr := range parts.Policies {
			p := SOrgPolicy{
				Arn:         *pPtr.Arn,
				AwsManaged:  pPtr.AwsManaged,
				Description: *pPtr.Description,
				Id:          *pPtr.Id,
				Name:        *pPtr.Name,
				Type:        string(pPtr.Type),
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
	cfg, err := r.getConfig()
	if err != nil {
		return nil, errors.Wrap(err, "getConfig")
	}
	orgCli := organizations.NewFromConfig(cfg)
	var nextToken *string
	policies := make([]SOrgPolicy, 0)

	for {
		input := organizations.ListPoliciesForTargetInput{}
		input.Filter = types.PolicyType(filter)
		input.TargetId = &targetId
		if nextToken != nil {
			input.NextToken = nextToken
		}
		parts, err := orgCli.ListPoliciesForTarget(context.Background(), &input)
		if err != nil {
			return nil, errors.Wrap(err, "ListPoliciesForTarget")
		}
		for _, pPtr := range parts.Policies {
			p := SOrgPolicy{
				Arn:         *pPtr.Arn,
				AwsManaged:  pPtr.AwsManaged,
				Description: *pPtr.Description,
				Id:          *pPtr.Id,
				Name:        *pPtr.Name,
				Type:        string(pPtr.Type),
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
	cfg, err := r.getConfig()
	if err != nil {
		return nil, errors.Wrap(err, "getConfig")
	}
	orgCli := organizations.NewFromConfig(cfg)
	input := organizations.DescribePolicyInput{}
	input.PolicyId = &pId
	output, err := orgCli.DescribePolicy(context.Background(), &input)
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
	cfg, err := r.getConfig()
	if err != nil {
		return nil, errors.Wrap(err, "getConfig")
	}
	orgCli := organizations.NewFromConfig(cfg)
	input := organizations.DescribeOrganizationInput{}
	orgOutput, err := orgCli.DescribeOrganization(context.Background(), &input)
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
		parts, err := orgCli.ListAccounts(context.Background(), &input)
		if err != nil {
			return nil, errors.Wrap(err, "ListAccounts")
		}
		for _, actPtr := range parts.Accounts {
			account := SAccount{
				ID:              *actPtr.Id,
				Name:            *actPtr.Id,
				Arn:             *actPtr.Arn,
				Email:           *actPtr.Email,
				State:           string(actPtr.State),
				JoinedMethod:    string(actPtr.JoinedMethod),
				JoinedTimestamp: *actPtr.JoinedTimestamp,
			}
			if actPtr.Name != nil && len(*actPtr.Name) > 0 {
				account.Name = *actPtr.Name
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

func (awscli *SAwsClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	defRegion, err := awscli.getDefaultRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "getDefaultRegion")
	}
	accounts, err := defRegion.ListAccounts()
	if err != nil {
		// find errors
		if strings.Contains(err.Error(), "AWSOrganizationsNotInUseException") || strings.Contains(err.Error(), "AccessDeniedException") {
			// permission denied, fall back to single account mode
			subAccount := cloudprovider.SSubAccount{}
			subAccount.Name = awscli.cpcfg.Name
			subAccount.Account = awscli.accessKey
			subAccount.Id = awscli.accountId
			if len(subAccount.Id) == 0 {
				subAccount.Id = awscli.GetAccountId()
			}
			subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
			return []cloudprovider.SSubAccount{subAccount}, nil
		} else {
			return nil, errors.Wrap(err, "ListAccounts")
		}
	} else {
		// check if caller is a root caller
		caller, _ := awscli.GetCallerIdentity()
		isRootAccount := false
		// arn:aws:iam::285906155448:root
		if caller != nil && strings.HasSuffix(caller.Arn, ":root") {
			log.Debugf("root %s", caller.Arn)
			isRootAccount = true
		}
		subAccounts := []cloudprovider.SSubAccount{}
		for _, account := range accounts {
			subAccount := cloudprovider.SSubAccount{}
			switch account.State {
			case "ACTIVE":
				subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
			case "PENDING_ACTIVATION":
				subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_PENDING
			default:
				subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_SUSPENDED
			}
			if account.IsMaster {
				subAccount.Name = fmt.Sprintf("%s/%s", account.Name, awscli.cpcfg.Name)
				subAccount.Account = awscli.accessKey
				subAccount.Id = account.ID
			} else {
				if isRootAccount {
					log.Warningf("Cannot access non-master account with root account!!")
					subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NO_PERMISSION
				}
				subAccount.Name = account.ID
				if len(account.Name) > 0 {
					subAccount.Name = fmt.Sprintf("%s/%s", account.Name, account.ID)
				}
				subAccount.Account = fmt.Sprintf("%s/%s", awscli.accessKey, account.ID)
				subAccount.Id = account.ID
			}
			subAccounts = append(subAccounts, subAccount)
		}
		return subAccounts, nil
	}
}

func (r *SRegion) ListParents(childId string) error {
	cfg, err := r.getConfig()
	if err != nil {
		return errors.Wrap(err, "getConfig")
	}
	orgCli := organizations.NewFromConfig(cfg)
	input := organizations.ListParentsInput{}
	input.ChildId = &childId
	parents, err := orgCli.ListParents(context.Background(), &input)
	if err != nil {
		return errors.Wrap(err, "ListParents")
	}
	log.Debugf("%#v", parents)
	return nil
}

func (r *SRegion) DescribeOrganizationalUnit(ouId string) error {
	cfg, err := r.getConfig()
	if err != nil {
		return errors.Wrap(err, "getConfig")
	}
	orgCli := organizations.NewFromConfig(cfg)
	input := organizations.DescribeOrganizationalUnitInput{}
	input.OrganizationalUnitId = &ouId
	output, err := orgCli.DescribeOrganizationalUnit(context.Background(), &input)
	if err != nil {
		return errors.Wrap(err, "DescribeOrganizationUnit")
	}
	log.Debugf("%#v", output)
	return nil
}
