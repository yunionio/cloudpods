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

package structarg

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/pkg/utils"
)

type BaseOptions struct {
	PidFile string `help:"Pid file path"`
	Config  string `help:"Configuration file"`
	Help    bool   `help:"Show help and exit"`
	Version bool   `help:"Show version and exit"`
}

type Argument interface {
	NeedData() bool
	Token() string
	AliasToken() string
	ShortToken() string
	NegativeToken() string
	MetaVar() string
	IsPositional() bool
	IsRequired() bool
	IsMulti() bool
	IsSubcommand() bool
	HelpString(indent string) string
	String() string
	SetValue(val string) error
	Reset()
	DoAction(nega bool) error
	Validate() error
	SetDefault()
	IsSet() bool
}

type SingleArgument struct {
	token      string
	aliasToken string
	shortToken string
	negaToken  string
	metavar    string
	positional bool
	required   bool
	help       string
	choices    []string
	useDefault bool
	defValue   reflect.Value
	value      reflect.Value
	ovalue     reflect.Value
	isSet      bool
	parser     *ArgumentParser
}

type MultiArgument struct {
	SingleArgument
	minCount int64
	maxCount int64
}

type SubcommandArgumentData struct {
	parser   *ArgumentParser
	callback reflect.Value
}

type SubcommandArgument struct {
	SingleArgument
	subcommands map[string]SubcommandArgumentData
}

type ArgumentParser struct {
	target      interface{}
	prog        string
	description string
	epilog      string
	help        bool
	optArgs     []Argument
	posArgs     []Argument
}

type sHelpArg struct {
}

func (self *sHelpArg) AliasToken() string {
	return ""
}

func (self *sHelpArg) DoAction(nega bool) error {
	return nil
}

func (self *sHelpArg) HelpString(indent string) string {
	return indent + "Print usage and this help message and exit."
}

func (self *sHelpArg) NeedData() bool {
	return false
}

func (self *sHelpArg) Token() string {
	return "help"
}

func (self *sHelpArg) ShortToken() string {
	return ""
}

func (self *sHelpArg) NegativeToken() string {
	return ""
}

func (self *sHelpArg) MetaVar() string {
	return ""
}

func (self *sHelpArg) IsPositional() bool {
	return false
}

func (self *sHelpArg) IsRequired() bool {
	return false
}

func (self *sHelpArg) IsMulti() bool {
	return false
}

func (self *sHelpArg) IsSubcommand() bool {
	return false
}

func (self *sHelpArg) String() string {
	return "[--help]"
}

func (self *sHelpArg) SetValue(val string) error {
	return nil
}

func (self *sHelpArg) Reset() {
	return
}

func (self *sHelpArg) Validate() error {
	return nil
}

func (self *sHelpArg) SetDefault() {

}

func (self *sHelpArg) IsSet() bool {
	return false
}

func newArgumentParser(target interface{}, prog, desc, epilog string) (*ArgumentParser, error) {
	parser := ArgumentParser{prog: prog, description: desc,
		epilog: epilog, target: target}
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("target must be a pointer")
	}
	targetValue = targetValue.Elem()
	e := parser.addStructArgument("", targetValue)
	if e != nil {
		return nil, e
	}
	// always add a help argument --help
	helpArg := &sHelpArg{}
	parser.AddArgument(helpArg)
	return &parser, nil
}

func NewArgumentParser(target interface{}, prog, desc, epilog string) (*ArgumentParser, error) {
	return newArgumentParser(target, prog, desc, epilog)
}

func NewArgumentParserWithHelp(target interface{}, prog, desc, epilog string) (*ArgumentParser, error) {
	return newArgumentParser(target, prog, desc, epilog)
}

const (
	/*
	   help text of the argument
	   the argument is optional.
	*/
	TAG_HELP = "help"
	/*
	   command-line token for the optional argument, e.g. token:"url"
	   the command-line argument will be "--url http://127.0.0.1:3306"
	   the tag is optional.
	   if the tag is missing, the variable name will be used as token.
	   If the variable name is CamelCase, the token will be transformed
	   into kebab-case, e.g. if the variable is "AuthURL", the token will
	   be "--auth-url"
	*/
	TAG_TOKEN = "token"
	/*
	   short form of command-line token, e.g. short-token:"u"
	   the command-line argument will be "-u http://127.0.0.1:3306"
	   the tag is optional
	*/
	TAG_SHORT_TOKEN = "short-token"
	/*
	   Metavar of the argument
	   the tag is optional
	*/
	TAG_METAVAR = "metavar"
	/*
	   The default value of the argument.
	   the tag is optional
	*/
	TAG_DEFAULT = "default"
	/*
	   The possible values of an arguments. All choices are are concatenatd by "|".
	   e.g. `choices:"1|2|3"`
	   the tag is optional
	*/
	TAG_CHOICES = "choices"
	/*
	   A boolean value explicitly declare whether the argument is optional,
	   the tag is optional
	*/
	TAG_POSITIONAL = "positional"
	/*
	   A boolean value explicitly declare whether the argument is required.
	   The tag is optional.  This is for optional arguments.  Positional
	   arguments must be "required"
	*/
	TAG_REQUIRED = "required"
	/*
	   A boolean value explicitly decalre whther the argument is an subcommand
	   A subcommand argument must be the last positional argument.
	   the tag is optional, the default value is false
	*/
	TAG_SUBCOMMAND = "subcommand"
	/*
	   The attribute defines the possible number of argument. Possible values
	   are:
	       * positive integers, e.g. "1", "2"
	       * "*" any number of arguments
	       * "+" at lease one argument
	       * "?" at most one argument
	   the tag is optional, the default value is "1"
	*/
	TAG_NARGS = "nargs"
	/*
		Alias name of argument
	*/
	TAG_ALIAS = "alias"
	/*
		Token for negative value, applicable to boolean values
	*/
	TAG_NEGATIVE_TOKEN = "negative"
	/*
	   Token for ignore
	*/
	TAG_IGNORE = "ignore"
)

func (this *ArgumentParser) addStructArgument(prefix string, tpVal reflect.Value) error {
	sets := reflectutils.FetchAllStructFieldValueSetForWrite(tpVal)
	for i := range sets {
		if sets[i].Value.Kind() == reflect.Struct && sets[i].Value.Type() != gotypes.TimeType {
			tagMap := sets[i].Info.Tags
			if _, ok := tagMap[reflectutils.TAG_DEPRECATED_BY]; ok {
				// deprecated field, ignore
				return nil
			}
			token, ok := tagMap[TAG_TOKEN]
			if !ok {
				token = sets[i].Info.MarshalName()
			}
			token = prefix + token + "-"
			err := this.addStructArgument(token, sets[i].Value)
			if err != nil {
				return errors.Wrap(err, "addStructArgument")
			}
		} else {
			err := this.addArgument(prefix, sets[i].Value, sets[i].Info)
			if err != nil {
				return errors.Wrap(err, "addArgument")
			}
		}
	}
	return nil
}

func (this *ArgumentParser) addArgument(prefix string, fv reflect.Value, info *reflectutils.SStructFieldInfo) error {
	tagMap := info.Tags
	if _, ok := tagMap[reflectutils.TAG_DEPRECATED_BY]; ok {
		// deprecated field, ignore
		return nil
	}
	if val, ok := tagMap[TAG_IGNORE]; ok && val == "true" {
		// ignore field
		return nil
	}
	help := tagMap[TAG_HELP]
	token, ok := tagMap[TAG_TOKEN]
	if !ok {
		token = info.MarshalName()
	}
	token = prefix + token
	shorttoken := tagMap[TAG_SHORT_TOKEN]
	alias := tagMap[TAG_ALIAS]
	negative := tagMap[TAG_NEGATIVE_TOKEN]
	metavar := tagMap[TAG_METAVAR]
	defval := tagMap[TAG_DEFAULT]
	if len(defval) > 0 {
		for _, dv := range strings.Split(defval, "|") {
			if dv[0] == '$' {
				dv = os.Getenv(strings.TrimLeft(dv, "$"))
			}
			defval = dv
			if len(defval) > 0 {
				break
			}
		}
	}
	if len(negative) > 0 && !valueIsBool(fv) {
		return fmt.Errorf("negative token is applicable to boolean option ONLY")
	}
	use_default := true
	if len(defval) == 0 {
		use_default = false
	}
	var choices []string
	if choices_str, ok := tagMap[TAG_CHOICES]; ok {
		choices = strings.Split(choices_str, "|")
	}
	// heuristic guessing "positional"
	var positional bool
	if info.FieldName == strings.ToUpper(info.FieldName) {
		positional = true
	} else {
		positional = false
	}
	if positionalTag := tagMap[TAG_POSITIONAL]; len(positionalTag) > 0 {
		switch positionalTag {
		case "true":
			positional = true
		case "false":
			positional = false
		default:
			return fmt.Errorf("Invalid positional tag %q, neither true nor false", positionalTag)
		}
	}
	required := positional
	if requiredTag := tagMap[TAG_REQUIRED]; len(requiredTag) > 0 {
		switch requiredTag {
		case "true":
			required = true
		case "false":
			required = false
		default:
			return fmt.Errorf("Invalid required tag %q, neither true nor false", requiredTag)
		}
	}
	if positional {
		if !required {
			return fmt.Errorf("positional %s must not have required:false", token)
		}
		if use_default {
			return fmt.Errorf("positional %s must not have default value", token)
		}
	}
	if !positional && use_default && required {
		return fmt.Errorf("non-positional argument with default value should not have required:true set")
	}
	subcommand, err := strconv.ParseBool(tagMap[TAG_SUBCOMMAND])
	if err != nil {
		subcommand = false
	}
	var defval_t reflect.Value
	if use_default {
		defval_t, err = gotypes.ParseValue(defval, fv.Type())
		if err != nil {
			return err
		}
	}
	if subcommand {
		positional = true
	}
	var arg Argument = nil
	ovalue := reflect.New(fv.Type()).Elem()
	ovalue.Set(fv)
	sarg := SingleArgument{
		token:      token,
		shortToken: shorttoken,
		aliasToken: alias,
		negaToken:  negative,
		positional: positional,
		required:   required,
		metavar:    metavar,
		help:       help,
		choices:    choices,
		useDefault: use_default,
		defValue:   defval_t,
		value:      fv,
		ovalue:     ovalue,
		parser:     this,
	}
	// fmt.Println(token, f.Type, f.Type.Kind())
	if subcommand {
		arg = &SubcommandArgument{SingleArgument: sarg,
			subcommands: make(map[string]SubcommandArgumentData)}
	} else if fv.Kind() == reflect.Array || fv.Kind() == reflect.Slice || fv.Kind() == reflect.Map {
		var min, max int64
		var err error
		nargs := tagMap[TAG_NARGS]
		if nargs == "*" {
			min = 0
			max = -1
		} else if nargs == "?" {
			min = 0
			max = 1
		} else if nargs == "+" {
			min = 1
			max = -1
		} else {
			min, err = strconv.ParseInt(nargs, 10, 64)
			if err == nil {
				max = min
			} else if positional {
				min = 1
				max = -1
			} else if !required {
				min = 0
				max = -1
			}
		}
		arg = &MultiArgument{SingleArgument: sarg,
			minCount: min, maxCount: max}
	} else {
		arg = &sarg
	}
	err = this.AddArgument(arg)
	if err != nil {
		return fmt.Errorf("AddArgument %s: %v", arg, err)
	}
	return nil
}

func (this *ArgumentParser) AddArgument(arg Argument) error {
	if arg.IsPositional() {
		if len(this.posArgs) > 0 {
			last_arg := this.posArgs[len(this.posArgs)-1]
			switch {
			case last_arg.IsMulti():
				return fmt.Errorf("Cannot append positional argument after an array positional argument")
			case last_arg.IsSubcommand():
				return fmt.Errorf("Cannot append positional argument after a subcommand argument")
			}
		}
		this.posArgs = append(this.posArgs, arg)
	} else {
		for _, argOld := range this.optArgs {
			if argOld.Token() == arg.Token() {
				if arg.Token() == "help" {
					// silently ignore help arguments
					return nil
				}
				rt := reflect.TypeOf(this.target)
				if rt.Kind() == reflect.Ptr || rt.Kind() == reflect.Interface {
					rt = rt.Elem()
				}
				return fmt.Errorf("%s: Duplicate argument %s", rt.Name(), argOld.Token())
			}
		}
		// Put required at the end and try to be stable
		if arg.IsRequired() {
			this.optArgs = append(this.optArgs, arg)
		} else {
			var i int
			var opt Argument
			for i, opt = range this.optArgs {
				if opt.IsRequired() {
					break
				}
			}
			this.optArgs = append(this.optArgs, nil)
			copy(this.optArgs[i+1:], this.optArgs[i:])
			this.optArgs[i] = arg
		}
	}
	return nil
}

func (this *ArgumentParser) SetDefault() {
	for _, arg := range this.posArgs {
		arg.SetDefault()
	}
	for _, arg := range this.optArgs {
		arg.SetDefault()
	}
}

func (this *ArgumentParser) Options() interface{} {
	return this.target
}

func valueIsBool(rv reflect.Value) bool {
	if rv.Kind() == reflect.Bool {
		return true
	}

	if rv.Kind() == reflect.Ptr && rv.Type().Elem().Kind() == reflect.Bool {
		return true
	}
	return false
}

func valueIsMap(rv reflect.Value) bool {
	if rv.Kind() == reflect.Map {
		return true
	}
	return false
}

func (this *SingleArgument) defaultBoolValue() bool {
	rv := this.defValue
	if rv.Kind() == reflect.Bool {
		return rv.Bool()
	}

	if rv.Kind() == reflect.Ptr && rv.Type().Elem().Kind() == reflect.Bool {
		return rv.Elem().Bool()
	}
	panic("expecting bool or *bool type: got " + rv.Type().String())
}

func (this *SingleArgument) NeedData() bool {
	if valueIsBool(this.value) {
		return false
	} else {
		return true
	}
}

func (this *SingleArgument) MetaVar() string {
	if len(this.metavar) > 0 {
		return this.metavar
	} else if len(this.choices) > 0 {
		return fmt.Sprintf("{%s}", strings.Join(this.choices, ","))
	} else {
		return strings.ToUpper(strings.Replace(this.Token(), "-", "_", -1))
	}
}

func splitCamelString(str string) string {
	return utils.CamelSplit(str, "-")
}

func (this *SingleArgument) AllToken() string {
	ret := this.Token()
	if len(this.AliasToken()) != 0 {
		ret = fmt.Sprintf("%s|--%s", ret, this.AliasToken())
	}
	if len(this.ShortToken()) != 0 {
		ret = fmt.Sprintf("%s|-%s", ret, this.ShortToken())
	}
	if len(this.NegativeToken()) != 0 {
		ret = fmt.Sprintf("%s/--%s", ret, this.NegativeToken())
	}
	return ret
}

func (this *SingleArgument) Token() string {
	return splitCamelString(this.token)
}

func (this *SingleArgument) AliasToken() string {
	return splitCamelString(this.aliasToken)
}

func (this *SingleArgument) ShortToken() string {
	return this.shortToken
}

func (this *SingleArgument) NegativeToken() string {
	return strings.ReplaceAll(this.negaToken, "_", "-")
}

func (this *SingleArgument) String() string {
	var start, end byte
	if this.IsRequired() {
		start = '<'
		end = '>'
	} else {
		start = '['
		end = ']'
	}
	if this.IsPositional() {
		return fmt.Sprintf("%c%s%c", start, this.MetaVar(), end)
	} else {
		if this.NeedData() {
			return fmt.Sprintf("%c--%s %s%c", start, this.AllToken(), this.MetaVar(), end)
		} else {
			return fmt.Sprintf("%c--%s%c", start, this.AllToken(), end)
		}
	}
}

func (this *SingleArgument) IsRequired() bool {
	return this.required
}

func (this *SingleArgument) IsPositional() bool {
	return this.positional
}

func (this *SingleArgument) IsMulti() bool {
	return false
}

func (this *SingleArgument) IsSubcommand() bool {
	return false
}

func (this *SingleArgument) HelpString(indent string) string {
	return indent + strings.Join(strings.Split(this.help, "\n"), "\n"+indent)
}

func (this *SingleArgument) InChoices(val string) bool {
	if len(this.choices) > 0 {
		for _, s := range this.choices {
			if s == val {
				return true
			}
		}
		return false
	} else {
		return true
	}
}

func (this *SingleArgument) Choices() []string {
	return this.choices
}

func (this *SingleArgument) SetValue(val string) error {
	if !this.InChoices(val) {
		return this.choicesErr(val)
	}
	e := gotypes.SetValue(this.value, val)
	if e != nil {
		return e
	}
	this.isSet = true
	return nil
}

func (this *SingleArgument) choicesErr(val string) error {
	cands := FindSimilar(val, this.choices, -1, 0.5)
	if len(cands) > 3 {
		cands = cands[:3]
	}
	msg := fmt.Sprintf("Unknown argument '%s' for %s", val, this.Token())
	if len(cands) > 0 {
		msg += fmt.Sprintf(", did you mean %s?", quotedChoicesString(cands))
	} else if len(this.choices) > 0 {
		msg += fmt.Sprintf(", accepts %s", quotedChoicesString(this.choices))
	}
	return fmt.Errorf("%s", msg)
}

func (this *SingleArgument) Reset() {
	this.value.Set(this.ovalue)
	this.isSet = false
}

func (this *SingleArgument) DoAction(nega bool) error {
	if valueIsBool(this.value) {
		var v bool
		if this.useDefault {
			v = !this.defaultBoolValue()
		} else if nega {
			v = false
		} else {
			v = true
		}
		gotypes.SetValue(this.value, fmt.Sprintf("%t", v))
		this.isSet = true
	}
	return nil
}

func (this *SingleArgument) SetDefault() {
	if !this.isSet && this.useDefault {
		this.value.Set(this.defValue)
	}
}

func (this *SingleArgument) Validate() error {
	if this.required && !this.isSet && !this.useDefault {
		return fmt.Errorf("Non-optional argument %s not set", this.token)
	}
	return nil
}

func (this *SingleArgument) IsSet() bool {
	return this.isSet
}

func (this *MultiArgument) IsMulti() bool {
	return true
}

func (this *MultiArgument) setKeyValue(val string) error {
	pos := strings.IndexByte(val, '=')
	var key, value string
	if pos >= 0 {
		key = val[:pos]
		value = val[pos+1:]
	} else {
		key = val
	}
	keyType := this.value.Type().Key()
	keyValue, err := gotypes.ParseValue(key, keyType)
	if err != nil {
		return errors.Wrapf(err, "ParseValue for key %s", key)
	}
	valType := this.value.Type().Elem()
	valValue, err := gotypes.ParseValue(value, valType)
	if err != nil {
		return errors.Wrapf(err, "ParseValue for value %s", value)
	}
	if this.value.Len() == 0 {
		this.value.Set(reflect.MakeMap(this.value.Type()))
	}
	this.value.SetMapIndex(keyValue, valValue)
	this.isSet = true
	return nil
}

func (this *MultiArgument) SetValue(val string) error {
	if valueIsMap(this.value) {
		return this.setKeyValue(val)
	}
	if !this.InChoices(val) {
		return this.choicesErr(val)
	}
	var e error = nil
	e = gotypes.AppendValue(this.value, val)
	if e != nil {
		return e
	}
	this.isSet = true
	return nil
}

func (this *MultiArgument) Validate() error {
	var e = this.SingleArgument.Validate()
	if e != nil {
		return e
	}
	var vallen int64 = int64(this.value.Len())
	if this.minCount >= 0 && vallen < this.minCount {
		return fmt.Errorf("Argument count requires at least %d", this.minCount)
	}
	if this.maxCount >= 0 && vallen > this.maxCount {
		return fmt.Errorf("Argument count requires at most %d", this.maxCount)
	}
	return nil
}

func (this *SubcommandArgument) IsSubcommand() bool {
	return true
}

func (this *SubcommandArgument) String() string {
	return fmt.Sprintf("<%s>", strings.ToUpper(this.token))
}

func (this *SubcommandArgument) AddSubParser(target interface{}, command string, desc string, callback interface{}) (*ArgumentParser, error) {
	return this.addSubParser(target, command, desc, callback)
}

func (this *SubcommandArgument) AddSubParserWithHelp(target interface{}, command string, desc string, callback interface{}) (*ArgumentParser, error) {
	return this.addSubParser(target, command, desc, callback)
}

func (this *SubcommandArgument) addSubParser(target interface{}, command string, desc string, callback interface{}) (*ArgumentParser, error) {
	prog := fmt.Sprintf("%s %s", this.parser.prog, command)
	parser, e := newArgumentParser(target, prog, desc, "")
	if e != nil {
		return nil, e
	}
	cbfunc := reflect.ValueOf(callback)
	this.subcommands[command] = SubcommandArgumentData{parser: parser,
		callback: cbfunc}
	this.choices = append(this.choices, command)
	return parser, nil
}

func (this *SubcommandArgument) HelpString(indent string) string {
	var buf bytes.Buffer
	for k, data := range this.subcommands {
		buf.WriteString(indent)
		buf.WriteString(k)
		buf.WriteByte('\n')
		buf.WriteString(indent)
		buf.WriteString("  ")
		buf.WriteString(data.parser.ShortDescription())
		buf.WriteByte('\n')
	}
	return buf.String()
}

func (this *SubcommandArgument) SubHelpString(cmd string) (string, error) {
	val, ok := this.subcommands[cmd]
	if ok {
		return val.parser.HelpString(), nil
	} else {
		return "", fmt.Errorf("No such command %s", cmd)
	}
}

func (this *SubcommandArgument) GetSubParser() *ArgumentParser {
	var cmd = this.value.String()
	val, ok := this.subcommands[cmd]
	if ok {
		return val.parser
	} else {
		return nil
	}
}

func (this *SubcommandArgument) Invoke(args ...interface{}) error {
	var inargs = make([]reflect.Value, 0)
	for _, arg := range args {
		inargs = append(inargs, reflect.ValueOf(arg))
	}
	var cmd = this.value.String()
	val, ok := this.subcommands[cmd]
	if !ok {
		return fmt.Errorf("Unknown subcommand %s", cmd)
	}
	out := val.callback.Call(inargs)
	if len(out) == 1 {
		if out[0].IsNil() {
			return nil
		} else {
			return out[0].Interface().(error)
		}
	} else {
		return fmt.Errorf("Callback return %d unknown outputs", len(out))
	}
}

func (this *ArgumentParser) ShortDescription() string {
	return strings.Split(this.description, "\n")[0]
}

func (this *ArgumentParser) Usage() string {
	var buf bytes.Buffer
	buf.WriteString("Usage: ")
	buf.WriteString(this.prog)
	for _, arg := range this.optArgs {
		buf.WriteByte(' ')
		buf.WriteString(arg.String())
	}
	for _, arg := range this.posArgs {
		buf.WriteByte(' ')
		buf.WriteString(arg.String())
		if arg.IsSubcommand() || arg.IsMulti() {
			buf.WriteString(" ...")
		}
	}
	buf.WriteByte('\n')
	buf.WriteByte('\n')
	return buf.String()
}

func (this *ArgumentParser) HelpString() string {
	var buf bytes.Buffer
	buf.WriteString(this.Usage())
	buf.WriteString(this.description)
	buf.WriteByte('\n')
	buf.WriteByte('\n')
	if len(this.posArgs) > 0 {
		buf.WriteString("Positional arguments:\n")
		for _, arg := range this.posArgs {
			buf.WriteString("    ")
			buf.WriteString(arg.String())
			buf.WriteByte('\n')
			buf.WriteString(arg.HelpString("        "))
			buf.WriteByte('\n')
		}
		buf.WriteByte('\n')
	}
	if len(this.optArgs) > 0 {
		buf.WriteString("Optional arguments:\n")
		for _, arg := range this.optArgs {
			buf.WriteString("    ")
			buf.WriteString(arg.String())
			buf.WriteByte('\n')
			buf.WriteString(arg.HelpString("        "))
			buf.WriteByte('\n')
		}
		buf.WriteByte('\n')
	}
	if len(this.epilog) > 0 {
		buf.WriteString(this.epilog)
		buf.WriteByte('\n')
		buf.WriteByte('\n')
	}
	return buf.String()
}

func tokenMatch(argToken, input string, exactMatch bool) bool {
	if exactMatch {
		return argToken == input
	} else {
		return strings.HasPrefix(argToken, input)
	}
}

func (this *ArgumentParser) findOptionalArgument(token string, exactMatch bool) (Argument, bool) {
	var match_arg Argument = nil
	match_len := -1
	negative := false
	for _, arg := range this.optArgs {
		if tokenMatch(arg.Token(), token, exactMatch) {
			if match_len < 0 || match_len > len(arg.Token()) {
				match_len = len(arg.Token())
				match_arg = arg
				negative = false
			}
		} else if tokenMatch(arg.ShortToken(), token, exactMatch) {
			if match_len < 0 || match_len > len(arg.ShortToken()) {
				match_len = len(arg.ShortToken())
				match_arg = arg
				negative = false
			}
		} else if tokenMatch(arg.AliasToken(), token, exactMatch) {
			if match_len < 0 || match_len > len(arg.AliasToken()) {
				match_len = len(arg.AliasToken())
				match_arg = arg
				negative = false
			}
		} else if tokenMatch(arg.NegativeToken(), token, exactMatch) {
			if match_len < 0 || match_len > len(arg.AliasToken()) {
				match_len = len(arg.AliasToken())
				match_arg = arg
				negative = true
			}
		}
	}
	return match_arg, negative
}

func validateArgs(args []Argument) error {
	for _, arg := range args {
		e := arg.Validate()
		if e != nil {
			return fmt.Errorf("%s error: %s", arg.Token(), e)
		}
	}
	return nil
}

func (this *ArgumentParser) Validate() error {
	var e error = nil
	e = validateArgs(this.posArgs)
	if e != nil {
		return e
	}
	e = validateArgs(this.optArgs)
	if e != nil {
		return e
	}
	return nil
}

func (this *ArgumentParser) reset() {
	for _, arg := range this.posArgs {
		arg.Reset()
	}
	for _, arg := range this.optArgs {
		arg.Reset()
	}
	this.help = false
}

func (this *ArgumentParser) ParseArgs(args []string, ignore_unknown bool) error {
	return this.ParseArgs2(args, ignore_unknown, true)
}

func (this *ArgumentParser) ParseArgs2(args []string, ignore_unknown bool, setDefaults bool) error {
	var pos_idx int
	var err error
	var argStr string

	this.reset()

	for i := 0; i < len(args) && err == nil; i++ {
		argStr = args[i]
		if argStr == "--help" {
			// shortcut to show help
			fmt.Println(this.HelpString())
			this.help = true
			continue
		}
		if strings.HasPrefix(argStr, "-") {
			arg, nega := this.findOptionalArgument(strings.TrimLeft(argStr, "-"), false)
			if arg != nil {
				if arg.NeedData() {
					if i+1 < len(args) {
						err = arg.SetValue(args[i+1])
						if err != nil {
							break
						}
						i++
					} else {
						err = fmt.Errorf("Missing arguments for %s", argStr)
						break
					}
				} else {
					err = arg.DoAction(nega)
					if err != nil {
						break
					}
				}
			} else if !ignore_unknown {
				err = fmt.Errorf("Unknown optional argument %s", argStr)
				break
			}
		} else {
			if pos_idx >= len(this.posArgs) {
				if len(this.posArgs) > 0 {
					last_arg := this.posArgs[len(this.posArgs)-1]
					if last_arg.IsMulti() {
						last_arg.SetValue(argStr)
					} else if !ignore_unknown {
						err = fmt.Errorf("Unknown positional argument %s", argStr)
						break
					}
				} else if !ignore_unknown {
					err = fmt.Errorf("Unknown positional argument %s", argStr)
					break
				}
			} else {
				arg := this.posArgs[pos_idx]
				pos_idx += 1
				err = arg.SetValue(argStr)
				if err != nil {
					break
				}
				if arg.IsSubcommand() {
					subarg := arg.(*SubcommandArgument)
					var subparser = subarg.GetSubParser()
					err = subparser.ParseArgs(args[i+1:], ignore_unknown)
					break
				}
			}
		}
	}
	if err == nil && pos_idx < len(this.posArgs) {
		err = &NotEnoughArgumentsError{argument: this.posArgs[pos_idx]}
	}
	if err == nil {
		err = this.Validate()
	}
	if setDefaults {
		this.SetDefault()
	}
	return err
}

func isQuotedByChar(str string, quoteChar byte) bool {
	return len(str) >= 2 && str[0] == quoteChar && str[len(str)-1] == quoteChar
}

func isQuoted(str string) bool {
	return isQuotedByChar(str, '"') || isQuotedByChar(str, '\'')
}

func (this *ArgumentParser) parseKeyValue(key, value string) error {
	arg, nega := this.findOptionalArgument(key, true)
	if arg != nil {
		if nega {
			log.Warningf("Ignore negative token when parse %s=%v", key, value)
			return nil
		}
		if arg.IsSet() {
			return nil
		}
		if arg.IsMulti() {
			if value[0] == '(' {
				value = strings.Trim(value, "()")
			} else {
				value = strings.Trim(value, "[]")
			}
			values := utils.FindWords([]byte(value), 0)
			for _, v := range values {
				e := arg.SetValue(v)
				if e != nil {
					return e
				}
			}
		} else {
			if !isQuoted(value) {
				value = fmt.Sprintf("\"%s\"", value)
			}
			values := utils.FindWords([]byte(value), 0)
			if len(values) == 1 {
				return arg.SetValue(values[0])
			} else {
				log.Warningf("too many arguments %#v for %s", values, key)
			}
		}
	} else {
		log.Warningf("Cannot find argument %s", key)
	}
	return nil
}

func removeComments(line string) string {
	pos := strings.IndexByte(line, '#')
	if pos >= 0 {
		return line[:pos]
	} else {
		return line
	}
}

func line2KeyValue(line string) (string, string, error) {
	// first remove comments
	pos := strings.IndexByte(line, '=')
	if pos > 0 && pos < len(line) {
		key := keyToToken(line[:pos])
		val := strings.Trim(line[pos+1:], " ")
		return key, val, nil
	} else {
		return "", "", fmt.Errorf("Misformated line: %s", line)
	}
}

func removeCharacters(input, charSet string) string {
	filter := func(r rune) rune {
		if strings.IndexRune(charSet, r) < 0 {
			return r
		}
		return -1
	}
	return strings.Map(filter, input)
}

func (this *ArgumentParser) ParseYAMLFile(filepath string) error {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("read file %s: %v", filepath, err)
	}
	obj, err := jsonutils.ParseYAML(string(content))
	if err != nil {
		return fmt.Errorf("parse yaml to json object: %v", err)
	}
	dict, ok := obj.(*jsonutils.JSONDict)
	if !ok {
		return fmt.Errorf("object %s is not JSONDict", obj.String())
	}
	return this.parseJSONDict(dict)
}

func (this *ArgumentParser) parseJSONDict(dict *jsonutils.JSONDict) error {
	mapJson, err := dict.GetMap()
	if err != nil {
		return errors.Wrap(err, "GetMap")
	}
	for key, obj := range mapJson {
		if err := this.parseJSONKeyValue(key, obj); err != nil {
			return fmt.Errorf("parse json %s: %s: %v", key, obj.String(), err)
		}
	}
	return nil
}

func keyToToken(key string) string {
	return strings.Replace(strings.Trim(key, " "), "_", "-", -1)
}

func (this *ArgumentParser) parseJSONKeyValue(key string, obj jsonutils.JSONObject) error {
	token := keyToToken(key)
	arg, nega := this.findOptionalArgument(token, true)
	if arg == nil {
		log.Warningf("Cannot find argument %s", token)
		return nil
	}
	if nega {
		log.Warningf("Ignore negative token when parse JSONKeyValue %s", token)
		return nil
	}
	if arg.IsSet() {
		return nil
	}
	// process multi argument
	if arg.IsMulti() {
		array, err := obj.GetArray()
		if err != nil {
			return errors.Wrap(err, "GetArray")
		}
		for _, item := range array {
			str, err := item.GetString()
			if err != nil {
				return err
			}
			if err := arg.SetValue(str); err != nil {
				return err
			}
		}
		return nil
	}
	// process single argument
	str, err := obj.GetString()
	if err != nil {
		return err
	}
	return arg.SetValue(str)
}

func (this *ArgumentParser) ParseFile(filepath string) error {
	if err := this.ParseYAMLFile(filepath); err == nil {
		return nil
	}
	return this.ParseTornadoFile(filepath)
}

func (this *ArgumentParser) parseReader(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(removeComments(line))
		// line = removeCharacters(line, `"'`)
		if len(line) > 0 {
			if line[0] == '[' {
				continue
			}
			key, val, e := line2KeyValue(line)
			if e == nil {
				this.parseKeyValue(key, val)
			} else {
				return e
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (this *ArgumentParser) ParseTornadoFile(filepath string) error {
	file, e := os.Open(filepath)
	if e != nil {
		return e
	}
	defer file.Close()

	return this.parseReader(file)
}

func (this *ArgumentParser) GetSubcommand() *SubcommandArgument {
	if len(this.posArgs) > 0 {
		last_arg := this.posArgs[len(this.posArgs)-1]
		if last_arg.IsSubcommand() {
			return last_arg.(*SubcommandArgument)
		}
	}
	return nil
}

func (this *ArgumentParser) ParseKnownArgs(args []string) error {
	return this.ParseArgs(args, true)
}

func (this *ArgumentParser) GetOptArgs() []Argument {
	return this.optArgs
}

func (this *ArgumentParser) GetPosArgs() []Argument {
	return this.posArgs
}

func (this *ArgumentParser) IsHelpSet() bool {
	return this.help
}
