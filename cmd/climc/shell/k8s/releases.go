package k8s

import (
	"fmt"
	"io/ioutil"

	json "yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initRelease() {
	cmdN := func(suffix string) string {
		return resourceCmdN("release", suffix)
	}
	type listOpt struct {
		namespaceListOptions
		baseListOptions
		Name       string `help:"Search by name"`
		Filter     string `help:"Filter, split by space"`
		Admin      bool   `help:"Admin to show all namespace releases"`
		Deployed   bool   `help:"Show deployed status releases"`
		Deleted    bool   `help:"Show deleted status releases"`
		Deleting   bool   `help:"Show deleting status releases"`
		Failed     bool   `help:"Show failed status releases"`
		Superseded bool   `help:"Show superseded status releases"`
		Pending    bool   `help:"Show pending status releases"`
	}
	R(&listOpt{}, cmdN("list"), "List k8s cluster helm releases", func(s *mcclient.ClientSession, args *listOpt) error {
		params := fetchNamespaceParams(args.namespaceListOptions)
		params.Update(fetchPagingParams(args.baseListOptions))
		params.Update(args.ClusterParams())
		if args.Filter != "" {
			params.Add(json.NewString(args.Filter), "filter")
		}
		if args.Namespace != "" {
			params.Add(json.NewString(args.Namespace), "namespace")
		}
		if args.Name != "" {
			params.Add(json.NewString(args.Name), "name")
		}
		params.Add(json.JSONTrue, "all")
		if args.Admin {
			params.Add(json.JSONTrue, "admin")
		}
		if args.Deployed {
			params.Add(json.JSONTrue, "deployed")
		}
		if args.Deleted {
			params.Add(json.JSONTrue, "deleted")
		}
		if args.Deleting {
			params.Add(json.JSONTrue, "deleting")
		}
		if args.Failed {
			params.Add(json.JSONTrue, "failed")
		}
		if args.Superseded {
			params.Add(json.JSONTrue, "superseded")
		}
		if args.Pending {
			params.Add(json.JSONTrue, "pending")
		}
		ret, err := k8s.Releases.List(s, params)
		if err != nil {
			return err
		}
		printList(ret, k8s.Releases.GetColumns(s))
		return nil
	})

	type showOpt struct {
		clusterBaseOptions
		NAME string `help:"Release instance name"`
	}
	R(&showOpt{}, cmdN("show"), "Get helm release details", func(s *mcclient.ClientSession, args *showOpt) error {
		ret, err := k8s.Releases.Get(s, args.NAME, args.ClusterParams())
		if err != nil {
			return err
		}
		resources, err := ret.GetString("info", "status", "resources")
		if err != nil {
			return err
		}
		fmt.Println(resources)
		return nil
	})

	type releaseCUOpts struct {
		Values  string `help:"Specify values in a YAML file (can specify multiple)" short-token:"f"`
		Version string `help:"Specify the exact chart version to install. If not specified, latest version installed"`
		//Set     []string `help:"set STRING values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)"`
		DryRun  bool  `help:"Simulate an install"`
		Details bool  `help:"Show release deploy details, include kubernetes created resources"`
		Timeout int64 `help:"Time in seconds to wait for any individual kubernetes operation (like Jobs for hooks)" default:"600"`
	}
	releaseCUDict := func(args releaseCUOpts) (*json.JSONDict, error) {
		params := json.NewDict()
		if args.Version != "" {
			params.Add(json.NewString(args.Version), "version")
		}
		if args.DryRun {
			params.Add(json.JSONTrue, "dry_run")
		}
		params.Add(json.NewInt(args.Timeout), "timeout")
		if args.Values != "" {
			//vals, err := helm.MergeValuesF(args.Values, args.Set, []string{})
			vals, err := ioutil.ReadFile(args.Values)
			if err != nil {
				return nil, err
			}
			params.Add(json.NewString(string(vals)), "values")
		}
		return params, nil
	}

	type releaseCreateOpts struct {
		namespaceOptions
		releaseCUOpts
		Name      string `help:"Release name, If unspecified, it will autogenerate one for you"`
		CHARTNAME string `help:"Helm chart name, e.g stable/etcd"`
	}
	R(&releaseCreateOpts{}, cmdN("create"), "Create release with specified helm chart", func(s *mcclient.ClientSession, args *releaseCreateOpts) error {
		params, err := releaseCUDict(args.releaseCUOpts)
		if err != nil {
			return err
		}
		params.Update(args.ClusterParams())
		params.Add(json.NewString(args.CHARTNAME), "chart_name")
		if args.Namespace != "" {
			params.Add(json.NewString(args.Namespace), "namespace")
		}
		if args.Name != "" {
			params.Add(json.NewString(args.Name), "release_name")
		}
		ret, err := k8s.Releases.Create(s, params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	type releaseUpgradeOpts struct {
		clusterBaseOptions
		releaseCUOpts
		NAME        string `help:"Release instance name"`
		CHARTNAME   string `help:"Helm chart name, e.g stable/etcd"`
		ReuseValues bool   `help:"When upgrading, reuse the last release's values, and merge in any new values. If '--reset-values' is specified, this is ignored"`
		ResetValues bool   `help:"When upgrading, reset the values to the ones built into the chart"`
	}
	R(&releaseUpgradeOpts{}, cmdN("upgrade"), "Upgrade release", func(s *mcclient.ClientSession, args *releaseUpgradeOpts) error {
		params, err := releaseCUDict(args.releaseCUOpts)
		if err != nil {
			return err
		}
		params.Update(args.ClusterParams())
		params.Add(json.NewString(args.CHARTNAME), "chart_name")
		params.Add(json.NewString(args.NAME), "release_name")
		if args.ReuseValues {
			params.Add(json.JSONTrue, "reuse_values")
		}
		if args.ResetValues {
			params.Add(json.JSONTrue, "reset_values")
		}

		res, err := k8s.Releases.Put(s, args.NAME, params)
		if err != nil {
			return err
		}
		printObject(res)
		return nil
	})

	type deleteOpt struct {
		clusterBaseOptions
		NAME string `help:"Release instance name"`
	}
	R(&deleteOpt{}, cmdN("delete"), "Delete release", func(s *mcclient.ClientSession, args *deleteOpt) error {
		_, err := k8s.Releases.Delete(s, args.NAME, args.ClusterParams())
		return err
	})
}
