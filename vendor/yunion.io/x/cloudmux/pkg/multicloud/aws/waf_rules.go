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

package aws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/wafv2"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type sWafRule struct {
	waf *SWebAcl
	*wafv2.Rule
}

func (self *sWafRule) GetAction() *cloudprovider.DefaultAction {
	ret := &cloudprovider.DefaultAction{}
	if self.Action == nil {
		ret.Action = cloudprovider.WafActionNone
	} else if self.Action.Allow != nil {
		ret.Action = cloudprovider.WafActionAllow
	} else if self.Action.Block != nil {
		ret.Action = cloudprovider.WafActionBlock
	} else if self.Action.Count != nil {
		ret.Action = cloudprovider.WafActionCount
	}
	return ret
}

func (self *sWafRule) GetDesc() string {
	return ""
}

func (self *sWafRule) GetName() string {
	return *self.Rule.Name
}

func (self *sWafRule) GetGlobalId() string {
	return self.GetName()
}

func (self *sWafRule) GetPriority() int {
	return int(*self.Rule.Priority)
}

func (self *sWafRule) Delete() error {
	input := wafv2.UpdateWebACLInput{}
	rules := []*wafv2.Rule{}
	for _, rule := range self.waf.Rules {
		if *rule.Name == *self.Name {
			continue
		}
		rules = append(rules, rule)
	}
	input.SetRules(rules)
	input.SetLockToken(self.waf.LockToken)
	input.SetId(*self.waf.Id)
	input.SetName(*self.waf.Name)
	input.SetScope(self.waf.scope)
	input.SetDescription(*self.waf.Description)
	input.SetDefaultAction(self.waf.DefaultAction)
	input.SetVisibilityConfig(self.waf.WebACL.VisibilityConfig)
	client, err := self.waf.region.getWafClient()
	if err != nil {
		return errors.Wrapf(err, "getWafClient")
	}
	_, err = client.UpdateWebACL(&input)
	return errors.Wrapf(err, "UpdateWebACL")
}

func (self *sWafRule) Update(opts *cloudprovider.SWafRule) error {
	return cloudprovider.ErrNotImplemented
}

func (self *sWafRule) GetStatementCondition() cloudprovider.TWafStatementCondition {
	if self.Rule.Statement == nil {
		return cloudprovider.WafStatementConditionNone
	}
	if self.Rule.Statement.AndStatement != nil {
		return cloudprovider.WafStatementConditionAnd
	} else if self.Rule.Statement.OrStatement != nil {
		return cloudprovider.WafStatementConditionOr
	} else if self.Rule.Statement.NotStatement != nil {
		return cloudprovider.WafStatementConditionNot
	}
	return cloudprovider.WafStatementConditionNone
}

type sWafStatement struct {
	*wafv2.Statement
}

func (self *sWafStatement) convert() cloudprovider.SWafStatement {
	statement := cloudprovider.SWafStatement{
		Transformations: &cloudprovider.TextTransformations{},
	}
	if self.ByteMatchStatement != nil {
		statement.Type = cloudprovider.WafStatementTypeByteMatch
		if self.ByteMatchStatement.PositionalConstraint != nil {
			operator := strings.ReplaceAll(utils.CamelSplit(*self.ByteMatchStatement.PositionalConstraint, "_"), "_", "")
			if operator == "None" {
				operator = ""
			}
			statement.Operator = cloudprovider.TWafOperator(operator)
		}
		fillStatement(&statement, self.ByteMatchStatement.FieldToMatch)
		statement.SearchString = string(self.ByteMatchStatement.SearchString)
		fillTransformations(&statement, self.ByteMatchStatement.TextTransformations)
	} else if self.GeoMatchStatement != nil {
		statement.Type = cloudprovider.WafStatementTypeGeoMatch
		statement.MatchFieldKey = "CountryCodes"
		values := cloudprovider.TWafMatchFieldValues{}
		for i := range self.GeoMatchStatement.CountryCodes {
			values = append(values, *self.GeoMatchStatement.CountryCodes[i])
		}
		statement.MatchFieldValues = &values
		if self.GeoMatchStatement.ForwardedIPConfig != nil {
			statement.ForwardedIPHeader = *self.GeoMatchStatement.ForwardedIPConfig.HeaderName
		}
	} else if self.IPSetReferenceStatement != nil {
		statement.Type = cloudprovider.WafStatementTypeIPSet
		statement.IPSetId = *self.IPSetReferenceStatement.ARN
		if self.IPSetReferenceStatement.IPSetForwardedIPConfig != nil {
			statement.ForwardedIPHeader = *self.IPSetReferenceStatement.IPSetForwardedIPConfig.HeaderName
		}
	} else if self.ManagedRuleGroupStatement != nil {
		statement.Type = cloudprovider.WafStatementTypeManagedRuleGroup
		statement.ManagedRuleGroupName = *self.ManagedRuleGroupStatement.Name
		fillExcludeRules(&statement, self.ManagedRuleGroupStatement.ExcludedRules)
	} else if self.RateBasedStatement != nil {
		statement.Type = cloudprovider.WafStatementTypeRate
		statement.MatchFieldValues = &cloudprovider.TWafMatchFieldValues{fmt.Sprintf("%d", *self.RateBasedStatement.Limit)}
		if self.RateBasedStatement.ForwardedIPConfig != nil {
			statement.ForwardedIPHeader = *self.RateBasedStatement.ForwardedIPConfig.HeaderName
		}
	} else if self.RegexPatternSetReferenceStatement != nil {
		statement.Type = cloudprovider.WafStatementTypeRegexSet
		statement.RegexSetId = *self.RegexPatternSetReferenceStatement.ARN
		fillStatement(&statement, self.RegexPatternSetReferenceStatement.FieldToMatch)
	} else if self.RuleGroupReferenceStatement != nil {
		statement.Type = cloudprovider.WafStatementTypeRuleGroup
		statement.RuleGroupId = *self.RuleGroupReferenceStatement.ARN
		fillExcludeRules(&statement, self.RuleGroupReferenceStatement.ExcludedRules)
	} else if self.SizeConstraintStatement != nil {
		statement.Type = cloudprovider.WafStatementTypeSize
		statement.Operator = cloudprovider.TWafOperator(*self.SizeConstraintStatement.ComparisonOperator)
		statement.MatchFieldValues = &cloudprovider.TWafMatchFieldValues{fmt.Sprintf("%d", self.SizeConstraintStatement.Size)}
		fillStatement(&statement, self.SizeConstraintStatement.FieldToMatch)
		fillTransformations(&statement, self.SizeConstraintStatement.TextTransformations)
	} else if self.SqliMatchStatement != nil {
		statement.Type = cloudprovider.WafStatementTypeSqliMatch
		fillStatement(&statement, self.SqliMatchStatement.FieldToMatch)
		fillTransformations(&statement, self.SqliMatchStatement.TextTransformations)
	} else if self.XssMatchStatement != nil {
		statement.Type = cloudprovider.WafStatementTypeXssMatch
		fillStatement(&statement, self.XssMatchStatement.FieldToMatch)
		fillTransformations(&statement, self.XssMatchStatement.TextTransformations)
	} else if self.LabelMatchStatement != nil {
		statement.Type = cloudprovider.WafStatementTypeLabelMatch
		if self.LabelMatchStatement.Scope != nil {
			statement.MatchFieldKey = *self.LabelMatchStatement.Scope
		}
		if self.LabelMatchStatement.Key != nil {
			statement.MatchFieldValues = &cloudprovider.TWafMatchFieldValues{*self.LabelMatchStatement.Key}
		}
	} else if self.NotStatement != nil {
		s := &sWafStatement{Statement: self.NotStatement.Statement}
		statement = s.convert()
		statement.Negation = true
	}
	return statement
}

func fillStatement(statement *cloudprovider.SWafStatement, field *wafv2.FieldToMatch) {
	if field.AllQueryArguments != nil {
		statement.MatchField = cloudprovider.WafMatchFieldQuery
		statement.MatchFieldKey = "AllArguments"
	} else if field.Body != nil {
		statement.MatchField = cloudprovider.WafMatchFieldBody
	} else if field.Method != nil {
		statement.MatchField = cloudprovider.WafMatchFieldMethod
	} else if field.QueryString != nil {
		statement.MatchField = cloudprovider.WafMatchFieldQuery
	} else if field.SingleHeader != nil {
		statement.MatchField = cloudprovider.WafMatchFiledHeader
		statement.MatchFieldKey = *field.SingleHeader.Name
	} else if field.SingleQueryArgument != nil {
		statement.MatchField = cloudprovider.WafMatchFieldQuery
		statement.MatchFieldKey = "SingleArgument"
	} else if field.UriPath != nil {
		statement.MatchField = cloudprovider.WafMatchFiledUriPath
	}
}

func fillTransformations(statement *cloudprovider.SWafStatement, trans []*wafv2.TextTransformation) {
	values := cloudprovider.TextTransformations{}
	for _, tran := range trans {
		switch *tran.Type {
		case wafv2.TextTransformationTypeNone:
			values = append(values, cloudprovider.WafTextTransformationNone)
		case wafv2.TextTransformationTypeLowercase:
			values = append(values, cloudprovider.WafTextTransformationLowercase)
		case wafv2.TextTransformationTypeCmdLine:
			values = append(values, cloudprovider.WafTextTransformationCmdLine)
		case wafv2.TextTransformationTypeUrlDecode:
			values = append(values, cloudprovider.WafTextTransformationUrlDecode)
		case wafv2.TextTransformationTypeHtmlEntityDecode:
			values = append(values, cloudprovider.WafTextTransformationHtmlEntityDecode)
		case wafv2.TextTransformationTypeCompressWhiteSpace:
			values = append(values, cloudprovider.WafTextTransformationCompressWithSpace)
		default:
			values = append(values, cloudprovider.TWafTextTransformation(*tran.Type))
		}
	}
	statement.Transformations = &values
}

func fillExcludeRules(statement *cloudprovider.SWafStatement, rules []*wafv2.ExcludedRule) {
	values := cloudprovider.SExcludeRules{}
	for _, rule := range rules {
		values = append(values, cloudprovider.SExcludeRule{Name: *rule.Name})
	}
	statement.ExcludeRules = &values
}

func (self *sWafRule) GetStatements() ([]cloudprovider.SWafStatement, error) {
	if self.Rule.Statement == nil {
		return []cloudprovider.SWafStatement{}, nil
	}
	ret := []cloudprovider.SWafStatement{}
	if self.Rule.Statement.AndStatement != nil {
		for i := range self.Rule.Statement.AndStatement.Statements {
			statement := sWafStatement{self.Rule.Statement.AndStatement.Statements[i]}
			ret = append(ret, statement.convert())
		}
	} else if self.Rule.Statement.OrStatement != nil {
		for i := range self.Rule.Statement.OrStatement.Statements {
			statement := sWafStatement{self.Rule.Statement.OrStatement.Statements[i]}
			ret = append(ret, statement.convert())
		}
	} else if self.Rule.Statement.NotStatement != nil {
		statement := sWafStatement{self.Rule.Statement.NotStatement.Statement}
		ret = append(ret, statement.convert())
	} else {
		statement := sWafStatement{self.Rule.Statement}
		ret = append(ret, statement.convert())
	}
	return ret, nil
}

func (self *SWebAcl) GetRules() ([]cloudprovider.ICloudWafRule, error) {
	ret := []cloudprovider.ICloudWafRule{}
	if len(self.Rules) == 0 {
		err := self.Refresh()
		if err != nil {
			return nil, errors.Wrapf(err, "Refresh")
		}
	}
	for i := range self.Rules {
		ret = append(ret, &sWafRule{
			waf:  self,
			Rule: self.Rules[i],
		})
	}
	return ret, nil
}
