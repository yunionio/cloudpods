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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	monitorapi "yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modmonitor "yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
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

func (lb *SLoadbalancer) getInfluxdbDBName() (string, error) {
	lbagents, err := LoadbalancerAgentManager.getByClusterId(lb.ClusterId)
	if err != nil {
		return "", err
	}
	if len(lbagents) == 0 {
		return "", errors.Wrapf(errors.ErrNotFound, "lbcluster %s has no agent", lb.ClusterId)
	}
	var dbName string
	for i := range lbagents {
		lbagent := &lbagents[i]
		params := lbagent.Params
		if params == nil {
			continue
		}
		if params.Telegraf.InfluxDbOutputName != "" {
			dbName = params.Telegraf.InfluxDbOutputName
			if lbagent.HaState == api.LB_HA_STATE_MASTER {
				break
			}
		}
	}
	if dbName == "" {
		return "", errors.Wrap(errors.ErrNotFound, "no lbagent has influxdb db name")
	}
	return dbName, nil
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

	dbName, err := lb.getInfluxdbDBName()
	if err != nil {
		return nil, errors.Wrapf(err, "find influxdb db name for loadbalancer %s", lb.Id)
	}

	queryInput := modmonitor.NewMetricQueryInputWithDB(dbName, "haproxy").
		From(time.Now().Add(-5 * time.Minute)).
		To(time.Now()).
		Scope("system").
		SkipCheckSeries(true)
	sels := queryInput.Selects()
	sels.Select("check_code").LAST()
	// lastsess is always written by the haproxy telegraf plugin, so include it as a
	// fallback to guarantee the series (and its check_status tag) is returned even
	// when check_code has no data for a backend.
	sels.Select("lastsess").LAST()
	queryInput.Where().Equal("pxname", pxname).REGEX("svname", "........-....-....-....-............")
	// check_status is stored as a tag in newer telegraf haproxy plugin output;
	// pull it into the grouped result so it appears in series.Tags.
	queryInput.GroupBy().TAG("pxname").TAG("svname")

	s := auth.GetAdminSession(ctx, options.Options.Region)
	resp, err := modmonitor.UnifiedMonitorManager.PerformQuery(s, queryInput.ToQueryData())
	if err != nil {
		return nil, errors.Wrap(err, "query monitor")
	}

	result := new(monitorapi.MetricsQueryResult)
	if err := resp.Unmarshal(result); err != nil {
		return nil, errors.Wrap(err, "unmarshal monitor query result")
	}

	// VictoriaMetrics driver returns the column as "<measurement>_<field>" for
	// single-select queries (e.g. "haproxy_check_code") or as
	// "<aggr>_<measurement>_<field>" for multi-select union queries (e.g.
	// "last_haproxy_check_code") because the influxql→metricsql converter drops AS
	// aliases. Strip both prefixes to recover the original field name; for the
	// influxdb driver the column already matches.
	stripCol := func(col string) string {
		return strings.TrimPrefix(strings.TrimPrefix(col, "last_"), "haproxy_")
	}

	// Tracks backends whose check_status came from the check_code-bearing
	// series; that source takes precedence over the lastsess-bearing series.
	csFromCheckCode := make(map[int]bool)

	for _, series := range result.Series {
		svname := series.Tags["svname"]
		ok, i := utils.InStringArray(svname, backendIds)
		if !ok {
			continue
		}
		backendJson := backendJsons[i].(*jsonutils.JSONDict)
		hasCheckCode := false
		hasLastsess := false
		for _, colName := range series.Columns {
			switch stripCol(colName) {
			case "check_code":
				hasCheckCode = true
			case "lastsess":
				hasLastsess = true
			}
		}
		// check_status is a tag in the haproxy measurement. Prefer the value from
		// the check_code-bearing series; fall back to the always-present lastsess
		// series only when check_code didn't supply one. Set it before the point
		// loop so it's recorded even if the loop finds no valid values.
		if cs, ok := series.Tags["check_status"]; ok {
			if hasCheckCode {
				backendJson.Set("check_status", jsonutils.NewString(cs))
				csFromCheckCode[i] = true
			} else if hasLastsess && !csFromCheckCode[i] {
				backendJson.Set("check_status", jsonutils.NewString(cs))
			}
		}
		var latest monitorapi.TimePoint
		for _, p := range series.Points {
			hasValue := false
			for k := 0; k < len(p)-1; k++ {
				if p[k] != nil {
					hasValue = true
					break
				}
			}
			if !hasValue {
				continue
			}
			if latest == nil || p.Timestamp() > latest.Timestamp() {
				latest = p
			}
		}
		if latest == nil {
			continue
		}
		backendJson.Set("check_time", jsonutils.NewTimeString(latest.Time()))
		for j, colName := range series.Columns {
			if colName == "time" {
				continue
			}
			if j >= len(latest)-1 {
				break
			}
			fieldName := stripCol(colName)
			v := latest[j]
			if v == nil {
				backendJson.Set(fieldName, jsonutils.JSONNull)
				continue
			}
			backendJson.Set(fieldName, jsonutils.Marshal(v))
		}
	}
	return jsonutils.NewArray(backendJsons...), nil
}
