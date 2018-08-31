package modules

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	WebConsole WebConsoleManager
)

func init() {
	WebConsole = WebConsoleManager{NewWebConsoleManager()}
}

type WebConsoleManager struct {
	ResourceManager
}

func NewWebConsoleManager() ResourceManager {
	return ResourceManager{BaseManager: BaseManager{serviceType: "webconsole"},
		Keyword: "webconsole", KeywordPlural: "webconsole"}
}

func (m WebConsoleManager) DoConnect(
	s *mcclient.ClientSession,
	connType, id, action string,
	params jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	if len(connType) == 0 {
		return nil, fmt.Errorf("Empty connection resource type")
	}
	url := fmt.Sprintf("/webconsole/%s", connType)
	if id != "" {
		url = fmt.Sprintf("%s/%s", url, id)
	}
	if action != "" {
		url = fmt.Sprintf("%s/%s", url, action)
	}
	return m._post(s, url, params, "webconsole")
}

func (m WebConsoleManager) DoK8sConnect(
	s *mcclient.ClientSession,
	id, action string,
	params jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	return m.DoConnect(s, "k8s", id, action, params)
}

func (m WebConsoleManager) DoK8sShellConnect(
	s *mcclient.ClientSession,
	id string, params jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	return m.DoK8sConnect(s, id, "shell", params)
}

func (m WebConsoleManager) DoK8sLogConnect(
	s *mcclient.ClientSession,
	id string, params jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	return m.DoK8sConnect(s, id, "log", params)
}
