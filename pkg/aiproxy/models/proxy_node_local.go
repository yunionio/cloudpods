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

package models

import (
	"context"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/aiproxy/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var localProxyNodeId string

// InitLocalProxyNodeId records the ai_proxy_node id for this running aiproxy process.
func InitLocalProxyNodeId(opts *options.SAiProxyOptions, isSlave bool) error {
	if isSlave {
		addr, err := AdvertiseAddressFromOptions(opts)
		if err != nil {
			return err
		}
		localProxyNodeId = aiProxyNodeId(addr)
		return nil
	}
	localProxyNodeId = defaultPrimaryAiProxyNodeId
	return nil
}

// CurrentProxyNodeId returns the ai_proxy_node id of this process.
func CurrentProxyNodeId() string {
	return localProxyNodeId
}

func validateAiProxyNodeId(ctx context.Context, userCred mcclient.TokenCredential, idOrName string) (string, error) {
	idOrName = strings.TrimSpace(idOrName)
	if idOrName == "" {
		return "", nil
	}
	obj, err := AiProxyNodeManager.FetchByIdOrName(ctx, userCred, idOrName)
	if err != nil {
		return "", errors.Wrap(err, "fetch ai_proxy_node")
	}
	node := obj.(*SAiProxyNode)
	if !node.GetEnabled() {
		return "", errors.Wrapf(httperrors.ErrInvalidStatus, "ai_proxy_node %q is disabled", idOrName)
	}
	return node.Id, nil
}

func resolveAiProxyNodeIdForCreate(ctx context.Context, userCred mcclient.TokenCredential, idOrName string) (string, error) {
	if strings.TrimSpace(idOrName) == "" {
		idOrName = defaultPrimaryAiProxyNodeId
	}
	return validateAiProxyNodeId(ctx, userCred, idOrName)
}

func proxyNodeScopeMatches(routingNodeId, currentNodeId string) bool {
	routingNodeId = strings.TrimSpace(routingNodeId)
	if routingNodeId == "" {
		return true
	}
	return routingNodeId == strings.TrimSpace(currentNodeId)
}
