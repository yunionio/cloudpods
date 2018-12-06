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
	PvcTemplate []string `help:"PVC volume desc, format is <pvc_name>:<size>:<mount_point>"`
}

func (o StatefulSetCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := o.K8sAppBaseCreateOptions.Params()
	if err != nil {
		return nil, err
	}
	vols := jsonutils.NewArray()
	volMounts := jsonutils.NewArray()
	for _, pvc := range o.PvcTemplate {
		vol, volMount, err := parsePvcTemplate(pvc)
		if err != nil {
			return nil, err
		}
		vols.Add(vol)
		volMounts.Add(volMount)
	}
	params.Add(vols, "volumeClaimTemplates")
	params.Add(volMounts, "volumeMounts")
	return params, nil
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
