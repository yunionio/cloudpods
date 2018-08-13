package k8s

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initRepo() {
	cmdN := func(suffix string) string {
		return resourceCmdN("repo", suffix)
	}
	type listOpt struct {
		BaseListOptions
	}
	R(&listOpt{}, cmdN("list"), "List k8s global helm repos", func(s *mcclient.ClientSession, args *listOpt) error {
		params := FetchPagingParams(args.BaseListOptions)
		result, err := k8s.Repos.List(s, params)
		if err != nil {
			return err
		}
		printList(result, k8s.Repos.GetColumns(s))
		return nil
	})

	type getOpt struct {
		NAME string `help:"ID or name of the repo"`
	}
	R(&getOpt{}, cmdN("show"), "Show details fo a repo", func(s *mcclient.ClientSession, args *getOpt) error {
		repo, err := k8s.Repos.Get(s, args.NAME, nil)
		if err != nil {
			return err
		}
		printObject(repo)
		return nil
	})

	type createOpt struct {
		getOpt
		URL    string `help:"Repository url"`
		Public bool   `help:"Make repository public"`
	}
	R(&createOpt{}, cmdN("create"), "Add repository", func(s *mcclient.ClientSession, args *createOpt) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString(args.URL), "url")
		if args.Public {
			params.Add(jsonutils.JSONTrue, "is_public")
		}
		repo, err := k8s.Repos.Create(s, params)
		if err != nil {
			return err
		}
		printObject(repo)
		return nil
	})

	type updateOpt struct {
		getOpt
		Name string `help:"Repository name to change"`
		Url  string `help:"Repository url to change"`
	}
	R(&updateOpt{}, cmdN("update"), "Update helm repository", func(s *mcclient.ClientSession, args *updateOpt) error {
		params := jsonutils.NewDict()
		if args.Name != "" {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if args.Url != "" {
			params.Add(jsonutils.NewString(args.Url), "url")
		}
		repo, err := k8s.Repos.Update(s, args.NAME, params)
		if err != nil {
			return err
		}
		printObject(repo)
		return nil
	})

	R(&getOpt{}, cmdN("private"), "Make repository private", func(s *mcclient.ClientSession, args *getOpt) error {
		repo, err := k8s.Repos.PerformAction(s, args.NAME, "private", nil)
		if err != nil {
			return err
		}
		printObject(repo)
		return nil
	})

	R(&getOpt{}, cmdN("public"), "Make repository public", func(s *mcclient.ClientSession, args *getOpt) error {
		repo, err := k8s.Repos.PerformAction(s, args.NAME, "public", nil)
		if err != nil {
			return err
		}
		printObject(repo)
		return nil
	})

	R(&getOpt{}, cmdN("delete"), "Delete a repository", func(s *mcclient.ClientSession, args *getOpt) error {
		repo, err := k8s.Repos.Delete(s, args.NAME, nil)
		if err != nil {
			return err
		}
		printObject(repo)
		return nil
	})
}
