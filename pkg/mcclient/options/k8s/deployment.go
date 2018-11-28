package k8s

import (
	"yunion.io/x/jsonutils"
)

type DeploymentCreateOptions struct {
	K8sAppBaseCreateOptions
}

func (o DeploymentCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := o.K8sAppBaseCreateOptions.Params()
	if err != nil {
		return nil, err
	}
	return params, nil
}

type StatefulSetCreateOptions struct {
	K8sAppBaseCreateOptions
}

type JobCreateOptions struct {
	K8sAppBaseCreateOptions
	Parallelism int64 `help:"Specifies the maximum desired number of pods the job should run at any given time"`
}

func (o JobCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := o.K8sAppBaseCreateOptions.Params()
	if err != nil {
		return nil, err
	}
	if o.Parallelism > 0 {
		params.Add(jsonutils.NewInt(o.Parallelism), "parallelism")
	}
	return params, nil
}

type CronJobCreateOptions struct {
	JobCreateOptions
	Schedule string `help:"The chedule in Cron format, e.g. '*/10 * * * *'" required:"true"`
}

func (o CronJobCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := o.JobCreateOptions.Params()
	if err != nil {
		return nil, err
	}
	params.Add(jsonutils.NewString(o.Schedule), "schedule")
	return params, nil
}
