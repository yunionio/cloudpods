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

package ft

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
	apmodules "yunion.io/x/onecloud/pkg/mcclient/modules/aiproxy"
)

type providerConfigSnapshot struct {
	providerKey string
	config      *jsonutils.JSONDict
}

// ResourceTracker records resources created during a test run for automatic cleanup.
type ResourceTracker struct {
	KeepResources bool

	createdRouting    string
	createdVirtualKey string
	createdAiKey      string
	createdAiModel    string
	createdProvider   string

	providerConfigRestore *providerConfigSnapshot
}

func NewResourceTracker(keepResources bool) *ResourceTracker {
	return &ResourceTracker{KeepResources: keepResources}
}

func envKeepResources(explicit bool) bool {
	if explicit {
		return true
	}
	return envTruthy("AIPROXY_TEST_KEEP_RESOURCES", "AIPROXY_FT_KEEP_RESOURCES")
}

func (t *ResourceTracker) hasCreated() bool {
	if t == nil {
		return false
	}
	return t.createdRouting != "" ||
		t.createdVirtualKey != "" ||
		t.createdAiKey != "" ||
		t.createdAiModel != "" ||
		t.createdProvider != "" ||
		t.providerConfigRestore != nil
}

func (t *ResourceTracker) Cleanup(session *mcclient.ClientSession) {
	if t == nil {
		return
	}
	if t.KeepResources {
		if t.hasCreated() {
			fmt.Println()
			fmt.Println("Keeping test resources (--keep-resources / AIPROXY_TEST_KEEP_RESOURCES=1)")
			t.printKeepHint()
		}
		return
	}
	if !t.hasCreated() {
		return
	}

	fmt.Println()
	Step("cleanup test resources")

	if t.createdRouting != "" {
		t.deleteRouting(session, t.createdRouting)
	}
	if t.createdVirtualKey != "" {
		t.deleteVirtualKey(session, t.createdVirtualKey)
	}
	if t.createdAiKey != "" {
		t.deleteAiKey(session, t.createdAiKey)
	}
	if t.createdAiModel != "" {
		t.deleteAiModel(session, t.createdAiModel)
	}
	if t.providerConfigRestore != nil {
		t.restoreProviderConfig(session)
	}
	if t.createdProvider != "" {
		t.deleteProvider(session, t.createdProvider)
	}
}

func (t *ResourceTracker) printKeepHint() {
	if t.createdRouting != "" {
		fmt.Printf("  climc ai-routing-delete %s\n", t.createdRouting)
	}
	if t.createdVirtualKey != "" {
		fmt.Printf("  climc ai-virtual-key-delete %s\n", t.createdVirtualKey)
	}
	if t.createdAiKey != "" {
		fmt.Printf("  climc ai-key-delete %s\n", t.createdAiKey)
	}
	if t.createdAiModel != "" {
		fmt.Printf("  climc ai-model-delete %s\n", t.createdAiModel)
	}
	if t.createdProvider != "" {
		fmt.Printf("  climc ai-provider-delete %s\n", t.createdProvider)
	}
}

func (t *ResourceTracker) deleteRouting(session *mcclient.ClientSession, name string) {
	if _, err := apmodules.AiRoutings.Delete(session, name, nil); err != nil {
		fmt.Printf("WARN: delete ai_routing %s: %v\n", name, err)
		return
	}
	fmt.Printf("deleted ai_routing %s\n", name)
}

func (t *ResourceTracker) deleteVirtualKey(session *mcclient.ClientSession, name string) {
	if _, err := apmodules.AiVirtualKeys.Delete(session, name, nil); err != nil {
		fmt.Printf("WARN: delete ai_virtual_key %s: %v\n", name, err)
		return
	}
	fmt.Printf("deleted ai_virtual_key %s\n", name)
}

func (t *ResourceTracker) deleteAiKey(session *mcclient.ClientSession, name string) {
	if _, err := apmodules.AiKeys.Delete(session, name, nil); err != nil {
		fmt.Printf("WARN: delete ai_key %s: %v\n", name, err)
		return
	}
	fmt.Printf("deleted ai_key %s\n", name)
}

func (t *ResourceTracker) deleteAiModel(session *mcclient.ClientSession, name string) {
	if _, err := apmodules.AiModels.Delete(session, name, nil); err != nil {
		fmt.Printf("WARN: delete ai_model %s: %v\n", name, err)
		return
	}
	fmt.Printf("deleted ai_model %s\n", name)
}

func (t *ResourceTracker) deleteProvider(session *mcclient.ClientSession, name string) {
	if _, err := apmodules.AiProviders.Delete(session, name, nil); err != nil {
		fmt.Printf("WARN: delete ai_provider %s: %v\n", name, err)
		return
	}
	fmt.Printf("deleted ai_provider %s\n", name)
}

func (t *ResourceTracker) restoreProviderConfig(session *mcclient.ClientSession) {
	snap := t.providerConfigRestore
	if snap == nil {
		return
	}
	params := jsonutils.NewDict()
	if snap.config != nil {
		params.Set("config", snap.config)
	} else {
		params.Set("config", jsonutils.NewDict())
	}
	if _, err := apmodules.AiProviders.Update(session, snap.providerKey, params); err != nil {
		fmt.Printf("WARN: restore ai_provider %s config: %v\n", snap.providerKey, err)
		return
	}
	fmt.Printf("restored ai_provider %s config\n", snap.providerKey)
}

func cloneJSONDict(obj jsonutils.JSONObject) *jsonutils.JSONDict {
	if obj == nil {
		return nil
	}
	if d, ok := obj.(*jsonutils.JSONDict); ok {
		out := jsonutils.NewDict()
		out.Update(d)
		return out
	}
	parsed, err := jsonutils.Parse([]byte(obj.String()))
	if err != nil {
		return nil
	}
	if d, ok := parsed.(*jsonutils.JSONDict); ok {
		return d
	}
	return nil
}

func snapshotProviderConfig(session *mcclient.ClientSession, providerKey string) (*providerConfigSnapshot, error) {
	obj, err := apmodules.AiProviders.Get(session, providerKey, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "ai_provider %s not found", providerKey)
	}
	snap := &providerConfigSnapshot{providerKey: providerKey}
	if obj.Contains("config") {
		cfg, _ := obj.Get("config")
		snap.config = cloneJSONDict(cfg)
	}
	return snap, nil
}
