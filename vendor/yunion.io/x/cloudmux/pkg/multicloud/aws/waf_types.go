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

const (
	wafTextTransformationTypeNone               = "NONE"
	wafTextTransformationTypeLowercase          = "LOWERCASE"
	wafTextTransformationTypeCmdLine            = "CMD_LINE"
	wafTextTransformationTypeUrlDecode          = "URL_DECODE"
	wafTextTransformationTypeHtmlEntityDecode   = "HTML_ENTITY_DECODE"
	wafTextTransformationTypeCompressWhiteSpace = "COMPRESS_WHITE_SPACE"
)

type sWafEmpty struct{}

type sWafAllowAction struct{}
type sWafBlockAction struct{}
type sWafCountAction struct{}

type sWafDefaultAction struct {
	Allow *sWafAllowAction
	Block *sWafBlockAction
}

type sWafRuleAction struct {
	Allow *sWafAllowAction
	Block *sWafBlockAction
	Count *sWafCountAction
}

type sWafVisibilityConfig struct {
	CloudWatchMetricsEnabled bool
	MetricName               string
	SampledRequestsEnabled   bool
}

type sWafBody struct{}
type sWafMethod struct{}
type sWafQueryString struct{}
type sWafUriPath struct{}
type sWafAllQueryArguments struct{}
type sWafSingleQueryArgument struct {
	Name *string
}
type sWafSingleHeader struct {
	Name *string
}

type sWafFieldToMatch struct {
	AllQueryArguments   *sWafAllQueryArguments
	Body                *sWafBody
	Method              *sWafMethod
	QueryString         *sWafQueryString
	SingleHeader        *sWafSingleHeader
	SingleQueryArgument *sWafSingleQueryArgument
	UriPath             *sWafUriPath
}

type sWafTextTransformation struct {
	Priority *int64
	Type     *string
}

type sWafExcludedRule struct {
	Name *string
}

type sWafForwardedIPConfig struct {
	HeaderName *string
}

type sWafIPSetForwardedIPConfig struct {
	HeaderName *string
}

type sWafAndStatement struct {
	Statements []*sWafStatement
}

type sWafOrStatement struct {
	Statements []*sWafStatement
}

type sWafNotStatement struct {
	Statement *sWafStatement
}

type sWafRateBasedStatement struct {
	Limit              *int64
	ForwardedIPConfig  *sWafForwardedIPConfig
	AggregateKeyType   *string
	ScopeDownStatement *sWafStatement
}

type sWafIPSetReferenceStatement struct {
	ARN                     *string
	IPSetForwardedIPConfig  *sWafIPSetForwardedIPConfig
}

type sWafXssMatchStatement struct {
	FieldToMatch        *sWafFieldToMatch
	TextTransformations []*sWafTextTransformation
}

type sWafSizeConstraintStatement struct {
	ComparisonOperator  *string
	FieldToMatch        *sWafFieldToMatch
	Size                *int64
	TextTransformations []*sWafTextTransformation
}

type sWafGeoMatchStatement struct {
	CountryCodes      []*string
	ForwardedIPConfig *sWafForwardedIPConfig
}

type sWafRegexPatternSetReferenceStatement struct {
	ARN                 *string
	FieldToMatch        *sWafFieldToMatch
	TextTransformations []*sWafTextTransformation
}

type sWafByteMatchStatement struct {
	FieldToMatch          *sWafFieldToMatch
	PositionalConstraint  *string
	SearchString          []byte
	TextTransformations   []*sWafTextTransformation
}

type sWafRuleGroupReferenceStatement struct {
	ARN           *string
	ExcludedRules []*sWafExcludedRule
}

type sWafSqliMatchStatement struct {
	FieldToMatch        *sWafFieldToMatch
	TextTransformations []*sWafTextTransformation
}

type sWafManagedRuleGroupStatement struct {
	Name          *string
	VendorName    *string
	ExcludedRules []*sWafExcludedRule
}

type sWafLabelMatchStatement struct {
	Key   *string
	Scope *string
}

type sWafStatement struct {
	AndStatement                      *sWafAndStatement
	ByteMatchStatement                *sWafByteMatchStatement
	GeoMatchStatement                 *sWafGeoMatchStatement
	IPSetReferenceStatement           *sWafIPSetReferenceStatement
	LabelMatchStatement               *sWafLabelMatchStatement
	ManagedRuleGroupStatement         *sWafManagedRuleGroupStatement
	NotStatement                      *sWafNotStatement
	OrStatement                       *sWafOrStatement
	RateBasedStatement                *sWafRateBasedStatement
	RegexPatternSetReferenceStatement *sWafRegexPatternSetReferenceStatement
	RuleGroupReferenceStatement       *sWafRuleGroupReferenceStatement
	SizeConstraintStatement           *sWafSizeConstraintStatement
	SqliMatchStatement                *sWafSqliMatchStatement
	XssMatchStatement                 *sWafXssMatchStatement
}

type sWafRuleItem struct {
	Action           *sWafRuleAction
	Name             *string
	Priority         *int64
	Statement        *sWafStatement
	VisibilityConfig *sWafVisibilityConfig
}

type sWafWebACL struct {
	ARN              *string
	Capacity         *int64
	DefaultAction    *sWafDefaultAction
	Description      *string
	Id               *string
	Name             *string
	Rules            []*sWafRuleItem
	VisibilityConfig *sWafVisibilityConfig
}

func awsWafString(s string) *string {
	return &s
}

func awsWafInt64(i int64) *int64 {
	return &i
}
