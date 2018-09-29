package models

import (
	"encoding/base64"
	"fmt"
	"text/template"

	"yunion.io/x/onecloud/pkg/mcclient/models"
)

type AgentParams struct {
	AgentModel           *models.LoadbalancerAgent
	KeepalivedConfigTmpl *template.Template
	HaproxyConfigTmpl    *template.Template
	Data                 map[string]interface{}
}

func NewAgentParams(agent *models.LoadbalancerAgent) (*AgentParams, error) {
	b64s := map[string]string{
		"keepalived_conf_tmpl": agent.Params.KeepalivedConfTmpl,
		"haproxy_conf_tmpl":    agent.Params.HaproxyConfTmpl,
	}
	tmpls := map[string]*template.Template{}
	for name, b64 := range b64s {
		d, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil, fmt.Errorf("%s: invalid base64 string: %s", name, err)
		}
		tmpl, err := template.New(name).Parse(string(d))
		if err != nil {
			return nil, fmt.Errorf("%s: invalid template: %s", name, err)
		}
		tmpls[name] = tmpl
	}
	dataAgent := map[string]interface{}{
		"id":   agent.Id,
		"name": agent.Name,
	}
	dataVrrp := map[string]interface{}{
		"priority":            agent.Params.Vrrp.Priority,
		"virtual_router_id":   agent.Params.Vrrp.VirtualRouterId,
		"garp_master_refresh": agent.Params.Vrrp.GarpMasterRefresh,
		"preempt":             agent.Params.Vrrp.Preempt,
		"interface":           agent.Params.Vrrp.Interface,
		"advert_int":          agent.Params.Vrrp.AdvertInt,
		"pass":                agent.Params.Vrrp.Pass,
	}
	dataHaproxy := map[string]interface{}{
		"global_log":      agent.Params.Haproxy.GlobalLog,
		"global_nbthread": agent.Params.Haproxy.GlobalNbthread,
		"log_http":        agent.Params.Haproxy.LogHttp,
		"log_tcp":         agent.Params.Haproxy.LogTcp,
		"log_normal":      agent.Params.Haproxy.LogNormal,
	}
	data := map[string]interface{}{
		"agent":   dataAgent,
		"vrrp":    dataVrrp,
		"haproxy": dataHaproxy,
	}
	agentParams := &AgentParams{
		AgentModel:           agent,
		KeepalivedConfigTmpl: tmpls["keepalived_conf_tmpl"],
		HaproxyConfigTmpl:    tmpls["haproxy_conf_tmpl"],
		Data:                 data,
	}
	return agentParams, nil
}

func (p *AgentParams) Equal(p2 *AgentParams) bool {
	if p == nil && p2 == nil {
		return true
	}
	if p == nil || p2 == nil {
		return false
	}
	agentP := p.AgentModel
	agentP2 := p2.AgentModel
	if agentP.Params != agentP2.Params {
		return false
	}
	return true
}

func (p *AgentParams) setXxParams(xx, k string, v interface{}) map[string]interface{} {
	var dt map[string]interface{}
	d, ok := p.Data[xx]
	if !ok {
		dt = map[string]interface{}{}
		p.Data[xx] = dt
	} else {
		dt = d.(map[string]interface{})
	}
	dt[k] = v
	return dt
}

func (p *AgentParams) SetVrrpParams(k string, v interface{}) map[string]interface{} {
	return p.setXxParams("vrrp", k, v)
}

func (p *AgentParams) SetHaproxyParams(k string, v interface{}) map[string]interface{} {
	return p.setXxParams("haproxy", k, v)
}

func (p *AgentParams) KeepalivedConfig() {
}
