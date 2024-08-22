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

package validators

import (
	"strings"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/httperrors"
	merrors "yunion.io/x/onecloud/pkg/monitor/errors"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
)

const (
	ErrMissingParameterThreshold = errors.Error("Condition is missing the threshold parameter")
	ErrMissingParameterType      = errors.Error("Condition is missing the type parameter")
	ErrInvalidEvaluatorType      = errors.Error("Invalid condition evaluator type")
	ErrAlertConditionUnknown     = errors.Error("Unknown alert condition")
	ErrAlertConditionEmpty       = errors.Error("Alert is missing conditions")
)

var (
	EvaluatorDefaultTypes = []string{"gt", "lt", "eq"}
	EvaluatorRangedTypes  = []string{"within_range", "outside_range"}

	CommonAlertType = []string{
		monitor.CommonAlertNomalAlertType,
		monitor.CommonAlertSystemAlertType,
		monitor.CommonAlertServiceAlertType,
	}
	CommonAlertReducerFieldOpts = []string{"/"}
	CommonAlertNotifyTypes      = []string{"email", "mobile", "dingtalk", "webconsole", "feishu"}

	ConditionTypes = []string{"query", "nodata_query"}
)

func ValidateAlertCreateInput(input monitor.AlertCreateInput) error {
	if len(input.Settings.Conditions) == 0 {
		return httperrors.NewInputParameterError("input condition is empty")
	}
	for _, condition := range input.Settings.Conditions {
		if err := ValidateAlertCondition(condition); err != nil {
			return err
		}
	}
	return nil
}

func ValidateAlertCondition(input monitor.AlertCondition) error {
	condType := input.Type
	if err := ValidateAlertConditionType(condType); err != nil {
		return err
	}
	if err := ValidateAlertConditionQuery(input.Query); err != nil {
		return err
	}
	if err := ValidateAlertConditionReducer(input.Reducer); err != nil {
		return err
	}
	if err := ValidateAlertConditionEvaluator(input.Evaluator); err != nil {
		return err
	}
	if input.Operator == "" {
		input.Operator = "and"
	}
	if !utils.IsInStringArray(input.Operator, []string{"and", "or"}) {
		return httperrors.NewInputParameterError("Unkown operator %s", input.Operator)
	}
	return nil
}

func ValidateAlertConditionQuery(input monitor.AlertQuery) error {
	if err := ValidateFromValue(input.From); err != nil {
		return err
	}
	if err := ValidateToValue(input.To); err != nil {
		return err
	}
	if err := ValidateAlertQueryModel(input.Model); err != nil {
		return err
	}
	return nil
}

func ValidateAlertQueryModel(input monitor.MetricQuery) error {
	if len(input.Selects) == 0 {
		return merrors.NewArgIsEmptyErr("select")
	}
	if len(input.Database) == 0 {
		return merrors.NewArgIsEmptyErr("database")
	}
	if len(input.Measurement) == 0 {
		return merrors.NewArgIsEmptyErr("measurement")
	}
	return nil
}

func ValidateSelectOfMetricQuery(input monitor.AlertQuery) error {
	if err := ValidateFromAndToValue(input); err != nil {
		return errors.Wrap(err, "ValidateFromAndToValue")
	}

	if err := ValidateAlertQueryModel(input.Model); err != nil {
		return errors.Wrap(err, "ValidateAlertQueryModel")
	}

	for _, sel := range input.Model.Selects {
		if len(sel) == 0 {
			return httperrors.NewInputParameterError("select for nothing in query")
		}
	}
	if input.ResultReducer != nil {
		if !monitor.ValidateReducerTypes.Has(input.ResultReducer.Type) {
			return httperrors.NewInputParameterError("invalid result reducer type %s", input.ResultReducer.Type)
		}
	}
	return nil
}

func ValidateFromAndToValue(input monitor.AlertQuery) error {
	if err := ValidateFromValue(input.From); err != nil {
		return errors.Wrap(err, "ValidateFromValue")
	}
	if err := ValidateToValue(input.To); err != nil {
		return errors.Wrap(err, "ValidateToValue")
	}
	return nil
}

func ValidateAlertConditionReducer(input monitor.Condition) error {
	return nil
}

func ValidateAlertConditionEvaluator(input monitor.Condition) error {
	typ := input.Type
	if typ == "" {
		return ErrMissingParameterType
	}
	if utils.IsInStringArray(typ, EvaluatorDefaultTypes) {
		return ValidateAlertConditionThresholdEvaluator(input)
	}
	if utils.IsInStringArray(typ, EvaluatorRangedTypes) {
		return ValidateAlertConditionRangedEvaluator(input)
	}
	if typ != "no_value" {
		return errors.Wrapf(ErrInvalidEvaluatorType, "type: %s", typ)
	}
	return nil
}

func ValidateAlertConditionType(typ string) error {
	if typ == "" {
		return httperrors.NewInputParameterError("alert condition type is empty")
	}
	if !utils.IsInStringArray(typ, ConditionTypes) {
		return httperrors.NewInputParameterError("Unkown alert condition type: %s", typ)
	}
	return nil
}

func ValidateAlertConditionThresholdEvaluator(input monitor.Condition) error {
	if len(input.Params) == 0 {
		return errors.Wrapf(ErrMissingParameterThreshold, "Evaluator %s", HumanThresholdType(input.Type))
	}
	return nil
}

func ValidateAlertConditionRangedEvaluator(input monitor.Condition) error {
	if len(input.Params) == 0 {
		return errors.Wrapf(ErrMissingParameterThreshold, "Evaluator %s", HumanThresholdType(input.Type))
	}
	if len(input.Params) == 1 {
		return errors.Wrap(ErrMissingParameterThreshold, "RangedEvaluator parameter second parameter is missing")
	}
	return nil
}

// HumanThresholdType converts a threshold "type" string to a string that matches the UI
// so errors are less confusing.
func HumanThresholdType(typ string) string {
	switch typ {
	case "gt":
		return "IS ABOVE"
	case "lt":
		return "IS BELOW"
	case "within_range":
		return "IS WITHIN RANGE"
	case "outside_range":
		return "IS OUTSIDE RANGE"
	}
	return ""
}

func ValidateFromValue(from string) error {
	_, ok := tsdb.TryParseUnixMsEpoch(from)
	if ok {
		return nil
	}

	fromRaw := strings.Replace(from, "now-", "", 1)
	fromRawStr := "-" + fromRaw
	_, err := time.ParseDuration(fromRawStr)
	if err != nil {
		return errors.Wrapf(err, "parse duration: %s", fromRawStr)
	}
	return nil
}

func ValidateToValue(to string) error {
	if to == "now" {
		return nil
	} else if strings.HasPrefix(to, "now-") {
		withoutNow := strings.Replace(to, "now-", "", 1)
		_, err := time.ParseDuration("-" + withoutNow)
		if err == nil {
			return nil
		}
	}

	_, ok := tsdb.TryParseUnixMsEpoch(to)
	if ok {
		return nil
	}
	_, err := time.ParseDuration(to)
	return errors.Wrapf(err, "parse to: %s", to)
}
