package modules

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
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

func (this *SchedulerManager) DoScheduleListResult(s *mcclient.ClientSession, params jsonutils.JSONObject, count int) (*ListResult, error) {
	candidates, err := this.DoSchedule(s, params, count)
	if err != nil {
		return nil, err
	}
	ret := ListResult{Data: make([]jsonutils.JSONObject, len(candidates))}
	for i, candidate := range candidates {
		host, err := candidate.Get("candidate")
		if err == nil {
			ret.Data[i] = host
		} else {
			ret.Data[i] = candidate
		}
	}
	return &ret, nil
}

func (this *SchedulerManager) DoSchedule(s *mcclient.ClientSession, params jsonutils.JSONObject, count int) ([]jsonutils.JSONObject, error) {
	url := fmt.Sprintf("/%s", this.Keyword)
	body := jsonutils.NewDict()
	body.Add(params, this.Keyword)
	if count <= 0 {
		count = 1
	}
	body.Add(jsonutils.NewInt(int64(count)), "count")
	cands, err := this._post(s, url, body, this.Keyword)
	if err != nil {
		return nil, err
	}
	candidates, err := cands.GetArray()
	if err != nil {
		return nil, fmt.Errorf("Not a valid response")
	}
	return candidates, nil
}

func newSchedURL(action string) string {
	return fmt.Sprintf("/scheduler/%s", action)
}

func newSchedIdentURL(action, ident string) string {
	return fmt.Sprintf("%s/%s", newSchedURL(action), ident)
}

func (this *SchedulerManager) Test(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	url := newSchedURL("test")
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
	_, err := this.rawRequest(s, "POST", url, nil, nil)
	return err
}
