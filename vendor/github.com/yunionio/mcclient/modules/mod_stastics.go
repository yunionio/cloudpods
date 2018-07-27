package modules

import (
	"fmt"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient"
)

type StatisticsManager struct {
	ResourceManager
}

var (
	Statistics StatisticsManager
)

func (this *StatisticsManager) GetByEnv(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	node_labels, err := params.GetString("node_labels")
	ret := jsonutils.NewDict()
	if err != nil {
		return ret, err
	}

	path := fmt.Sprintf("/%s/env?node_labels=%s", this.KeywordPlural, node_labels)
	return this._get(s, path, this.Keyword)
}

func (this *StatisticsManager) GetByResType(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	node_labels, err := params.GetString("node_labels")
	ret := jsonutils.NewDict()
	if err != nil {
		return ret, err
	}

	path := fmt.Sprintf("/%s/res_type?node_labels=%s", this.KeywordPlural, node_labels)
	return this._get(s, path, this.Keyword)
}

func (this *StatisticsManager) GetHardware(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	node_labels, err := params.GetString("node_labels")
	ret := jsonutils.NewDict()
	if err != nil {
		return ret, err
	}

	path := fmt.Sprintf("/%s/hardware?node_labels=%s", this.KeywordPlural, node_labels)
	return this._get(s, path, this.Keyword)
}

func init() {
	Statistics = StatisticsManager{NewMonitorManager("statistic", "statistics",
		[]string{"ID"},
		[]string{})}

	register(&Statistics)
}
