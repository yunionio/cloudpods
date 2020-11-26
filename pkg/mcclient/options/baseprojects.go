package options

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

type SharableProjectizedResourceBaseCreateInput struct {
	apis.ProjectizedResourceCreateInput
	apis.SharableResourceBaseCreateInput
}

func (opts *SharableProjectizedResourceBaseCreateInput) Params() (*jsonutils.JSONDict, error) {
	params, err := optionsStructToParams(opts.SharableResourceBaseCreateInput)
	if err != nil {
		return nil, err
	}

	projectInput, err := optionsStructToParams(opts.ProjectizedResourceCreateInput.ProjectizedResourceInput)
	if err != nil {
		return nil, err
	}

	domainInput, err := optionsStructToParams(opts.ProjectizedResourceCreateInput.DomainizedResourceInput)
	if err != nil {
		return nil, err
	}

	params.Update(projectInput)
	params.Update(domainInput)
	return params, nil
}

type SharableResourcePublicBaseOptions struct {
	Scope          string   `help:"sharing scope" choices:"system|domain|project"`
	SharedProjects []string `help:"Share to projects"`
	SharedDomains  []string `help:"Share to domains"`
}
