package modules

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient"
	"github.com/yunionio/pkg/utils"
)

type SpecsManager struct {
	ResourceManager
}

func generateSpecURL(model, ident, action string, params jsonutils.JSONObject) string {
	url := utils.ComposeURL("specs", model, ident, action)
	if params != nil {
		qs := params.QueryString()
		if len(qs) > 0 {
			url = fmt.Sprintf("%s?%s", url, qs)
		}
	}
	return url
}

func newSpecActionURL(model, ident, action string, params jsonutils.JSONObject) string {
	return generateSpecURL(model, ident, action, params)
}

func newSpecURL(model string, params jsonutils.JSONObject) string {
	return generateSpecURL(model, "", "", params)
}

func (this *SpecsManager) GetHostSpecs(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.GetModelSpecs(s, "hosts", params)
}

func (this *SpecsManager) GetIsolatedDevicesSpecs(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.GetModelSpecs(s, "isolated_devices", params)
}

func (this *SpecsManager) GetAllSpecs(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.GetModelSpecs(s, "", params)
}

func (this *SpecsManager) GetModelSpecs(s *mcclient.ClientSession, model string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	url := newSpecURL(model, params)
	return this._get(s, url, this.Keyword)
}

func (this *SpecsManager) SpecsQueryModelObjects(s *mcclient.ClientSession, model string, specKeys []string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if len(specKeys) == 0 {
		return nil, fmt.Errorf("Spec keys must provided")
	}
	specKey := url.QueryEscape(strings.Join(specKeys, "/"))
	url := newSpecActionURL(model, specKey, "resource", params)
	return this._get(s, url, this.Keyword)
}

var (
	Specs SpecsManager
)

func init() {
	Specs = SpecsManager{NewComputeManager("spec", "specs",
		[]string{},
		[]string{})}

	registerCompute(&Specs)
}
