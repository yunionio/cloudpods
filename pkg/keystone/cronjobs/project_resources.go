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

package cronjobs

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/keystone/tokens"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

var (
	serviceBlackList map[string]time.Time
)

func init() {
	serviceBlackList = make(map[string]time.Time)
}

type sServiceEndpoints struct {
	regionId  string
	serviceId string
	internal  string
	external  string
}

func FetchScopeResourceCount(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	err := refreshScopeResourceCount(ctx)
	if err != nil {
		log.Errorf("refreshScopeResourceCount error: %v", err)
	}
}
func refreshScopeResourceCount(ctx context.Context) error {
	log.Debugf("FetchScopeResourceCount")
	eps, err := models.EndpointManager.FetchAll()
	if err != nil {
		return errors.Wrapf(err, "EndpointManager.FetchAll")
	}
	serviceTbl := make(map[string]*sServiceEndpoints)
	for _, ep := range eps {
		if utils.IsInStringArray(ep.ServiceType, apis.NO_RESOURCE_SERVICES) {
			// skip self and offline cloudmeta
			continue
		}
		key := fmt.Sprintf("%s-%s", ep.RegionId, ep.ServiceId)
		if _, ok := serviceTbl[key]; !ok {
			serviceTbl[key] = &sServiceEndpoints{
				regionId:  ep.RegionId,
				serviceId: ep.ServiceId,
			}
		}
		switch ep.Interface {
		case api.EndpointInterfacePublic:
			serviceTbl[key].external = ep.Url
		case api.EndpointInterfaceInternal:
			serviceTbl[key].internal = ep.Url
		}
	}
	for srvId, ep := range serviceTbl {
		if to, ok := serviceBlackList[srvId]; ok {
			if to.IsZero() || to.After(time.Now()) {
				continue
			}
		}
		url := ep.internal
		if url == "" {
			url = ep.external
		}
		url = httputils.JoinPath(url, "scope-resources")
		tk := tokens.GetDefaultToken()
		hdr := http.Header{}
		hdr.Add("X-Auth-Token", tk)
		_, ret, err := httputils.JSONRequest(
			httputils.GetTimeoutClient(time.Minute),
			ctx, "GET",
			url,
			hdr,
			nil, false)
		if err != nil {
			// ignore errors
			// log.Errorf("fetch from %s fail: %s", url, err)
			errCode := httputils.ErrorCode(err)
			if errCode == 404 {
				serviceBlackList[srvId] = time.Time{}
			} else {
				serviceBlackList[srvId] = time.Now().Add(time.Hour)
			}
			continue
		}
		if gotypes.IsNil(ret) {
			continue
		}
		if _, ok := serviceBlackList[srvId]; ok {
			delete(serviceBlackList, srvId)
		}
		projectResCounts := make(map[string][]db.SScopeResourceCount)
		err = ret.Unmarshal(&projectResCounts)
		if err != nil {
			continue
		}
		syncScopeResourceCount(ctx, ep.regionId, ep.serviceId, projectResCounts)
	}
	return nil
}

func syncScopeResourceCount(ctx context.Context, regionId string, serviceId string, projResCnt map[string][]db.SScopeResourceCount) {
	for res, resCnts := range projResCnt {
		projList := make([]string, 0)
		domainList := []string{}
		ownerList := []string{}

		for i := range resCnts {
			if len(resCnts[i].TenantId) == 0 && len(resCnts[i].DomainId) == 0 && len(resCnts[i].OwnerId) == 0 {
				continue
			}

			scopeRes := models.SScopeResource{
				DomainId:  resCnts[i].DomainId,
				ProjectId: resCnts[i].TenantId,
				OwnerId:   resCnts[i].OwnerId,
			}
			scopeRes.RegionId = regionId
			scopeRes.ServiceId = serviceId
			scopeRes.Resource = res
			scopeRes.Count = resCnts[i].ResCount

			if len(scopeRes.ProjectId) > 0 {
				projList = append(projList, scopeRes.ProjectId)
			}
			if len(scopeRes.DomainId) > 0 {
				domainList = append(domainList, scopeRes.DomainId)
			}
			if len(scopeRes.OwnerId) > 0 {
				ownerList = append(ownerList, scopeRes.OwnerId)
			}

			scopeRes.SetModelManager(models.ScopeResourceManager, &scopeRes)

			err := models.ScopeResourceManager.TableSpec().InsertOrUpdate(ctx, &scopeRes)
			if err != nil {
				log.Errorf("table insert error %s", err)
			}
		}

		q := models.ScopeResourceManager.Query()
		if len(projList) > 0 {
			q = q.NotIn("project_id", projList)
		}
		if len(domainList) > 0 {
			q = q.NotIn("domain_id", domainList)
		}
		if len(ownerList) > 0 {
			q = q.NotIn("owner_id", ownerList)
		}
		q = q.Equals("region_id", regionId)
		q = q.Equals("service_id", serviceId)
		q = q.Equals("resource", res)
		q = q.NotEquals("count", 0)

		emptySets := make([]models.SScopeResource, 0)
		err := db.FetchModelObjects(models.ScopeResourceManager, q, &emptySets)
		if err != nil {
			log.Errorf("db.FetchModelObjects %s", err)
		}

		for i := range emptySets {
			_, err := db.Update(&emptySets[i], func() error {
				emptySets[i].Count = 0
				return nil
			})
			if err != nil {
				log.Errorf("db.Update %s", err)
			}
		}
	}
}

func AddRefreshHandler(prefix string, app *appsrv.Application) {
	app.AddHandler2("POST", fmt.Sprintf("%s/scope-resource/refresh", prefix), auth.Authenticate(refreshHandler), nil, "scope_resource_refresh", nil)
}

func refreshHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, nil)
	if userCred == nil || db.IsDomainAllowList(userCred, models.DomainManager).Result.IsDeny() {
		httperrors.ForbiddenError(ctx, w, "not enough privilege")
		return
	}

	err := refreshScopeResourceCount(ctx)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	ret := map[string]interface{}{
		"scope-resource": map[string]string{
			"status": "ok",
		},
	}
	fmt.Fprintf(w, jsonutils.Marshal(ret).String())
}
