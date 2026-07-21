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

func (sWafEmpty) MarshalJSON() ([]byte, error) {
	return []byte("{}"), nil
}

type sWafAllowAction struct{ sWafEmpty }
type sWafBlockAction struct{ sWafEmpty }
type sWafCountAction struct{ sWafEmpty }

type sWafDefaultAction struct {
	Allow *sWafAllowAction `json:"Allow"`
	Block *sWafBlockAction `json:"Block"`
}

type sWafRuleAction struct {
	Allow *sWafAllowAction `json:"Allow"`
	Block *sWafBlockAction `json:"Block"`
	Count *sWafCountAction `json:"Count"`
}

type sWafVisibilityConfig struct {
	CloudWatchMetricsEnabled bool   `json:"CloudWatchMetricsEnabled"`
	MetricName               string `json:"MetricName"`
	SampledRequestsEnabled   bool   `json:"SampledRequestsEnabled"`
}

type sWafBody struct{ sWafEmpty }
type sWafMethod struct{ sWafEmpty }
type sWafQueryString struct{ sWafEmpty }
type sWafUriPath struct{ sWafEmpty }
type sWafAllQueryArguments struct{ sWafEmpty }
type sWafSingleQueryArgument struct {
	Name string `json:"Name"`
}
type sWafSingleHeader struct {
	Name string `json:"Name"`
}

type sWafFieldToMatch struct {
	AllQueryArguments   *sWafAllQueryArguments   `json:"AllQueryArguments"`
	Body                *sWafBody                `json:"Body"`
	Method              *sWafMethod              `json:"Method"`
	QueryString         *sWafQueryString         `json:"QueryString"`
	SingleHeader        *sWafSingleHeader        `json:"SingleHeader"`
	SingleQueryArgument *sWafSingleQueryArgument `json:"SingleQueryArgument"`
	UriPath             *sWafUriPath             `json:"UriPath"`
}

type sWafTextTransformation struct {
	Priority int64  `json:"Priority"`
	Type     string `json:"Type"`
}

type sWafExcludedRule struct {
	Name string `json:"Name"`
}

type sWafForwardedIPConfig struct {
	HeaderName string `json:"HeaderName"`
}

type sWafIPSetForwardedIPConfig struct {
	HeaderName string `json:"HeaderName"`
}

type sWafAndStatement struct {
	Statements []sWafStatement `json:"Statements"`
}

type sWafOrStatement struct {
	Statements []sWafStatement `json:"Statements"`
}

type sWafNotStatement struct {
	Statement *sWafStatement `json:"Statement"`
}

type sWafRateBasedStatement struct {
	Limit              int64                  `json:"Limit"`
	ForwardedIPConfig  *sWafForwardedIPConfig `json:"ForwardedIPConfig"`
	AggregateKeyType   string                 `json:"AggregateKeyType"`
	ScopeDownStatement *sWafStatement         `json:"ScopeDownStatement"`
}

type sWafIPSetReferenceStatement struct {
	ARN                    string                      `json:"ARN"`
	IPSetForwardedIPConfig *sWafIPSetForwardedIPConfig `json:"IPSetForwardedIPConfig"`
}

type sWafXssMatchStatement struct {
	FieldToMatch        *sWafFieldToMatch       `json:"FieldToMatch"`
	TextTransformations []sWafTextTransformation `json:"TextTransformations"`
}

type sWafSizeConstraintStatement struct {
	ComparisonOperator  string                   `json:"ComparisonOperator"`
	FieldToMatch        *sWafFieldToMatch        `json:"FieldToMatch"`
	Size                int64                    `json:"Size"`
	TextTransformations []sWafTextTransformation `json:"TextTransformations"`
}

type sWafGeoMatchStatement struct {
	CountryCodes      []string               `json:"CountryCodes"`
	ForwardedIPConfig *sWafForwardedIPConfig `json:"ForwardedIPConfig"`
}

type sWafRegexPatternSetReferenceStatement struct {
	ARN                 string                   `json:"ARN"`
	FieldToMatch        *sWafFieldToMatch        `json:"FieldToMatch"`
	TextTransformations []sWafTextTransformation `json:"TextTransformations"`
}

type sWafByteMatchStatement struct {
	FieldToMatch         *sWafFieldToMatch        `json:"FieldToMatch"`
	PositionalConstraint string                   `json:"PositionalConstraint"`
	SearchString         []byte                   `json:"SearchString"`
	TextTransformations  []sWafTextTransformation `json:"TextTransformations"`
}

type sWafRuleGroupReferenceStatement struct {
	ARN           string             `json:"ARN"`
	ExcludedRules []sWafExcludedRule `json:"ExcludedRules"`
}

type sWafSqliMatchStatement struct {
	FieldToMatch        *sWafFieldToMatch       `json:"FieldToMatch"`
	TextTransformations []sWafTextTransformation `json:"TextTransformations"`
}

type sWafManagedRuleGroupStatement struct {
	Name          string             `json:"Name"`
	VendorName    string             `json:"VendorName"`
	ExcludedRules []sWafExcludedRule `json:"ExcludedRules"`
}

type sWafLabelMatchStatement struct {
	Key   string `json:"Key"`
	Scope string `json:"Scope"`
}

type sWafStatement struct {
	AndStatement                      *sWafAndStatement                      `json:"AndStatement"`
	ByteMatchStatement                *sWafByteMatchStatement                `json:"ByteMatchStatement"`
	GeoMatchStatement                 *sWafGeoMatchStatement                 `json:"GeoMatchStatement"`
	IPSetReferenceStatement           *sWafIPSetReferenceStatement           `json:"IPSetReferenceStatement"`
	LabelMatchStatement               *sWafLabelMatchStatement               `json:"LabelMatchStatement"`
	ManagedRuleGroupStatement         *sWafManagedRuleGroupStatement         `json:"ManagedRuleGroupStatement"`
	NotStatement                      *sWafNotStatement                      `json:"NotStatement"`
	OrStatement                       *sWafOrStatement                       `json:"OrStatement"`
	RateBasedStatement                *sWafRateBasedStatement                `json:"RateBasedStatement"`
	RegexPatternSetReferenceStatement *sWafRegexPatternSetReferenceStatement `json:"RegexPatternSetReferenceStatement"`
	RuleGroupReferenceStatement       *sWafRuleGroupReferenceStatement       `json:"RuleGroupReferenceStatement"`
	SizeConstraintStatement           *sWafSizeConstraintStatement           `json:"SizeConstraintStatement"`
	SqliMatchStatement                *sWafSqliMatchStatement                `json:"SqliMatchStatement"`
	XssMatchStatement                 *sWafXssMatchStatement                 `json:"XssMatchStatement"`
}

// SWafRuleItem must be exported: anonymous embed field name must be exportable for jsonutils.
type SWafRuleItem struct {
	Action           *sWafRuleAction       `json:"Action"`
	Name             string                `json:"Name"`
	Priority         int64                 `json:"Priority"`
	Statement        *sWafStatement        `json:"Statement"`
	VisibilityConfig *sWafVisibilityConfig `json:"VisibilityConfig"`
}

// SWafWebACL must be exported: anonymous embed field name must be exportable for jsonutils.
type SWafWebACL struct {
	ARN              string                `json:"ARN"`
	Capacity         int64                 `json:"Capacity"`
	DefaultAction    *sWafDefaultAction    `json:"DefaultAction"`
	Description      string                `json:"Description"`
	Id               string                `json:"Id"`
	Name             string                `json:"Name"`
	Rules            []SWafRuleItem        `json:"Rules"`
	VisibilityConfig *sWafVisibilityConfig `json:"VisibilityConfig"`
}
