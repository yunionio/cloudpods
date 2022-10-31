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

package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

func (lblis *SLoadbalancerListener) GetDetailsBackendStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	provider := lblis.GetCloudprovider()
	if provider != nil {
		return jsonutils.NewArray(), nil
	}
	if lblis.BackendGroupId == "" {
		return jsonutils.NewArray(), nil
	}
	lb, err := lblis.GetLoadbalancer()
	if err != nil {
		return nil, errors.Wrapf(err, "GetLoadbalancer")
	}
	var pxname string
	switch lblis.ListenerType {
	case api.LB_LISTENER_TYPE_TCP:
		pxname = fmt.Sprintf("backends_listener-%s", lblis.Id)
	case api.LB_LISTENER_TYPE_HTTP, api.LB_LISTENER_TYPE_HTTPS:
		pxname = fmt.Sprintf("backends_listener_default-%s", lblis.Id)
	}
	return lb.GetBackendGroupCheckStatus(ctx, userCred, pxname, lblis.BackendGroupId)
}

func (lbr *SLoadbalancerListenerRule) GetDetailsBackendStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	provider := lbr.GetCloudprovider()
	if provider != nil {
		return jsonutils.NewArray(), nil
	}
	lblis, err := lbr.GetLoadbalancerListener()
	if err != nil {
		return nil, err
	}
	lb, err := lblis.GetLoadbalancer()
	if err != nil {
		return nil, errors.Wrapf(err, "GetLoadbalancer")
	}
	pxname := fmt.Sprintf("backends_rule-%s", lbr.Id)
	return lb.GetBackendGroupCheckStatus(ctx, userCred, pxname, lbr.BackendGroupId)
}

func (lb *SLoadbalancer) GetInfluxdbByLbId() (*influxdb.SInfluxdb, string, error) {
	lbagents, err := LoadbalancerAgentManager.getByClusterId(lb.ClusterId)
	if err != nil {
		return nil, "", err
	}
	if len(lbagents) == 0 {
		return nil, "", errors.Wrapf(errors.ErrNotFound, "lbcluster %s has no agent", lb.ClusterId)
	}
	var (
		dbUrl  string
		dbName string
	)
	for i := range lbagents {
		lbagent := &lbagents[i]
		params := lbagent.Params
		if params == nil {
			continue
		}
		paramsTelegraf := params.Telegraf
		if paramsTelegraf.InfluxDbOutputUrl != "" && paramsTelegraf.InfluxDbOutputName != "" {
			dbUrl = paramsTelegraf.InfluxDbOutputUrl
			dbName = paramsTelegraf.InfluxDbOutputName
			if lbagent.HaState == api.LB_HA_STATE_MASTER {
				// prefer the one on master
				break
			}
		}
	}
	if dbUrl == "" || dbName == "" {
		return nil, "", errors.Wrap(errors.ErrNotFound, "no lbagent has influxdb url or db name")
	}
	dbinst := influxdb.NewInfluxdb(dbUrl)
	return dbinst, dbName, nil
}

func (lb *SLoadbalancer) GetBackendGroupCheckStatus(ctx context.Context, userCred mcclient.TokenCredential, pxname string, groupId string) (*jsonutils.JSONArray, error) {
	var (
		backendJsons []jsonutils.JSONObject
		backendIds   []string
	)
	{
		var err error
		q := LoadbalancerBackendManager.Query().Equals("backend_group_id", groupId)
		backendJsons, err = db.Query2List(LoadbalancerBackendManager, ctx, userCred, q, jsonutils.NewDict(), false)
		if err != nil {
			return nil, errors.Wrapf(err, "query backends of backend group %s", groupId)
		}
		if len(backendJsons) == 0 {
			return jsonutils.NewArray(), nil
		}
		for _, backendJson := range backendJsons {
			id, err := backendJson.GetString("id")
			if err != nil {
				return nil, errors.Wrap(err, "get backend id from json")
			}
			if id == "" {
				return nil, errors.Wrap(err, "get backend id from json: id empty")
			}
			backendIds = append(backendIds, id)
		}
	}

	dbinst, dbName, err := lb.GetInfluxdbByLbId()
	if err != nil {
		return nil, errors.Wrapf(err, "find influxdb for loadbalancer %s", lb.Id)
	}

	queryFmt := "select check_status, check_code from %s..haproxy where pxname = '%s' and svname =~ /........-....-....-....-............/ group by pxname, svname order by time desc limit 1"
	querySql := fmt.Sprintf(queryFmt, dbName, pxname)
	queryRes, err := dbinst.Query(querySql)
	if err != nil {
		return nil, errors.Wrap(err, "query influxdb")
	}
	if len(queryRes) != 1 {
		return nil, fmt.Errorf("query influxdb: expecting 1 set of results, got %d", len(queryRes))
	}
	type Tags struct {
		PxName string `json:"pxname"`
		SvName string `json:"svname"`
	}
	for _, resSeries := range queryRes[0] {
		if len(resSeries.Values) == 0 {
			continue
		}
		resColumns := resSeries.Values[0]
		if len(resColumns) != 3 {
			continue
		}
		tags := Tags{}
		if err := resSeries.Tags.Unmarshal(&tags); err != nil {
			return nil, errors.Wrap(err, "unmarshal tags in influxdb query result")
		}
		ok, i := utils.InStringArray(tags.SvName, backendIds)
		if !ok {
			continue
		}
		backendJson := backendJsons[i].(*jsonutils.JSONDict)
		for j, colName := range resSeries.Columns {
			colVal := resColumns[j]
			if colVal == nil {
				colVal = jsonutils.JSONNull
			}
			if colName == "time" {
				colName = "check_time"
			}
			backendJson.Set(colName, colVal)
		}
	}
	return jsonutils.NewArray(backendJsons...), nil
}
