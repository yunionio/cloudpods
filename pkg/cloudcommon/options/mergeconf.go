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

package options

import (
	"context"
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
)

func GetServiceIdByType(s *mcclient.ClientSession, typeStr string, verStr string) (string, error) {
	params := jsonutils.NewDict()
	if len(verStr) > 0 {
		typeStr += "_" + verStr
	}
	params.Add(jsonutils.NewString(typeStr), "type")
	result, err := identity.ServicesV3.List(s, params)
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

func GetServiceConfig(s *mcclient.ClientSession, serviceId string) (jsonutils.JSONObject, error) {
	conf, err := identity.ServicesV3.GetSpecific(s, serviceId, "config", nil)
	if err != nil {
		return nil, errors.Wrap(err, "modules.ServicesV3.GetSpecific config")
	}
	defConf, _ := conf.Get("config", "default")
	return defConf, nil
}

type IServiceConfigSession interface {
	Merge(opts interface{}, serviceType string, serviceVersion string) bool
	Upload()
	IsRemote() bool
}

type mcclientServiceConfigSession struct {
	session   *mcclient.ClientSession
	serviceId string
	config    *jsonutils.JSONDict

	commonServiceId string
}

func newServiceConfigSession() IServiceConfigSession {
	return &mcclientServiceConfigSession{}
}

func (s *mcclientServiceConfigSession) Merge(opts interface{}, serviceType string, serviceVersion string) bool {
	merged := false
	s.config = jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	region, _ := s.config.GetString("region")
	// epType, _ := s.config.GetString("session_endpoint_type")
	s.session = auth.GetAdminSession(context.Background(), region)
	if len(serviceType) > 0 {
		s.serviceId, _ = GetServiceIdByType(s.session, serviceType, serviceVersion)
		if len(s.serviceId) > 0 {
			serviceConf, err := GetServiceConfig(s.session, s.serviceId)
			if err != nil {
				log.Errorf("GetServiceConfig for %s failed: %s", serviceType, err)
			} else if serviceConf != nil {
				s.config.Update(serviceConf)
				merged = true
			} else {
				// not initialized
				// s.Upload()
			}
		}
	}
	s.commonServiceId, _ = GetServiceIdByType(s.session, consts.COMMON_SERVICE, "")
	if len(s.commonServiceId) > 0 {
		commonConf, err := GetServiceConfig(s.session, s.commonServiceId)
		if err != nil {
			log.Errorf("GetServiceConfig for %s failed: %s", consts.COMMON_SERVICE, err)
		} else if commonConf != nil {
			s.config.Update(commonConf)
			merged = true
		} else {
			// common not initialized
		}
	}
	if merged {
		err := s.config.Unmarshal(opts)
		if err == nil {
			return true
		}
		log.Errorf("s.config.Unmarshal fail %s", err)
	}
	return false
}

func (s *mcclientServiceConfigSession) Upload() {
	// upload service config
	if len(s.serviceId) > 0 {
		nconf := jsonutils.NewDict()
		nconf.Add(s.config, "config", "default")
		_, err := identity.ServicesV3.PerformAction(s.session, s.serviceId, "config", nconf)
		if err != nil {
			// ignore the error
			log.Errorf("fail to save config: %s", err)
		}
		_, err = identity.ServicesV3.PerformAction(s.session, s.commonServiceId, "config", nconf)
		if err != nil {
			// ignore the error
			log.Errorf("fail to save common config: %s", err)
		}
	}
}

func (s *mcclientServiceConfigSession) IsRemote() bool {
	return true
}
