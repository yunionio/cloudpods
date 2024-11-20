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

package monitor

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
)

// AlertConfig is a helper to generate monitor service alert related api input
type AlertConfig struct {
	name           string
	frequencyStr   string
	frequency      int64
	forTime        int64
	level          string
	enabled        bool
	conditions     []*AlertCondition
	execErrorState string
	noDataState    string
	UsedBy         string
}

func NewAlertConfig(name string, frequency string, enabled bool) (*AlertConfig, error) {
	freq, err := time.ParseDuration(frequency)
	if err != nil {
		return nil, err
	}
	input := &AlertConfig{
		name:         name,
		frequencyStr: frequency,
		frequency:    int64(freq / time.Second),
		level:        "",
		enabled:      enabled,
		conditions:   make([]*AlertCondition, 0),
	}
	return input, nil
}

func (c *AlertConfig) ExecutionErrorState(s string) *AlertConfig {
	c.execErrorState = s
	return c
}

func (c *AlertConfig) NoDataState(s string) *AlertConfig {
	c.noDataState = s
	return c
}

func (c *AlertConfig) Level(l string) *AlertConfig {
	c.level = l
	return c
}

func (c *AlertConfig) Enable(e bool) *AlertConfig {
	c.enabled = e
	return c
}

func (c *AlertConfig) ToAlertCreateInput() monitor.AlertCreateInput {
	return monitor.AlertCreateInput{
		Name:                c.name,
		Frequency:           c.frequency,
		Settings:            c.ToAlertSetting(),
		Enabled:             &c.enabled,
		Level:               c.level,
		For:                 c.forTime,
		ExecutionErrorState: c.execErrorState,
		NoDataState:         c.noDataState,
		UsedBy:              c.UsedBy,
	}
}

func (c *AlertConfig) ToCommonMetricInputQuery() monitor.CommonMetricInputQuery {
	conds := make([]*monitor.CommonAlertQuery, len(c.conditions))
	for i, cc := range c.conditions {
		tmp := cc.ToCommonAlertQuery()
		conds[i] = &tmp
	}
	return monitor.CommonMetricInputQuery{
		MetricQuery: conds,
	}
}

func (c *AlertConfig) ToCommonAlertCreateInput(bi *monitor.CommonAlertCreateBaseInput) monitor.CommonAlertCreateInput {
	mq := c.ToCommonMetricInputQuery()
	ai := c.ToAlertCreateInput()
	if bi == nil {
		bi = new(monitor.CommonAlertCreateBaseInput)
	}
	input := monitor.CommonAlertCreateInput{
		CommonMetricInputQuery:     mq,
		AlertCreateInput:           ai,
		CommonAlertCreateBaseInput: *bi,
		Period:                     c.frequencyStr,
	}
	input.UsedBy = c.UsedBy
	return input
}

func (c *AlertConfig) ToAlertSetting() monitor.AlertSetting {
	conds := make([]monitor.AlertCondition, len(c.conditions))
	for i, cc := range c.conditions {
		conds[i] = cc.ToCondition()
	}
	return monitor.AlertSetting{
		Conditions: conds,
	}
}

func (c *AlertConfig) Condition(database string, measurement string) *AlertCondition {
	cc := NewAlertCondition(database, measurement)
	c.conditions = append(c.conditions, cc)
	return cc
}

func (c *AlertConfig) AND(cs ...*AlertCondition) *AlertConfig {
	if len(cs) == 0 {
		return c
	}
	for _, cond := range cs {
		cond.setOperator("AND")
	}
	return c
}

func (c *AlertConfig) OR(cs ...*AlertCondition) *AlertConfig {
	if len(cs) == 0 {
		return c
	}
	for _, cond := range cs {
		cond.setOperator("OR")
	}
	return c
}

type AlertCondition struct {
	operator  string
	reducer   *monitor.Condition
	evaluator *monitor.Condition
	query     *AlertQuery
}

func NewAlertCondition(
	database string,
	measurement string,
) *AlertCondition {
	c := &AlertCondition{
		query: NewAlertQuery(database, measurement),
	}
	// set default avg reducer
	c.Avg()
	return c
}

func (c *AlertCondition) ToCondition() monitor.AlertCondition {
	return monitor.AlertCondition{
		Type:      "query",
		Query:     c.query.ToAlertQuery(),
		Reducer:   *c.reducer,
		Evaluator: *c.evaluator,
		Operator:  c.operator,
	}
}

func (c *AlertCondition) ToCommonAlertQuery() monitor.CommonAlertQuery {
	aq := c.query.ToAlertQuery()
	eval := c.evaluator.Type
	if !utils.IsInStringArray(eval, []string{"lt", "gt", "eq"}) {
		panic(fmt.Sprintf("Invalid evaluator %q", eval))
	}
	var comp string
	switch eval {
	case "lt":
		comp = "<="
	case "gt":
		comp = ">="
	case "eq":
		comp = "=="
	}
	return monitor.CommonAlertQuery{
		AlertQuery: &aq,
		Reduce:     c.reducer.Type,
		Comparator: comp,
		Threshold:  c.evaluator.Params[0],
		// TODO: figure out what's the meaning of FieldOpt
		FieldOpt:      "",
		ConditionType: "query",
	}
}

func (c *AlertCondition) setOperator(op string) *AlertCondition {
	c.operator = op
	return c
}

func (c *AlertCondition) setReducer(typ string, params ...float64) *AlertCondition {
	c.reducer = &monitor.Condition{
		Type: typ,
	}
	c.reducer.Params = params
	return c
}

func (c *AlertCondition) Avg() *AlertCondition {
	return c.setReducer("avg")
}

func (c *AlertCondition) Sum() *AlertCondition {
	return c.setReducer("sum")
}

func (c *AlertCondition) Min() *AlertCondition {
	return c.setReducer("min")
}

func (c *AlertCondition) Max() *AlertCondition {
	return c.setReducer("max")
}

func (c *AlertCondition) Count() *AlertCondition {
	return c.setReducer("count")
}

func (c *AlertCondition) Last() *AlertCondition {
	return c.setReducer("last")
}

func (c *AlertCondition) Median() *AlertCondition {
	return c.setReducer("median")
}

func (c *AlertCondition) setEvaluator(typ string, threshold float64) *AlertCondition {
	c.evaluator = &monitor.Condition{
		Type:   typ,
		Params: []float64{threshold},
	}
	return c
}

// LessThan is evaluator part
func (c *AlertCondition) LT(threshold float64) *AlertCondition {
	return c.setEvaluator("lt", threshold)
}

// GreaterThan is evaluator part
func (c *AlertCondition) GT(threshold float64) *AlertCondition {
	return c.setEvaluator("gt", threshold)
}

func (c *AlertCondition) Query() *AlertQuery {
	return c.query
}

type AlertQuery struct {
	from         string
	to           string
	alias        string
	tz           string
	database     string
	measurement  string
	interval     string
	policy       string
	resultFormat string

	selects            *AlertQuerySelects
	where              *AlertQueryWhere
	groupBy            *AlertQueryGroupBy
	resultReducer      *monitor.Condition
	resultReducerOrder monitor.ResultReducerOrder
}

func NewAlertQuery(database string, measurement string) *AlertQuery {
	q := &AlertQuery{
		selects: new(AlertQuerySelects),
		where:   new(AlertQueryWhere),
		groupBy: new(AlertQueryGroupBy),
	}
	q = q.Database(database).Measurement(measurement)
	return q
}

func (q *AlertQuery) ToAlertQuery() monitor.AlertQuery {
	return monitor.AlertQuery{
		Model: q.ToMetricQuery(),
		From:  q.from,
		To:    q.to,
	}
}

func (q *AlertQuery) ToMetricQuery() monitor.MetricQuery {
	return monitor.MetricQuery{
		Alias:        q.alias,
		Tz:           q.tz,
		Database:     q.database,
		Measurement:  q.measurement,
		Tags:         q.where.ToTags(),
		GroupBy:      q.groupBy.ToGroupBy(),
		Selects:      q.selects.ToSelects(),
		Interval:     q.interval,
		Policy:       q.policy,
		ResultFormat: q.resultFormat,
	}
}

func (q *AlertQuery) ToTsdbQuery() *tsdb.TsdbQuery {
	timeRange := tsdb.NewTimeRange(q.from, q.to)
	tsdbQ := &tsdb.TsdbQuery{
		TimeRange: timeRange,
		Queries: []*tsdb.Query{
			{
				MetricQuery: q.ToMetricQuery(),
			},
		},
	}
	return tsdbQ
}

func (q *AlertQuery) From(from string) *AlertQuery {
	q.from = from
	return q
}

func (q *AlertQuery) To(to string) *AlertQuery {
	q.to = to
	return q
}

func (q *AlertQuery) Alias(alias string) *AlertQuery {
	q.alias = alias
	return q
}

func (q *AlertQuery) Tz(tz string) *AlertQuery {
	q.tz = tz
	return q
}

func (q *AlertQuery) Database(db string) *AlertQuery {
	q.database = db
	return q
}

func (q *AlertQuery) Measurement(m string) *AlertQuery {
	q.measurement = m
	return q
}

func (q *AlertQuery) Interval(i string) *AlertQuery {
	q.interval = i
	return q
}

func (q *AlertQuery) Policy(p string) *AlertQuery {
	q.policy = p
	return q
}

func (q *AlertQuery) Selects() *AlertQuerySelects {
	q.selects = &AlertQuerySelects{
		parts: make([]*AlertQuerySelect, 0),
	}
	return q.selects
}

func (q *AlertQuery) Where() *AlertQueryWhere {
	w := &AlertQueryWhere{
		parts: make([]monitor.MetricQueryTag, 0),
	}
	w.AND()
	q.where = w
	return w
}

func (q *AlertQuery) GroupBy() *AlertQueryGroupBy {
	g := &AlertQueryGroupBy{parts: make([]monitor.MetricQueryPart, 0)}
	q.groupBy = g
	return g
}

func (q *AlertQuery) Reducer(rType string, params []float64) *AlertQuery {
	q.resultReducer = &monitor.Condition{
		Type:   rType,
		Params: params,
	}
	return q
}

func (q *AlertQuery) ReducerOrder(rType monitor.ResultReducerOrder) *AlertQuery {
	q.resultReducerOrder = rType
	return q
}

type AlertQuerySelects struct {
	parts []*AlertQuerySelect
}

type AlertQuerySelect struct {
	monitor.MetricQuerySelect
}

func (s *AlertQuerySelects) Select(fieldName string) *AlertQuerySelect {
	if s.parts == nil {
		s.parts = make([]*AlertQuerySelect, 0)
	}
	sel := make([]monitor.MetricQueryPart, 0)
	sel = append(sel, monitor.NewMetricQueryPartField(fieldName))
	part := &AlertQuerySelect{sel}
	s.parts = append(s.parts, part)
	return part
}

func (s *AlertQuerySelects) ToSelects() []monitor.MetricQuerySelect {
	ret := make([]monitor.MetricQuerySelect, len(s.parts))
	for i, p := range s.parts {
		ret[i] = p.MetricQuerySelect
	}
	return ret
}

func (s *AlertQuerySelect) addPart(p monitor.MetricQueryPart) *AlertQuerySelect {
	s.MetricQuerySelect = append(s.MetricQuerySelect, p)
	return s
}

// Aggregations method
func (s *AlertQuerySelect) MEAN() *AlertQuerySelect {
	return s.addPart(monitor.NewMetricQueryPartMean())
}

func (s *AlertQuerySelect) COUNT() *AlertQuerySelect {
	return s.addPart(monitor.NewMetricQueryPartCount())
}

func (s *AlertQuerySelect) DISTINCT() *AlertQuerySelect {
	return s.addPart(monitor.NewMetricQueryPartDistinct())
}

func (s *AlertQuerySelect) SUM() *AlertQuerySelect {
	return s.addPart(monitor.NewMetricQueryPartSum())
}

func (s *AlertQuerySelect) MIN() *AlertQuerySelect {
	return s.addPart(monitor.NewMetricQueryPartMin())
}

func (s *AlertQuerySelect) MAX() *AlertQuerySelect {
	return s.addPart(monitor.NewMetricQueryPartMax())
}

func (s *AlertQuerySelect) LAST() *AlertQuerySelect {
	return s.addPart(monitor.NewMetricQueryPartLast())
}

// AS is alias method
func (s *AlertQuerySelect) AS(alias string) *AlertQuerySelect {
	return s.addPart(monitor.NewMetricQueryPartAS(alias))
}

// MATH method
func (s *AlertQuerySelect) MATH(op string, val string) *AlertQuerySelect {
	return s.addPart(monitor.NewMetricQueryPartMath(op, val))
}

type AlertQueryWhere struct {
	parts []monitor.MetricQueryTag
	cond  string
}

func (w *AlertQueryWhere) add(tags ...monitor.MetricQueryTag) *AlertQueryWhere {
	w.parts = append(w.parts, tags...)
	return w
}

func (w *AlertQueryWhere) condAdd(cond string, tags ...monitor.MetricQueryTag) *AlertQueryWhere {
	for i := range tags {
		tags[i].Condition = cond
	}
	w.add(tags...)
	return w
}

func (w *AlertQueryWhere) filter(op string, key string, value string) *AlertQueryWhere {
	if len(w.parts) == 0 {
		w.add(w.newTag(op, key, value))
		return w
	}
	return w.condAdd(w.cond, w.newTag(op, key, value))
}

func (w *AlertQueryWhere) Equal(key string, value string) *AlertQueryWhere {
	return w.filter("=", key, value)
}

func (w *AlertQueryWhere) NotEqual(key string, value string) *AlertQueryWhere {
	return w.filter("!=", key, value)
}

func (w *AlertQueryWhere) LT(key string, value string) *AlertQueryWhere {
	return w.filter("<", key, value)
}

func (w *AlertQueryWhere) GT(key string, value string) *AlertQueryWhere {
	return w.filter(">", key, value)
}

func (w *AlertQueryWhere) AND() *AlertQueryWhere {
	w.cond = "AND"
	return w
}

func (w *AlertQueryWhere) OR() *AlertQueryWhere {
	w.cond = "OR"
	return w
}

func (w *AlertQueryWhere) REGEX(key, val string) *AlertQueryWhere {
	return w.filter("=~", key, fmt.Sprintf("/%s/", val))
}

func (w *AlertQueryWhere) IN(key string, vals []string) *AlertQueryWhere {
	if len(vals) == 0 {
		return w
	}
	valStr := strings.Join(vals, "|")
	return w.REGEX(key, valStr)
}

func (w *AlertQueryWhere) AddTag(tag *monitor.MetricQueryTag) *AlertQueryWhere {
	if tag == nil {
		return w
	}
	if tag.Condition != "" {
		w.cond = tag.Condition
	}
	return w.filter(tag.Operator, tag.Key, tag.Value)
}

func (w *AlertQueryWhere) newTag(op string, key string, value string) monitor.MetricQueryTag {
	return monitor.MetricQueryTag{
		Key:      key,
		Operator: op,
		Value:    value,
	}
}

func (w *AlertQueryWhere) ToTags() []monitor.MetricQueryTag {
	return w.parts
}

type AlertQueryGroupBy struct {
	parts []monitor.MetricQueryPart
}

func (g *AlertQueryGroupBy) addPart(typ string, params ...string) *AlertQueryGroupBy {
	g.parts = append(g.parts, monitor.MetricQueryPart{
		Type:   typ,
		Params: params,
	})
	return g
}

func (g *AlertQueryGroupBy) TIME(val string) *AlertQueryGroupBy {
	return g.addPart("time", val)
}

func (g *AlertQueryGroupBy) TAG(val string) *AlertQueryGroupBy {
	return g.addPart("tag", val)
}

func (g *AlertQueryGroupBy) FILL_NULL() *AlertQueryGroupBy {
	return g.FILL("null")
}

func (g *AlertQueryGroupBy) FILL(val string) *AlertQueryGroupBy {
	return g.addPart("fill", val)
}

func (g *AlertQueryGroupBy) ToGroupBy() []monitor.MetricQueryPart {
	return g.parts
}

type MetricQueryInput struct {
	from            string `json:"from"`
	to              string `json:"to"`
	scope           string `json:"scope"`
	slimit          string `json:"slimit"`
	soffset         string `json:"soffset"`
	unit            bool   `json:"unit"`
	interval        string `json:"interval"`
	domainId        string `json:"domain_id"`
	projectId       string `json:"project_id"`
	showMeta        bool   `json:"show_meta"`
	skipCheckSeries bool   `json:"skip_check_series"`
	query           *AlertQuery
}

func NewMetricQueryInput(measurement string) *MetricQueryInput {
	return NewMetricQueryInputWithDB("", measurement)
}

func NewMetricQueryInputWithDB(db string, measurement string) *MetricQueryInput {
	return &MetricQueryInput{
		query: NewAlertQuery(db, measurement),
	}
}

func (input *MetricQueryInput) Interval(interval string) *MetricQueryInput {
	input.interval = interval
	return input
}

func (input *MetricQueryInput) From(from time.Time) *MetricQueryInput {
	input.from = fmt.Sprintf("%d", from.UnixMilli())
	return input
}

func (input *MetricQueryInput) To(to time.Time) *MetricQueryInput {
	input.to = fmt.Sprintf("%d", to.UnixMilli())
	return input
}

func (input *MetricQueryInput) Scope(scope string) *MetricQueryInput {
	input.scope = scope
	return input
}

func (input *MetricQueryInput) SkipCheckSeries(skip bool) *MetricQueryInput {
	input.skipCheckSeries = skip
	return input
}

func (input *MetricQueryInput) Selects() *AlertQuerySelects {
	return input.query.Selects()
}

func (input *MetricQueryInput) Where() *AlertQueryWhere {
	return input.query.Where()
}

func (input *MetricQueryInput) GroupBy() *AlertQueryGroupBy {
	return input.query.GroupBy()
}

func (input *MetricQueryInput) Reducer(rType string, params []float64) *AlertQuery {
	return input.query.Reducer(rType, params)
}

func (input *MetricQueryInput) ToQueryData() *monitor.MetricQueryInput {
	data := &monitor.MetricQueryInput{
		From:            input.from,
		To:              input.to,
		Scope:           input.scope,
		Slimit:          input.slimit,
		Soffset:         input.soffset,
		Unit:            input.unit,
		Interval:        input.interval,
		DomainId:        input.domainId,
		ProjectId:       input.projectId,
		ShowMeta:        input.showMeta,
		SkipCheckSeries: input.skipCheckSeries,
	}
	data.MetricQuery = []*monitor.AlertQuery{
		{
			Model:              input.query.ToMetricQuery(),
			ResultReducer:      input.query.resultReducer,
			ResultReducerOrder: input.query.resultReducerOrder,
		},
	}

	jsonData := jsonutils.Marshal(data).(*jsonutils.JSONDict)
	digest := monitor.DigestQuerySignature(jsonData)
	data.Signature = digest

	return data
}
