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

package servicetree

import (
	"fmt"
	"strconv"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/servicetree"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	/**
	 * 创建新的服务树节点
	 */
	type ServiceTreeCreateOptions struct {
		NAME    string   `help:"Name of the new service-tree node"`
		Pid     int64    `help:"PID of the service-tree node" default:"-1"`
		Label   []string `help:"Labels to this node"`
		Project string   `help:"Project id of the if create PDL node"`
		Remark  string   `help:"Remark or description of the new service-tree node"`
	}
	R(&ServiceTreeCreateOptions{}, "servicetree-node-create", "Create a service-tree node", func(s *mcclient.ClientSession, args *ServiceTreeCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		pid := args.Pid
		var err error
		if pid < 0 {
			if len(args.Label) <= 0 {
				return fmt.Errorf("Must provide either pid or labels")
			}
			pid, err = modules.TreeNodes.GetNodeIDByLabels(s, args.Label)
			if err != nil {
				return err
			}
			if pid < 0 {
				return fmt.Errorf("Invalid labels: %s", args.Label)
			}
		}
		params.Add(jsonutils.NewInt(pid), "pid")
		if len(args.Project) > 0 {
			projId, err := identity.Projects.GetId(s, args.Project, nil)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewString(projId), "project_id")
		}
		if len(args.Remark) > 0 {
			params.Add(jsonutils.NewString(args.Remark), "remark")
		}
		serviceTree, err := modules.TreeNodes.Create(s, params)
		if err != nil {
			return err
		}
		printObject(serviceTree)
		return nil
	})

	type ServiceTreeNodeListOptions struct {
		options.BaseListOptions
		Label string `help:"Label of node"`
		Pid   int64  `help:"Pid of node" default:"-1"`
	}
	R(&ServiceTreeNodeListOptions{}, "servicetree-node-list", "List servicetree nodes", func(s *mcclient.ClientSession, args *ServiceTreeNodeListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		if args.Pid >= 0 {
			params.Add(jsonutils.NewInt(args.Pid), "pid")
		}
		if len(args.Label) > 0 {
			params.Add(jsonutils.NewString(args.Label), "name")
		}
		result, err := modules.TreeNodes.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.TreeNodes.GetColumns(s))
		return nil
	})

	/**
	 * 查看服务树节点配置详情
	 */
	type ServiceTreeNodeShowOptions struct {
		Id    int64    `help:"ID of tree node"`
		Label []string `help:"Labels to this tree node"`
	}
	R(&ServiceTreeNodeShowOptions{}, "servicetree-node-show", "Show details of a servicetree node", func(s *mcclient.ClientSession, args *ServiceTreeNodeShowOptions) error {
		pid := args.Id
		var err error
		if pid <= 0 {
			pid, err = modules.TreeNodes.GetNodeIDByLabels(s, args.Label)
			if err != nil {
				return err
			}
			if pid <= 0 {
				return fmt.Errorf("Invalid tree node ID")
			}
		}
		idstr := strconv.Itoa(int(pid))
		result, err := modules.TreeNodes.Get(s, idstr, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	/**
	 * 修改服务树节点配置信息
	 */
	type ServiceTreeNodeUpdateOptions struct {
		ServiceTreeNodeShowOptions
		Name    string `help:"Name of the service-tree"`
		Project string `help:"Project associate with this tree node"`
		Remark  string `help:"Remark or description of the service-tree"`
	}
	R(&ServiceTreeNodeUpdateOptions{}, "servicetree-node-update", "Update a servicetree node", func(s *mcclient.ClientSession, args *ServiceTreeNodeUpdateOptions) error {
		pid := args.Id
		var err error
		if pid <= 0 {
			pid, err = modules.TreeNodes.GetNodeIDByLabels(s, args.Label)
			if err != nil {
				return err
			}
			if pid <= 0 {
				return fmt.Errorf("Invalid tree node ID")
			}
		}
		idstr := strconv.Itoa(int(pid))
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Project) > 0 {
			projId, err := identity.Projects.GetId(s, args.Project, nil)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewString(projId), "project_id")
		}
		if len(args.Remark) > 0 {
			params.Add(jsonutils.NewString(args.Remark), "remark")
		}
		result, err := modules.TreeNodes.Update(s, idstr, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	/**
	 * 修改服务树节点的项目类型
	 */
	type ServiceTreeNodeChangeProjectTypeOptions struct {
		PROJECTID   string `help:"ID of project"`
		PROJECTTYPE string `help:"TYPE of project" choices:"CreateNewProject|RelatedExisting"`
	}
	R(&ServiceTreeNodeChangeProjectTypeOptions{}, "servicetree-node-change-project-type", "servicetree-node-change-project-type", func(s *mcclient.ClientSession, args *ServiceTreeNodeChangeProjectTypeOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.PROJECTTYPE), "project_type")
		_, err := modules.TreeNodes.PerformAction(s, args.PROJECTID, "change-project-type", params)
		if err != nil {
			return err
		}
		return nil
	})

	/**
	 * 删除一个服务树节点
	 */
	R(&ServiceTreeNodeShowOptions{}, "servicetree-node-delete", "Delete a servicetree node", func(s *mcclient.ClientSession, args *ServiceTreeNodeShowOptions) error {
		pid := args.Id
		var err error
		if pid <= 0 {
			pid, err = modules.TreeNodes.GetNodeIDByLabels(s, args.Label)
			if err != nil {
				return err
			}
			if pid <= 0 {
				return fmt.Errorf("Invalid tree node ID")
			}
		}
		idstr := strconv.Itoa(int(pid))
		result, err := modules.TreeNodes.Delete(s, idstr, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
