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
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/aiproxy/chatlog"
	"yunion.io/x/onecloud/pkg/aiproxy/options"
	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
	defaultAiProxyNodeHbTimeout    = 120
	defaultPrimaryAiProxyNodeId    = "primary"
	maxAiProxyNodeAccessAddressLen = 256
)

// SAiProxyNode records an aiproxy instance reachable address and optional public access URL.
type SAiProxyNode struct {
	db.SEnabledStatusStandaloneResourceBase

	Address       string    `width:"256" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user" index:"true"`
	AccessAddress string    `width:"256" charset:"utf8" nullable:"true" list:"user" create:"optional" update:"user"`
	LastSeen      time.Time `nullable:"true" list:"user"`
	HbTimeout     int       `nullable:"false" default:"120" list:"user" create:"optional" update:"user"`
}

type SAiProxyNodeManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
}

var AiProxyNodeManager *SAiProxyNodeManager

func init() {
	AiProxyNodeManager = &SAiProxyNodeManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SAiProxyNode{},
			"ai_proxy_nodes_tbl",
			"ai_proxy_node",
			"ai_proxy_nodes",
		),
	}
	AiProxyNodeManager.SetVirtualObject(AiProxyNodeManager)
}

func (manager *SAiProxyNodeManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeUser
}

func (manager *SAiProxyNodeManager) InitializeData() error {
	ctx := context.Background()
	addr, err := AdvertiseAddressFromOptions(nil)
	if err != nil {
		return err
	}
	node := SAiProxyNode{}
	node.SetModelManager(manager, &node)
	node.Id = defaultPrimaryAiProxyNodeId
	node.Name = defaultPrimaryAiProxyNodeId
	node.Description = "Default primary aiproxy node"
	node.Address = addr
	accessAddress := ""
	if existing, err := manager.FetchById(defaultPrimaryAiProxyNodeId); err == nil {
		accessAddress = strings.TrimSpace(existing.(*SAiProxyNode).AccessAddress)
	}
	if accessAddress == "" {
		a, err := AccessAddressFromApiServer(nil)
		if err != nil {
			return err
		}
		accessAddress = a
	}
	node.AccessAddress = accessAddress
	node.HbTimeout = defaultAiProxyNodeHbTimeout
	node.LastSeen = time.Now()
	node.SetEnabled(true)
	node.Status = apis.STATUS_AVAILABLE
	node.Progress = 100
	if err := manager.TableSpec().InsertOrUpdate(ctx, &node); err != nil {
		return errors.Wrap(err, "insert or update default primary ai_proxy_node")
	}
	return nil
}

func aiProxyNodeId(address string) string {
	return stringutils2.GenId("aiproxy.node", address)
}

func normalizeAiProxyNodeAddress(raw string) (string, error) {
	address := strings.TrimSpace(raw)
	if address == "" {
		return "", errors.Wrap(httperrors.ErrInputParameter, "address is required")
	}
	if strings.Contains(address, "://") {
		u, err := url.Parse(address)
		if err != nil {
			return "", errors.Wrapf(httperrors.ErrInputParameter, "invalid address URL: %v", err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return "", errors.Wrap(httperrors.ErrInputParameter, "address scheme must be http or https")
		}
		if strings.TrimSpace(u.Host) == "" {
			return "", errors.Wrap(httperrors.ErrInputParameter, "address must include host")
		}
		return strings.TrimRight(address, "/"), nil
	}
	if _, _, err := net.SplitHostPort(address); err != nil {
		return "", errors.Wrapf(httperrors.ErrInputParameter, "invalid address %q: %v", address, err)
	}
	return fmt.Sprintf("http://%s", address), nil
}

func normalizeAiProxyNodeAccessAddress(raw string) (string, error) {
	accessAddress := strings.TrimSpace(raw)
	if accessAddress == "" {
		return "", nil
	}
	if len(accessAddress) > maxAiProxyNodeAccessAddressLen {
		return "", errors.Wrapf(httperrors.ErrInputParameter, "access_address too long (max %d)", maxAiProxyNodeAccessAddressLen)
	}
	return normalizeAiProxyNodeAddress(accessAddress)
}

func aiProxyNodeDisplayName(address string) string {
	u, err := url.Parse(address)
	if err != nil || strings.TrimSpace(u.Host) == "" {
		return address
	}
	return u.Host
}

// AdvertiseAddressFromOptions returns the service URL advertised by this instance.
func AdvertiseAddressFromOptions(opts *options.SAiProxyOptions) (string, error) {
	if opts == nil {
		opts = &options.Options
	}
	if addr := strings.TrimRight(strings.TrimSpace(opts.AdvertiseAddress), "/"); addr != "" {
		return normalizeAiProxyNodeAddress(addr)
	}
	scheme := "http"
	if opts.EnableSsl {
		scheme = "https"
	}
	host := strings.TrimSpace(opts.Address)
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	return normalizeAiProxyNodeAddress(fmt.Sprintf("%s://%s:%d", scheme, host, opts.Port))
}

// AccessAddressFromApiServer derives ai_proxy_node.access_address from --api-server.
func AccessAddressFromApiServer(opts *options.SAiProxyOptions) (string, error) {
	if opts == nil {
		opts = &options.Options
	}
	raw := strings.TrimSpace(opts.ApiServer)
	if raw == "" {
		return "", nil
	}
	return normalizeAiProxyNodeAccessAddress(raw)
}

func (manager *SAiProxyNodeManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AiProxyNodeListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.ListItemFilter")
	}
	if addr := strings.TrimSpace(query.Address); addr != "" {
		q = q.Equals("address", addr)
	}
	if accessAddress := strings.TrimSpace(query.AccessAddress); accessAddress != "" {
		q = q.Equals("access_address", accessAddress)
	}
	return q, nil
}

func (manager *SAiProxyNodeManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.AiProxyNodeDetails {
	rows := make([]api.AiProxyNodeDetails, len(objs))
	baseRows := manager.SEnabledStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range objs {
		rows[i].EnabledStatusStandaloneResourceDetails = baseRows[i]
		node := objs[i].(*SAiProxyNode)
		rows[i].IsActive = node.IsActive()
	}
	return rows
}

func (manager *SAiProxyNodeManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.AiProxyNodeCreateInput,
) (api.AiProxyNodeCreateInput, error) {
	var err error
	input.EnabledStatusStandaloneResourceCreateInput, err = manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusStandaloneResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData")
	}
	input.Address, err = normalizeAiProxyNodeAddress(input.Address)
	if err != nil {
		return input, err
	}
	input.AccessAddress, err = normalizeAiProxyNodeAccessAddress(input.AccessAddress)
	if err != nil {
		return input, err
	}
	if input.HbTimeout <= 0 {
		input.HbTimeout = defaultAiProxyNodeHbTimeout
	}
	if strings.TrimSpace(input.Name) == "" {
		input.Name = aiProxyNodeDisplayName(input.Address)
	}
	return input, nil
}

func (manager *SAiProxyNodeManager) PerformRegister(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.AiProxyNodeRegisterInput,
) (jsonutils.JSONObject, error) {
	addr, err := normalizeAiProxyNodeAddress(input.Address)
	if err != nil {
		return nil, err
	}
	hbTimeout := input.HbTimeout
	if hbTimeout <= 0 {
		hbTimeout = defaultAiProxyNodeHbTimeout
	}
	nodeId := aiProxyNodeId(addr)
	accessAddress := ""
	if existing, err := manager.FetchById(nodeId); err == nil {
		accessAddress = existing.(*SAiProxyNode).AccessAddress
	} else if errors.Cause(err) != sql.ErrNoRows {
		return nil, errors.Wrap(err, "fetch ai_proxy_node")
	}
	node := SAiProxyNode{}
	node.SetModelManager(manager, &node)
	node.Id = nodeId
	node.Name = aiProxyNodeDisplayName(addr)
	node.Address = addr
	node.AccessAddress = accessAddress
	node.HbTimeout = hbTimeout
	node.LastSeen = time.Now()
	node.SetEnabled(true)
	node.Status = apis.STATUS_AVAILABLE
	node.Progress = 100
	if err := manager.TableSpec().InsertOrUpdate(ctx, &node); err != nil {
		return nil, errors.Wrap(err, "insert or update ai_proxy_node")
	}
	return jsonutils.Marshal(api.AiProxyNodeRegisterOutput{Id: node.Id}), nil
}

func (node *SAiProxyNode) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.AiProxyNodeUpdateInput,
) (*api.AiProxyNodeUpdateInput, error) {
	var err error
	input.EnabledStatusStandaloneResourceBaseUpdateInput, err = node.SEnabledStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.EnabledStatusStandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEnabledStatusStandaloneResourceBase.ValidateUpdateData")
	}
	if input.Address != "" {
		input.Address, err = normalizeAiProxyNodeAddress(input.Address)
		if err != nil {
			return input, err
		}
	}
	if query.Contains("access_address") {
		input.AccessAddress, err = normalizeAiProxyNodeAccessAddress(input.AccessAddress)
		if err != nil {
			return input, err
		}
	}
	if input.HbTimeout < 0 {
		return input, errors.Wrap(httperrors.ErrInputParameter, "hb_timeout must be >= 0")
	}
	return input, nil
}

func (node *SAiProxyNode) IsActive() bool {
	if node.Id == defaultPrimaryAiProxyNodeId {
		return node.GetEnabled()
	}
	if node.LastSeen.IsZero() {
		return false
	}
	timeout := node.HbTimeout
	if timeout <= 0 {
		timeout = defaultAiProxyNodeHbTimeout
	}
	return int(time.Since(node.LastSeen).Seconds()) < timeout
}

func (node *SAiProxyNode) GetDetailsChatLogs(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	input := struct {
		Start     string `json:"start"`
		End       string `json:"end"`
		Limit     int    `json:"limit"`
		RequestID string `json:"request_id"`
	}{}
	if query != nil {
		if err := query.Unmarshal(&input); err != nil {
			return nil, errors.Wrap(httperrors.ErrInputParameter, err.Error())
		}
	}
	var start time.Time
	if strings.TrimSpace(input.Start) != "" {
		ts, err := time.Parse(time.RFC3339, input.Start)
		if err != nil {
			return nil, errors.Wrapf(httperrors.ErrInputParameter, "invalid start: %v", err)
		}
		start = ts
	}
	var end time.Time
	if strings.TrimSpace(input.End) != "" {
		ts, err := time.Parse(time.RFC3339, input.End)
		if err != nil {
			return nil, errors.Wrapf(httperrors.ErrInputParameter, "invalid end: %v", err)
		}
		end = ts
	}
	ret, err := chatlog.Read(ctx, chatlog.ReadOptions{
		Start:     start,
		End:       end,
		Limit:     input.Limit,
		RequestID: input.RequestID,
		Instance:  node.Id,
	})
	if err != nil {
		return nil, errors.Wrap(err, "read aiproxy chat logs")
	}
	return jsonutils.Marshal(ret), nil
}
