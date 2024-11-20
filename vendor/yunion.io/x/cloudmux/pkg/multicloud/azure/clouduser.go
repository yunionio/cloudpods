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
	"time"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SClouduserPasswordProfile struct {
	Password                     string
	forceChangePasswordNextLogin bool
	enforceChangePasswordPolicy  bool
}

type SClouduser struct {
	client *SAzureClient
	multicloud.SBaseClouduser

	OdataType                        string `json:"odata.type"`
	ObjectType                       string
	Id                               string
	DeletionTimestamp                string
	AccountEnabled                   bool
	AgeGroup                         string
	City                             string
	CompanyName                      string
	ConsentProvidedForMinor          string
	Country                          string
	CreatedDateTime                  time.Time
	CreationType                     string
	Department                       string
	DirSyncEnabled                   string
	DisplayName                      string
	EmployeeId                       string
	FacsimileTelephoneNumber         string
	GivenName                        string
	ImmutableId                      string
	IsCompromised                    string
	JobTitle                         string
	LastDirSyncTime                  string
	LegalAgeGroupClassification      string
	Mail                             string
	MailNickname                     string
	Mobile                           string
	OnPremisesDistinguishedName      string
	OnPremisesSecurityIdentifier     string
	PasswordPolicies                 string
	PasswordProfile                  SClouduserPasswordProfile
	PhysicalDeliveryOfficeName       string
	PostalCode                       string
	PreferredLanguage                string
	RefreshTokensValidFromDateTime   time.Time
	ShowInAddressList                string
	SipProxyAddress                  string
	State                            string
	StreetAddress                    string
	Surname                          string
	TelephoneNumber                  string
	ThumbnailPhotoOdataMediaEditLink string `json:"thumbnailPhoto@odata.mediaEditLink"`
	UsageLocation                    string
	UserPrincipalName                string
	UserState                        string
	UserStateChangedOn               string
	UserType                         string

	inviteRedeemUrl string
}

func (user *SClouduser) GetName() string {
	return user.UserPrincipalName
}

func (user *SClouduser) GetGlobalId() string {
	return user.Id
}

func (user *SClouduser) GetEmailAddr() string {
	return user.Mail
}

func (user *SClouduser) GetInviteUrl() string {
	return user.inviteRedeemUrl
}

func (user *SClouduser) GetICloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := user.client.GetPrincipalPolicy(user.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetCloudpolicies(%s)", user.Id)
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		ret = append(ret, &SCloudpolicy{Id: policies[i].RoleDefinitionId})
	}
	return ret, nil
}

func (user *SClouduser) AttachPolicy(policyId string, policyType api.TPolicyType) error {
	return user.client.AssignPolicy(user.Id, policyId)
}

func (user *SClouduser) DetachPolicy(policyId string, policyType api.TPolicyType) error {
	policys, err := user.client.GetPrincipalPolicy(user.Id)
	if err != nil {
		return err
	}
	for _, policy := range policys {
		if policy.RoleDefinitionId == policyId {
			return user.client.DeletePrincipalPolicy(policy.Id)
		}
	}
	return nil
}

func (user *SClouduser) IsConsoleLogin() bool {
	return true
}

// 需要当前应用有User administrator权限
func (user *SClouduser) Delete() error {
	return user.client.DeleteClouduser(user.UserPrincipalName)
}

func (user *SClouduser) ResetPassword(password string) error {
	return user.client.ResetClouduserPassword(user.Id, password)
}

func (user *SClouduser) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	groups, err := user.client.GetUserGroups(user.Id)
	if err != nil {
		return nil, errors.Wrap(err, "GetUserGroups")
	}
	ret := []cloudprovider.ICloudgroup{}
	for i := range groups {
		groups[i].client = user.client
		ret = append(ret, &groups[i])
	}
	return ret, nil
}

func (self *SAzureClient) GetUserGroups(userId string) ([]SCloudgroup, error) {
	resource := fmt.Sprintf("users/%s/memberOf", userId)
	resp, err := self._list_v2(SERVICE_GRAPH, resource, "", nil)
	if err != nil {
		return nil, err
	}
	groups := []SCloudgroup{}
	err = resp.Unmarshal(&groups, "value")
	if err != nil {
		return nil, err
	}
	return groups, err
}

func (self *SAzureClient) ResetClouduserPassword(id, password string) error {
	body := map[string]interface{}{
		"passwordPolicies": "DisablePasswordExpiration, DisableStrongPassword",
		"passwordProfile": map[string]interface{}{
			"password": password,
		},
	}
	resource := fmt.Sprintf("%s/users/%s", self.tenantId, id)
	_, err := self._patch_v2(SERVICE_GRAPH, resource, "", body)
	return err
}

func (self *SAzureClient) GetClouduser(name string) (*SClouduser, error) {
	users, err := self.GetCloudusers()
	if err != nil {
		return nil, err
	}
	for i := range users {
		if users[i].DisplayName == name || users[i].UserPrincipalName == name {
			users[i].client = self
			return &users[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SAzureClient) GetCloudusers() ([]SClouduser, error) {
	users := []SClouduser{}
	resp, err := self._list_v2(SERVICE_GRAPH, "users", "", url.Values{})
	if err != nil {
		return nil, err
	}
	err = resp.Unmarshal(&users, "value")
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (self *SAzureClient) DeleteClouduser(id string) error {
	_, err := self._delete_v2(SERVICE_GRAPH, "users/"+id, "")
	return err
}

func (self *SAzureClient) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	users, err := self.GetCloudusers()
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudusers")
	}
	ret := []cloudprovider.IClouduser{}
	for i := range users {
		users[i].client = self
		ret = append(ret, &users[i])
	}
	return ret, nil
}

func (self *SAzureClient) GetIClouduserByName(name string) (cloudprovider.IClouduser, error) {
	user, err := self.GetClouduser(name)
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudusers")
	}
	return user, nil
}

func (self *SAzureClient) CreateIClouduser(conf *cloudprovider.SClouduserCreateConfig) (cloudprovider.IClouduser, error) {
	if conf.UserType == "Guest" {
		return self.InviteUser(conf.Email)
	}
	return self.CreateClouduser(conf.Name, conf.Password)
}

type SDomain struct {
	Name                             string
	Id                               string
	AuthenticationType               string
	AvailabilityStatus               string
	IsAdminManaged                   bool
	IsDefault                        bool
	IsDefaultForCloudRedirections    bool
	IsInitial                        bool
	IsRoot                           bool
	IsVerified                       bool
	ForceDeleteState                 string
	State                            string
	PasswordValidityPeriodInDays     string
	PasswordNotificationWindowInDays string
}

func (self *SAzureClient) GetDomains() ([]SDomain, error) {
	domains := []SDomain{}
	resp, err := self._list_v2(SERVICE_GRAPH, "domains", "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "list domains")
	}
	err = resp.Unmarshal(&domains, "value")
	if err != nil {
		return nil, err
	}
	return domains, nil
}

func (self *SAzureClient) GetDefaultDomain() (string, error) {
	domains, err := self.GetDomains()
	if err != nil {
		return "", errors.Wrapf(err, "ListGraphUsers")
	}
	for i := range domains {
		if domains[i].IsDefault && domains[i].IsVerified && domains[i].IsRoot {
			return domains[i].Id, nil
		}
	}
	return "", cloudprovider.ErrNotFound
}

func (self *SAzureClient) CreateClouduser(name, password string) (*SClouduser, error) {
	passwordProfile := map[string]interface{}{
		"password": "Lomo1824",
	}
	if len(password) > 0 {
		passwordProfile["password"] = password
	}
	params := map[string]interface{}{
		"accountEnabled":    true,
		"displayName":       name,
		"mailNickname":      name,
		"passwordProfile":   passwordProfile,
		"userPrincipalName": name,
	}
	domain, err := self.GetDefaultDomain()
	if err != nil {
		return nil, errors.Wrap(err, "GetDefaultDomain")
	}
	params["userPrincipalName"] = fmt.Sprintf("%s@%s", name, domain)
	user := &SClouduser{client: self}
	resp, err := self._post_v2(SERVICE_GRAPH, "users", "", params)
	if err != nil {
		return nil, errors.Wrap(err, "Create")
	}
	err = resp.Unmarshal(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}
