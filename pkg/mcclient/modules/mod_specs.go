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
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type SpecsManager struct {
	modulebase.ResourceManager
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
	return modulebase.Get(this.ResourceManager, s, url, this.Keyword)
}

func (this *SpecsManager) GetObjects(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	dict := params.(*jsonutils.JSONDict)
	model, err := dict.GetString("kind")
	if err != nil {
		return nil, httperrors.NewInputParameterError("Not found kind in query: %v", err)
	}
	dict.Remove("kind")
	specKey, err := params.GetString("key")
	if err != nil {
		return nil, httperrors.NewInputParameterError("Not found key in query: %v", err)
	}
	dict.Remove("key")
	specKey = url.QueryEscape(specKey)
	url := newSpecActionURL(model, specKey, "resource", dict)
	return modulebase.Get(this.ResourceManager, s, url, this.Keyword)
}

func (this *SpecsManager) SpecsQueryModelObjects(s *mcclient.ClientSession, model string, specKeys []string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if len(specKeys) == 0 {
		return nil, fmt.Errorf("Spec keys must provided")
	}
	specKey := url.QueryEscape(strings.Join(specKeys, "/"))
	url := newSpecActionURL(model, specKey, "resource", params)
	return modulebase.Get(this.ResourceManager, s, url, this.Keyword)
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
