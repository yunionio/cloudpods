package options

import (
	"strings"
	"time"

	"yunion.io/x/jsonutils"
)

type LoadbalancerAgentParamsOptions struct {
	KeepalivedConfTmpl string
	HaproxyConfTmpl    string

	VrrpPriority          *int // required
	VrrpVirtualRouterId   *int // required
	VrrpGarpMasterRefresh *int
	VrrpPreempt           string `choices:"true|false"`
	VrrpInterface         string // required
	VrrpAdvertInt         *int
	VrrpPass              string

	HaproxyGlobalLog      string
	HaproxyGlobalNbthread *int   `help:"enable experimental multi-threading support available since haproxy 1.8"`
	HaproxyLogHttp        string `choices:"true|false"`
	HaproxyLogTcp         string `choices:"true|false"`
	HaproxyLogNormal      string `choices:"true|false"`

	TelegrafInfluxDbOutputUrl    string
	TelegrafInfluxDbOutputName   string
	TelegrafHaproxyInputInterval int
}

func (opts *LoadbalancerAgentParamsOptions) setPrefixedParams(params *jsonutils.JSONDict, pref string) {
	pref_ := pref + "_"
	pref_len := len(pref_)
	pp := jsonutils.NewDict()
	pMap, _ := params.GetMap()
	for k, v := range pMap {
		if strings.HasPrefix(k, pref_) && !strings.HasSuffix(k, "_conf_tmpl") {
			params.Remove(k)
			pp.Set(k[pref_len:], v)
		}
	}
	if pp.Length() > 0 {
		params.Set(pref, pp)
	}
}

func (opts *LoadbalancerAgentParamsOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := optionsStructToParams(opts)
	if err != nil {
		return nil, err
	}
	opts.setPrefixedParams(params, "vrrp")
	opts.setPrefixedParams(params, "haproxy")
	opts.setPrefixedParams(params, "telegraf")
	return params, nil
}

type LoadbalancerAgentCreateOptions struct {
	NAME      string
	HbTimeout *int

	LoadbalancerAgentParamsOptions
}

func (opts *LoadbalancerAgentCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := StructToParams(opts)
	if err != nil {
		return nil, err
	}
	paramsSub, err := opts.LoadbalancerAgentParamsOptions.Params()
	if err != nil {
		return nil, err
	}
	params.Set("params", paramsSub)
	return params, nil
}

type LoadbalancerAgentListOptions struct {
	BaseListOptions
}

type LoadbalancerAgentGetOptions struct {
	ID string
}

type LoadbalancerAgentUpdateOptions struct {
	ID string

	HbTimeout *int

	Loadbalancers             *time.Time
	LoadbalancerListeners     *time.Time
	LoadbalancerListenerRules *time.Time
	LoadbalancerBackendGroups *time.Time
	LoadbalancerBackends      *time.Time
	LoadbalancerAcls          *time.Time
	LoadbalancerCertificates  *time.Time
}

type LoadbalancerAgentDeleteOptions struct {
	ID string
}

type LoadbalancerAgentActionHbOptions struct {
	ID string
}

type LoadbalancerAgentActionPatchParamsOptions struct {
	ID string

	LoadbalancerAgentParamsOptions
}

func (opts *LoadbalancerAgentActionPatchParamsOptions) Params() (*jsonutils.JSONDict, error) {
	return opts.LoadbalancerAgentParamsOptions.Params()
}
