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

	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/keystone/tokens"
	"yunion.io/x/onecloud/pkg/mcclient"
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

func FetchProjectResourceCount(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	//log.Debugf("FetchProjectResourceCount")
	eps, err := models.EndpointManager.FetchAll()
	if err != nil {
		return
	}
	serviceTbl := make(map[string]*sServiceEndpoints)
	for _, ep := range eps {
		if ep.ServiceType == api.SERVICE_TYPE {
			// skip self
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
		url = httputils.JoinPath(url, "project-resources")
		tk, _ := tokens.GetDefaultToken()
		hdr := http.Header{}
		hdr.Add("X-Auth-Token", tk)
		// log.Debugf("request %s", url)
		_, ret, err := httputils.JSONRequest(
			httputils.GetDefaultClient(),
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
		if _, ok := serviceBlackList[srvId]; ok {
			delete(serviceBlackList, srvId)
		}
		projectResCounts := make(map[string][]db.SProjectResourceCount)
		err = ret.Unmarshal(&projectResCounts)
		if err != nil {
			continue
		}
		syncProjectResourceCount(ep.regionId, ep.serviceId, projectResCounts)
	}
}

func syncProjectResourceCount(regionId string, serviceId string, projResCnt map[string][]db.SProjectResourceCount) {
	projList := make([]string, 0)
	for res, resCnts := range projResCnt {
		for i := range resCnts {
			if resCnts[i].TenantId == "" {
				continue
			}

			projRes := models.SProjectResource{}
			projRes.ProjectId = resCnts[i].TenantId
			projRes.RegionId = regionId
			projRes.ServiceId = serviceId
			projRes.Resource = res
			projRes.Count = resCnts[i].ResCount

			projList = append(projList, resCnts[i].TenantId)

			err := models.ProjectResourceManager.TableSpec().InsertOrUpdate(&projRes)
			if err != nil {
				log.Errorf("table insert error %s", err)
			}
		}

		q := models.ProjectResourceManager.Query()
		if len(projList) > 0 {
			q = q.NotIn("project_id", projList)
		}
		q = q.Equals("region_id", regionId)
		q = q.Equals("service_id", serviceId)
		q = q.Equals("resource", res)
		q = q.NotEquals("count", 0)

		emptySets := make([]models.SProjectResource, 0)
		err := db.FetchModelObjects(models.ProjectResourceManager, q, &emptySets)
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
