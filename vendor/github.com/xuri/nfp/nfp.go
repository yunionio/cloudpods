// Copyright 2022 The nfp Authors. All rights reserved. Use of this source code
// is governed by a BSD-style license that can be found in the LICENSE file.
//
// This package NFP (Number Format Parser) produce syntax trees for number
// format expression. Excel Number format controls options such the number of
// decimal digits, the currency sign, commas to separate thousands, and
// display of negative numbers. The number format of an index applies wherever
// that index is used, including row or column headers of a table, or graph
// axis that uses that index.
//
// Implementation with Go language by Ri Xu: https://xuri.me

package nfp

import "strings"

// Asterisk, At and other's constants are token definitions.
const (
	// Character constants
	Asterisk       = "*"
	At             = "@"
	BackSlash      = "\\"
	BlockDelimiter = ";"
	BracketClose   = "]"
	BracketOpen    = "["
	Colon          = ":"
	Comma          = ","
	Dash           = "-"
	Dollar         = "$"
	Dot            = "."
	Hash           = "#"
	ParenClose     = ")"
	ParenOpen      = "("
	Percent        = "%"
	Plus           = "+"
	Question       = "?"
	QuoteDouble    = "\""
	QuoteSingle    = "'"
	Slash          = "/"
	Underscore     = "_"
	Whitespace     = " "
	Zero           = "0"
	// DatesTimesCodeChars defined dates and times control codes in upper case
	DatesTimesCodeChars = "EYMDHSG"
	// NumCodeChars defined numeric code character
	NumCodeChars = "0123456789"
	// Token section types
	TokenSectionNegative = "Negative"
	TokenSectionPositive = "Positive"
	TokenSectionText     = "Text"
	TokenSectionZero     = "Zero"
	// Token subtypes
	TokenSubTypeCurrencyString = "CurrencyString"
	TokenSubTypeLanguageInfo   = "LanguageInfo"
	TokenTypeColor             = "Color"
	// Token types
	TokenTypeCondition          = "Condition"
	TokenTypeCurrencyLanguage   = "CurrencyLanguage"
	TokenTypeDateTimes          = "DateTimes"
	TokenTypeDecimalPoint       = "DecimalPoint"
	TokenTypeDenominator        = "Denominator"
	TokenTypeDigitalPlaceHolder = "DigitalPlaceHolder"
	TokenTypeElapsedDateTimes   = "ElapsedDateTimes"
	TokenTypeExponential        = "Exponential"
	TokenTypeFraction           = "Fraction"
	TokenTypeGeneral            = "General"
	TokenTypeHashPlaceHolder    = "HashPlaceHolder"
	TokenTypeLiteral            = "Literal"
	TokenTypeOperand            = "Operand"
	TokenTypeOperator           = "Operator"
	TokenTypePercent            = "Percent"
	TokenTypeRepeatsChar        = "RepeatsChar"
	TokenTypeSwitchArgument     = "SwitchArgument"
	TokenTypeTextPlaceHolder    = "TextPlaceHolder"
	TokenTypeThousandsSeparator = "ThousandsSeparator"
	TokenTypeUnknown            = "Unknown"
	TokenTypeZeroPlaceHolder    = "ZeroPlaceHolder"
)

// ColorNames defined colors name used in for a section of the format, use the
// name of one of the following eight colors in square brackets in the
// section. The color code shall be the first item in the section.
var ColorNames = []string{
	"black",
	"blue",
	"cyan",
	"green",
	"magenta",
	"red",
	"white",
	"yellow",
}

// GeneralFormattingSwitchArguments defined switch-arguments apply to fields
// whose field result is a numeric value. If the result type of the field is
// not numeric, then these switches have no effect.
var GeneralFormattingSwitchArguments = []string{
	"AIUEO",
	"ALPHABETIC",
	"alphabetic",
	"Arabic",
	"ARABICABJAD",
	"ARABICALPHA",
	"ArabicDash",
	"BAHTTEXT",
	"CardText",
	"CHINESENUM1",
	"CHINESENUM2",
	"CHINESENUM3",
	"CHOSUNG",
	"CIRCLENUM",
	"DBCHAR",
	"DBNUM1",
	"DBNUM2",
	"DBNUM3",
	"DBNUM4",
	"DollarText",
	"GANADA",
	"GB1",
	"GB2",
	"GB3",
	"GB4",
	"HEBREW1",
	"HEBREW2",
	"Hex",
	"HINDIARABIC",
	"HINDICARDTEXT",
	"HINDILETTER1",
	"HINDILETTER2",
	"IROHA",
	"KANJINUM1",
	"KANJINUM2",
	"KANJINUM3",
	"Ordinal",
	"OrdText",
	"Roman",
	"roman",
	"SBCHAR",
	"THAIARABIC",
	"THAICARDTEXT",
	"THAILETTER",
	"VIETCARDTEXT",
	"ZODIAC1",
	"ZODIAC2",
	"ZODIAC3",
}

// AmPm defined the AM and PM with international considerations.
var AmPm = []string{"AM/PM", "A/P", "上午/下午"}

// ConditionOperators defined the condition operators.
var ConditionOperators = []string{"<", "<=", ">", ">=", "<>", "="}

// Part directly maps the sub part of the token.
type Part struct {
	Token Token
	Value string
}

// Token encapsulate a number format token.
type Token struct {
	TValue string
	TType  string
	Parts  []Part
}

// Section directly maps sections of the number format. Up to four sections of
// format codes can be specified. The format codes, separated by semicolons,
// define the formats for positive numbers, negative numbers, zero values, and
// text, in that order. If only two sections are specified, the first is used
// for positive numbers and zeros, and the second is used for negative
// numbers. If only one section is specified, it is used for all numbers. To
// skip a section, the ending semicolon for that section shall be written.
type Section struct {
	Type  string
	Items []Token
}

// Tokens directly maps the ordered list of tokens.
// Attributes:
//
//    Index        - Current position in the number format expression
//    SectionIndex - Current position in section
//    Sections     - Ordered section of token sequences
//
type Tokens struct {
	Index        int
	SectionIndex int
	Sections     []Section
}

// fTokens provides function to handle an ordered list of tokens.
func fTokens() Tokens {
	return Tokens{
		Index: -1,
	}
}

// fToken provides function to encapsulate a number format token.
func fToken(value, tokenType string, parts []Part) Token {
	return Token{
		TValue: value,
		TType:  tokenType,
		Parts:  parts,
	}
}

// add provides function to add a token to the end of the list.
func (tk *Tokens) add(value, tokenType string, parts []Part) Token {
	token := fToken(value, tokenType, parts)
	tk.addRef(token)
	return token
}

// addRef provides function to add a token to the end of the list.
func (tk *Tokens) addRef(token Token) {
	if len(tk.Sections) <= tk.SectionIndex {
		sectionType := []string{TokenSectionPositive, TokenSectionNegative, TokenSectionZero, TokenSectionText}[tk.SectionIndex]
		for i := len(tk.Sections) - 1; i < tk.SectionIndex; i++ {
			tk.Sections = append(tk.Sections, Section{Type: sectionType})
		}
	}
	tk.Sections[tk.SectionIndex].Items = append(tk.Sections[tk.SectionIndex].Items, token)
}

// reset provides function to reset the index to -1.
func (tk *Tokens) reset() {
	tk.Index = -1
}

// Parser inheritable container.
type Parser struct {
	InBracket     bool
	InString      bool
	InPlaceholder bool
	NumFmt        string
	Offset        int
	Tokens        Tokens
	Token         Token
}

// NumberFormatParser provides function to parse an Excel number format into a
// stream of tokens.
func NumberFormatParser() Parser {
	return Parser{}
}

// EOF provides function to check whether end of tokens stack.
func (ps *Parser) EOF() bool {
	return ps.Offset >= len([]rune(ps.NumFmt))
}

// getTokens return a token stream (list).
func (ps *Parser) getTokens() Tokens {
	ps.NumFmt = strings.TrimSpace(ps.NumFmt)
	// state-dependent character evaluation (order is important)
	for !ps.EOF() {
		if ps.InBracket {
			if ps.Token.TType == TokenTypeCurrencyLanguage {
				if ps.currentChar() != Dash && ps.currentChar() != BracketClose {
					ps.Token.Parts[1].Token.TValue += ps.currentChar()
				}
				if ps.currentChar() == Dash {
					ps.Token.Parts[0].Token.TValue, ps.Token.Parts[1].Token.TValue = ps.Token.Parts[1].Token.TValue, ps.Token.Parts[0].Token.TValue
				}
			}

			if len(ps.Token.TValue) > 1 && inStrSlice(ConditionOperators, ps.Token.TValue[1:], true) != -1 {
				if ps.currentChar() == Dash || strings.ContainsAny(NumCodeChars, ps.currentChar()) {
					ps.Token.TType = TokenTypeCondition
					ps.Token.Parts = []Part{
						{Token: Token{TType: TokenTypeOperator, TValue: ps.Token.TValue[1:]}},
						{Token: Token{TType: TokenTypeOperand}},
					}
					ps.Token.TValue = ""
					ps.Token.TValue += ps.currentChar()
					ps.Offset++
					continue
				}
			}

			if ps.currentChar() == BracketClose {
				ps.InBracket = false
				if ps.Token.TType == TokenTypeCondition && len(ps.Token.Parts) == 2 {
					ps.Token.Parts[1].Token.TValue = ps.Token.TValue
					ps.Tokens.add(ps.Token.Parts[0].Token.TValue+ps.Token.Parts[1].Token.TValue, ps.Token.TType, ps.Token.Parts)
					ps.Token = Token{}
					ps.Offset++
					continue
				}
				ps.Token.TValue += ps.currentChar()
				if l := len(ps.Token.TValue); l > 2 {
					lit := ps.Token.TValue[1 : l-1]

					if idx := inStrSlice(ColorNames, lit, false); idx != -1 {
						ps.Tokens.add(lit, TokenTypeColor, nil)
						ps.Token = Token{}
						ps.Offset++
						continue
					}

					if idx := inStrSlice(GeneralFormattingSwitchArguments, lit, false); idx != -1 {
						ps.Tokens.add(ps.Token.TValue, TokenTypeSwitchArgument, nil)
						ps.Token = Token{}
						ps.Offset++
						continue
					}

					if ps.Token.TType == TokenTypeCurrencyLanguage {
						if ps.Token.Parts[0].Token.TValue == "" {
							ps.Token.Parts = []Part{{Token: Token{TType: ps.Token.Parts[1].Token.TType, TValue: ps.Token.Parts[1].Token.TValue}}}
						}

						ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
						ps.Token = Token{}
						ps.Offset++
						continue
					}
					ps.Token.TType, ps.Token.TValue = TokenTypeUnknown, lit
					isDateTime := true
					for _, ch := range lit {
						if !strings.ContainsAny(DatesTimesCodeChars, strings.ToUpper(string(ch))) {
							isDateTime = false
						}
					}
					if isDateTime {
						ps.Token.TType = TokenTypeElapsedDateTimes
					}
					ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
					ps.Token = Token{}
					ps.Offset++
					continue
				}
			}
		}

		if !ps.InBracket {
			if strings.ContainsAny(NumCodeChars, ps.currentChar()) {
				if ps.Token.TType == TokenTypeZeroPlaceHolder || ps.Token.TType == TokenTypeDenominator {
					ps.Token.TValue += ps.currentChar()
					ps.Offset++
					continue
				}
				if ps.Token.TType == TokenTypeFraction {
					ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
					ps.Token = Token{TType: TokenTypeDenominator, TValue: ps.currentChar()}
					ps.Offset++
					continue
				}
				if ps.Token.TType != "" {
					ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
					ps.Token = Token{}
				}
				ps.Token.TType = TokenTypeZeroPlaceHolder
				if ps.currentChar() != Zero {
					ps.Token.TType = TokenTypeLiteral
				}
				ps.Token.TValue += ps.currentChar()
				ps.Offset++
				continue
			}

			if ps.currentChar() == Hash {
				if ps.Token.TType != TokenTypeHashPlaceHolder && ps.Token.TType != "" {
					if ps.Token.TValue == Dot {
						ps.Token.TType = TokenTypeDecimalPoint
					}
					ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
					ps.Token = Token{}
				}
				ps.Token.TType = TokenTypeHashPlaceHolder
				ps.Token.TValue += ps.currentChar()
				ps.Offset++
				continue
			}

			if ps.currentChar() == Dot {
				if ps.Token.TType == TokenTypeZeroPlaceHolder || ps.Token.TType == TokenTypeHashPlaceHolder {
					ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
					ps.Tokens.add(ps.currentChar(), TokenTypeDecimalPoint, ps.Token.Parts)
					ps.Token = Token{}
					ps.Offset++
					continue
				}
				if !ps.InString {
					if ps.Token.TType != "" && strings.ContainsAny(NumCodeChars, ps.nextChar()) {
						ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
						ps.Token = Token{}
					}
					ps.Token.TType = TokenTypeDecimalPoint
				}
				ps.Token.TValue += ps.currentChar()
				ps.Offset++
				continue
			}
		}

		if strings.ContainsAny(Dollar+Dash+Plus+ParenOpen+ParenClose+Colon+Whitespace, ps.currentChar()) {
			if ps.InBracket {
				if len(ps.Token.Parts) == 0 {
					ps.Token.Parts = []Part{
						{Token: Token{TType: TokenSubTypeCurrencyString}},
						{Token: Token{TType: TokenSubTypeLanguageInfo}},
					}
				}
				ps.Token.TValue += ps.currentChar()
				ps.Token.TType = TokenTypeCurrencyLanguage
				ps.Offset++
				continue
			}

			if ps.Token.TType != TokenTypeLiteral && ps.Token.TType != TokenTypeDateTimes && ps.Token.TType != "" {
				ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
				ps.Token = Token{TType: TokenTypeLiteral, TValue: ps.currentChar()}
				ps.Offset++
				continue
			}

			if ps.Token.TValue != BackSlash && ps.Token.TType == "" && inStrSlice(AmPm, ps.Token.TValue, false) == -1 {
				ps.Token.TType = TokenTypeLiteral
			}

			if ps.Token.TType == TokenTypeLiteral {
				ps.Token.TValue += ps.currentChar()
				ps.Offset++
				continue
			}
		}

		if ps.currentChar() == Underscore {
			ps.Offset += 2
			continue
		}

		if ps.currentChar() == Asterisk {
			ps.Tokens.add(ps.nextChar(), TokenTypeRepeatsChar, ps.Token.Parts)
			ps.Token = Token{}
			ps.Offset += 2
			continue
		}

		if ps.currentChar() == BackSlash {
			if ps.Token.TValue != "" {
				ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
				ps.Token = Token{}
			}
			ps.Tokens.add(ps.nextChar(), TokenTypeLiteral, ps.Token.Parts)
			ps.Token = Token{}
			ps.Offset += 2
			continue
		}

		if ps.currentChar() == Dash {
			if ps.Token.TType != "" {
				ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
			}
			ps.Token.TType = TokenTypeLiteral
			if ps.currentChar() != ps.nextChar() {
				ps.Tokens.add(ps.currentChar(), ps.Token.TType, ps.Token.Parts)
			}
			ps.Token = Token{}
			ps.Offset++
			continue
		}

		if ps.currentChar() == Comma {
			if ps.Token.TType == TokenTypeZeroPlaceHolder || ps.Token.TType == TokenTypeHashPlaceHolder {
				ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
				ps.Tokens.add(ps.currentChar(), TokenTypeThousandsSeparator, ps.Token.Parts)
				ps.Token = Token{}
				ps.Offset++
				continue
			}
			if !ps.InString {
				if ps.Token.TType == TokenTypeLiteral {
					ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
					ps.Token = Token{TType: TokenTypeThousandsSeparator}
				}
				if ps.Token.TType == TokenTypeDateTimes {
					ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
					ps.Token = Token{TType: TokenTypeLiteral}
				}
				if ps.currentChar() != ps.nextChar() {
					if ps.Token.TType == "" {
						ps.Token.TType = TokenTypeLiteral
					}
					ps.Tokens.add(ps.currentChar(), ps.Token.TType, ps.Token.Parts)
				}
				ps.Token = Token{}
				ps.Offset++
				continue
			}
			ps.Token.TType = TokenTypeLiteral
			ps.Token.TValue += ps.currentChar()
			ps.Offset++
			continue
		}

		if ps.currentChar() == Whitespace {
			if inStrSlice(AmPm, ps.Token.TValue, false) != -1 {
				ps.Token.TType = TokenTypeDateTimes
				ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
				ps.Token = Token{}
				ps.Offset++
				continue
			}
			if ps.Token.TType != "" && ps.Token.TType != TokenTypeLiteral {
				ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
			}
			ps.Token.TType = TokenTypeLiteral
			ps.Tokens.add(ps.currentChar(), ps.Token.TType, ps.Token.Parts)
			ps.Token = Token{}
			ps.Offset++
			continue
		}

		if ps.currentChar() == Slash {
			if ps.Token.TType == TokenTypeDateTimes {
				ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
				ps.Tokens.add(ps.currentChar(), TokenTypeLiteral, ps.Token.Parts)
				ps.Token = Token{}
				ps.Offset++
				continue
			}
			if ps.Token.TType == TokenTypeDigitalPlaceHolder {
				ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
				ps.Token = Token{TType: TokenTypeFraction, TValue: ps.currentChar()}
				ps.Offset++
				continue
			}
		}

		if ps.currentChar() == Colon && ps.Token.TType == TokenTypeDateTimes {
			ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
			ps.Tokens.add(ps.currentChar(), TokenTypeLiteral, ps.Token.Parts)
			ps.Token = Token{}
			ps.Offset++
			continue
		}

		if ps.currentChar() == QuoteDouble {
			ps.Offset++
			if ps.InString && len(ps.Token.TValue) > 0 {
				ps.Tokens.add(ps.Token.TValue, TokenTypeLiteral, ps.Token.Parts)
				ps.Token = Token{}
				ps.InString = false
				continue
			}
			if ps.Token.TValue != "" {
				ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
			}
			ps.InString = true
			ps.Token = Token{TType: TokenTypeLiteral}
			continue
		}

		if ps.currentChar() == At {
			if len(ps.Tokens.Sections) <= ps.Tokens.SectionIndex {
				ps.Tokens.Sections = append(ps.Tokens.Sections, Section{Type: TokenSectionText})
			}
			ps.Tokens.Sections[ps.Tokens.SectionIndex].Type = TokenSectionText
			if ps.Token.TType != "" && !ps.InBracket {
				ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
			}
			ps.Token = Token{TType: TokenTypeTextPlaceHolder, TValue: ps.currentChar()}
			ps.Offset++
			continue
		}

		if ps.currentChar() == BracketOpen {
			if ps.Token.TType != "" && !ps.InBracket {
				ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
				ps.Token = Token{}
			}
			ps.InBracket = true
			ps.Token.TValue += ps.currentChar()
			ps.Offset++
			continue
		}

		if ps.currentChar() == Question {
			if ps.Token.TType != "" && ps.Token.TType != TokenTypeDigitalPlaceHolder {
				ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
				ps.Token = Token{}
			}
			ps.Token.TType = TokenTypeDigitalPlaceHolder
			ps.Token.TValue += ps.currentChar()
			ps.Offset++
			continue
		}

		if ps.currentChar() == Percent {
			if ps.Token.TType != "" {
				ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
				ps.Token = Token{}
			}
			ps.Token.TType = TokenTypePercent
			ps.Token.TValue += ps.currentChar()
			ps.Offset++
			continue
		}

		if ps.currentChar() == BlockDelimiter {
			sectionTypes := []string{TokenSectionPositive, TokenSectionNegative, TokenSectionZero, TokenSectionText}
			if ps.Token.TType != "" {
				ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
			}
			if len(ps.Tokens.Sections) <= ps.Tokens.SectionIndex {
				ps.Tokens.Sections = append(ps.Tokens.Sections, Section{Type: sectionTypes[ps.Tokens.SectionIndex]})
			}
			ps.Tokens.SectionIndex++
			if ps.Tokens.SectionIndex > 3 {
				tokens := fTokens()
				tokens.reset()
				return Tokens{}
			}
			ps.Token = Token{}
			ps.Tokens.Sections = append(ps.Tokens.Sections, Section{Type: sectionTypes[ps.Tokens.SectionIndex]})
			ps.Offset++
			continue
		}

		if strings.EqualFold("E+", ps.doubleChar()) {
			if ps.Token.TType != "" {
				ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
				ps.Token = Token{}
			}
			ps.Token.TType = TokenTypeExponential
			ps.Token.TValue += ps.doubleChar()
			ps.Offset += 2
			continue
		}

		if ap, matched := ps.apPattern(); ap != -1 {
			ps.Tokens.add(matched, TokenTypeDateTimes, ps.Token.Parts)
			ps.Token = Token{}
			ps.Offset += len(matched)
			continue
		}

		if general, matched := ps.generalPattern(); general != -1 {
			ps.Tokens.add(matched, TokenTypeGeneral, ps.Token.Parts)
			ps.Token = Token{}
			ps.Offset += len(matched)
			continue
		}

		// token accumulation
		if !ps.InBracket && !ps.InString {
			if strings.ContainsAny(DatesTimesCodeChars, strings.ToUpper(ps.currentChar())) {
				if inStrSlice(AmPm, ps.Token.TValue, false) != -1 {
					ps.Token.TType = TokenTypeDateTimes
					ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
					ps.Token = Token{}
				}
				if ps.Token.TType == TokenTypeLiteral || ps.Token.TType == TokenTypeDateTimes && !strings.ContainsAny(ps.Token.TValue, ps.currentChar()) {
					ps.Tokens.add(ps.Token.TValue, ps.Token.TType, ps.Token.Parts)
					ps.Token = Token{}
				}
				ps.Token.TType = TokenTypeDateTimes
				ps.Token.TValue += ps.currentChar()
				ps.Offset++
				continue
			}
			if strings.ContainsAny(DatesTimesCodeChars, strings.ToUpper(ps.Token.TValue)) {
				ps.Tokens.add(ps.Token.TValue, TokenTypeDateTimes, ps.Token.Parts)
				ps.Token = Token{TType: TokenTypeLiteral, TValue: ps.currentChar()}
				ps.Offset++
				continue
			}
			if strings.ContainsAny(DatesTimesCodeChars, strings.ToUpper(ps.nextChar())) {
				ps.Token.TValue += ps.currentChar()
				ps.Token.TType = TokenTypeLiteral
				ps.Offset++
				continue
			}
			if ps.currentChar() == QuoteSingle {
				ps.Offset++
				continue
			}
		}
		ps.Token.TValue += ps.currentChar()
		ps.Offset++
	}

	// dump remaining accumulation
	if len(ps.Token.TValue) > 0 {
		tokenType := TokenTypeLiteral
		if ps.Token.TType != "" {
			tokenType = ps.Token.TType
		}
		ps.Tokens.add(ps.Token.TValue, tokenType, nil)
	}

	tokens := fTokens()
	tokens.reset()
	return ps.Tokens
}

// Parse provides function to parse number format as a token stream (list).
func (ps *Parser) Parse(numFmt string) []Section {
	ps.NumFmt = numFmt
	ps.Tokens = ps.getTokens()
	return ps.Tokens.Sections
}

// doubleChar provides function to get two characters after the current
// position.
func (ps *Parser) doubleChar() string {
	if len([]rune(ps.NumFmt)) >= ps.Offset+2 {
		return string([]rune(ps.NumFmt)[ps.Offset : ps.Offset+2])
	}
	return ""
}

// currentChar provides function to get the character of the current position.
func (ps *Parser) currentChar() string {
	return string([]rune(ps.NumFmt)[ps.Offset])
}

// nextChar provides function to get the next character of the current
// position.
func (ps *Parser) nextChar() string {
	if len([]rune(ps.NumFmt)) >= ps.Offset+2 {
		return string([]rune(ps.NumFmt)[ps.Offset+1 : ps.Offset+2])
	}
	return ""
}

// apPattern infers whether the subsequent characters match the AM/PM pattern,
// it will be returned matched index and result.
func (ps *Parser) apPattern() (int, string) {
	for i, pattern := range AmPm {
		l := len(pattern)
		if len([]rune(ps.NumFmt)) >= ps.Offset+l {
			matched := string([]rune(ps.NumFmt)[ps.Offset : ps.Offset+l])
			if strings.EqualFold(matched, pattern) {
				return i, matched
			}
		}
	}
	return -1, ""
}

// generalPattern infers whether the subsequent characters match the
// general pattern, it will be returned matched result and result.
func (ps *Parser) generalPattern() (int, string) {
	l := len(TokenTypeGeneral)
	if len([]rune(ps.NumFmt)) >= ps.Offset+l {
		matched := string([]rune(ps.NumFmt)[ps.Offset : ps.Offset+l])
		if strings.EqualFold(matched, TokenTypeGeneral) {
			return 0, matched
		}
	}
	return -1, ""
}

// inStrSlice provides a method to check if an element is present in an array,
// and return the index of its location, otherwise return -1.
func inStrSlice(a []string, x string, caseSensitive bool) int {
	for idx, n := range a {
		if !caseSensitive && strings.EqualFold(x, n) {
			return idx
		}
		if x == n {
			return idx
		}
	}
	return -1
}

// PrettyPrint provides function to pretty the parsed result with the indented
// format.
func (ps *Parser) PrettyPrint() string {
	indent, output := 0, ""
	for _, section := range ps.Tokens.Sections {
		output += "<" + section.Type + ">" + "\n"
		for _, item := range section.Items {
			indent++
			for i := 0; i < indent; i++ {
				output += "\t"
			}
			if len(item.Parts) == 0 {
				output += item.TValue + " <" + item.TType + ">" + "\n"
			} else {
				output += "<" + item.TType + ">" + "\n"
			}
			for _, part := range item.Parts {
				indent++
				for i := 0; i < indent; i++ {
					output += "\t"
				}
				output += part.Token.TValue + " <" + part.Token.TType + ">" + "\n"
				indent--
			}
			indent--
		}
	}
	return output
}
