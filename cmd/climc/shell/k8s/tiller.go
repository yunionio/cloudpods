package k8s

import (
	json "yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initTiller() {
	cmdN := func(suffix string) string {
		return resourceCmdN("tiller", suffix)
	}
	type createOpt struct {
		clusterBaseOptions
		KubeContext string `json:"kube_context"`
		Namespace   string `json:"namespace" default:"kube-system"`
		// Upgrade if Tiller is already installed
		Upgrade bool `json:"upgrade"`
		// Name of service account
		ServiceAccount string `json:"service_account" default:"tiller"`
		// Use the canary Tiller image
		Canary bool `json:"canary_image"`

		// Override Tiller image
		Image string `json:"tiller_image" default:"yunion/tiller:v2.9.0"`
		// Limit the maximum number of revisions saved per release. Use 0 for no limit.
		MaxHistory int `json:"history_max"`
	}
	R(&createOpt{}, cmdN("create"), "Install helm tiller server to Kubernetes cluster", func(s *mcclient.ClientSession, args *createOpt) error {
		params := json.NewDict()
		if len(args.KubeContext) > 0 {
			params.Add(json.NewString(args.KubeContext), "kube_context")
		}
		params.Add(json.NewString(args.Namespace), "namespace")
		params.Add(json.NewString(args.ServiceAccount), "service_account")
		if args.Canary {
			params.Add(json.JSONTrue, "canary_image")
		}
		if args.Upgrade {
			params.Add(json.JSONTrue, "upgrade")
		}
		if len(args.Image) > 0 {
			params.Add(json.NewString(args.Image), "tiller_image")
		}
		if args.MaxHistory > 0 {
			params.Add(json.NewInt(int64(args.MaxHistory)), "history_max")
		}
		ret, err := k8s.Tiller.CreateInContexts(s, params, args.ClusterContext())
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
}
