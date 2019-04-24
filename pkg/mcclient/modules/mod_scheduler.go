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

package modules

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"

	api "yunion.io/x/onecloud/pkg/apis/scheduler"
)

var (
	SchedManager SchedulerManager
)

func init() {
	SchedManager = SchedulerManager{NewSchedulerManager("scheduler", "schedulers",
		[]string{}, []string{})}
	register(&SchedManager)
}

type SchedulerManager struct {
	ResourceManager
}

func (this *SchedulerManager) DoSchedule(s *mcclient.ClientSession, input *api.ScheduleInput, count int) (*api.ScheduleOutput, error) {
	url := fmt.Sprintf("/%s", this.Keyword)
	if count <= 0 {
		count = 1
	}
	input.Count = count
	body := input.JSON(input)
	ret, err := this._post(s, url, body, "")
	if err != nil {
		return nil, err
	}
	output := new(api.ScheduleOutput)
	err = ret.Unmarshal(output)
	if err != nil {
		return nil, fmt.Errorf("Not a valid response: %v", err)
	}
	return output, nil
}

func (this *SchedulerManager) DoScheduleForecast(s *mcclient.ClientSession, params *api.ScheduleInput, count int) (bool, error) {
	if count <= 0 {
		count = 1
	}
	params.Count = count
	res, err := this.DoForecast(s, params.JSON(params))
	if err != nil {
		return false, err
	}
	canCreate := jsonutils.QueryBoolean(res, "can_create", false)
	return canCreate, nil
}

func newSchedURL(action string) string {
	return fmt.Sprintf("/scheduler/%s", action)
}

func newSchedIdentURL(action, ident string) string {
	return fmt.Sprintf("%s/%s", newSchedURL(action), ident)
}

func (this *SchedulerManager) Test(s *mcclient.ClientSession, params *api.ScheduleInput) (jsonutils.JSONObject, error) {
	url := newSchedURL("test")
	_, obj, err := this.jsonRequest(s, "POST", url, nil, params.JSON(params))
	if err != nil {
		return nil, err
	}
	return obj, err
}

func (this *SchedulerManager) DoForecast(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	url := newSchedURL("forecast")
	_, obj, err := this.jsonRequest(s, "POST", url, nil, params)
	if err != nil {
		return nil, err
	}
	return obj, err
}

func (this *SchedulerManager) Cleanup(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	url := newSchedURL("cleanup")
	return this._post(s, url, params, "")
}

func (this *SchedulerManager) SyncSku(s *mcclient.ClientSession, wait bool) (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewBool(wait), "wait")
	url := newSchedURL("sync-sku")
	return this._post(s, url, params, "")
}

func (this *SchedulerManager) Kill(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, fmt.Errorf("Not impl")
}

func (this *SchedulerManager) CandidateList(s *mcclient.ClientSession, params jsonutils.JSONObject) (obj jsonutils.JSONObject, err error) {
	url := newSchedURL("candidate-list")
	_, obj, err = this.jsonRequest(s, "POST", url, nil, params)
	if err != nil {
		return
	}

	_parse := func(property string, o jsonutils.JSONObject) (key, val string, err error) {
		omap, err := o.GetMap()
		if err != nil {
			return
		}
		resObj, ok := omap[property]
		if !ok {
			err = fmt.Errorf("Get key %q error", property)
			return
		}
		free, err := resObj.Int("free")
		reserverd, err := resObj.Int("reserverd")
		total, err := resObj.Int("total")
		if err != nil {
			return
		}
		key = fmt.Sprintf("%s(free/reserverd/total)", property)
		val = fmt.Sprintf("%d/%d/%d", free, reserverd, total)
		return
	}

	parseAdd := func(o jsonutils.JSONObject) error {
		odict := o.(*jsonutils.JSONDict)
		for _, k := range []string{"cpu", "mem", "storage"} {
			k, v, err := _parse(k, o)
			if err != nil {
				return err
			}
			odict.Add(jsonutils.NewString(v), k)
		}
		return nil
	}

	aggregate := func(result jsonutils.JSONObject) error {
		data, _ := result.GetArray("data")
		for _, o := range data {
			err := parseAdd(o)
			if err != nil {
				return err
			}
		}
		return nil
	}
	err = aggregate(obj)
	return
}

func (this *SchedulerManager) CandidateDetail(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	url := newSchedIdentURL("candidate-detail", id)
	return this._post(s, url, params, "candidate")
}

func (this *SchedulerManager) HistoryList(s *mcclient.ClientSession, params jsonutils.JSONObject) (obj jsonutils.JSONObject, err error) {
	url := newSchedURL("history-list")
	_, obj, err = this.jsonRequest(s, "POST", url, nil, params)
	if err != nil {
		return
	}
	return
}

func (this *SchedulerManager) HistoryShow(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	url := newSchedIdentURL("history-detail", id)
	return this._post(s, url, params, "history")
}

func (this *SchedulerManager) CleanCache(s *mcclient.ClientSession, hostId string) error {
	url := newSchedURL("clean-cache")
	if len(hostId) > 0 {
		url = fmt.Sprintf("%s/%s", url, hostId)
	}
	resp, err := this.rawRequest(s, "POST", url, nil, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
