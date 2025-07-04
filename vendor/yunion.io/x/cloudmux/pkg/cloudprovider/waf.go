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

package cloudprovider

import (
	"fmt"
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
)

type TWafStatementType string
type TWafStatementCondition string
type TWafAction string
type TWafMatchField string
type TWafType string
type TWafOperator string

type TWafTextTransformation string

var (
	WafTypeCloudFront = TWafType("CloudFront")
	WafTypeRegional   = TWafType("Regional")
	WafTypeDefault    = TWafType("Default")
	WafTypeAppGateway = TWafType("AppGateway")

	WafTypeSaaS         = TWafType("SaaS")
	WafTypeLoadbalancer = TWafType("Loadbalancer")

	WafStatementTypeByteMatch        = TWafStatementType("ByteMatch")
	WafStatementTypeGeoMatch         = TWafStatementType("GeoMatch")
	WafStatementTypeIPSet            = TWafStatementType("IPSet")
	WafStatementTypeLabelMatch       = TWafStatementType("LabelMatch")
	WafStatementTypeManagedRuleGroup = TWafStatementType("ManagedRuleGroup")
	WafStatementTypeRate             = TWafStatementType("Rate")
	WafStatementTypeRegexSet         = TWafStatementType("RegexSet")
	WafStatementTypeRuleGroup        = TWafStatementType("RuleGroup")
	WafStatementTypeSize             = TWafStatementType("Size")
	WafStatementTypeSqliMatch        = TWafStatementType("SqliMatch")
	WafStatementTypeXssMatch         = TWafStatementType("XssMatch")

	WafStatementConditionAnd  = TWafStatementCondition("And")
	WafStatementConditionOr   = TWafStatementCondition("Or")
	WafStatementConditionNot  = TWafStatementCondition("Not")
	WafStatementConditionNone = TWafStatementCondition("")

	WafActionAllow      = TWafAction("Allow")
	WafActionBlock      = TWafAction("Block")
	WafActionLog        = TWafAction("Log")
	WafActionCount      = TWafAction("Count")
	WafActionAlert      = TWafAction("Alert")
	WafActionDetection  = TWafAction("Detection")
	WafActionPrevention = TWafAction("Prevention")
	WafActionNone       = TWafAction("")

	WafMatchFieldBody     = TWafMatchField("Body")
	WafMatchFieldJsonBody = TWafMatchField("JsonBody")
	WafMatchFieldQuery    = TWafMatchField("Query")
	WafMatchFieldMethod   = TWafMatchField("Method")
	WafMatchFiledHeader   = TWafMatchField("Header")
	WafMatchFiledUriPath  = TWafMatchField("UriPath")
	WafMatchFiledPostArgs = TWafMatchField("PostArgs")
	WafMatchFiledCookie   = TWafMatchField("Cookie")

	// size
	WafOperatorEQ = TWafOperator("EQ")
	WafOperatorNE = TWafOperator("NE")
	WafOperatorLE = TWafOperator("LE")
	WafOperatorLT = TWafOperator("LT")
	WafOperatorGE = TWafOperator("GE")
	WafOperatorGT = TWafOperator("GT")

	// string
	WafOperatorExactly      = TWafOperator("Exactly")
	WafOperatorStartsWith   = TWafOperator("StartsWith")
	WafOperatorEndsWith     = TWafOperator("EndsWith")
	WafOperatorContains     = TWafOperator("Contains")
	WafOperatorContainsWord = TWafOperator("ContainsWord")
	WafOperatorRegex        = TWafOperator("Regex")

	WafTextTransformationNone              = TWafTextTransformation("")
	WafTextTransformationCompressWithSpace = TWafTextTransformation("CompressWithSpace")
	WafTextTransformationHtmlEntityDecode  = TWafTextTransformation("HtmlEntityDecode")
	WafTextTransformationLowercase         = TWafTextTransformation("Lowercase")
	WafTextTransformationCmdLine           = TWafTextTransformation("CmdLine")
	WafTextTransformationUrlDecode         = TWafTextTransformation("UrlDecode")

	// azure
	WafTextTransformationTrim        = TWafTextTransformation("Trim")
	WafTextTransformationUrlEncode   = TWafTextTransformation("UrlEncode")
	WafTextTransformationRemoveNulls = TWafTextTransformation("RemoveNulls")
)

type TWafMatchFieldValues []string

func (values TWafMatchFieldValues) IsZero() bool {
	return len(values) == 0
}

func (values TWafMatchFieldValues) String() string {
	return jsonutils.Marshal(values).String()
}

type TextTransformations []TWafTextTransformation

func (transformations TextTransformations) IsZero() bool {
	return len(transformations) == 0
}

func (transformations TextTransformations) String() string {
	return jsonutils.Marshal(transformations).String()
}

type SExcludeRule struct {
	Name string
}

type SExcludeRules []SExcludeRule

func (rules SExcludeRules) IsZero() bool {
	return len(rules) == 0
}

func (rules SExcludeRules) String() string {
	return jsonutils.Marshal(rules).String()
}

type SWafRule struct {
	Name               string
	Desc               string
	Action             *DefaultAction
	StatementCondition TWafStatementCondition
	Priority           int
	Statements         []SWafStatement
}

// +onecloud:model-api-gen
type SWafStatement struct {
	// 管理规则组名称
	ManagedRuleGroupName string `width:"64" charset:"utf8" nullable:"false" list:"user"`
	// 不包含的规则列表
	ExcludeRules *SExcludeRules `width:"200" charset:"utf8" nullable:"false" list:"user"`
	// 表达式类别
	// enmu: ByteMatch, GeoMatch, IPSet, LabelMatch, ManagedRuleGroup, Rate, RegexSet, RuleGroup, Size, SqliMatch, XssMatch
	Type TWafStatementType `width:"20" charset:"ascii" nullable:"false" list:"user"`
	// 是否取反操作, 仅对Azure生效
	Negation bool `nullable:"false" list:"user"`
	// 操作类型
	// enum: ["EQ", "NE", "LE", "LT", "GE", "GT"]
	Operator TWafOperator `width:"20" charset:"ascii" nullable:"false" list:"user"`
	// 匹配字段
	// enmu: Body, JsonBody, Query, Method, Header, UriPath, PostArgs, Cookie
	MatchField TWafMatchField `width:"20" charset:"utf8" nullable:"false" list:"user"`
	// 匹配字段的key
	MatchFieldKey string `width:"20" charset:"utf8" nullable:"false" list:"user"`
	// 匹配字段的值列表
	MatchFieldValues *TWafMatchFieldValues `width:"250" charset:"utf8" nullable:"false" list:"user"`
	// 进行转换操作
	// enmu: CompressWithSpace, HtmlEntityDecode, Lowercase, CmdLine, UrlDecode, Trim, UrlEncode, RemoveNulls
	Transformations   *TextTransformations `width:"250" charset:"ascii" nullable:"false" list:"user"`
	ForwardedIPHeader string               `width:"20" charset:"ascii" nullable:"false" list:"user"`
	// 搜索字段, 仅Aws有用
	SearchString string `width:"64" charset:"utf8" nullable:"false" list:"user"`
	IPSetId      string `width:"36" charset:"ascii" nullable:"false" list:"user"`
	// 正则表达式Id, 目前只读
	RegexSetId string `width:"36" charset:"ascii" nullable:"false" list:"user"`
	// 自定义规则组Id, 目前只读
	RuleGroupId string `width:"36" charset:"ascii" nullable:"false" list:"user"`
}

func (statement SWafStatement) GetGlobalId() string {
	id := fmt.Sprintf("%s-%s-%s-%s-%s",
		statement.Type,
		statement.MatchField,
		statement.MatchFieldKey,
		statement.ManagedRuleGroupName,
		statement.SearchString,
	)
	if statement.Type == WafStatementTypeGeoMatch || statement.Type == WafStatementTypeRate || statement.Type == WafStatementTypeLabelMatch {
		id = fmt.Sprintf("%s-%s", id, statement.MatchFieldValues)
	}
	return id
}

func (statement SWafStatement) GetExternalId() string {
	return statement.GetGlobalId()
}

// +onecloud:model-api-gen
type DefaultAction struct {
	// Allow, Block, Log, Count, Alert, Detection, Prevention
	Action TWafAction

	// 仅Action为Allow时生效
	InsertHeaders map[string]string
	// 仅Action为Block时生效
	Response string
	// 仅Action为Block时生效
	ResponseCode *int
	// 仅Action为Block时生效
	ResponseHeaders map[string]string
}

type WafSourceIps []string

// +onecloud:model-api-gen
type WafRegexPatterns []string

func (patterns WafRegexPatterns) IsZero() bool {
	return len(patterns) == 0
}

func (patterns WafRegexPatterns) String() string {
	return jsonutils.Marshal(patterns).String()
}

// +onecloud:model-api-gen
type WafAddresses []string

func (addresses WafAddresses) IsZero() bool {
	return len(addresses) == 0
}

func (addresses WafAddresses) String() string {
	return jsonutils.Marshal(addresses).String()
}

func (action DefaultAction) IsZero() bool {
	return false
}

func (action DefaultAction) String() string {
	return jsonutils.Marshal(action).String()
}

type SCloudResource struct {
	// 资源Id
	Id string
	// 资源名称
	Name string
	// 资源类型
	Type string
	// 资源映射端口
	Port int
	// 是否可以解除关联
	CanDissociate bool
}

type SCloudResources struct {
	Data  []SCloudResource `json:"data,allowempty"`
	Total int
}

type WafCreateOptions struct {
	Name           string
	Desc           string
	CloudResources []SCloudResource
	SourceIps      WafSourceIps
	Type           TWafType
	DefaultAction  *DefaultAction
}

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&DefaultAction{}), func() gotypes.ISerializable {
		return &DefaultAction{}
	})

	gotypes.RegisterSerializable(reflect.TypeOf(&WafAddresses{}), func() gotypes.ISerializable {
		return &WafAddresses{}
	})

	gotypes.RegisterSerializable(reflect.TypeOf(&TextTransformations{}), func() gotypes.ISerializable {
		return &TextTransformations{}
	})

	gotypes.RegisterSerializable(reflect.TypeOf(&TWafMatchFieldValues{}), func() gotypes.ISerializable {
		return &TWafMatchFieldValues{}
	})

	gotypes.RegisterSerializable(reflect.TypeOf(&SExcludeRules{}), func() gotypes.ISerializable {
		return &SExcludeRules{}
	})

	gotypes.RegisterSerializable(reflect.TypeOf(&WafRegexPatterns{}), func() gotypes.ISerializable {
		return &WafRegexPatterns{}
	})

}
