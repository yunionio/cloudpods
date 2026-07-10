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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
	apmodules "yunion.io/x/onecloud/pkg/mcclient/modules/aiproxy"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
)

func normalizeAiproxyBaseURL(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}

// PickProxyNodeBaseURL prefers access_address (client-facing URL) over internal address.
func PickProxyNodeBaseURL(accessAddress, address string) string {
	if u := normalizeAiproxyBaseURL(accessAddress); u != "" {
		return u
	}
	return normalizeAiproxyBaseURL(address)
}

// ResolveRoutingProxyNodeURL returns the ai_proxy_node access URL bound to an ai_routing.
func ResolveRoutingProxyNodeURL(session *mcclient.ClientSession, routingNameOrID string) (string, error) {
	routingNameOrID = strings.TrimSpace(routingNameOrID)
	if routingNameOrID == "" {
		return "", nil
	}
	obj, err := apmodules.AiRoutings.Get(session, routingNameOrID, nil)
	if err != nil {
		return "", errors.Wrapf(err, "ai-routing-show %s", routingNameOrID)
	}
	nodeID, _ := obj.GetString("ai_proxy_node_id")
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return "", nil
	}
	nodeObj, err := apmodules.AiProxyNodes.Get(session, nodeID, nil)
	if err != nil {
		return "", errors.Wrapf(err, "ai-proxy-node-show %s", nodeID)
	}
	access, _ := nodeObj.GetString("access_address")
	addr, _ := nodeObj.GetString("address")
	url := PickProxyNodeBaseURL(access, addr)
	if url == "" {
		return "", nil
	}
	return url, nil
}

// ResolveAiproxyURLForRouting resolves the client-facing aiproxy base URL.
// When routing is set, prefers the bound ai_proxy_node access_address over Keystone public endpoint.
func ResolveAiproxyURLForRouting(session *mcclient.ClientSession, override, routing string) (string, error) {
	if override = normalizeAiproxyBaseURL(override); override != "" {
		return override, nil
	}
	if routing = strings.TrimSpace(routing); routing != "" {
		url, err := ResolveRoutingProxyNodeURL(session, routing)
		if err != nil {
			return "", err
		}
		if url != "" {
			return url, nil
		}
	}
	return ResolveAiproxyURL(session, "")
}

func ResolveAiproxyURL(session *mcclient.ClientSession, override string) (string, error) {
	if override = normalizeAiproxyBaseURL(override); override != "" {
		return override, nil
	}
	if url := resolveAiproxyURLFromEnv(); url != "" {
		return url, nil
	}
	query := jsonutils.NewDict()
	query.Set("service", jsonutils.NewString("aiproxy"))
	query.Set("interface", jsonutils.NewString("public"))
	query.Set("limit", jsonutils.NewInt(1))
	result, err := modules.EndpointsV3.List(session, query)
	if err != nil {
		return "", errors.Wrap(err, "endpoint-list aiproxy public")
	}
	if len(result.Data) == 0 {
		return "", errors.Error("cannot resolve aiproxy public URL; set AIPROXY_URL")
	}
	url, _ := result.Data[0].GetString("url")
	url = strings.TrimRight(strings.TrimSpace(url), "/")
	if url == "" {
		return "", errors.Error("cannot resolve aiproxy public URL; set AIPROXY_URL")
	}
	return url, nil
}
