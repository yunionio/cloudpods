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

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

/*
 * {"arn":"arn:aws:organizations::285906155448:account/o-vgh74bqhdw/285906155448","email":"swordqiu@gmail.com","id":"285906155448","joined_method":"INVITED","joined_timestamp":"2021-02-09T03:55:27.724000Z","name":"qiu jian","status":"ACTIVE"}
 */
type SAccount struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Arn   string   `json:"arn"`
	Email string   `json:"email"`
	State string   `json:"state"`
	Path  []string `json:"path"`

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

type sOrgPolicySummary struct {
	Arn         string
	AwsManaged  bool
	Description string
	Id          string
	Name        string
	Type        string
}

type sOrgAccount struct {
	Id              string
	Name            string
	Arn             string
	Email           string
	State           string
	Paths           []string
	JoinedMethod    string
	JoinedTimestamp float64
}

type sOrganization struct {
	MasterAccountId string
}

type sOrganizationalUnit struct {
	Name string
}

func (r *SRegion) orgRequest(apiName string, params map[string]interface{}, retval interface{}) error {
	return r.client.orgRequest(apiName, params, retval)
}

func (r *SRegion) ListPolicies(filter string) ([]SOrgPolicy, error) {
	params := map[string]interface{}{
		"Filter": filter,
	}
	policies := make([]SOrgPolicy, 0)
	for {
		ret := struct {
			Policies  []sOrgPolicySummary
			NextToken string
		}{}
		err := r.orgRequest("ListPolicies", params, &ret)
		if err != nil {
			return nil, errors.Wrap(err, "ListPolicies")
		}
		for _, p := range ret.Policies {
			policies = append(policies, SOrgPolicy{
				Arn:         p.Arn,
				AwsManaged:  p.AwsManaged,
				Description: p.Description,
				Id:          p.Id,
				Name:        p.Name,
				Type:        p.Type,
			})
		}
		if len(ret.NextToken) == 0 {
			break
		}
		params["NextToken"] = ret.NextToken
	}
	return policies, nil
}

func (r *SRegion) ListPoliciesForTarget(filter string, targetId string) ([]SOrgPolicy, error) {
	params := map[string]interface{}{
		"Filter":   filter,
		"TargetId": targetId,
	}
	policies := make([]SOrgPolicy, 0)
	for {
		ret := struct {
			Policies  []sOrgPolicySummary
			NextToken string
		}{}
		err := r.orgRequest("ListPoliciesForTarget", params, &ret)
		if err != nil {
			return nil, errors.Wrap(err, "ListPoliciesForTarget")
		}
		for _, p := range ret.Policies {
			policies = append(policies, SOrgPolicy{
				Arn:         p.Arn,
				AwsManaged:  p.AwsManaged,
				Description: p.Description,
				Id:          p.Id,
				Name:        p.Name,
				Type:        p.Type,
			})
		}
		if len(ret.NextToken) == 0 {
			break
		}
		params["NextToken"] = ret.NextToken
	}
	return policies, nil
}

func (r *SRegion) DescribeOrgPolicy(pId string) (jsonutils.JSONObject, error) {
	params := map[string]interface{}{
		"PolicyId": pId,
	}
	ret := struct {
		Policy struct {
			Content string
		}
	}{}
	err := r.orgRequest("DescribePolicy", params, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "DescribePolicy")
	}
	content, err := jsonutils.ParseString(ret.Policy.Content)
	if err != nil {
		return nil, errors.Wrap(err, "ParseJSON")
	}
	return content, nil
}

func (r *SRegion) ListAccounts() ([]SAccount, error) {
	orgRet := struct {
		Organization sOrganization
	}{}
	err := r.orgRequest("DescribeOrganization", map[string]interface{}{}, &orgRet)
	if err != nil {
		log.Errorf("%#v", err)
		return nil, errors.Wrap(err, "DescribeOrganization")
	}

	params := map[string]interface{}{}
	accounts := make([]SAccount, 0)
	for {
		part := struct {
			Accounts  []sOrgAccount
			NextToken string
		}{}
		err := r.orgRequest("ListAccounts", params, &part)
		if err != nil {
			return nil, errors.Wrap(err, "ListAccounts")
		}
		for _, act := range part.Accounts {
			account := SAccount{
				ID:              act.Id,
				Name:            act.Id,
				Arn:             act.Arn,
				Email:           act.Email,
				State:           act.State,
				JoinedMethod:    act.JoinedMethod,
				JoinedTimestamp: time.Unix(0, int64(act.JoinedTimestamp*float64(time.Second))),
				Path:            act.Paths,
			}
			if len(act.Name) > 0 {
				account.Name = act.Name
			}
			if orgRet.Organization.MasterAccountId == act.Id {
				account.IsMaster = true
			}
			accounts = append(accounts, account)
		}
		if len(part.NextToken) == 0 {
			break
		}
		params["NextToken"] = part.NextToken
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

		// Best-effort: fill Tags by AWS Organizations OU hierarchy.
		// Azure uses a parent display-name chain to build L1/L2/L3/... tags.
		// Here we use SAccount.Path traversal to build L1/L2/L3/... tags.
		// For OU IDs in path (prefix "ou-"), we also best-effort resolve OU name via DescribeOrganizationalUnit.
		tagsByAccount := map[string]map[string]string{}
		ouNameCache := map[string]string{}
		for _, account := range accounts {
			tags, tErr := getAccountPathTags(awscli, account.Path, ouNameCache)
			if tErr != nil {
				log.Debugf("getAccountPathTags %s: %v", account.ID, tErr)
				continue
			}
			if len(tags) > 0 {
				tagsByAccount[account.ID] = tags
			}
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
				if len(account.Name) > 0 && account.Name != account.ID {
					subAccount.Name = fmt.Sprintf("%s/%s", account.Name, account.ID)
				}
				subAccount.Account = fmt.Sprintf("%s/%s", awscli.accessKey, account.ID)
				subAccount.Id = account.ID
			}

			if tags, ok := tagsByAccount[account.ID]; ok {
				subAccount.Tags = tags
			}

			subAccounts = append(subAccounts, subAccount)
		}
		return subAccounts, nil
	}
}

func getAccountPathTags(client *SAwsClient, paths []string, ouNameCache map[string]string) (map[string]string, error) {
	// AWS Organizations account "Paths" can be:
	// - multiple path strings (account may exist in multiple hierarchies)
	// - each path string could be a concatenation like "r-xxxx/ou-yyyy/ou-zzzz"
	// - or, in some cases, already a segment list-like format.
	//
	// We iterate through each path string and build tags from the first usable one.
	for _, pathStr := range paths {
		if len(pathStr) == 0 {
			continue
		}

		// If the path string is a concatenation, split by "/".
		if strings.Contains(pathStr, "/") {
			segs := strings.Split(pathStr, "/")
			ouIds := make([]string, 0, len(segs))
			tagVals := make([]string, 0, len(segs))
			for _, seg := range segs {
				seg = strings.TrimSpace(seg)
				if len(seg) == 0 {
					continue
				}
				if strings.HasPrefix(seg, "ou-") {
					ouIds = append(ouIds, seg)
				} else if strings.HasPrefix(seg, "r-") {
					// skip root ID
					continue
				} else if strings.HasPrefix(seg, "o-") {
					// skip organization ID segment (AWS Organizations "o-xxxxx")
					continue
				} else {
					// Skip account id segments (some Organizations APIs expose them as path segments),
					// otherwise we'd incorrectly generate Lx tags from account id.
					if isAWSAccountID(seg) {
						continue
					}
					// fallback: treat it as a value directly
					tagVals = append(tagVals, seg)
				}
			}

			if len(ouIds) > 0 {
				tags, err := getOuIdsTags(client, ouIds, ouNameCache)
				if err != nil {
					return nil, err
				}
				if len(tags) > 0 {
					return tags, nil
				}
				continue
			}

			if len(tagVals) > 0 {
				tags := map[string]string{}
				for idx, v := range tagVals {
					if len(v) == 0 {
						continue
					}
					tags[fmt.Sprintf("L%d", idx+1)] = v
				}
				if len(tags) > 0 {
					return tags, nil
				}
			}
			continue
		}

		// Non-concatenated single segment.
		if strings.HasPrefix(pathStr, "ou-") {
			tags, err := getOuIdsTags(client, []string{pathStr}, ouNameCache)
			if err != nil {
				return nil, err
			}
			if len(tags) > 0 {
				return tags, nil
			}
		} else if strings.HasPrefix(pathStr, "o-") {
			// organization root id alone should not produce Lx tags
			return map[string]string{}, nil
		} else {
			return map[string]string{"L1": pathStr}, nil
		}
	}

	return map[string]string{}, nil
}

func getOuIdsTags(client *SAwsClient, ouIds []string, ouNameCache map[string]string) (map[string]string, error) {
	tags := map[string]string{}
	level := 1
	for _, ouId := range ouIds {
		ouId = strings.TrimSpace(ouId)
		if len(ouId) == 0 {
			continue
		}

		ouName, ok := ouNameCache[ouId]
		if !ok {
			params := map[string]interface{}{
				"OrganizationalUnitId": ouId,
			}
			ret := struct {
				OrganizationalUnit sOrganizationalUnit
			}{}
			descErr := client.orgRequest("DescribeOrganizationalUnit", params, &ret)
			if descErr != nil {
				return nil, errors.Wrap(descErr, "DescribeOrganizationalUnit")
			}
			ouName = ret.OrganizationalUnit.Name
			ouNameCache[ouId] = ouName
		}

		// If OU name is empty, still keep the ID to avoid missing level ordering.
		if len(ouName) == 0 {
			ouName = ouId
		}
		tags[fmt.Sprintf("L%d", level)] = ouName
		level++
	}

	return tags, nil
}

func isAWSAccountID(s string) bool {
	if len(s) != 12 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (r *SRegion) ListParents(childId string) error {
	params := map[string]interface{}{
		"ChildId": childId,
	}
	ret := struct {
		Parents []interface{}
	}{}
	err := r.orgRequest("ListParents", params, &ret)
	if err != nil {
		return errors.Wrap(err, "ListParents")
	}
	log.Debugf("%#v", ret)
	return nil
}

func (r *SRegion) DescribeOrganizationalUnit(ouId string) error {
	params := map[string]interface{}{
		"OrganizationalUnitId": ouId,
	}
	ret := struct {
		OrganizationalUnit sOrganizationalUnit
	}{}
	err := r.orgRequest("DescribeOrganizationalUnit", params, &ret)
	if err != nil {
		return errors.Wrap(err, "DescribeOrganizationUnit")
	}
	log.Debugf("%#v", ret)
	return nil
}
