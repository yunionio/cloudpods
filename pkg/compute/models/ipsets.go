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
	"net"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=ipset
// +onecloud:swagger-gen-model-plural=ipsets
type SIpSetManager struct {
	db.SSharableVirtualResourceBaseManager
}

var IpSetManager *SIpSetManager

func init() {
	IpSetManager = &SIpSetManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SIpSet{},
			"ipsets_tbl",
			"ipset",
			"ipsets",
		),
	}
	IpSetManager.SetVirtualObject(IpSetManager)
}

type SIpSet struct {
	db.SSharableVirtualResourceBase

	// IP集合类型
	IpSetType api.TIpSetType `width:"32" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	// IP/CIDR 列表，逗号分隔
	Data string `width:"2048" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
}

// IP集合列表
func (manager *SIpSetManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.IpSetListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	if len(query.IpSetType) > 0 {
		types := make([]string, 0, len(query.IpSetType))
		for _, t := range query.IpSetType {
			types = append(types, string(t))
		}
		q = q.In("ip_set_type", types)
	}
	if len(query.Ip) > 0 {
		q = q.Contains("data", query.Ip)
	}
	return q, nil
}

func (manager *SIpSetManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.IpSetListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SSharableVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SIpSetManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SSharableVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SIpSetManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.IpSetDetails {
	rows := make([]api.IpSetDetails, len(objs))
	virtRows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.IpSetDetails{
			SharableVirtualResourceDetails: virtRows[i],
			SecurityGroupCount:             objs[i].(*SIpSet).GetSecurityGroupCount(),
		}
	}
	return rows
}

func (manager *SIpSetManager) ListItemExportKeys(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SSharableVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (manager *SIpSetManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input *api.IpSetCreateInput,
) (*api.IpSetCreateInput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	input.Data = normalizeIpSetData(input.IpSetType, input.Data)
	if len(input.Data) == 0 {
		return nil, httperrors.NewInputParameterError("Data is empty")
	}

	var err error
	input.SharableVirtualResourceCreateInput, err = manager.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return nil, err
	}
	return input, nil
}

func (ipset *SIpSet) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.IpSetUpdateInput,
) (*api.IpSetUpdateInput, error) {
	if input.Data != nil {
		normalized := normalizeIpSetData(ipset.IpSetType, *input.Data)
		if len(normalized) == 0 {
			return nil, httperrors.NewInputParameterError("Data is empty")
		}
		input.Data = &normalized
	}

	var err error
	input.SharableVirtualResourceBaseUpdateInput, err = ipset.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.SharableVirtualResourceBaseUpdateInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func normalizeIpSetData(ipSetType api.TIpSetType, data string) string {
	parts := strings.Split(data, ",")
	normalized := make([]string, 0, len(parts))
	for i := range parts {
		cidr := strings.TrimSpace(parts[i])
		if len(cidr) == 0 {
			continue
		}
		switch ipSetType {
		case api.IpSetTypeIpv4CidrList:
			if pref, err := netutils.NewIPV4Prefix(cidr); err == nil {
				normalized = append(normalized, pref.String())
			}
		case api.IpSetTypeIpv6CidrList:
			if pref, err := netutils.NewIPV6Prefix(cidr); err == nil {
				normalized = append(normalized, pref.String())
			}
		}
	}
	return strings.Join(normalized, ",")
}

func (ipset *SIpSet) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if info != nil && info.Contains("security_group_count") {
		securityGroupCount, _ := info.Int("security_group_count")
		if securityGroupCount > 0 {
			return errors.Wrapf(httperrors.ErrNotEmpty, "ipset is being used by %d security groups", securityGroupCount)
		}
	} else if ipset.GetSecurityGroupCount() > 0 {
		return errors.Wrapf(httperrors.ErrNotEmpty, "ipset is being used by %d security groups", ipset.GetSecurityGroupCount())
	}
	err := ipset.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx, info)
	if err != nil {
		return errors.Wrapf(err, "SSharableVirtualResourceBase.ValidateDeleteCondition")
	}
	return nil
}

func (ipset *SIpSet) GetSecurityGroupCount() int {
	q := SecurityGroupRuleManager.Query("secgroup_id").Equals("target_type", api.SecurityGroupRuleTargetTypeIpSet).Equals("cidr", ipset.Id).Distinct()
	return q.Count()
}

func (ipset *SIpSet) getIpNets() []*net.IPNet {
	return netutils2.Str2IPNets(ipset.Data)
}
