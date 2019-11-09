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

package app

import (
	"context"
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func getServiceIdByType(s *mcclient.ClientSession, typeStr string, verStr string) (string, error) {
	params := jsonutils.NewDict()
	if len(verStr) > 0 {
		typeStr += "_" + verStr
	}
	params.Add(jsonutils.NewString(typeStr), "type")
	result, err := modules.ServicesV3.List(s, params)
	if err != nil {
		return "", errors.Wrap(err, "modules.ServicesV3.List")
	}
	if len(result.Data) == 0 {
		return "", errors.Wrap(sql.ErrNoRows, "modules.ServicesV3.List")
	} else if len(result.Data) > 1 {
		return "", errors.Wrap(sqlchemy.ErrDuplicateEntry, "modules.ServicesV3.List")
	}
	return result.Data[0].GetString("id")
}

func getServiceConfig(s *mcclient.ClientSession, serviceId string) (jsonutils.JSONObject, error) {
	conf, err := modules.ServicesV3.GetSpecific(s, serviceId, "config", nil)
	if err != nil {
		return nil, errors.Wrap(err, "modules.ServicesV3.GetSpecific config")
	}
	defConf, _ := conf.Get("config", "default")
	return defConf, nil
}

func MergeServiceConfig(opts interface{}, serviceType string, serviceVersion string) error {
	merged := false
	conf := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	region, _ := conf.GetString("region")
	epType, _ := conf.GetString("session_endpoint_type")
	s := auth.AdminSession(context.Background(), region, "", epType, "")
	serviceId, _ := getServiceIdByType(s, serviceType, serviceVersion)
	if len(serviceId) > 0 {
		serviceConf, err := getServiceConfig(s, serviceId)
		if err != nil {
			return errors.Wrap(err, "getServiceConfig")
		}
		conf.Update(serviceConf)
		merged = true
	}
	commonServiceId, _ := getServiceIdByType(s, consts.COMMON_SERVICE, "")
	if len(commonServiceId) > 0 {
		commonConf, err := getServiceConfig(s, commonServiceId)
		if err != nil {
			return errors.Wrap(err, "getServiceConfig common service")
		}
		conf.Update(commonConf)
		merged = true
	}
	if merged {
		err := conf.Unmarshal(opts)
		if err != nil {
			return errors.Wrap(err, "conf.Unmarshal")
		}
		if len(serviceId) > 0 {
			nconf := jsonutils.NewDict()
			nconf.Add(conf, "config", "default")
			_, err := modules.ServicesV3.PerformAction(s, serviceId, "config", nconf)
			if err != nil {
				return errors.Wrap(err, "modules.ServicesV3.PerformAction")
			}
		}
	}
	return nil
}
