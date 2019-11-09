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

package k8s

import (
	"yunion.io/x/jsonutils"
)

type JobTemplateOptions struct {
	K8sLabelOptions
	K8sPodTemplateOptions
	Parallelism int64 `help:"Specifies the maximum desired number of pods the job should run at any given time"`
}

func (o JobTemplateOptions) Params(name string) (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	o.K8sPodTemplateOptions.setContainerName(name)
	if err := o.K8sPodTemplateOptions.Attach(params); err != nil {
		return nil, err
	}
	if o.Parallelism > 0 {
		params.Add(jsonutils.NewInt(o.Parallelism), "parallelism")
	}
	return params, nil
}

func (o JobTemplateOptions) Attach(params *jsonutils.JSONDict, name string, key ...string) error {
	ret, err := o.Params(name)
	if err != nil {
		return err
	}
	if len(key) == 0 {
		params.Update(ret)
	} else {
		params.Add(ret, key...)
	}
	return nil
}

type JobCreateOptions struct {
	NamespaceWithClusterOptions
	JobTemplateOptions

	NAME string `help:"Name of job"`
}

func (o JobCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params := o.NamespaceWithClusterOptions.Params()
	if err := o.JobTemplateOptions.Attach(params, o.NAME); err != nil {
		return nil, err
	}
	params.Add(jsonutils.NewString(o.NAME), "name")
	return params, nil
}

type CronJobCreateOptions struct {
	JobTemplateOptions
	NamespaceWithClusterOptions
	NAME     string `help:"Name of cronjob"`
	Schedule string `help:"The chedule in Cron format, e.g. '*/10 * * * *'" required:"true"`
}

func (o CronJobCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params := o.NamespaceWithClusterOptions.Params()

	if err := o.JobTemplateOptions.Attach(params, o.NAME, "jobTemplate", "spec"); err != nil {
		return nil, err
	}
	params.Add(jsonutils.NewString(o.NAME), "name")
	params.Add(jsonutils.NewString(o.Schedule), "schedule")
	return params, nil
}
