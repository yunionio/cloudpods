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

package misc

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/yunionconf"
)

func init() {
	var features = []string{
		"onestack",
		"baremetal",
		"lb",
		"aliyun",
		"aws",
		"azure",
		"ctyun",
		"google",
		"huawei",
		"qcloud",
		"ucloud",
		"ecloud",
		"jdcloud",
		"vmware",
		"openstack",
		"dstack",
		"zstack",
		"apsara",
		"cloudpods",
		"hcso",
		"hcsop",
		"hcs",
		"nutanix",
		"s3",
		"ceph",
		"xsky",
		"bill",
		"auth",
		"onecloud",
		"incloudsphere",
		"proxmox",
		"public",
		"private",
		"storage",
		"default",
		"k8s",
		"monitor",
	}

	const (
		GlobalSettings = "global-settings"
		SystemScope    = "system"
		YunionAgent    = "yunionagent"
	)

	type FeatureCfgOpts struct {
		Switch string `help:"Config feature on or off" choices:"on|off"`
	}

	featureR := func(name string) {
		R(&FeatureCfgOpts{}, fmt.Sprintf("feature-config-%s", name), fmt.Sprintf("Set feature %s on or off", name), func(s *mcclient.ClientSession, args *FeatureCfgOpts) error {
			enable := true
			if args.Switch == "off" {
				enable = false
			}
			items, err := yunionconf.Parameters.List(s, jsonutils.Marshal(map[string]string{
				"name":  GlobalSettings,
				"scope": "system"}))
			if err != nil {
				return errors.Wrapf(err, "get %s from yunionconf", GlobalSettings)
			}

			if len(items.Data) == 0 {
				// create it if enabled
				if enable {
					input := map[string]interface{}{
						"name":       GlobalSettings,
						"service_id": YunionAgent,
						"value": map[string]interface{}{
							"setupKeys":                []string{name},
							"setupOneStackInitialized": true,
						},
					}
					params := jsonutils.Marshal(input)
					if _, err := yunionconf.Parameters.Create(s, params); err != nil {
						return errors.Errorf("create %s for feature %q", GlobalSettings, name)
					}
					return nil
				} else {
					return errors.Errorf("not found %s", GlobalSettings)
				}
			}

			if len(items.Data) != 1 {
				return errors.Errorf("found %d %q from yunionconf", len(items.Data), GlobalSettings)
			}

			// update it
			ss := items.Data[0]
			value, err := ss.Get("value")
			if err != nil {
				return errors.Wrap(err, "get value")
			}
			setupKeys := []string{}
			if value.Contains("setupKeys") {
				if err := value.Unmarshal(&setupKeys, "setupKeys"); err != nil {
					return errors.Wrap(err, "unmarshal setupKeys")
				}
			}
			if !enable {
				if utils.IsInStringArray(name, setupKeys) {
					newKeys := []string{}
					for _, key := range setupKeys {
						if key != name {
							newKeys = append(newKeys, key)
						}
					}
					setupKeys = newKeys
				}
			} else if enable {
				if !utils.IsInStringArray(name, setupKeys) {
					setupKeys = append(setupKeys, name)
				}
			}
			value.(*jsonutils.JSONDict).Set("setupKeys", jsonutils.Marshal(setupKeys))
			id, err := ss.GetString("id")
			if err != nil {
				return errors.Errorf("get id from %s", ss)
			}
			ss.(*jsonutils.JSONDict).Set("value", value)
			obj, err := yunionconf.Parameters.Update(s, id, ss)
			if err != nil {
				return errors.Wrapf(err, "update %s(%s)", GlobalSettings, id)
			}
			printObject(obj)
			return nil
		})
	}
	for _, name := range features {
		featureR(name)
	}
}
