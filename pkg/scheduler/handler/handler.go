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

package handler

import (
	"fmt"
	"net/http"
	"time"

	simplejson "github.com/bitly/go-simplejson"
	gin "github.com/gin-gonic/gin"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	computemodels "yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	skuman "yunion.io/x/onecloud/pkg/scheduler/data_manager/sku"
	schedman "yunion.io/x/onecloud/pkg/scheduler/manager"
)

// InstallHandler is an interface that registes route and
// handles scheduler's services.
func InstallHandler(r *gin.Engine) {
	r.POST("/scheduler", timer(scheduleHandler))
	r.POST("/scheduler/:action", timer(schedulerActionHandler))
	r.POST("/scheduler/:action/:ident", timer(schedulerActionIdentHandler))
	InstallPingHandler(r)
	InstallVersionHandler(r)
	InstallK8sSchedExtenderHandler(r)
}

func timer(f gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		f(c)
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		if raw != "" {
			path = path + "?" + raw
		}
		log.Infof("Handler %q cost: %v", path, time.Since(startTime))
	}
}

func scheduleHandler(c *gin.Context) {
	doSyncSchedule(c)
}

func schedulerActionHandler(c *gin.Context) {
	act := c.Param("action")
	switch act {
	case "test":
		doSchedulerTest(c)
	case "forecast":
		doSchedulerForecast(c)
	case "candidate-list":
		doCandidateList(c)
	case "cleanup":
		doCleanup(c)
	case "history-list":
		doHistoryList(c)
	case "clean-cache":
		doCleanAllHostCache(c)
	case "sync-sku":
		doSyncSku(c)
	//case "reserved-resources":
	//doReservedResources(c)
	default:
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("action: %s not support", act))
	}
}

func schedulerActionIdentHandler(c *gin.Context) {
	act := c.Param("action")
	id := c.Param("ident")
	switch act {
	case "clean-cache":
		doCleanHostCache(c, id)
	case "candidate-detail":
		doCandidateDetail(c, id)
	case "history-detail":
		doHistoryDetail(c, id)
	case "completed":
		doCompleted(c, id)
	default:
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("action: %s not support", act))
	}
}

func doSchedulerTest(c *gin.Context) {
	if !schedman.IsReady() {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("Global scheduler not init"))
		return
	}

	schedInfo, err := api.FetchSchedInfo(c.Request)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	schedInfo.IsSuggestion = true
	result, err := schedman.Schedule(schedInfo)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result.TestResult)
}

func doSchedulerForecast(c *gin.Context) {
	if !schedman.IsReady() {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("Global scheduler not init"))
		return
	}

	schedInfo, err := api.FetchSchedInfo(c.Request)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	schedInfo.IsSuggestion = true
	schedInfo.ShowSuggestionDetails = true
	schedInfo.SuggestionAll = true
	result, err := schedman.Schedule(schedInfo)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, result.ForecastResult)
}

func doCandidateList(c *gin.Context) {
	args, err := api.NewCandidateListArgs(c.Request.Body)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	result, err := schedman.GetCandidateList(args)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func doCandidateDetail(c *gin.Context, id string) {
	hs, err := computemodels.HostManager.FetchById(id)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if hs == nil {
		c.AbortWithError(http.StatusNotFound, fmt.Errorf("Candidate %s not found.", id))
		return
	}

	host := hs.(*computemodels.SHost)

	args := new(api.CandidateDetailArgs)
	args.ID = id
	if host.HostType == computeapi.HOST_TYPE_BAREMETAL {
		args.Type = api.HostTypeBaremetal
	} else {
		args.Type = api.HostTypeHost
	}

	result, err := schedman.GetCandidateDetail(args)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	SendJSON(c, http.StatusOK, result)
}

func doCleanup(c *gin.Context) {
	sjson, err := simplejson.NewFromReader(c.Request.Body)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	args, err := api.NewCleanupArgs(sjson)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	result, err := schedman.Cleanup(args)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func doHistoryList(c *gin.Context) {
	sjson, err := simplejson.NewFromReader(c.Request.Body)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	args, err := api.NewHistoryArgs(sjson)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	result, err := schedman.GetHistoryList(args)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func doHistoryDetail(c *gin.Context, id string) {
	sjson, err := simplejson.NewFromReader(c.Request.Body)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	args, err := api.NewHistoryDetailArgs(sjson, id)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	result, err := schedman.GetHistoryDetail(args)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func doSyncSchedule(c *gin.Context) {
	if !schedman.IsReady() {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("Global scheduler not init"))
		return
	}
	schedInfo, err := api.FetchSchedInfo(c.Request)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	result, err := schedman.Schedule(schedInfo)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, result.Result)
}

func regionResponse(v interface{}) interface{} {
	return struct {
		Result interface{} `json:"scheduler"`
	}{Result: v}
}

func newExpireArgsByHostIDs(ids []string, sid string) (*api.ExpireArgs, error) {
	hs := []computemodels.SHost{}
	q := computemodels.HostManager.Query().In("id", ids)
	if err := db.FetchModelObjects(computemodels.HostManager, q, &hs); err != nil {
		return nil, fmt.Errorf("Fetch hosts by ids %v: %v", ids, err)
	}

	expireArgs := &api.ExpireArgs{
		DirtyBaremetals: []string{},
		DirtyHosts:      []string{},
		SessionId:       sid,
	}
	for _, host := range hs {
		if host.HostType == computeapi.HOST_TYPE_BAREMETAL {
			expireArgs.DirtyBaremetals = append(expireArgs.DirtyBaremetals, host.GetId())
		} else {
			expireArgs.DirtyHosts = append(expireArgs.DirtyHosts, host.GetId())
		}
	}
	return expireArgs, nil
}

func doCleanAllHostCache(c *gin.Context) {
	idMap, err := computemodels.HostManager.Query("id").AllStringMap()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	ids := []string{}
	for _, obj := range idMap {
		ids = append(ids, obj["id"])
	}
	sid := getSessionId(c)
	args, err := newExpireArgsByHostIDs(ids, sid)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	doCleanHostCacheByArgs(c, args)
}

type SyncSkuArgs struct {
	Wait bool `json:"wait"`
}

func doSyncSku(c *gin.Context) {
	args := new(SyncSkuArgs)
	if err := c.BindJSON(args); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if err := skuman.SyncOnce(args.Wait); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, nil)
}

func getSessionId(c *gin.Context) string {
	query, err := jsonutils.ParseQueryString(c.Request.URL.RawQuery)
	if err != nil {
		log.Warningf("not found session id in query")
		return ""
	}
	return jsonutils.GetAnyString(query, []string{"session", "session_id"})
}

func doCleanHostCache(c *gin.Context, hostID string) {
	sid := getSessionId(c)
	args, err := newExpireArgsByHostIDs([]string{hostID}, sid)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	doCleanHostCacheByArgs(c, args)
}

func doCleanHostCacheByArgs(c *gin.Context, args *api.ExpireArgs) {
	result, err := schedman.Expire(args)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, regionResponse(result))
}

func doCompleted(c *gin.Context, id string) {
	sjson, err := simplejson.NewFromReader(c.Request.Body)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	completedNotifyArgs, err := api.NewCompletedNotifyArgs(sjson, id)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	result, err := schedman.CompletedNotify(completedNotifyArgs)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, result)
}
