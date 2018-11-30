package models

import (
	"context"
	"encoding/base64"
	"net/url"
	"reflect"
	"text/template"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SLoadbalancerAgentManager struct {
	db.SStandaloneResourceBaseManager
	SInfrastructureManager
}

var LoadbalancerAgentManager *SLoadbalancerAgentManager

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&SLoadbalancerAgentParams{}), func() gotypes.ISerializable {
		return &SLoadbalancerAgentParams{}
	})
	LoadbalancerAgentManager = &SLoadbalancerAgentManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SLoadbalancerAgent{},
			"loadbalanceragents_tbl",
			"loadbalanceragent",
			"loadbalanceragents",
		),
	}
}

// TODO
//
//  - scrub stale backends: Guests with deleted=1
//  - agent configuration params
//
type SLoadbalancerAgent struct {
	db.SStandaloneResourceBase
	SInfrastructure

	HbLastSeen time.Time                 `nullable:"true" list:"admin" update:"admin"`
	HbTimeout  int                       `nullable:"true" list:"admin" update:"admin" create:"optional" default:"3600"`
	Params     *SLoadbalancerAgentParams `create:"optional" get:"admin"`

	Loadbalancers             time.Time `nullable:"true" list:"admin" update:"admin"`
	LoadbalancerListeners     time.Time `nullable:"true" list:"admin" update:"admin"`
	LoadbalancerListenerRules time.Time `nullable:"true" list:"admin" update:"admin"`
	LoadbalancerBackendGroups time.Time `nullable:"true" list:"admin" update:"admin"`
	LoadbalancerBackends      time.Time `nullable:"true" list:"admin" update:"admin"`
	LoadbalancerAcls          time.Time `nullable:"true" list:"admin" update:"admin"`
	LoadbalancerCertificates  time.Time `nullable:"true" list:"admin" update:"admin"`
}

type SLoadbalancerAgentParamsVrrp struct {
	Priority          int
	VirtualRouterId   int
	GarpMasterRefresh int
	Preempt           bool
	Interface         string
	AdvertInt         int
	Pass              string
}

type SLoadbalancerAgentParamsHaproxy struct {
	GlobalLog      string
	GlobalNbthread int
	LogHttp        bool
	LogTcp         bool
	LogNormal      bool
}

type SLoadbalancerAgentParamsTelegraf struct {
	InfluxDbOutputUrl    string
	InfluxDbOutputName   string
	HaproxyInputInterval int
}

type SLoadbalancerAgentParams struct {
	KeepalivedConfTmpl string
	HaproxyConfTmpl    string
	TelegrafConfTmpl   string
	Vrrp               SLoadbalancerAgentParamsVrrp
	Haproxy            SLoadbalancerAgentParamsHaproxy
	Telegraf           SLoadbalancerAgentParamsTelegraf
}

func (p *SLoadbalancerAgentParamsVrrp) Validate(data *jsonutils.JSONDict) error {
	if len(p.Interface) == 0 || len(p.Interface) > 16 {
		// TODO printable exclude white space
		return httperrors.NewInputParameterError("invalid vrrp interface %q", p.Interface)
	}
	if len(p.Pass) == 0 || len(p.Pass) > 8 {
		// TODO printable exclude white space
		return httperrors.NewInputParameterError("invalid vrrp authentication pass size: %d, want [1,8]", len(p.Pass))
	}
	if p.Priority < 1 || p.Priority > 255 {
		return httperrors.NewInputParameterError("invalid vrrp priority %d: want [1,255]", p.Priority)
	}
	if p.VirtualRouterId < 1 || p.VirtualRouterId > 255 {
		return httperrors.NewInputParameterError("invalid vrrp virtual_router_id %d: want [1,255]", p.VirtualRouterId)
	}
	return nil
}

func (p *SLoadbalancerAgentParamsVrrp) initDefault(data *jsonutils.JSONDict) {
	if !data.Contains("params", "vrrp", "advert_int") {
		p.AdvertInt = 1
	}
	if !data.Contains("params", "vrrp", "garp_master_refresh") {
		p.GarpMasterRefresh = 27
	}
	if !data.Contains("params", "vrrp", "pass") {
		p.Pass = "YunionLB"
	}
}

func (p *SLoadbalancerAgentParamsHaproxy) Validate(data *jsonutils.JSONDict) error {
	if p.GlobalNbthread < 1 {
		p.GlobalNbthread = 1
	}
	if p.GlobalNbthread > 64 {
		// This is a limit imposed by haproxy and arch word size
		p.GlobalNbthread = 64
	}
	return nil
}

func (p *SLoadbalancerAgentParamsHaproxy) initDefault(data *jsonutils.JSONDict) {
	if !data.Contains("params", "haproxy", "global_nbthread") {
		p.GlobalNbthread = 1
	}
	if !data.Contains("params", "haproxy", "global_log") {
		p.GlobalLog = "log /dev/log local0 info"
	}
	if !data.Contains("params", "haproxy", "log_http") {
		p.LogHttp = true
	}
	if !data.Contains("params", "haproxy", "log_normal") {
		p.LogNormal = true
	}
}

func (p *SLoadbalancerAgentParamsTelegraf) Validate(data *jsonutils.JSONDict) error {
	if p.InfluxDbOutputUrl != "" {
		_, err := url.Parse(p.InfluxDbOutputUrl)
		if err != nil {
			return httperrors.NewInputParameterError("telegraf params: invalid influxdb url: %s", err)
		}
	}
	if p.HaproxyInputInterval <= 0 {
		p.HaproxyInputInterval = 5
	}
	if p.InfluxDbOutputName == "" {
		p.InfluxDbOutputName = "telegraf"
	}
	return nil
}

func (p *SLoadbalancerAgentParamsTelegraf) initDefault(data *jsonutils.JSONDict) {
	if p.HaproxyInputInterval == 0 {
		p.HaproxyInputInterval = 5
	}
	if p.InfluxDbOutputName == "" {
		p.InfluxDbOutputName = "telegraf"
	}
}

func (p *SLoadbalancerAgentParams) validateTmpl(k, s string) error {
	d, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return httperrors.NewInputParameterError("%s: bad base64 encoded string: %s", k, err)
	}
	s = string(d)
	_, err = template.New("").Parse(s)
	if err != nil {
		return httperrors.NewInputParameterError("%s: bad template: %s", k, err)
	}
	return nil
}

func (p *SLoadbalancerAgentParams) initDefault(data *jsonutils.JSONDict) {
	if p.KeepalivedConfTmpl == "" {
		p.KeepalivedConfTmpl = loadbalancerKeepalivedConfTmplDefaultEncoded
	}
	if p.HaproxyConfTmpl == "" {
		p.HaproxyConfTmpl = loadbalancerHaproxyConfTmplDefaultEncoded
	}
	if p.TelegrafConfTmpl == "" {
		p.TelegrafConfTmpl = loadbalancerTelegrafConfTmplDefaultEncoded
	}
	p.Vrrp.initDefault(data)
	p.Haproxy.initDefault(data)
	p.Telegraf.initDefault(data)
}

func (p *SLoadbalancerAgentParams) Validate(data *jsonutils.JSONDict) error {
	p.initDefault(data)
	if err := p.validateTmpl("keepalived_conf_tmpl", p.KeepalivedConfTmpl); err != nil {
		return err
	}
	if err := p.validateTmpl("haproxy_conf_tmpl", p.HaproxyConfTmpl); err != nil {
		return err
	}
	if err := p.validateTmpl("telegraf_conf_tmpl", p.TelegrafConfTmpl); err != nil {
		return err
	}
	if err := p.Vrrp.Validate(data); err != nil {
		return err
	}
	if err := p.Haproxy.Validate(data); err != nil {
		return err
	}
	if err := p.Telegraf.Validate(data); err != nil {
		return err
	}
	return nil
}

func (p *SLoadbalancerAgentParams) String() string {
	return jsonutils.Marshal(p).String()
}

func (p *SLoadbalancerAgentParams) IsZero() bool {
	if *p == (SLoadbalancerAgentParams{}) {
		return true
	}
	return false
}

func (man *SLoadbalancerAgentManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	{
		keyV := map[string]validators.IValidator{
			"hb_timeout": validators.NewNonNegativeValidator("hb_timeout").Default(3600),
			"params":     validators.NewStructValidator("params", &SLoadbalancerAgentParams{}),
		}
		for _, v := range keyV {
			if err := v.Validate(data); err != nil {
				return nil, err
			}
		}
	}
	return man.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (man *SLoadbalancerAgentManager) CleanPendingDeleteLoadbalancers(ctx context.Context, userCred mcclient.TokenCredential) {
	agents := []SLoadbalancerAgent{}
	{
		// find active agents
		err := man.Query().All(&agents)
		if err != nil {
			log.Errorf("query agents failed")
			return
		}
		i := 0
		for _, agent := range agents {
			if !agent.IsActive() {
				continue
			}
			agents[i] = agent
			i++
		}
		agents = agents[:i]
	}
	men := map[string]db.IModelManager{
		"loadbalancers":               LoadbalancerManager,
		"loadbalancer_listeners":      LoadbalancerListenerManager,
		"loadbalancer_listener_rules": LoadbalancerListenerRuleManager,
		"loadbalancer_backend_groups": LoadbalancerBackendGroupManager,
		"loadbalancer_backends":       LoadbalancerBackendManager,
		"loadbalancer_acls":           LoadbalancerAclManager,
		"loadbalancer_certificates":   LoadbalancerCertificateManager,
	}
	agentsData := jsonutils.Marshal(&agents).(*jsonutils.JSONArray)
	for fieldName, man := range men {
		keyPlural := man.KeywordPlural()
		now := time.Now()
		minT := now
		if len(agents) > 0 {
			// find min updated_at seen by these active agents
			for i := 0; i < agentsData.Length(); i++ {
				agentData, _ := agentsData.GetAt(i)
				t, err := agentData.GetTime(fieldName)
				if err != nil {
					continue
				}
				if minT.After(t) {
					minT = t
				}
			}
			if minT.Equal(now) {
				log.Warningf("%s: no agents has reported yet", keyPlural)
				continue
			}
		} else {
			// when no active agents exists, we are free to go
		}
		{
			// find resources pending deleted before minT
			q := man.Query().IsTrue("pending_deleted").LT("pending_deleted_at", minT)
			rows, err := q.Rows()
			if err != nil {
				log.Errorf("%s: query pending_deleted_at < %s: %s", keyPlural, minT, err)
				continue
			}
			m, err := db.NewModelObject(man)
			if err != nil {
				log.Errorf("%s: new model object failed: %s", keyPlural, err)
				continue
			}
			mInitValue := reflect.Indirect(reflect.ValueOf(m))
			m, _ = db.NewModelObject(man)
			for rows.Next() {
				reflect.Indirect(reflect.ValueOf(m)).Set(mInitValue)
				err := q.Row2Struct(rows, m)
				if err != nil {
					log.Errorf("%s: Row2Struct: %s", keyPlural, err)
					continue
				}
				{
					// find real delete method
					rv := reflect.Indirect(reflect.ValueOf(m))
					baseRv := rv.FieldByName("SVirtualResourceBase")
					if !baseRv.IsValid() {
						baseRv = rv.FieldByName("SSharableVirtualResourceBase")
					}
					if !baseRv.IsValid() {
						log.Errorf("%s: cannot find base resource field", keyPlural)
						break // no need to try again
					}
					// now update deleted,deleted_at fields
					realDeleteMethod := baseRv.Addr().MethodByName("Delete")
					retRv := realDeleteMethod.Call([]reflect.Value{
						reflect.ValueOf(ctx),
						reflect.ValueOf(userCred),
					})
					err := retRv[0].Interface()
					if !gotypes.IsNil(err) {
						log.Errorf("%s: real delete failed: %s", keyPlural, err.(error))
					}
				}
			}
		}
	}
}

func (man *SLoadbalancerAgentManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), man.KeywordPlural(), policy.PolicyActionCreate)
}

func (lbagent *SLoadbalancerAgent) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	{
		keyV := map[string]validators.IValidator{
			"hb_timeout": validators.NewNonNegativeValidator("hb_timeout").Optional(true),
		}
		for _, v := range keyV {
			if err := v.Validate(data); err != nil {
				return nil, err
			}
		}
	}
	keys := map[string]time.Time{
		"loadbalancers":               lbagent.Loadbalancers,
		"loadbalancer_listeners":      lbagent.LoadbalancerListeners,
		"loadbalancer_listener_rules": lbagent.LoadbalancerListenerRules,
		"loadbalancer_backend_groups": lbagent.LoadbalancerBackendGroups,
		"loadbalancer_backends":       lbagent.LoadbalancerBackends,
		"loadbalancer_acls":           lbagent.LoadbalancerAcls,
		"loadbalancer_certificates":   lbagent.LoadbalancerCertificates,
	}
	for k, curValue := range keys {
		if !data.Contains(k) {
			continue
		}
		newValue, err := data.GetTime(k)
		if err != nil {
			return nil, httperrors.NewInputParameterError("%s: time error: %s", k, err)
		}
		if newValue.Before(curValue) {
			// this is possible with objects deleted
			data.Remove(k)
			continue
		}
		if now := time.Now(); newValue.After(now) {
			return nil, httperrors.NewInputParameterError("%s: new time is in the future: %s > %s",
				k, newValue, now)
		}
	}
	data.Set("hb_last_seen", jsonutils.NewTimeString(time.Now()))
	return data, nil
}

func (lbagent *SLoadbalancerAgent) AllowPerformHb(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), lbagent.KeywordPlural(), policy.PolicyActionPerform, "hb")
}

func (lbagent *SLoadbalancerAgent) PerformHb(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	_, err := lbagent.GetModelManager().TableSpec().Update(lbagent, func() error {
		lbagent.HbLastSeen = time.Now()
		return nil
	})
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (lbagent *SLoadbalancerAgent) IsActive() bool {
	if lbagent.HbLastSeen.IsZero() {
		return false
	}
	duration := time.Since(lbagent.HbLastSeen).Seconds()
	if int(duration) >= lbagent.HbTimeout {
		return false
	}
	return true
}

func (lbagent *SLoadbalancerAgent) AllowPerformParamsPatch(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), lbagent.KeywordPlural(), policy.PolicyActionPerform, "params-patch")
}

func (lbagent *SLoadbalancerAgent) PerformParamsPatch(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	params := gotypes.DeepCopy(*lbagent.Params).(SLoadbalancerAgentParams)
	d := jsonutils.NewDict()
	d.Set("params", data)
	paramsV := validators.NewStructValidator("params", &params)
	if err := paramsV.Validate(d); err != nil {
		return nil, err
	}
	{
		_, err := lbagent.GetModelManager().TableSpec().Update(lbagent, func() error {
			lbagent.Params = &params
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

const (
	loadbalancerKeepalivedConfTmplDefault = `
global_defs {
	router_id {{ .agent.id }}
	#vrrp_strict
	vrrp_skip_check_adv_addr
	enable_script_security
}

vrrp_instance YunionLB {
	interface {{ .vrrp.interface }}
	virtual_router_id {{ .vrrp.virtual_router_id }}
	authentication {
		auth_type PASS
		auth_pass {{ .vrrp.pass }}
	}
	priority {{ .vrrp.priority }}
	advert_int {{ .vrrp.advert_int }}
	garp_master_refresh {{ .vrrp.garp_master_refresh }}
	{{ if .vrrp.preempt -}} preempt {{- else -}} nopreempt {{- end }}
	virtual_ipaddress {
		{{- printf "\n" }}
		{{- range .vrrp.addresses }}		{{ println . }} {{- end }}
		{{- printf "\t" -}}
	}
}
`
	loadbalancerHaproxyConfTmplDefault = `
global
	maxconn 20480
	tune.ssl.default-dh-param 2048
	{{- println }}
	{{- if .haproxy.global_stats_socket }}	{{ println .haproxy.global_stats_socket }} {{- end }}
	{{- if .haproxy.global_nbthread }}	nbthread {{ println .haproxy.global_nbthread }} {{- end }}
	{{- if .haproxy.global_log }}	{{ println .haproxy.global_log }} {{- end }}

defaults
	timeout connect 10s
	timeout client 60s
	timeout server 60s
	timeout tunnel 1h
	{{- println }}
	{{- if .haproxy.global_log }}	{{ println "log global" }} {{- end }}
	{{- if not .haproxy.log_normal }}	{{ println "option dontlog-normal" }} {{- end }}

listen stats
	mode http
	bind :778
	stats enable
	stats hide-version
	stats realm "Haproxy Statistics"
	stats auth Yunion:LBStats
	stats uri /
`

	loadbalancerTelegrafConfTmplDefault = `
[[outputs.influxdb]]
	urls = ["{{ .telegraf.influx_db_output_url }}"]
	database = "{{ .telegraf.influx_db_output_name }}"

[[inputs.haproxy]]
	interval = "{{ .telegraf.haproxy_input_interval }}s"
	servers = ["{{ .telegraf.haproxy_input_stats_socket }}"]
	keep_field_names = true
`
)

var (
	loadbalancerKeepalivedConfTmplDefaultEncoded = base64.StdEncoding.EncodeToString([]byte(loadbalancerKeepalivedConfTmplDefault))
	loadbalancerHaproxyConfTmplDefaultEncoded    = base64.StdEncoding.EncodeToString([]byte(loadbalancerHaproxyConfTmplDefault))
	loadbalancerTelegrafConfTmplDefaultEncoded   = base64.StdEncoding.EncodeToString([]byte(loadbalancerTelegrafConfTmplDefault))
)
