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

package lbagent

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/version"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/netutils2"
)

func register(ctx context.Context, opts *Options) (string, error) {
	_, ips, err := netutils2.WaitIfaceIps(opts.ListenInterface)
	if err != nil {
		return "", errors.Wrap(err, "netutils2.WaitIfaceIps")
	}

	if len(ips) == 0 {
		return "", errors.Wrapf(httperrors.ErrNotFound, "no valid ip on interface %s", opts.ListenInterface)
	}

	if len(ips) > 1 && len(opts.AccessIp) == 0 {
		return "", errors.Wrap(httperrors.ErrDuplicateResource, "multiple IPs, must specified a valid access IP")
	}

	if len(opts.AccessIp) == 0 {
		opts.AccessIp = ips[0].String()
	} else {
		find := false
		for i := range ips {
			if ips[i].String() == opts.AccessIp {
				find = true
				break
			}
		}
		if !find {
			return "", errors.Wrapf(httperrors.ErrConflict, "access IP %s not present on interface %s", opts.AccessIp, opts.ListenInterface)
		}
	}

	s := auth.GetAdminSession(ctx, opts.Region)
	params := jsonutils.NewDict()
	params.Set(api.LBAGENT_QUERY_ORIG_KEY, jsonutils.NewString(api.LBAGENT_QUERY_ORIG_VAL))
	params.Set("ip", jsonutils.NewString(opts.AccessIp))
	results, err := modules.LoadbalancerAgents.List(s, params)
	if err != nil {
		return "", errors.Wrap(err, "LoadbalancerAgents.List")
	}
	if len(results.Data) > 1 {
		// multiple lbagent with ident IP? conflict
		return "", errors.Wrapf(httperrors.ErrDuplicateResource, "multiple lbagent with same IP %s", opts.AccessIp)
	}
	if len(results.Data) == 0 {
		// not found, to create a new one
		createParams := jsonutils.NewDict()
		createParams.Set("generate_name", jsonutils.NewString("lbagent"))
		createParams.Set("ip", jsonutils.NewString(opts.AccessIp))
		createParams.Set("interface", jsonutils.NewString(opts.ListenInterface))
		createParams.Set("version", jsonutils.NewString(version.Get().GitVersion))
		data, err := modules.LoadbalancerAgents.Create(s, createParams)
		if err != nil {
			return "", errors.Wrap(err, "LoadbalancerAgents.Create")
		}
		results.Data = []jsonutils.JSONObject{data}
	}
	idstr, _ := results.Data[0].GetString("id")
	log.Infof("register lbagent with ID %s", idstr)
	return idstr, nil
}
