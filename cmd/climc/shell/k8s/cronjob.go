package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initCronJob() {
	cmd := initK8sNamespaceResource("cronjob", k8s.CronJobs)
	cmdN := cmd.CommandNameFactory
	createCmd := NewCommand(
		&o.CronJobCreateOptions{},
		cmdN("create"),
		"Create cronjob resource",
		func(s *mcclient.ClientSession, args *o.JobCreateOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := k8s.CronJobs.Create(s, params)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})

	cmd.AddR(createCmd)
}
