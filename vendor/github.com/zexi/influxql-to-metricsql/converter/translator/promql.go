package translator

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/influxdata/influxql"
	"github.com/influxdata/promql/v2"
	"github.com/influxdata/promql/v2/pkg/labels"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
)

const UNION_RESULT_NAME = "__union_result__"

const (
	CALL_TOP        = "top"
	CALL_BOTTOM     = "bottom"
	CALL_PERCENTILE = "percentile"
	CALL_SUM        = "sum"
	CALL_MIN        = "min"
	CALL_MAX        = "max"
)

var MUL_ARGS_AGGREGATOR MulArgsAggregator = []string{CALL_TOP, CALL_PERCENTILE, CALL_BOTTOM}

type MulArgsAggregator []string

func (m MulArgsAggregator) Has(aggr string) bool {
	for _, a := range m {
		if a == aggr {
			return true
		}
	}
	return false
}

type promQL struct {
	groupByWildcard bool
	timeRange       *influxql.TimeRange
	fieldIsWildcard bool
	fieldIsRegex    bool
	measurement     string
	labelsVisitor   *labelsVisitor
}

func NewPromQL() Translator {
	return &promQL{
		labelsVisitor: newLabelsVisitor(),
	}
}

func (m *promQL) Translate(s influxql.Statement) (string, error) {
	selectS, ok := s.(*influxql.SelectStatement)
	if !ok {
		return "", errors.Errorf("Only SelectStatement is supported, input %t", s)
	}
	return m.translate(selectS)
}

func (m *promQL) GetTimeRange() *influxql.TimeRange {
	return m.timeRange
}

type fieldResult struct {
	metricName string
	aggrOps    []*AggrOperator
	expr       promql.Expr
}

func newFieldResult(metricName string, ops []*AggrOperator, expr promql.Expr) *fieldResult {
	return &fieldResult{
		metricName: metricName,
		aggrOps:    ops,
		expr:       expr,
	}
}

func (m *promQL) translateField(s *influxql.SelectStatement, field *influxql.Field) (*fieldResult, error) {
	metricName, err := getMetricName(s.Sources, field)
	if err != nil {
		if errors.Cause(err) == ErrVariableIsWildcard {
			m.measurement = metricName
			m.fieldIsWildcard = true
		} else {
			return nil, errors.Wrap(err, "getMetricName")
		}
	}

	aggrOps, err := getAggrOperators(field)
	if err != nil {
		return nil, errors.Wrap(err, "get field aggregate operator")
	}
	cond, timeRange, err := getTimeRange(s.Condition)
	if err != nil {
		return nil, errors.Wrap(err, "getTimeRange")
	}
	m.timeRange = timeRange

	matchers, err := m.getLabels(m.labelsVisitor, cond)
	if err != nil {
		return nil, errors.Wrap(err, "get matchers")
	}
	m.fieldIsRegex = false
	if !m.fieldIsWildcard {
		if isRegexMetricName(metricName) {
			m.fieldIsRegex = true
			regexPattern := trimRegexDelimiters(metricName)
			nameMatcher, _ := labels.NewMatcher(labels.MatchRegexp, labels.MetricName, regexPattern)
			matchers = append(matchers, nameMatcher)
		} else {
			nameMatcher, _ := labels.NewMatcher(labels.MatchEqual, labels.MetricName, metricName)
			matchers = append(matchers, nameMatcher)
		}
	}

	lookbehindWin, groups, err := m.getGroups(s.Dimensions)
	if err != nil {
		return nil, errors.Wrap(err, "get groups")
	}
	//interval, err := s.GroupByInterval()
	//if err != nil {
	//	return "", errors.Wrap(err, "GroupByInterval")
	//}
	//fmt.Printf("==get interval: %#v\n", interval)

	expr, err := m.generateExpr(metricName, matchers, lookbehindWin, aggrOps, groups)
	if err != nil {
		return nil, errors.Wrap(err, "generate expression")
	}
	if binExpr, ok := field.Expr.(*influxql.BinaryExpr); ok {
		expr, err = wrapBinaryExpr(binExpr, expr)
		if err != nil {
			return nil, errors.Wrap(err, "wrap binary expression")
		}
	}
	return newFieldResult(metricName, aggrOps, expr), nil
}

func (m *promQL) translate(s *influxql.SelectStatement) (string, error) {
	exprs := make([]*fieldResult, 0)
	var resultExpr promql.Expr
	for _, field := range s.Fields {
		m.labelsVisitor = newLabelsVisitor()
		expr, err := m.translateField(s, field)
		if err != nil {
			return "", errors.Wrapf(err, "translate field %s", field)
		}
		exprs = append(exprs, expr)
	}

	if len(exprs) == 1 {
		resultExpr = exprs[0].expr
	} else {
		// union field expr
		resultExpr = unionFieldsExpr(exprs)
	}

	return m.formatExpr(resultExpr), nil
}

func unionFieldsExpr(exprs []*fieldResult) promql.Expr {
	result := make([]promql.Expr, len(exprs))
	// 1. wrap each expr with label_set: https://docs.victoriametrics.com/MetricsQL.html#label_set
	setKey := UNION_RESULT_NAME
	for i := range exprs {
		expr := exprs[i]
		setValue := expr.metricName
		if len(expr.aggrOps) > 0 {
			opsNames := make([]string, len(expr.aggrOps))
			for i := range expr.aggrOps {
				opsNames[i] = expr.aggrOps[i].Name
			}
			setValue = fmt.Sprintf("%s_%s", strings.Join(opsNames, "_"), expr.metricName)
		}
		result[i] = &promql.Call{
			Func: &promql.Function{
				Name:       "label_set",
				ArgTypes:   []promql.ValueType{promql.ValueTypeVector, promql.ValueTypeString, promql.ValueTypeString},
				Variadic:   1,
				ReturnType: promql.ValueTypeVector,
			},
			Args: promql.Expressions{
				expr.expr,
				&promql.StringLiteral{setKey},
				&promql.StringLiteral{setValue},
			},
		}
	}
	// 2. use union: https://docs.victoriametrics.com/MetricsQL.html#union
	return &promql.Call{
		Func: &promql.Function{
			Name:       "union",
			Variadic:   1,
			ReturnType: promql.ValueTypeVector,
		},
		Args: result,
	}
}

func getTimeRange(cond influxql.Expr) (influxql.Expr, *influxql.TimeRange, error) {
	// parse time range
	//mustParseTime := func(value string) time.Time {
	//	ts, err := time.Parse(time.RFC3339, value)
	//	if err != nil {
	//		panic(fmt.Errorf("unable to parse time: %s", err))
	//	}
	//	return ts
	//}
	//now := mustParseTime("2000-01-01T00:00:00Z")
	valuer := influxql.NowValuer{
		Now:      time.Now(),
		Location: time.UTC,
	}
	cond, timeRange, err := influxql.ConditionExpr(cond, &valuer)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "parse time range from %q", cond)
	}
	if timeRange.IsZero() {
		return cond, nil, nil
	}
	// process maxTime
	if !timeRange.MaxTime().IsZero() {
		year, month, day := timeRange.Max.Date()
		// FIX: time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)
		if year == 1 && month == 1 && day == 1 {
			timeRange.Max = time.Now()
		}
	}
	return cond, &timeRange, nil
}

func (m promQL) generateExpr(
	metricName string,
	ls []*labels.Matcher,
	lookbehindWindow string,
	aggrOps []*AggrOperator,
	groups []string) (promql.Expr, error) {
	//fmt.Printf("=====name: %s, labels: %#v, lookbehindWindow: %q, aggrOps: %#v, groups: %#v\n", metricName, ls, lookbehindWindow, aggrOps, groups)
	for _, l := range ls {
		fmt.Printf("label: %s\n", l.String())
	}

	if m.fieldIsWildcard {
		measurementM, _ := labels.NewMatcher(labels.MatchRegexp, labels.MetricName, fmt.Sprintf("^%s_.*", m.measurement))
		ls = append(ls, measurementM)
	}

	var result promql.Expr
	if len(aggrOps) != 0 {
		if lookbehindWindow == "" {
			lookbehindWindow = "1m"
		}
		dur, err := model.ParseDuration(lookbehindWindow)
		if err != nil {
			return nil, errors.Wrapf(err, "ParseDuration: %q", lookbehindWindow)
		}
		ms := &promql.MatrixSelector{
			LabelMatchers: ls,
			Range:         time.Duration(dur),
		}
		if !m.fieldIsWildcard && !m.fieldIsRegex {
			ms.Name = metricName
		}
		result = ms
	} else {
		vs := &promql.VectorSelector{
			LabelMatchers: ls,
		}
		if !m.fieldIsWildcard && !m.fieldIsRegex {
			vs.Name = metricName
		}
		result = vs
	}

	if len(groups) != 0 && len(aggrOps) == 0 {
		return nil, errors.Errorf("Can't use group by when aggregate operator is empty")
	}

	result = getAggrExpr(aggrOps, result)

	//fmt.Printf("=====m.GroupByWildcard: %v, %#v, aggrOps: %#v\n", m.groupByWildcard, result, aggrOps)

	shouldSkipAggr := func(opName string) bool {
		switch opName {
		case CALL_PERCENTILE, CALL_TOP, CALL_BOTTOM, "last":
			return true
		}
		return false
	}

	if len(aggrOps) != 0 {
		op := promql.ItemAvg
		opName := aggrOps[0].Name
		if !shouldSkipAggr(opName) {
			switch opName {
			case CALL_SUM:
				op = promql.ItemSum
			case CALL_MAX:
				op = promql.ItemMax
			case CALL_MIN:
				op = promql.ItemMin
			}
			expr := &promql.AggregateExpr{
				Op:   op,
				Expr: result,
			}
			if len(groups) != 0 {
				expr.Grouping = groups
			}
			if !m.groupByWildcard {
				result = expr
			}
		}
	}

	return result, nil
}

func (m promQL) formatExpr(expr promql.Expr) string {
	initialExpr := expr.String()
	//fmt.Printf("---replaceLabels: %#v\n", m.labelsVisitor.replaceLabels)
	//fmt.Printf("--src: %s\n", initialExpr)
	if len(m.labelsVisitor.replaceLabels) > 0 && len(m.labelsVisitor.labels) > 1 {
		for src, replace := range m.labelsVisitor.replaceLabels {
			initialExpr = strings.ReplaceAll(initialExpr, src, replace)
		}
	}
	return initialExpr
}

func newAggrExpr(name string, argType promql.ValueType, returnType promql.ValueType, restExpr promql.Expr) promql.Expr {
	return newAggrExprWithArgs(name, []promql.ValueType{argType}, returnType, promql.Expressions{restExpr})
}

func newAggrExprWithArgs(name string, args []promql.ValueType, returnType promql.ValueType, restExprs promql.Expressions) promql.Expr {
	return &promql.Call{
		Func: &promql.Function{
			Name:       name,
			ArgTypes:   args,
			Variadic:   0,
			ReturnType: returnType,
		},
		Args: restExprs,
	}
}

func getAggrExpr(ops []*AggrOperator, expr promql.Expr) promql.Expr {
	if len(ops) == 0 {
		return expr
	}
	aggrOp := ops[0]
	restOps := ops[1:]
	restExpr := getAggrExpr(restOps, expr)
	switch aggrOp.Name {
	case "abs":
		// https://prometheus.io/docs/prometheus/latest/querying/functions/#abs
		expr = newAggrExpr("abs", promql.ValueTypeVector, promql.ValueTypeVector, restExpr)
	case "mean":
		// https://docs.victoriametrics.com/MetricsQL.html#avg_over_time
		expr = newAggrExpr("avg_over_time", promql.ValueTypeMatrix, promql.ValueTypeVector, restExpr)
	case "last":
		// https://docs.victoriametrics.com/MetricsQL.html#last_over_time
		expr = newAggrExpr("last_over_time", promql.ValueTypeMatrix, promql.ValueTypeVector, restExpr)
	case "stddev":
		// https://prometheus.io/docs/prometheus/latest/querying/functions/#aggregation_over_time
		expr = newAggrExpr("stddev_over_time", promql.ValueTypeMatrix, promql.ValueTypeVector, restExpr)
	case "count":
		// use count, not use 'count_over_time' https://docs.victoriametrics.com/MetricsQL.html#count_over_time
		expr = newAggrExpr("count", promql.ValueTypeMatrix, promql.ValueTypeVector, restExpr)
	case "median":
		// https://docs.victoriametrics.com/MetricsQL.html#median_over_time
		expr = newAggrExpr("median_over_time", promql.ValueTypeMatrix, promql.ValueTypeVector, restExpr)
	/*case "max":
		// https://docs.victoriametrics.com/MetricsQL.html#max_over_time
		expr = newAggrExpr("max", promql.ValueTypeMatrix, promql.ValueTypeVector, restExpr)
	case "min":
		// https://docs.victoriametrics.com/MetricsQL.html#min_over_time
		expr = newAggrExpr("min", promql.ValueTypeMatrix, promql.ValueTypeVector, restExpr)*/
	case "mode":
		// https://docs.victoriametrics.com/MetricsQL.html#mode_over_time
		expr = newAggrExpr("mode_over_time", promql.ValueTypeMatrix, promql.ValueTypeVector, restExpr)
	case "integral":
		// https://docs.victoriametrics.com/MetricsQL.html#integrate
		expr = newAggrExpr("integrate", promql.ValueTypeMatrix, promql.ValueTypeVector, restExpr)
	case "distinct":
		expr = newAggrExpr("distinct", promql.ValueTypeMatrix, promql.ValueTypeVector, restExpr)
	case CALL_TOP:
		expr = newAggrExprWithArgs("topk_avg",
			[]promql.ValueType{
				promql.ValueTypeString,
				promql.ValueTypeMatrix,
			}, promql.ValueTypeVector,
			promql.Expressions{
				aggrOp.Args[0],
				restExpr})
	case CALL_BOTTOM:
		expr = newAggrExprWithArgs("bottomk_avg",
			[]promql.ValueType{
				promql.ValueTypeString,
				promql.ValueTypeMatrix,
			}, promql.ValueTypeVector,
			promql.Expressions{
				aggrOp.Args[0],
				restExpr})
	case CALL_PERCENTILE:
		expr = newAggrExprWithArgs("quantile_over_time",
			[]promql.ValueType{
				promql.ValueTypeString,
				promql.ValueTypeMatrix,
			}, promql.ValueTypeVector,
			promql.Expressions{
				aggrOp.Args[0],
				restExpr})
	case "non_negative_derivative":
		// InfluxQL: non_negative_derivative(mean("field"), 1s) computes per-second non-negative rate of change
		// MetricsQL: rate() is the equivalent for counter-like metrics
		// Unwrap inner aggregation (e.g., avg_over_time) and apply rate() to the base metric
		if callExpr, ok := restExpr.(*promql.Call); ok && len(callExpr.Args) > 0 {
			expr = newAggrExpr("rate", promql.ValueTypeMatrix, promql.ValueTypeVector, callExpr.Args[0])
		} else {
			expr = newAggrExpr("rate", promql.ValueTypeMatrix, promql.ValueTypeVector, restExpr)
		}
	case "derivative":
		// InfluxQL: derivative(mean("field"), 1s) computes per-second rate of change (can be negative)
		// MetricsQL: deriv() is the closest equivalent
		if callExpr, ok := restExpr.(*promql.Call); ok && len(callExpr.Args) > 0 {
			expr = newAggrExpr("deriv", promql.ValueTypeMatrix, promql.ValueTypeVector, callExpr.Args[0])
		} else {
			expr = newAggrExpr("deriv", promql.ValueTypeMatrix, promql.ValueTypeVector, restExpr)
		}
	case "difference":
		// InfluxQL: difference(mean("field")) computes difference between consecutive points
		// MetricsQL: delta() is the closest equivalent
		if callExpr, ok := restExpr.(*promql.Call); ok && len(callExpr.Args) > 0 {
			expr = newAggrExpr("delta", promql.ValueTypeMatrix, promql.ValueTypeVector, callExpr.Args[0])
		} else {
			expr = newAggrExpr("delta", promql.ValueTypeMatrix, promql.ValueTypeVector, restExpr)
		}
	case "non_negative_difference":
		// InfluxQL: non_negative_difference() - like difference but only non-negative values
		// MetricsQL: increase() is the closest equivalent
		if callExpr, ok := restExpr.(*promql.Call); ok && len(callExpr.Args) > 0 {
			expr = newAggrExpr("increase", promql.ValueTypeMatrix, promql.ValueTypeVector, callExpr.Args[0])
		} else {
			expr = newAggrExpr("increase", promql.ValueTypeMatrix, promql.ValueTypeVector, restExpr)
		}
	case "elapsed":
		// elapsed is not directly supported, pass through
	case "moving_average":
		// moving_average is not directly supported, pass through as avg_over_time
		expr = newAggrExpr("avg_over_time", promql.ValueTypeMatrix, promql.ValueTypeVector, restExpr)
	}
	return expr
}

type AggrOperator struct {
	Name string
	Args promql.Expressions
}

func newAggrOperatorByName(name string) *AggrOperator {
	return &AggrOperator{
		Name: name,
	}
}

func getAggrOperator(op *influxql.Call) ([]*AggrOperator, error) {
	if len(op.Args) != 1 && !MUL_ARGS_AGGREGATOR.Has(op.Name) && !hasDurationLiteralExtraArgs(op) {
		return nil, errors.Errorf("not supported aggregator: %s with args: %#v", op.String(), op.Args)
	}
	aggOp := newAggrOperatorByName(op.Name)
	if op.Name == CALL_TOP || op.Name == CALL_BOTTOM || op.Name == CALL_PERCENTILE {
		numStr := op.Args[len(op.Args)-1].String()
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return nil, errors.Wrapf(err, "parse top/bottom aggregator: %s", op)
		}
		numFloat := float64(num)
		if op.Name == CALL_PERCENTILE {
			if numFloat > 100 {
				return nil, errors.Errorf("percentile %f is large than 100", numFloat)
			}
			numFloat = numFloat / 100
		}
		aggOp.Args = promql.Expressions{
			&promql.NumberLiteral{Val: numFloat},
		}
	}
	ret := []*AggrOperator{aggOp}
	args, ok := op.Args[0].(*influxql.Call)
	if !ok {
		return ret, nil
	}
	rest, err := getAggrOperator(args)
	if err != nil {
		return nil, errors.Wrapf(err, "get rest aggregate operator: %s", args.String())
	}
	ret = append(ret, rest...)
	return ret, nil
}

func getAggrOperators(field *influxql.Field) ([]*AggrOperator, error) {
	switch expr := field.Expr.(type) {
	case *influxql.Call:
		return getAggrOperator(expr)
	case *influxql.BinaryExpr:
		return getBinaryExprAggrOperators(expr)
	}
	return nil, nil
}

func getBinaryExprAggrOperators(expr *influxql.BinaryExpr) ([]*AggrOperator, error) {
	if call, ok := expr.LHS.(*influxql.Call); ok {
		return getAggrOperator(call)
	}
	if call, ok := expr.RHS.(*influxql.Call); ok {
		return getAggrOperator(call)
	}
	if binExpr, ok := expr.LHS.(*influxql.BinaryExpr); ok {
		return getBinaryExprAggrOperators(binExpr)
	}
	if binExpr, ok := expr.RHS.(*influxql.BinaryExpr); ok {
		return getBinaryExprAggrOperators(binExpr)
	}
	return nil, nil
}

func getMetricName(sources influxql.Sources, field *influxql.Field) (string, error) {
	if len(sources) != 1 {
		return "", errors.Errorf("sources %#v length doesn't equal 1", sources)
	}
	src := sources[0]
	measurement, ok := src.(*influxql.Measurement)
	if !ok {
		return "", errors.Errorf("source %#v is not measurement type", src)
	}

	var (
		fieldName string
		err       error
	)

	switch expr := field.Expr.(type) {
	case *influxql.VarRef:
		fieldName = expr.Val
	case *influxql.Call:
		fieldName, err = getCallVariable(expr)
	case *influxql.BinaryExpr:
		fieldName, err = getBinaryExprVariable(expr)
	default:
		return "", errors.Errorf("field.Expr %#v is not supported", expr)
	}

	if err != nil {
		return measurement.Name, err
	}

	return fmt.Sprintf("%s_%s", measurement.Name, fieldName), nil
}

var (
	ErrVariableIsWildcard = errors.New("variable field is wildcard")
)

// isRegexMetricName checks if a metric name contains a regex field pattern.
// In InfluxQL, regex fields are delimited by forward slashes, e.g., measurement_/pattern/
func isRegexMetricName(metricName string) bool {
	idx := strings.Index(metricName, "/")
	return idx >= 0 && strings.HasSuffix(metricName, "/")
}

// trimRegexDelimiters converts a metric name like "haproxy_/d(req|con)/" to "haproxy_d(req|con)"
func trimRegexDelimiters(metricName string) string {
	idx := strings.Index(metricName, "/")
	if idx < 0 {
		return metricName
	}
	prefix := metricName[:idx]
	regexPart := metricName[idx:]
	regexPart = strings.TrimPrefix(regexPart, "/")
	regexPart = strings.TrimSuffix(regexPart, "/")
	return prefix + regexPart
}

// hasDurationLiteralExtraArgs checks if a call has extra args that are all DurationLiterals or IntegerLiterals.
// This handles functions like non_negative_derivative(mean("field"), 1s) where 1s is a duration parameter.
func hasDurationLiteralExtraArgs(c *influxql.Call) bool {
	if len(c.Args) <= 1 {
		return false
	}
	for _, arg := range c.Args[1:] {
		switch arg.(type) {
		case *influxql.DurationLiteral, *influxql.IntegerLiteral:
			continue
		default:
			return false
		}
	}
	return true
}

func getCallVariable(c *influxql.Call) (string, error) {
	if len(c.Args) != 1 && !MUL_ARGS_AGGREGATOR.Has(c.Name) && !hasDurationLiteralExtraArgs(c) {
		return "", errors.Errorf("length of call %q args %#v != 1", c.Name, c.Args)
	}
	switch args := c.Args[0].(type) {
	case *influxql.VarRef:
		return args.Val, nil
	case *influxql.Wildcard:
		return "", ErrVariableIsWildcard
	case *influxql.Call:
		return getCallVariable(args)
	default:
		return "", errors.Errorf("unsupported args %#v", args)
	}
	return c.Args[0].String(), nil
}

func getBinaryExprVariable(expr *influxql.BinaryExpr) (string, error) {
	switch lhs := expr.LHS.(type) {
	case *influxql.Call:
		return getCallVariable(lhs)
	case *influxql.VarRef:
		return lhs.Val, nil
	case *influxql.BinaryExpr:
		return getBinaryExprVariable(lhs)
	}
	switch rhs := expr.RHS.(type) {
	case *influxql.Call:
		return getCallVariable(rhs)
	case *influxql.VarRef:
		return rhs.Val, nil
	case *influxql.BinaryExpr:
		return getBinaryExprVariable(rhs)
	}
	return "", errors.Errorf("BinaryExpr %#v doesn't contain a Call or VarRef", expr)
}

func influxqlOpToPromqlOp(op influxql.Token) (promql.ItemType, error) {
	switch op {
	case influxql.ADD:
		return promql.ItemADD, nil
	case influxql.SUB:
		return promql.ItemSUB, nil
	case influxql.MUL:
		return promql.ItemMUL, nil
	case influxql.DIV:
		return promql.ItemDIV, nil
	case influxql.MOD:
		return promql.ItemMOD, nil
	default:
		return 0, errors.Errorf("unsupported influxql binary operator: %s", op)
	}
}

func influxqlLiteralToPromqlExpr(expr influxql.Expr) (promql.Expr, error) {
	switch v := expr.(type) {
	case *influxql.IntegerLiteral:
		return &promql.NumberLiteral{Val: float64(v.Val)}, nil
	case *influxql.NumberLiteral:
		return &promql.NumberLiteral{Val: v.Val}, nil
	default:
		return nil, errors.Errorf("unsupported literal type %T in binary expression", expr)
	}
}

func wrapBinaryExpr(binExpr *influxql.BinaryExpr, innerExpr promql.Expr) (promql.Expr, error) {
	promOp, err := influxqlOpToPromqlOp(binExpr.Op)
	if err != nil {
		return nil, err
	}

	// Determine which side is the Call (already translated as innerExpr)
	// and which side is the literal
	var literalExpr promql.Expr
	var lhs, rhs promql.Expr

	switch binExpr.LHS.(type) {
	case *influxql.Call, *influxql.BinaryExpr:
		// LHS is the call side, RHS should be the literal
		literalExpr, err = influxqlLiteralToPromqlExpr(binExpr.RHS)
		if err != nil {
			return nil, err
		}
		lhs = innerExpr
		rhs = literalExpr
	default:
		// RHS is the call side, LHS should be the literal
		literalExpr, err = influxqlLiteralToPromqlExpr(binExpr.LHS)
		if err != nil {
			return nil, err
		}
		lhs = literalExpr
		rhs = innerExpr
	}

	return &promql.BinaryExpr{
		Op:  promOp,
		LHS: lhs,
		RHS: rhs,
	}, nil
}

type labelsVisitor struct {
	err           error
	labels        []*labels.Matcher
	curKey        string
	curOp         influxql.Token
	curVal        string
	replaceLabels map[string]string
}

func newLabelsVisitor() *labelsVisitor {
	return &labelsVisitor{
		err:           nil,
		labels:        make([]*labels.Matcher, 0),
		replaceLabels: map[string]string{},
	}
}

func (l *labelsVisitor) Error() error {
	return l.err
}

func (l *labelsVisitor) Labels() []*labels.Matcher {
	return l.labels
}

func (l *labelsVisitor) commitLabel() error {
	if l.err != nil {
		return l.err
	}
	var (
		label *labels.Matcher
		err   error
	)
	var promOP labels.MatchType
	switch l.curOp {
	case influxql.EQ:
		promOP = labels.MatchEqual
	case influxql.NEQ:
		promOP = labels.MatchNotEqual
	case influxql.EQREGEX:
		promOP = labels.MatchRegexp
	case influxql.NEQREGEX:
		promOP = labels.MatchNotRegexp
	default:
		return errors.Errorf("Not suport influxdb operator: %s", l.curOp)
	}
	label, err = labels.NewMatcher(promOP, l.curKey, l.curVal)
	if err != nil {
		return errors.Wrapf(err, "not supported operator: %q", l.curOp)
	}

	l.labels = append(l.labels, label)
	return nil
}

func (l *labelsVisitor) Visit(node influxql.Node) influxql.Visitor {
	//fmt.Printf("-- visit: %s, %#v\n", node, node)
	if l.err != nil {
		log.Printf("error happend: %v, visting skipped", l.err)
		return l
	}
	switch expr := node.(type) {
	case *influxql.BinaryExpr:
		if expr.Op != influxql.OR && expr.Op != influxql.AND {
			l.curOp = expr.Op
		}
		l.Visit(expr.LHS)
		if expr.Op == influxql.OR {
			curLabel := l.labels[len(l.labels)-1]
			key := fmt.Sprintf("%s,", curLabel.String())
			l.replaceLabels[key] = fmt.Sprintf("%s or ", curLabel.String())
		}
		l.Visit(expr.RHS)
		return nil
	case *influxql.VarRef:
		l.curKey = expr.Val
	case *influxql.StringLiteral:
		l.curVal = expr.Val
		if err := l.commitLabel(); err != nil {
			l.err = err
		}
	case *influxql.RegexLiteral:
		l.curVal = expr.Val.String()
		if err := l.commitLabel(); err != nil {
			l.err = err
		}
	}
	return l
}

func (m promQL) getLabels(v *labelsVisitor, cond influxql.Expr) ([]*labels.Matcher, error) {
	if cond == nil {
		return nil, nil
	}
	influxql.Walk(v, cond)
	return v.Labels(), v.Error()
}

func (m *promQL) getGroups(groups influxql.Dimensions) (string, []string, error) {
	result := []string{}
	var (
		lookbehindWindow string
	)
	for _, group := range groups {
		tmpWin, grp, err := m.getGroup(group)
		if err != nil {
			return "", result, errors.Wrapf(err, "getGroup %q", group)
		}
		if tmpWin != "" {
			lookbehindWindow = tmpWin
		}
		if grp != "" {
			result = append(result, grp)
		}
	}
	return lookbehindWindow, result, nil
}

func (m *promQL) getGroup(group *influxql.Dimension) (string, string, error) {
	//fmt.Printf("---try group: %#v\n", group)
	grp := group.Expr
	lookbehindWindow := ""
	switch expr := grp.(type) {
	case *influxql.Call:
		if expr.Name == "time" {
			lookbehindWindow = expr.Args[0].String()
		}
		return lookbehindWindow, "", nil
	case *influxql.VarRef:
		return "", expr.Val, nil
	case *influxql.Wildcard:
		m.groupByWildcard = true
		return "", "", nil
	}
	return "", "", errors.Errorf("not support %q", group.String())
}
