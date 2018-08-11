package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {

	/**
	 * 向指定的服务树节点添加机器
	 */
	type ServiceHostCreateOptions struct {
		LABELS    string   `help:"Labels for tree-node(split by comma)"`
		HOST_NAME []string `help:"Host names to add to"`
	}
	R(&ServiceHostCreateOptions{}, "servicehost-create", "Add host to tree-node", func(s *mcclient.ClientSession, args *ServiceHostCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.LABELS), "node_labels")
		arr := jsonutils.NewArray()
		if len(args.HOST_NAME) > 0 {
			for _, f := range args.HOST_NAME {
				tmpObj := jsonutils.NewDict()
				tmpObj.Add(jsonutils.NewString(f), "host_name")
				arr.Add(tmpObj)
			}
		}
		params.Add(arr, "service_hosts")

		rst, err := modules.ServiceHosts.Create(s, params)

		if err != nil {
			return err
		}

		printObject(rst)
		return nil
	})

	/**
	 * 从服务树节点删除机器
	 */
	type ServiceHostDeleteOptions struct {
		LABELS    string   `help:"Labels for tree-node(split by comma)"`
		HOST_NAME []string `help:"Host names to remove from"`
	}
	R(&ServiceHostDeleteOptions{}, "servicehost-delete", "Remove host from tree-node", func(s *mcclient.ClientSession, args *ServiceHostDeleteOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.LABELS), "node_labels")
		arr := jsonutils.NewArray()
		if len(args.HOST_NAME) > 0 {
			for _, f := range args.HOST_NAME {
				tmpObj := jsonutils.NewDict()
				tmpObj.Add(jsonutils.NewString(f), "host_name")
				arr.Add(tmpObj)
			}
		}
		params.Add(arr, "service_hosts")

		rst, err := modules.ServiceHosts.DoDeleteServiceHost(s, params)

		if err != nil {
			return err
		}

		printObject(rst)
		return nil
	})

	/**
	 * 查看指定服务树节点的机器
	 */
	type ServiceHostListOptions struct {
		BaseListOptions
		Labels string `help:"Labels for tree-node(split by comma)"`
	}
	R(&ServiceHostListOptions{}, "servicehost-list", "List all hosts for the tree-node", func(s *mcclient.ClientSession, args *ServiceHostListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)
		if len(args.Labels) > 0 {
			params.Add(jsonutils.NewString(args.Labels), "node_labels")
		}

		result, err := modules.ServiceHosts.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.ServiceHosts.GetColumns(s))
		return nil
	})

}
