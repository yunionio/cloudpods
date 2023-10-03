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

package cas

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/keystone/driver"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// apereo CAS (Central Authentication Server)
type SCASDriver struct {
	driver.SBaseIdentityDriver

	casConfig *api.SCASIdpConfigOptions

	isDebug bool
}

func NewCASDriver(idpId, idpName, template, targetDomainId string, conf api.TConfigs) (driver.IIdentityBackend, error) {
	base, err := driver.NewBaseIdentityDriver(idpId, idpName, template, targetDomainId, conf)
	if err != nil {
		return nil, errors.Wrap(err, "NewBaseIdentityDriver")
	}
	drv := SCASDriver{SBaseIdentityDriver: base}
	drv.SetVirtualObject(&drv)
	err = drv.prepareConfig()
	if err != nil {
		return nil, errors.Wrap(err, "prepareConfig")
	}
	return &drv, nil
}

func (self *SCASDriver) prepareConfig() error {
	if self.casConfig == nil {
		conf := api.SCASIdpConfigOptions{}
		confJson := jsonutils.Marshal(self.Config["cas"])
		err := confJson.Unmarshal(&conf)
		if err != nil {
			return errors.Wrap(err, "json.Unmarshal")
		}
		if len(conf.UserIdAttribute) == 0 {
			conf.UserIdAttribute = "cas:user"
		}
		if len(conf.UserNameAttribute) == 0 {
			conf.UserNameAttribute = "cas:user"
		}
		self.casConfig = &conf
		log.Debugf("%s %s %#v", self.Config, confJson, self.casConfig)
	}
	return nil
}

func (cas *SCASDriver) GetSsoRedirectUri(ctx context.Context, callbackUrl, state string) (string, error) {
	req := map[string]string{
		"service": callbackUrl,
	}
	urlStr := fmt.Sprintf("%s?%s", cas.casConfig.CASServerURL, jsonutils.Marshal(req).QueryString())
	return urlStr, nil
}

func (self *SCASDriver) request(ctx context.Context, method httputils.THttpMethod, path string) ([]byte, error) {
	cli := httputils.GetDefaultClient()
	urlStr := httputils.JoinPath(self.casConfig.CASServerURL, path)
	resp, err := httputils.Request(cli, ctx, method, urlStr, nil, nil, self.isDebug)
	_, body, err := httputils.ParseResponse("", resp, err, self.isDebug)
	return body, err
}

/*
serviceValidate response:
<cas:serviceResponse xmlns:cas='http://www.yale.edu/tp/cas'>
    <cas:authenticationSuccess>
        <cas:user>casuser</cas:user>
    </cas:authenticationSuccess>
</cas:serviceResponse>
<cas:serviceResponse xmlns:cas='http://www.yale.edu/tp/cas'>
    <cas:authenticationSuccess>
        <cas:user>casuser</cas:user>
        <cas:attributes>
            <cas:credentialType>UsernamePasswordCredential</cas:credentialType>
            <cas:isFromNewLogin>false</cas:isFromNewLogin>
            <cas:authenticationDate>2019-09-05T12:40:08.014Z[UTC]</cas:authenticationDate>
            <cas:authenticationMethod>AcceptUsersAuthenticationHandler</cas:authenticationMethod>
            <cas:successfulAuthenticationHandlers>AcceptUsersAuthenticationHandler</cas:successfulAuthenticationHandlers>
            <cas:longTermAuthenticationRequestTokenUsed>false</cas:longTermAuthenticationRequestTokenUsed>
            </cas:attributes>
    </cas:authenticationSuccess>
</cas:serviceResponse>
*/
/*type SCASServiceResponse struct {
	XMLName                  xml.Name `xml:"serviceResponse"`
	CASAuthenticationSuccess struct {
		CASUser string `xml:"user"`
	} `xml:"authenticationSuccess"`
}*/

func (self *SCASDriver) Authenticate(ctx context.Context, ident mcclient.SAuthenticationIdentity) (*api.SUserExtended, error) {
	query := jsonutils.NewDict()
	query.Set("ticket", jsonutils.NewString(ident.CASTicket.Id))
	query.Set("service", jsonutils.NewString(ident.CASTicket.Service))
	path := "serviceValidate?" + query.QueryString()
	resp, err := self.request(ctx, "GET", path)
	/*if err != nil && httputils.ErrorCode(err) == 404 {
		path = "serviceValidate?" + query.QueryString()
		resp, err = self.request(ctx, "GET", path)
	}*/
	if err != nil {
		return nil, errors.Wrap(err, "self.request")
	}
	log.Debugf("CAS response: %s qs: %s", resp, query.QueryString())
	attrs := fetchAttributes(resp)

	var domainId, domainName, usrId, usrName string
	if v, ok := attrs[self.casConfig.DomainIdAttribute]; ok && len(v) > 0 {
		domainId = v[0]
	}
	if v, ok := attrs[self.casConfig.DomainNameAttribute]; ok && len(v) > 0 {
		domainName = v[0]
	}
	if v, ok := attrs[self.casConfig.UserIdAttribute]; ok && len(v) > 0 {
		usrId = v[0]
	}
	if v, ok := attrs[self.casConfig.UserNameAttribute]; ok && len(v) > 0 {
		usrName = v[0]
	}

	idp, err := models.IdentityProviderManager.FetchIdentityProviderById(self.IdpId)
	if err != nil {
		return nil, errors.Wrap(err, "self.GetIdentityProvider")
	}
	domain, usr, err := idp.SyncOrCreateDomainAndUser(ctx, domainId, domainName, usrId, usrName)
	if err != nil {
		return nil, errors.Wrap(err, "idp.SyncOrCreateDomainAndUser")
	}
	extUser, err := models.UserManager.FetchUserExtended(usr.Id, "", "", "")
	if err != nil {
		return nil, errors.Wrap(err, "models.UserManager.FetchUserExtended")
	}

	idp.TryUserJoinProject(self.casConfig.SIdpAttributeOptions, ctx, usr, domain.Id, attrs)

	extUser.AuditIds = []string{
		ident.CASTicket.Id,
	}

	return extUser, nil
}

/*func (self *SCASDriver) userTryJoinProject(ctx context.Context, usr *models.SUser, domainId string, resp []byte) {
	var err error
	var targetProject *models.SProject
	log.Debugf("userTryJoinProject resp %s proj %s", string(resp), self.casConfig.CasProjectAttribute)
	if !consts.GetNonDefaultDomainProjects() {
		domainId = api.DEFAULT_DOMAIN_ID
	}
	if len(self.casConfig.CasProjectAttribute) > 0 {
		projName := fetchAttribute(resp, self.casConfig.CasProjectAttribute)
		if len(projName) > 0 {
			targetProject, err = models.ProjectManager.FetchProject("", projName, domainId, "")
			if err != nil {
				log.Errorf("fetch project %s fail %s", projName, err)
				if errors.Cause(err) == sql.ErrNoRows && self.casConfig.AutoCreateCasProject.IsTrue() {
					targetProject, err = models.ProjectManager.NewProject(ctx, projName, "cas project", domainId)
					if err != nil {
						log.Errorf("auto create project %s fail %s", projName, err)
					}
				}
			}
		}
	}
	if targetProject == nil && len(self.casConfig.DefaultCasProjectId) > 0 {
		targetProject, err = models.ProjectManager.FetchProjectById(self.casConfig.DefaultCasProjectId)
		if err != nil {
			log.Errorf("fetch default project %s fail %s", self.casConfig.DefaultCasProjectId, err)
		}
	}
	if targetProject != nil {
		// put user in project
		var targetRole *models.SRole
		if len(self.casConfig.CasRoleAttribute) > 0 {
			roleName := fetchAttribute(resp, self.casConfig.CasRoleAttribute)
			if len(roleName) > 0 {
				targetRole, err = models.RoleManager.FetchRole("", roleName, domainId, "")
				if err != nil {
					log.Errorf("fetch role %s fail %s", roleName, err)
				}
			}
		}
		if targetRole == nil && len(self.casConfig.DefaultCasRoleId) > 0 {
			targetRole, err = models.RoleManager.FetchRoleById(self.casConfig.DefaultCasRoleId)
			if err != nil {
				log.Errorf("fetch default role %s fail %s", self.casConfig.DefaultCasRoleId, err)
			}
		}
		if targetRole != nil {
			err = models.AssignmentManager.ProjectAddUser(ctx, models.GetDefaultAdminCred(), targetProject, usr, targetRole)
			if err != nil {
				log.Errorf("CAS user join project fail %s", err)
			}
		}
	}
}*/

func fetchAttributes(heystack []byte) map[string][]string {
	ret := make(map[string][]string)
	pattern := regexp.MustCompile(`<([^>/]+)>([^<]*)</([^>]+)>`)
	results := pattern.FindAllStringSubmatch(string(heystack), -1)
	for _, result := range results {
		key := result[1]
		value := strings.TrimSpace(result[2])
		var vs []string
		if _, ok := ret[key]; ok {
			vs = ret[key]
		} else {
			vs = make([]string, 0, 1)
		}
		ret[key] = append(vs, value)
	}
	return ret
}

func (self *SCASDriver) Sync(ctx context.Context) error {
	return nil
}

func (self *SCASDriver) Probe(ctx context.Context) error {
	_, err := self.request(ctx, "GET", "login")
	if err != nil && httputils.ErrorCode(err) != 401 {
		return errors.Wrap(err, "self.request")
	}
	return nil
}
