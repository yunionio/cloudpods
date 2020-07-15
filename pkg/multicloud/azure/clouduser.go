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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SClouduserPasswordProfile struct {
	Password                     string
	forceChangePasswordNextLogin bool
	enforceChangePasswordPolicy  bool
}

type SClouduser struct {
	client *SAzureClient

	OdataType                        string `json:"odata.type"`
	ObjectType                       string
	ObjectId                         string
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
}

func (user *SClouduser) GetName() string {
	return user.UserPrincipalName
}

func (user *SClouduser) GetGlobalId() string {
	return user.ObjectId
}

func (user *SClouduser) GetISystemCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := user.client.GetCloudpolicies(user.ObjectId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetCloudpolicies(%s)", user.ObjectId)
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		ret = append(ret, &policies[i])
	}
	return ret, nil
}

func (user *SClouduser) AttachSystemPolicy(policyId string) error {
	return user.client.AssignPolicy(user.ObjectId, policyId)
}

func (user *SClouduser) DetachSystemPolicy(policyId string) error {
	assignments, err := user.client.GetAssignments(user.ObjectId)
	if err != nil {
		return errors.Wrapf(err, "GetAssignments(%s)", user.ObjectId)
	}
	for _, assignment := range assignments {
		role, err := user.client.GetRole(assignment.Properties.RoleDefinitionId)
		if err != nil {
			return errors.Wrapf(err, "GetRule(%s)", assignment.Properties.RoleDefinitionId)
		}
		if role.Properties.RoleName == policyId {
			return user.client.Delete(assignment.Id)
		}
	}
	return nil
}

func (user *SClouduser) IsConsoleLogin() bool {
	return user.AccountEnabled
}

// 需要当前应用有User administrator权限
func (user *SClouduser) Delete() error {
	return user.client.DeleteClouduser(user.ObjectId)
}

func (user *SClouduser) ResetPassword(password string) error {
	return user.client.ResetClouduserPassword(user.ObjectId, password)
}

func (user *SClouduser) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	groups, err := user.client.GetUserGroups(user.ObjectId)
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
	cli, err := self.getGraphClient()
	if err != nil {
		return nil, err
	}

	resource := fmt.Sprintf("%s/users/%s/memberOf", self.tenantId, userId)
	resp, err := jsonRequest(cli, "GET", self.domain, resource, self.subscriptionId, "", GraphResource)
	if err != nil {
		return nil, err
	}

	groups := []SCloudgroup{}
	err = resp.Unmarshal(&groups, "value")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return groups, nil
}

func (self *SAzureClient) ResetClouduserPassword(id, password string) error {
	cli, err := self.getGraphClient()
	if err != nil {
		return err
	}
	body := jsonutils.Marshal(map[string]interface{}{
		"passwordPolicies": "DisablePasswordExpiration, DisableStrongPassword",
		"passwordProfile": map[string]interface{}{
			"password": password,
		},
	})
	resource := fmt.Sprintf("%s/users/%s", self.tenantId, id)
	_, err = jsonRequest(cli, "PATCH", self.domain, resource, self.subscriptionId, body.String(), GraphResource)
	return err
}

func (self *SAzureClient) GetCloudusers(name string) ([]SClouduser, error) {
	users := []SClouduser{}
	params := url.Values{}
	if len(name) > 0 {
		params.Set("$filter", fmt.Sprintf("userPrincipalName eq '%s'", name))
	}
	err := self.ListGraphResource("users", params, &users)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (self *SAzureClient) DeleteClouduser(id string) error {
	return self.DeleteGraph(fmt.Sprintf("%s/users/%s?api-version=1.6", self.tenantId, id))
}

func (self *SAzureClient) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	users, err := self.GetCloudusers("")
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
	users, err := self.GetCloudusers(name)
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudusers")
	}
	if len(users) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if len(users) > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	users[0].client = self
	return &users[0], nil
}

func (self *SAzureClient) CreateIClouduser(conf *cloudprovider.SClouduserCreateConfig) (cloudprovider.IClouduser, error) {
	return self.CreateClouduser(conf.Name, conf.Password)
}

type SDomain struct {
	Name string
}

func (self *SAzureClient) GetDomains() ([]SDomain, error) {
	domains := []SDomain{}
	err := self.ListGraphResource("domains", nil, &domains)
	if err != nil {
		return nil, errors.Wrap(err, "ListGraphResource")
	}
	return domains, nil
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
	domains, err := self.GetDomains()
	if err != nil {
		return nil, errors.Wrap(err, "GetDomains")
	}
	if len(domains) == 0 {
		return nil, errors.Wrap(err, "Missing domains")
	}
	params["userPrincipalName"] = fmt.Sprintf("%s@%s", name, domains[0].Name)
	user := SClouduser{client: self}
	err = self.CreateGraphResource("users", jsonutils.Marshal(params), &user)
	if err != nil {
		return nil, errors.Wrap(err, "Create")
	}
	return &user, nil
}
