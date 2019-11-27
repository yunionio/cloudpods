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

package shell

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

func init() {
	type ServiceListOptions struct {
		Limit  int64  `help:"Limit, default 0, i.e. no limit" default:"20"`
		Offset int64  `help:"Offset, default 0, i.e. no offset"`
		Name   string `help:"Search by name"`
		Type   string `help:"Search by type"`
		Search string `help:"search any fields"`
	}
	R(&ServiceListOptions{}, "service-list", "List services", func(s *mcclient.ClientSession, args *ServiceListOptions) error {
		query := jsonutils.NewDict()
		if args.Limit > 0 {
			query.Add(jsonutils.NewInt(args.Limit), "limit")
		}
		if args.Offset > 0 {
			query.Add(jsonutils.NewInt(args.Offset), "offset")
		}
		if len(args.Name) > 0 {
			query.Add(jsonutils.NewString(args.Name), "name__icontains")
		}
		if len(args.Type) > 0 {
			query.Add(jsonutils.NewString(args.Type), "type__icontains")
		}
		if len(args.Search) > 0 {
			query.Add(jsonutils.NewString(args.Search), "search")
		}
		result, err := modules.ServicesV3.List(s, query)
		if err != nil {
			return err
		}
		printList(result, modules.ServicesV3.GetColumns(s))
		return nil
	})

	type ServiceShowOptions struct {
		ID string `help:"ID of service"`
	}
	R(&ServiceShowOptions{}, "service-show", "Show details of a service", func(s *mcclient.ClientSession, args *ServiceShowOptions) error {
		srvId, err := modules.ServicesV3.GetId(s, args.ID, nil)
		if err != nil {
			return err
		}
		result, err := modules.ServicesV3.Get(s, srvId, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
	R(&ServiceShowOptions{}, "service-delete", "Delete a service", func(s *mcclient.ClientSession, args *ServiceShowOptions) error {
		srvId, err := modules.ServicesV3.GetId(s, args.ID, nil)
		if err != nil {
			return err
		}
		result, err := modules.ServicesV3.Delete(s, srvId, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServiceCreateOptions struct {
		TYPE     string `help:"Service type"`
		NAME     string `help:"Service name"`
		Desc     string `help:"Description"`
		Enabled  bool   `help:"Enabeld"`
		Disabled bool   `help:"Disabled"`
	}
	R(&ServiceCreateOptions{}, "service-create", "Create a service", func(s *mcclient.ClientSession, args *ServiceCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.TYPE), "type")
		params.Add(jsonutils.NewString(args.NAME), "name")
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if args.Enabled && !args.Disabled {
			params.Add(jsonutils.JSONTrue, "enabled")
		} else if !args.Enabled && args.Disabled {
			params.Add(jsonutils.JSONFalse, "enabled")
		}
		srv, err := modules.ServicesV3.Create(s, params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	type ServiceUpdateOptions struct {
		ID       string `help:"ID or name of the service"`
		Type     string `help:"Service type"`
		Name     string `help:"Service name"`
		Desc     string `help:"Description"`
		Enabled  bool   `help:"Enabeld"`
		Disabled bool   `help:"Disabled"`
	}
	R(&ServiceUpdateOptions{}, "service-update", "Update a service", func(s *mcclient.ClientSession, args *ServiceUpdateOptions) error {
		srvId, err := modules.ServicesV3.GetId(s, args.ID, nil)
		if err != nil {
			return err
		}
		params := jsonutils.NewDict()
		if len(args.Type) > 0 {
			params.Add(jsonutils.NewString(args.Type), "type")
		}
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if args.Enabled && !args.Disabled {
			params.Add(jsonutils.JSONTrue, "enabled")
		} else if !args.Enabled && args.Disabled {
			params.Add(jsonutils.JSONFalse, "enabled")
		}
		srv, err := modules.ServicesV3.Patch(s, srvId, params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	type ServiceConfigShowOptions struct {
		SERVICE string `help:"service name or id"`
	}
	R(&ServiceConfigShowOptions{}, "service-config-show", "Show configs of a service", func(s *mcclient.ClientSession, args *ServiceConfigShowOptions) error {
		conf, err := modules.ServicesV3.GetSpecific(s, args.SERVICE, "config", nil)
		if err != nil {
			return err
		}
		fmt.Println(conf.PrettyString())
		return nil
	})

	type ServiceConfigOptions struct {
		SERVICE string   `help:"service name or id"`
		Config  []string `help:"config options, can be a JSON, a YAML or a key=value pair, e.g:
    * JSON
      '{\"default\":{\"password_expiration_seconds\":300}}'
    * YAML
      default:
        password_expiration_seconds: 300
    * A key=value pair (under default section)
      password_expiration_seconds=300
"`
		Remove bool `help:"remove config"`
	}
	R(&ServiceConfigOptions{}, "service-config", "Add config to service", func(s *mcclient.ClientSession, args *ServiceConfigOptions) error {
		config := jsonutils.NewDict()
		if args.Remove {
			config.Add(jsonutils.NewString("remove"), "action")
		} else {
			config.Add(jsonutils.NewString("update"), "action")
		}
		for _, c := range args.Config {
			json, _ := jsonutils.ParseString(c)
			if json != nil {
				if _, ok := json.(*jsonutils.JSONDict); ok {
					subconf := jsonutils.NewDict()
					subconf.Add(json, "config")
					config.Update(subconf)
					continue
				}
			}
			yaml, _ := jsonutils.ParseYAML(c)
			if yaml != nil {
				if _, ok := yaml.(*jsonutils.JSONDict); ok {
					subconf := jsonutils.NewDict()
					subconf.Add(yaml, "config")
					config.Update(subconf)
					continue
				}
			}
			pos := strings.IndexByte(c, '=')
			if pos < 0 {
				return fmt.Errorf("%s is not a key=value pair", c)
			}
			key := strings.TrimSpace(c[:pos])
			value := strings.TrimSpace(c[pos+1:])
			config.Add(jsonutils.NewString(value), "config", "default", key)
		}
		nconf, err := modules.ServicesV3.PerformAction(s, args.SERVICE, "config", config)
		if err != nil {
			return err
		}
		fmt.Println(nconf.PrettyString())
		return nil
	})

	type ServiceConfigYamlOptions struct {
		SERVICE string `help:"service name or id"`
		YAML    string `help:"config yaml file"`
	}
	R(&ServiceConfigYamlOptions{}, "service-config-yaml", "Config service with a yaml file", func(s *mcclient.ClientSession, args *ServiceConfigYamlOptions) error {
		content, err := fileutils2.FileGetContents(args.YAML)
		if err != nil {
			return err
		}
		yamlJson, err := jsonutils.ParseYAML(content)
		if err != nil {
			return err
		}
		config := jsonutils.NewDict()
		config.Add(yamlJson, "config", "default")
		nconf, err := modules.ServicesV3.PerformAction(s, args.SERVICE, "config", config)
		if err != nil {
			return err
		}
		fmt.Println(nconf.PrettyString())
		return nil
	})

}
