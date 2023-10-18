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

package bingoiam

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	json "yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/keystone/models"
)

type SBase struct {
	Id         string    `json:"id"`
	Name       string    `json:"name"`
	Code       string    `json:"code"`
	Email      string    `json:"email"`
	Mobile     string    `json:"mobile"`
	ParentId   string    `json:"parentId"`
	ExternalId string    `json:"externalId"`
	Active     bool      `json:"active"`
	IsDeleted  bool      `json:"isDeleted"`
	CreatedAt  time.Time `json:"createdAt"`
	CreatedBy  string    `json:"createdBy"`
	UpdatedAt  time.Time `json:"updatedAt"`
	UpdatedBy  string    `json:"updatedBy"`
}

type STenant struct {
	SBase
}

type SOrganization struct {
	SBase
	TenantId string `json:"tenantId"`
}

type SUser struct {
	SBase
	OrgId    string `json:"orgId"`
	UserName string `json:"userName"`
	TenantId string `json:"tenantId"`
}

type SApp struct {
	SBase
	Name        string `json:"title"`
	TenantId    string `json:"tenantId"`
	AppType     string `json:"appType"`
	ProjectId   string `json:"projectId"`
	Description string `json:"description"`
}

type SProject struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	TenantId string `json:"tenantId"`
}

func (drv *SBingoIAMOAuth2Driver) syncTenants(ctx context.Context, idp *models.SIdentityProvider) error {
	var count, page = 0, 0
	for {
		page++
		total, tenants, err := drv.getTenants(ctx, page, 500)
		if err != nil {
			return err
		}
		for _, tenant := range tenants {
			domain, err := idp.SyncOrCreateDomain(ctx, tenant.Id, tenant.Name, fmt.Sprintf("Sync from %s", idp.Name), false)
			if err != nil {
				return errors.Wrap(err, "idp.SyncOrCreateDomain")
			}
			drv.domains[tenant.Id] = domain
		}
		count += len(tenants)
		if count >= total {
			break
		}
		time.Sleep(time.Millisecond * 200)
	}
	return nil
}

func (drv *SBingoIAMOAuth2Driver) syncOrganizations(ctx context.Context, idp *models.SIdentityProvider) error {
	var loader func(filters string, orgs chan []*SOrganization)
	loader = func(filters string, orgs chan []*SOrganization) {
		var page, count = 0, 0
		var childOrgs []*SOrganization
		for {
			page++
			total, orgs, err := drv.getOrganizations(ctx, filters, page, 500)
			if err != nil {
				log.Errorf("get organizations %s fail %s", filters, err)
				return
			}
			childOrgs = append(childOrgs, orgs...)
			count += len(orgs)
			if count >= total {
				break
			}
			time.Sleep(time.Millisecond * 200)
		}
		if len(childOrgs) > 0 {
			orgs <- childOrgs
		}
		for _, org := range childOrgs {
			loader("parentId eq "+org.Id, orgs)
		}
	}

	var allOrgs = make(chan []*SOrganization)
	go func() {
		for orgs := range allOrgs {
			for _, org := range orgs {
				domain := drv.domains[org.TenantId]
				if domain == nil {
					continue
				}
				//TODO save to db
			}
		}
	}()
	loader("parentId is null", allOrgs)
	close(allOrgs)
	return nil
}

func (drv *SBingoIAMOAuth2Driver) syncProjects(ctx context.Context, idp *models.SIdentityProvider) error {
	var count, page = 0, 0
	for {
		page++
		total, projects, err := drv.getProjects(ctx, page, 500)
		if err != nil {
			return err
		}
		for _, project := range projects {
			domain := drv.domains[project.TenantId]
			if domain == nil {
				continue
			}
			ret, err := models.ProjectManager.FetchProject("", project.Name, domain.Id, "")
			if err != nil {
				log.Errorf("fetch project %s fail %s", project.Name, err)
				if errors.Cause(err) == sql.ErrNoRows {
					ret, err = models.ProjectManager.NewProject(ctx, project.Name, fmt.Sprintf("Sync from %s", idp.Name), domain.Id)
					if err != nil {
						continue
					}
				}
			}
			drv.projects[project.Id] = ret
		}
		count += len(projects)
		if count >= total {
			break
		}
		time.Sleep(time.Millisecond * 200)
	}
	return nil
}

func (drv *SBingoIAMOAuth2Driver) syncUsers(ctx context.Context, idp *models.SIdentityProvider) error {
	var count, page = 0, 0
	for {
		page++
		total, users, err := drv.getUsers(ctx, page, 500)
		if err != nil {
			return err
		}
		for _, user := range users {
			domain := drv.domains[user.TenantId]
			if domain == nil {
				continue
			}
			_, err = idp.SyncOrCreateUser(ctx, user.Id, user.UserName, domain.Id, true, func(u *models.SUser) {
				u.Id = user.Id
				u.Name = user.UserName
				u.Displayname = user.Name
				u.Description = fmt.Sprintf("Sync from %s", idp.Name)
				u.Email = user.Email
				u.DomainId = domain.Id
				u.Mobile = user.Mobile
				u.Enabled = tristate.True
				u.SEnabledIdentityBaseResource.Enabled = tristate.True
			})
			if err != nil {
				return errors.Wrap(err, "idp.SyncOrCreateDomain")
			}
		}
		count += len(users)
		if count >= total {
			break
		}
		time.Sleep(time.Millisecond * 200)
	}
	return nil
}

func (drv *SBingoIAMOAuth2Driver) getTenants(ctx context.Context, page, pageSize int) (int, []*STenant, error) {
	var accessToken = drv.accessToken

	if accessToken == "" {
		accessToken, _ = drv.getAccessToken(ctx)
	}

	urlStrStr := fmt.Sprintf("%v/tenant?total=true&page_size=%v&page=%v", drv.getIAMApiEndpoint(ctx), pageSize, page)
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+accessToken)

	return doRequest[[]*STenant](ctx, urlStrStr, httputils.GET, headers, nil)
}

func (drv *SBingoIAMOAuth2Driver) getOrganizations(ctx context.Context, filters string, page, pageSize int) (int, []*SOrganization, error) {
	var accessToken = drv.accessToken

	if accessToken == "" {
		accessToken, _ = drv.getAccessToken(ctx)
	}

	urlStr := fmt.Sprintf("%v/organization?total=true&page_size=%v&page=%v&filters=%v", drv.getIAMApiEndpoint(ctx), pageSize, page, url.QueryEscape(filters))
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+accessToken)

	return doRequest[[]*SOrganization](ctx, urlStr, httputils.GET, headers, nil)
}

func (drv *SBingoIAMOAuth2Driver) getUsers(ctx context.Context, page, pageSize int) (int, []*SUser, error) {
	var accessToken = drv.accessToken

	if accessToken == "" {
		accessToken, _ = drv.getAccessToken(ctx)
	}

	urlStr := fmt.Sprintf("%v/user?total=true&page_size=%v&page=%v", drv.getIAMApiEndpoint(ctx), pageSize, page)
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+accessToken)

	return doRequest[[]*SUser](ctx, urlStr, httputils.GET, headers, nil)

}

func (drv *SBingoIAMOAuth2Driver) getProjects(ctx context.Context, page, pageSize int) (int, []*SProject, error) {
	var accessToken = drv.accessToken

	if accessToken == "" {
		accessToken, _ = drv.getAccessToken(ctx)
	}

	urlStr := fmt.Sprintf("%v/project?total=true&page_size=%v&page=%v", drv.getIAMApiEndpoint(ctx), pageSize, page)
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+accessToken)

	return doRequest[[]*SProject](ctx, urlStr, httputils.GET, headers, nil)
}

func doRequest[Result any](ctx context.Context, urlStr string, method httputils.THttpMethod, headers http.Header, body json.JSONObject) (total int, result Result, err error) {
	httpclient := httputils.GetDefaultClient()
	repsHeaders, resp, err := httputils.JSONRequest(httpclient, ctx, method, urlStr, headers, body, true)
	if err != nil {
		return 0, result, err
	}
	total, _ = strconv.Atoi(repsHeaders.Get("X-Total-Count"))
	err = resp.Unmarshal(&result)
	return
}
