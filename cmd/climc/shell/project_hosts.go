package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {

	/**
	 * move host to one tree-node
	 */
	type ProjectHostCreateOptions struct {
		LABELS    string   `help:"Labels for tree-node(split by comma)"`
		HOST_NAME []string `help:"Host names to move to"`
	}
	R(&ProjectHostCreateOptions{}, "projecthost-create", "move host to tree-node", func(s *mcclient.ClientSession, args *ProjectHostCreateOptions) error {
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

		rst, err := modules.ProjectHosts.Create(s, params)

		if err != nil {
			return err
		}

		printObject(rst)
		return nil
	})
}
