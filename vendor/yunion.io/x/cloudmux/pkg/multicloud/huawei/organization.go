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
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

// https://support.huaweicloud.com/api-organizations/ShowOrganization.html
type SOrganization struct {
	Id                    string
	Urn                   string
	ManagementAccountId   string
	ManagementAccountName string
	CreatedAt             time.Time
}

// https://support.huaweicloud.com/api-organizations/ListAccounts.html
type SOrganizationAccount struct {
	Id          string
	Urn         string
	Name        string
	Status      string
	JoinMethod  string
	JoinedAt    time.Time
	Description string
	Email       string
}

// https://support.huaweicloud.com/api-organizations/ListRoots.html
type SOrganizationRoot struct {
	Id        string
	Urn       string
	Name      string
	CreatedAt time.Time
}

// https://support.huaweicloud.com/api-organizations/ShowOrganizationalUnit.html
type SOrganizationalUnit struct {
	Id        string
	Urn       string
	Name      string
	CreatedAt time.Time
}

// https://support.huaweicloud.com/api-organizations/ListEntities.html
type SOrganizationEntity struct {
	Id   string
	Name string
	Type string
}

type sOrgPageInfo struct {
	NextMarker   string
	CurrentCount int
}

type sAgencyCredential struct {
	ExpiresAt     time.Time `json:"expires_at"`
	Access        string    `json:"access"`
	Secret        string    `json:"secret"`
	SecurityToken string    `json:"securitytoken"`
}

func (self *SHuaweiClient) forEachOrgPage(resource string, query url.Values, consume func(resp jsonutils.JSONObject) (int, string, error)) error {
	if query == nil {
		query = url.Values{}
	}
	if len(query.Get("limit")) == 0 {
		query.Set("limit", "1000")
	}
	for {
		resp, err := self.list(SERVICE_ORGANIZATIONS, "", resource, query)
		if err != nil {
			return err
		}
		n, marker, err := consume(resp)
		if err != nil {
			return err
		}
		marker = strings.TrimSpace(marker)
		if n == 0 || len(marker) == 0 || marker == "null" || marker == query.Get("marker") {
			break
		}
		query.Set("marker", marker)
	}
	return nil
}

func (self *SHuaweiClient) ShowOrganization() (*SOrganization, error) {
	resp, err := self.list(SERVICE_ORGANIZATIONS, "", "organizations", nil)
	if err != nil {
		return nil, err
	}
	ret := struct {
		Organization SOrganization
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return &ret.Organization, nil
}

func (self *SHuaweiClient) ListAccounts() ([]SOrganizationAccount, error) {
	query := url.Values{}
	query.Set("limit", "1000")
	ret := make([]SOrganizationAccount, 0)
	err := self.forEachOrgPage("organizations/accounts", query, func(resp jsonutils.JSONObject) (int, string, error) {
		part := struct {
			Accounts []SOrganizationAccount
			PageInfo sOrgPageInfo
		}{}
		err := resp.Unmarshal(&part)
		if err != nil {
			return 0, "", errors.Wrap(err, "Unmarshal")
		}
		ret = append(ret, part.Accounts...)
		return len(part.Accounts), part.PageInfo.NextMarker, nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SHuaweiClient) ListRoots() ([]SOrganizationRoot, error) {
	query := url.Values{}
	query.Set("limit", "1000")
	ret := make([]SOrganizationRoot, 0)
	err := self.forEachOrgPage("organizations/roots", query, func(resp jsonutils.JSONObject) (int, string, error) {
		part := struct {
			Roots    []SOrganizationRoot
			PageInfo sOrgPageInfo
		}{}
		err := resp.Unmarshal(&part)
		if err != nil {
			return 0, "", errors.Wrap(err, "Unmarshal")
		}
		ret = append(ret, part.Roots...)
		return len(part.Roots), part.PageInfo.NextMarker, nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SHuaweiClient) ListEntities(parentId string) ([]SOrganizationEntity, error) {
	query := url.Values{}
	query.Set("limit", "1000")
	if len(parentId) > 0 {
		query.Set("parent_id", parentId)
	}
	ret := make([]SOrganizationEntity, 0)
	err := self.forEachOrgPage("organizations/entities", query, func(resp jsonutils.JSONObject) (int, string, error) {
		part := struct {
			Entities []SOrganizationEntity
			PageInfo sOrgPageInfo
		}{}
		err := resp.Unmarshal(&part)
		if err != nil {
			return 0, "", errors.Wrap(err, "Unmarshal")
		}
		ret = append(ret, part.Entities...)
		return len(part.Entities), part.PageInfo.NextMarker, nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SHuaweiClient) ShowOrganizationalUnit(ouId string) (*SOrganizationalUnit, error) {
	resp, err := self.list(SERVICE_ORGANIZATIONS, "", "organizations/organizational-units/"+ouId, nil)
	if err != nil {
		return nil, err
	}
	ret := struct {
		OrganizationalUnit SOrganizationalUnit
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return &ret.OrganizationalUnit, nil
}

func organizationAccountHealth(status string) string {
	switch strings.ToLower(status) {
	case "active":
		return api.CLOUD_PROVIDER_HEALTH_NORMAL
	case "pending_closure":
		return api.CLOUD_PROVIDER_HEALTH_PENDING
	default:
		return api.CLOUD_PROVIDER_HEALTH_SUSPENDED
	}
}

func ouPathTags(path []string) map[string]string {
	if len(path) == 0 {
		return nil
	}
	tags := make(map[string]string, len(path))
	for i, name := range path {
		if len(name) == 0 {
			continue
		}
		tags[fmt.Sprintf("L%d", i+1)] = name
	}
	if len(tags) == 0 {
		return nil
	}
	return tags
}

// 未开通组织、或当前账号不是管理账号/委托管理员时，退回单账号模式。
func isOrgUnavailable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Cause(err) == cloudprovider.ErrNotFound {
		return true
	}
	if e, ok := errors.Cause(err).(*sHuaweiError); ok {
		switch e.ErrorCode {
		case "Organizations.1100", "Organizations.1001", "Organizations.1002", "Organizations.1008":
			return true
		}
		if e.StatusCode == http.StatusUnauthorized || e.StatusCode == http.StatusForbidden {
			return true
		}
	}
	msg := err.Error()
	for _, code := range []string{"Organizations.1100", "Organizations.1001", "Organizations.1002", "Organizations.1008"} {
		if strings.Contains(msg, code) {
			return true
		}
	}
	return false
}

func (self *SHuaweiClient) accountOuTags() map[string]map[string]string {
	roots, err := self.ListRoots()
	if err != nil {
		log.Debugf("ListRoots: %v", err)
		return nil
	}
	tags := map[string]map[string]string{}
	visited := map[string]struct{}{}
	var walk func(parentId string, path []string)
	walk = func(parentId string, path []string) {
		if len(parentId) == 0 {
			return
		}
		if _, ok := visited[parentId]; ok {
			return
		}
		visited[parentId] = struct{}{}
		entities, err := self.ListEntities(parentId)
		if err != nil {
			log.Debugf("ListEntities %s: %v", parentId, err)
			return
		}
		for _, ent := range entities {
			switch strings.ToLower(ent.Type) {
			case "account":
				if t := ouPathTags(path); len(t) > 0 {
					tags[ent.Id] = t
				}
			case "organizational_unit":
				name := ent.Name
				if len(name) == 0 {
					name = ent.Id
				}
				next := make([]string, len(path)+1)
				copy(next, path)
				next[len(path)] = name
				walk(ent.Id, next)
			}
		}
	}
	for _, root := range roots {
		walk(root.Id, nil)
	}
	return tags
}

func (self *SHuaweiClient) singleSubAccount() cloudprovider.SSubAccount {
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Id = self.GetAccountId()
	subAccount.Name = self.cpcfg.Name
	subAccount.Account = self.accessKey
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	subAccount.DefaultProjectId = "0"
	return subAccount
}

func (self *SHuaweiClient) organizationSubAccount(account SOrganizationAccount, currentId string, tags map[string]string) cloudprovider.SSubAccount {
	subAccount := cloudprovider.SSubAccount{}
	subAccount.HealthStatus = organizationAccountHealth(account.Status)
	subAccount.DefaultProjectId = "0"
	subAccount.Desc = account.Description
	display := account.Name
	if len(display) == 0 {
		display = account.Id
	}
	subAccount.Name = display
	// 当前凭证所属账号直接使用原始 AK，其余成员账号用 AK/账号ID，后续通过委托访问。
	if account.Id == currentId {
		subAccount.Account = self.accessKey
		subAccount.Id = account.Id
	} else {
		subAccount.Account = fmt.Sprintf("%s/%s", self.accessKey, account.Id)
		subAccount.Id = account.Id
	}
	if len(tags) > 0 {
		subAccount.Tags = tags
	}
	return subAccount
}

func (self *SHuaweiClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	current := self.singleSubAccount()
	accounts, err := self.ListAccounts()
	if err != nil {
		if isOrgUnavailable(err) {
			log.Warningf("huawei organizations unavailable, fallback to single account: %v", err)
			return []cloudprovider.SSubAccount{current}, nil
		}
		return nil, errors.Wrap(err, "ListAccounts")
	}

	tagsByAccount := self.accountOuTags()
	currentId := self.GetAccountId()
	ret := make([]cloudprovider.SSubAccount, 0, len(accounts))
	foundSelf := false
	for _, account := range accounts {
		if account.Id == currentId {
			foundSelf = true
		}
		var tags map[string]string
		if tagsByAccount != nil {
			tags = tagsByAccount[account.Id]
		}
		ret = append(ret, self.organizationSubAccount(account, currentId, tags))
	}
	if !foundSelf {
		ret = append([]cloudprovider.SSubAccount{current}, ret...)
	}
	return ret, nil
}

func (self *SHuaweiClient) tempCredValidLocked() bool {
	if self.tempCred == nil || len(self.tempCred.Access) == 0 || len(self.tempCred.SecurityToken) == 0 {
		return false
	}
	return time.Now().Add(5 * time.Minute).Before(self.tempCred.ExpiresAt)
}

func (self *SHuaweiClient) effectiveCredential() (ak, sk, token string) {
	self.credLock.Lock()
	defer self.credLock.Unlock()
	if self.tempCred != nil && len(self.tempCred.Access) > 0 {
		return self.tempCred.Access, self.tempCred.Secret, self.tempCred.SecurityToken
	}
	return self.accessKey, self.accessSecret, ""
}

func (self *SHuaweiClient) ensureAgency() error {
	if len(self.accountId) == 0 {
		return nil
	}
	self.credLock.Lock()
	defer self.credLock.Unlock()
	if self.tempCredValidLocked() {
		return nil
	}
	return self.refreshAgencyLocked()
}

func agencyUrn(accountId, agencyName string) string {
	return fmt.Sprintf("iam::%s:agency:%s", accountId, agencyName)
}

func (self *SHuaweiClient) refreshAgencyLocked() error {
	// 组织里的账号访问委托是信任委托，用 STS AssumeAgency 指定成员账号 URN。
	// 旧的 securitytokens 只校验调用方账号上的 iam:assume，成员账号委托会 403。
	stsErr := self.assumeAgencyByStsLocked()
	if stsErr == nil {
		return nil
	}
	iamErr := self.assumeAgencyByIamLocked()
	if iamErr == nil {
		return nil
	}
	return errors.Wrapf(stsErr, "iam assume: %s", iamErr)
}

func (self *SHuaweiClient) signedJsonPost(rawURL string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	req := httputils.NewJsonRequest(httputils.POST, rawURL, jsonutils.Marshal(params))
	client := &akClient{
		client: self.cpcfg.AdaptiveTimeoutHttpClient(),
		aksk: Signer{
			Key:    self.accessKey,
			Secret: self.accessSecret,
		},
	}
	hwErr := &sHuaweiError{}
	_, resp, err := httputils.NewJsonClient(client).Send(context.Background(), req, hwErr, self.debug)
	return resp, err
}

func (self *SHuaweiClient) storeAgencyCredential(cred sAgencyCredential) error {
	if len(cred.Access) == 0 || len(cred.SecurityToken) == 0 {
		return errors.Errorf("empty agency credential")
	}
	if cred.ExpiresAt.IsZero() {
		cred.ExpiresAt = time.Now().Add(time.Duration(assumeAgencyDurationSeconds) * time.Second)
	}
	self.tempCred = &cred
	return nil
}

// https://support.huaweicloud.com/api-iam5/AssumeAgency.html
func (self *SHuaweiClient) assumeAgencyByStsLocked() error {
	urn := agencyUrn(self.accountId, self.getAssumeAgencyName())
	params := map[string]interface{}{
		"duration_seconds":    assumeAgencyDurationSeconds,
		"agency_urn":          urn,
		"agency_session_name": "cloudmux",
	}
	rawURL := fmt.Sprintf("https://sts.%s.myhuaweicloud.com/v5/agencies/assume", HUAWEI_DEFAULT_REGION)
	resp, err := self.signedJsonPost(rawURL, params)
	if err != nil {
		return errors.Wrapf(err, "assume %s", urn)
	}
	ret := struct {
		Credentials struct {
			AccessKeyId     string
			SecretAccessKey string
			SecurityToken   string
			Expiration      time.Time
		}
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return errors.Wrap(err, "Unmarshal")
	}
	return self.storeAgencyCredential(sAgencyCredential{
		Access:        ret.Credentials.AccessKeyId,
		Secret:        ret.Credentials.SecretAccessKey,
		SecurityToken: ret.Credentials.SecurityToken,
		ExpiresAt:     ret.Credentials.Expiration,
	})
}

// https://support.huaweicloud.com/api-iam/iam_04_0101.html
func (self *SHuaweiClient) assumeAgencyByIamLocked() error {
	params := map[string]interface{}{
		"auth": map[string]interface{}{
			"identity": map[string]interface{}{
				"methods": []string{"assume_role"},
				"assume_role": map[string]interface{}{
					"agency_name":      self.getAssumeAgencyName(),
					"domain_id":        self.accountId,
					"duration_seconds": assumeAgencyDurationSeconds,
				},
			},
		},
	}
	resp, err := self.signedJsonPost("https://iam.myhuaweicloud.com/v3.0/OS-CREDENTIAL/securitytokens", params)
	if err != nil {
		return errors.Wrapf(err, "assume agency %s in %s", self.getAssumeAgencyName(), self.accountId)
	}
	ret := struct {
		Credential sAgencyCredential
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return errors.Wrap(err, "Unmarshal")
	}
	return self.storeAgencyCredential(ret.Credential)
}
