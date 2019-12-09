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

package models

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/choices"
	"yunion.io/x/onecloud/pkg/util/firewalld"
	"yunion.io/x/onecloud/pkg/util/rand"
)

type SRule struct {
	db.SStandaloneResourceBase

	Prio int `nullable:"false" list:"user" update:"user" create:"optional"`

	MatchSrcNet    string `length:"32" nullable:"false" list:"user" update:"user" create:"optional"`
	MatchDestNet   string `length:"32" nullable:"false" list:"user" update:"user" create:"optional"`
	MatchProto     string `length:"8" nullable:"false" list:"user" update:"user" create:"optional"`
	MatchSrcPort   int    `nullable:"false" list:"user" update:"user" create:"optional"`
	MatchDestPort  int    `nullable:"false" list:"user" update:"user" create:"optional"`
	MatchInIfname  string `length:"32" nullable:"false" list:"user" update:"user" create:"optional"`
	MatchOutIfname string `length:"32" nullable:"false" list:"user" update:"user" create:"optional"`

	Action        string `length:"32" nullable:"false" list:"user" update:"user" create:"required"`
	ActionOptions string `length:"32" nullable:"false" list:"user" update:"user" create:"optional"`

	RouterId string `length:"32" nullable:"false" list:"user" create:"optional"`

	IsSystem bool `nullable:"false" list:"user" create:"optional"`
}

const (
	MIN_PRIO                = 0
	MAX_PRIO                = 2000
	DEF_PRIO                = 0
	DEF_PRIO_ROUTER_FORWARD = 1000
	DEF_PRIO_MASQUERADE     = 1000

	ACT_SNAT           = "SNAT"
	ACT_DNAT           = "DNAT"
	ACT_MASQUERADE     = "MASQUERADE"
	ACT_TCPMSS         = "TCPMSS" // FORWARD chain for now
	ACT_INPUT_ACCEPT   = "INPUT_ACCEPT"
	ACT_FORWARD_ACCEPT = "FORWARD_ACCEPT"

	PROTO_TCP = "tcp"
	PROTO_UDP = "udp"
)

var (
	actionChoices = choices.NewChoices(
		ACT_SNAT,
		ACT_DNAT,
		ACT_MASQUERADE,
		ACT_TCPMSS,

		//"DROP",
		ACT_INPUT_ACCEPT,
		ACT_FORWARD_ACCEPT,
		//"REJECT",
	)
	protoChoices = choices.NewChoices(
		PROTO_TCP,
		PROTO_UDP,
	)
)

type SRuleManager struct {
	db.SStandaloneResourceBaseManager
}

var RuleManager *SRuleManager

func init() {
	RuleManager = &SRuleManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SRule{},
			"rules_tbl",
			"rule",
			"rules",
		),
	}
	RuleManager.SetVirtualObject(RuleManager)
}

func (man *SRuleManager) validateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict, rule *SRule) error {
	isUpdate := rule != nil
	routerV := validators.NewModelIdOrNameValidator("router", "router", ownerId)
	inIfnameV := validators.NewRegexpValidator("match_in_ifname", regexpIfname)
	outIfnameV := validators.NewRegexpValidator("match_out_ifname", regexpIfname)
	protoV := validators.NewStringChoicesValidator("match_proto", protoChoices)
	srcPortV := validators.NewPortValidator("match_src_port")
	destPortV := validators.NewPortValidator("match_dest_port")
	actionV := validators.NewStringChoicesValidator("action", actionChoices)
	actionOptsV := validators.NewStringLenRangeValidator("action_options", 0, 256)
	if isUpdate {
		inIfnameV.Default(rule.MatchInIfname)
		outIfnameV.Default(rule.MatchOutIfname)
		protoV.Default(rule.MatchProto)
		srcPortV.Default(int64(rule.MatchSrcPort))
		destPortV.Default(int64(rule.MatchDestPort))
		actionV.Default(rule.Action)
		actionOptsV.Default(rule.ActionOptions)
	}
	vs := []validators.IValidator{
		inIfnameV.Optional(true),
		outIfnameV.Optional(true),
		validators.NewIPv4PrefixValidator("match_src_net").Optional(true),
		validators.NewIPv4PrefixValidator("match_dest_net").Optional(true),
		protoV.Optional(true),
		srcPortV.Optional(true),
		destPortV.Optional(true),
		actionV,
		actionOptsV.Optional(true),
		routerV,
	}
	for _, v := range vs {
		if isUpdate {
			v.Optional(true)
		}
		if err := v.Validate(data); err != nil {
			return err
		}
	}
	if actionV.Value == ACT_TCPMSS {
		if protoV.Value != "" && protoV.Value != PROTO_TCP {
			return httperrors.NewBadRequestError("TCPMSS only works for proto tcp")
		}
		if protoV.Value == "" {
			data.Set("match_proto", jsonutils.NewString(PROTO_TCP))
		}
	} else if actionV.Value == ACT_DNAT {
		if outIfnameV.Value != "" {
			return httperrors.NewBadRequestError("cannot match out interface for DNAT")
		}
	} else if actionV.Value == ACT_SNAT {
		if inIfnameV.Value != "" {
			return httperrors.NewBadRequestError("cannot match in interface for SNAT")
		}
	}
	if (srcPortV.Value > 0 || destPortV.Value > 0) && protoV.Value == "" {
		return httperrors.NewBadRequestError("protocol must be specified when matching port")
	}

	{
		prioDefault := int64(0)
		if !isUpdate && actionV.Value == ACT_MASQUERADE && !data.Contains("prio") {
			prioDefault = DEF_PRIO_MASQUERADE
		}
		prioV := validators.NewRangeValidator("prio", MIN_PRIO, MAX_PRIO)
		if !isUpdate {
			prioV.Default(prioDefault)
		}
		if err := prioV.Validate(data); err != nil {
			return err
		}
	}

	// XXX validate interface against db
	// XXX validate action options

	if !isUpdate && !data.Contains("name") {
		router := routerV.Model.(*SRouter)
		data.Set("name", jsonutils.NewString(
			router.Name+"-"+rand.String(4),
		))
	}

	return nil
}

func (man *SRuleManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	input := apis.StandaloneResourceCreateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal StandaloneResourceCreateInput fail %s", err)
	}
	input, err = man.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))

	if err := man.validateData(ctx, userCred, ownerId, query, data, nil); err != nil {
		return nil, err
	}
	return data, nil
}

func (man *SRuleManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := man.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	data := query.(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "router", ModelKeyword: "router", OwnerId: userCred},
	})
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (rule *SRule) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if _, err := rule.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data); err != nil {
		return nil, err
	}
	if err := RuleManager.validateData(ctx, userCred, rule.GetOwnerId(), query, data, rule); err != nil {
		return nil, err
	}
	return nil, nil
}

func (man *SRuleManager) removeByRouter(ctx context.Context, userCred mcclient.TokenCredential, router *SRouter) error {
	rules, err := man.getByFilter(map[string]string{
		"router_id": router.Id,
	})
	if err != nil {
		return err
	}
	var errs []error
	for j := range rules {
		if err := rules[j].Delete(ctx, userCred); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.NewAggregate(errs)
}

func (man *SRuleManager) getByFilter(filter map[string]string) ([]SRule, error) {
	rules := []SRule{}
	q := man.Query()
	for key, val := range filter {
		q = q.Equals(key, val)
	}
	if err := db.FetchModelObjects(RuleManager, q, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (man *SRuleManager) getOneByFilter(filter map[string]string) (*SRule, error) {
	rules, err := man.getByFilter(filter)
	if err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		return nil, errNotFound(fmt.Errorf("cannot find rule: %#v", filter))
	}
	if len(rules) > 1 {
		return nil, errMoreThanOne(fmt.Errorf("found more than 1 rules: %#v", filter))
	}
	return &rules[0], nil
}

func (man *SRuleManager) checkExistenceByFilter(filter map[string]string) error {
	rules, err := man.getByFilter(filter)
	if err != nil {
		return err
	}
	if len(rules) > 0 {
		return fmt.Errorf("rule exist: %s(%s)", rules[0].Name, rules[0].Id)
	}
	return nil
}

func (man *SRuleManager) getByRouter(router *SRouter) ([]SRule, error) {
	filter := map[string]string{
		"router_id": router.Id,
	}
	rules, err := man.getByFilter(filter)
	if err != nil {
		return nil, err
	}
	return rules, nil
}

func (man *SRuleManager) addRouterRules(ctx context.Context, userCred mcclient.TokenCredential, router *SRouter) error {
	r := &SRule{
		Prio:     DEF_PRIO_ROUTER_FORWARD,
		RouterId: router.Id,
		Action:   ACT_FORWARD_ACCEPT,
	}
	r.IsSystem = true
	r.Name = router.Name + "-allow-forward-" + rand.String(4)
	r.SetModelManager(man, r)

	err := man.addRule(ctx, userCred, r)
	return err
}

func (man *SRuleManager) addWireguardIfaceRules(ctx context.Context, userCred mcclient.TokenCredential, iface *SIface) error {
	r := &SRule{
		RouterId:      iface.RouterId,
		MatchProto:    PROTO_UDP,
		MatchDestPort: iface.ListenPort,
		Action:        ACT_INPUT_ACCEPT,
	}
	r.IsSystem = true
	r.Name = iface.Name + "-allow-" + fmt.Sprintf("%d-", iface.ListenPort) + rand.String(4)
	r.SetModelManager(man, r)

	rules := man.ifaceTCPMSSRules(ctx, userCred, iface)
	rules = append(rules, r)
	err := man.addRules(ctx, userCred, rules)
	return err
}

func (man *SRuleManager) ifaceTCPMSSRules(ctx context.Context, userCred mcclient.TokenCredential, iface *SIface) []*SRule {
	rules := []*SRule{
		&SRule{
			RouterId:      iface.RouterId,
			MatchInIfname: iface.Ifname,
			MatchProto:    PROTO_TCP,
			Action:        ACT_TCPMSS,
			ActionOptions: "--clamp-mss-to-pmtu",
		},
		&SRule{
			RouterId:       iface.RouterId,
			MatchProto:     PROTO_TCP,
			MatchOutIfname: iface.Ifname,
			Action:         ACT_TCPMSS,
			ActionOptions:  "--clamp-mss-to-pmtu",
		},
	}
	for _, r := range rules {
		r.IsSystem = true
		r.Name = iface.Name + "-tcpmss-" + rand.String(4)
		r.SetModelManager(man, r)
	}
	return rules
}

func (man *SRuleManager) addRules(ctx context.Context, userCred mcclient.TokenCredential, rules []*SRule) error {
	errs := []error{}
	for _, r := range rules {
		err := man.addRule(ctx, userCred, r)
		if err != nil {
			errs = append(errs, err)
			continue
		}
	}
	return errors.NewAggregate(errs)
}

func (man *SRuleManager) addRule(ctx context.Context, userCred mcclient.TokenCredential, rule *SRule) error {
	return man.TableSpec().Insert(rule)
}

func (rule *SRule) firewalldRule() (*firewalld.Rule, error) {
	var (
		prio         int
		table        string
		chain        string
		action       string
		body         string
		matchOthers  []string
		actionOthers []string
	)
	prio = rule.Prio
	switch rule.Action {
	case ACT_SNAT, ACT_DNAT, ACT_MASQUERADE:
		table = "nat"
		if rule.Action == ACT_DNAT {
			chain = "PREROUTING"
		} else {
			chain = "POSTROUTING"
		}
		action = rule.Action
	case ACT_TCPMSS:
		table = "mangle"
		chain = "FORWARD" // save INPUT, OUTPUT for future occasions
		action = rule.Action
		matchOthers = []string{"-m", "tcp", "--tcp-flags", "SYN,RST", "SYN"}
		if rule.ActionOptions == "" {
			actionOthers = []string{"--clamp-mss-to-pmtu"}
		}
	case ACT_INPUT_ACCEPT:
		table = "filter"
		chain = "INPUT"
		action = "ACCEPT"
	case ACT_FORWARD_ACCEPT:
		table = "filter"
		chain = "FORWARD"
		action = "ACCEPT"
	default:
		return nil, fmt.Errorf("unknown rule action: %s", rule.Action)
	}

	{
		elms := []string{}
		if rule.MatchInIfname != "" {
			elms = append(elms, "-i", rule.MatchInIfname)
		}
		if rule.MatchOutIfname != "" {
			elms = append(elms, "-o", rule.MatchOutIfname)
		}
		if rule.MatchSrcNet != "" {
			elms = append(elms, "-s", rule.MatchSrcNet)
		}
		if rule.MatchDestNet != "" {
			elms = append(elms, "-d", rule.MatchDestNet)
		}
		if rule.MatchProto != "" {
			elms = append(elms, "-p", rule.MatchProto)
		}
		if rule.MatchSrcPort > 0 {
			elms = append(elms, "--sport", fmt.Sprintf("%d", rule.MatchSrcPort))
		}
		if rule.MatchDestPort > 0 {
			elms = append(elms, "--dport", fmt.Sprintf("%d", rule.MatchDestPort))
		}
		elms = append(elms, matchOthers...) // XXX empty elm
		elms = append(elms, "-j", action)
		if rule.ActionOptions != "" {
			elms = append(elms, rule.ActionOptions)
		}
		elms = append(elms, actionOthers...)
		body = strings.Join(elms, " ")
	}
	r := firewalld.NewIP4Rule(prio, table, chain, body)
	return r, nil
}

func (man *SRuleManager) firewalldDirectByRouter(router *SRouter) (*firewalld.Direct, error) {
	rules, err := man.getByRouter(router)
	if err != nil {
		return nil, err
	}
	rs := []*firewalld.Rule{}
	errs := []error{}
	for i := range rules {
		rule := &rules[i]
		r, err := rule.firewalldRule()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		rs = append(rs, r)
	}
	return firewalld.NewDirect(rs...), errors.NewAggregate(errs)
}
