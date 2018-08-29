package k8s

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Charts *ChartManager
)

type ChartManager struct {
	*ResourceManager
}

func init() {
	Charts = &ChartManager{NewResourceManager("chart", "charts",
		NewColumns("RepoWithName", "Version", "Description"),
		NewColumns())}
	modules.Register(Charts)
}

func (m ChartManager) GetRepoWithName(obj jsonutils.JSONObject) interface{} {
	repo, _ := obj.GetString("repo")
	name, _ := obj.GetString("chart", "name")
	return fmt.Sprintf("%s/%s", repo, name)
}

func (m ChartManager) GetVersion(obj jsonutils.JSONObject) interface{} {
	version, _ := obj.GetString("chart", "version")
	return version
}

func (m ChartManager) GetDescription(obj jsonutils.JSONObject) interface{} {
	desc, _ := obj.GetString("chart", "description")
	return desc
}
