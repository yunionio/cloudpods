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
	MetaVar() string
	IsPositional() bool
	IsRequired() bool
	IsMulti() bool
	IsSubcommand() bool
	HelpString(indent string) string
	String() string
	SetValue(val string) error
	Reset()
	DoAction() error
	Validate() error
	SetDefault()
	IsSet() bool
}

type SingleArgument struct {
	token      string
	aliasToken string
	shortToken string
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
	optArgs     []Argument
	posArgs     []Argument
}

func NewArgumentParser(target interface{}, prog, desc, epilog string) (*ArgumentParser, error) {
	parser := ArgumentParser{prog: prog, description: desc,
		epilog: epilog, target: target}
	target_type := reflect.TypeOf(target).Elem()
	target_value := reflect.ValueOf(target).Elem()
	e := parser.addStructArgument("", target_type, target_value)
	if e != nil {
		return nil, e
	}
	return &parser, nil
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
)

func (this *ArgumentParser) addStructArgument(prefix string, tp reflect.Type, val reflect.Value) error {
	for i := 0; i < tp.NumField(); i++ {
		v := val.Field(i)
		if !v.CanSet() {
			continue
		}
		f := tp.Field(i)
		if f.Type.Kind() == reflect.Struct {
			p := prefix
			if !f.Anonymous {
				p += f.Name + "-"
			}
			e := this.addStructArgument(p, f.Type, v)
			if e != nil {
				return e
			}
		} else {
			e := this.addArgument(prefix, f, v)
			if e != nil {
				return e
			}
		}
	}
	return nil
}

func (this *ArgumentParser) addArgument(prefix string, f reflect.StructField, v reflect.Value) error {
	info := reflectutils.ParseStructFieldJsonInfo(f)
	tagMap := info.Tags
	help := tagMap[TAG_HELP]
	token, ok := tagMap[TAG_TOKEN]
	if !ok {
		token = info.MarshalName()
	}
	token = prefix + token
	shorttoken := tagMap[TAG_SHORT_TOKEN]
	alias := tagMap[TAG_ALIAS]
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
	use_default := true
	if len(defval) == 0 {
		use_default = false
	}
	choices_str := tagMap[TAG_CHOICES]
	choices := make([]string, 0)
	if len(choices_str) > 0 {
		for _, s := range strings.Split(choices_str, "|") {
			if len(s) > 0 {
				choices = append(choices, s)
			}
		}
	}
	// heuristic guessing "positional"
	var positional bool
	if f.Name == strings.ToUpper(f.Name) {
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
		defval_t, err = gotypes.ParseValue(defval, f.Type)
		if err != nil {
			return err
		}
	}
	if subcommand {
		positional = true
	}
	var arg Argument = nil
	ovalue := reflect.New(v.Type()).Elem()
	ovalue.Set(v)
	sarg := SingleArgument{
		token:      token,
		shortToken: shorttoken,
		positional: positional,
		required:   required,
		metavar:    metavar,
		help:       help,
		choices:    choices,
		useDefault: use_default,
		aliasToken: alias,
		defValue:   defval_t,
		value:      v,
		ovalue:     ovalue,
		parser:     this,
	}
	// fmt.Println(token, f.Type, f.Type.Kind())
	if subcommand {
		arg = &SubcommandArgument{SingleArgument: sarg,
			subcommands: make(map[string]SubcommandArgumentData)}
	} else if f.Type.Kind() == reflect.Array || f.Type.Kind() == reflect.Slice {
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

func (this *SingleArgument) valueIsBool() bool {
	rv := this.value
	if rv.Kind() == reflect.Bool {
		return true
	}

	if rv.Kind() == reflect.Ptr && rv.Type().Elem().Kind() == reflect.Bool {
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
	if this.valueIsBool() {
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

func (this *SingleArgument) DoAction() error {
	if this.valueIsBool() {
		var v bool
		if this.useDefault {
			v = !this.defaultBoolValue()
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

func (this *MultiArgument) SetValue(val string) error {
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
	prog := fmt.Sprintf("%s %s", this.parser.prog, command)
	parser, e := NewArgumentParser(target, prog, desc, "")
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

func (this *ArgumentParser) findOptionalArgument(token string, exactMatch bool) Argument {
	var match_arg Argument = nil
	match_len := -1
	for _, arg := range this.optArgs {
		if tokenMatch(arg.Token(), token, exactMatch) {
			if match_len < 0 || match_len > len(arg.Token()) {
				match_len = len(arg.Token())
				match_arg = arg
			}
		} else if tokenMatch(arg.ShortToken(), token, exactMatch) {
			if match_len < 0 || match_len > len(arg.ShortToken()) {
				match_len = len(arg.ShortToken())
				match_arg = arg
			}
		} else if tokenMatch(arg.AliasToken(), token, exactMatch) {
			if match_len < 0 || match_len > len(arg.AliasToken()) {
				match_len = len(arg.AliasToken())
				match_arg = arg
			}
		}
	}
	return match_arg
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
}

func (this *ArgumentParser) ParseArgs(args []string, ignore_unknown bool) error {
	return this.ParseArgs2(args, ignore_unknown, true)
}

func (this *ArgumentParser) ParseArgs2(args []string, ignore_unknown bool, setDefaults bool) error {
	var pos_idx int = 0
	var arg Argument = nil
	var err error = nil
	var argStr string

	this.reset()

	for i := 0; i < len(args) && err == nil; i++ {
		argStr = args[i]
		if strings.HasPrefix(argStr, "-") {
			arg = this.findOptionalArgument(strings.TrimLeft(argStr, "-"), false)
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
					err = arg.DoAction()
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
				arg = this.posArgs[pos_idx]
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
	return str[0] == quoteChar && str[len(str)-1] == quoteChar
}

func isQuoted(str string) bool {
	return isQuotedByChar(str, '"') || isQuotedByChar(str, '\'')
}

func (this *ArgumentParser) parseKeyValue(key, value string) error {
	arg := this.findOptionalArgument(key, true)
	if arg != nil {
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
	for key, obj := range dict.Value() {
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
	arg := this.findOptionalArgument(token, true)
	if arg == nil {
		log.Warningf("Cannot find argument %s", token)
		return nil
	}
	if arg.IsSet() {
		return nil
	}
	// process multi argument
	if arg.IsMulti() {
		array, ok := obj.(*jsonutils.JSONArray)
		if !ok {
			return fmt.Errorf("%s object value is not array", key)
		}
		for _, item := range array.Value() {
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
