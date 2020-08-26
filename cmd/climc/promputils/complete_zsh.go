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

package promputils

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"yunion.io/x/structarg"
)

// ref: https://github.com/spf13/cobra/blob/master/zsh_completions.go

var (
	zshCompFuncMap = template.FuncMap{
		"genZshFuncName":              zshCompGenFuncName,
		"extractFlags":                zshCompExtractFlag,
		"genFlagEntryForZshArguments": zshCompGenFlagEntryForArguments,
		"extractArgsCompletions":      zshCompExtractArgumentCompletionHintsForRendering,
	}
	zshCompletionText = `
{{/* should accept Command (that contains subcommands) as parameter */}}
{{define "argumentsC" -}}
{{ $cmdPath := genZshFuncName .}}
function {{$cmdPath}} {
	local -a commands
	_arguments -C \{{- range extractFlags .}}
	  {{genFlagEntryForZshArguments .}} \{{- end}}
	  "1: :->cmnds" \
	  "*::arg:->args"

	case $state in
	cmnds)
	  commands=({{range .SubCmds}}
		"{{.Name}}:{{.Desc}}"{{end}}
	  )
	  _describe "command" commands
	  ;;
	esac

	case "$words[1]" in {{- range .SubCmds}}
	{{.Name}})
	  {{$cmdPath}}_{{.Name}}
	  ;;{{end}}
	esac
}
{{range .SubCmds}}
{{template "selectCmdTemplate" .}}
{{- end}}
{{- end}}

{{/* should accept Command without subcommands as parameter */}}
{{define "arguments" -}}
function {{genZshFuncName .}} {
{{"  _arguments"}}{{range extractFlags .}} \
     {{genFlagEntryForZshArguments . -}}
{{end}}{{range extractArgsCompletions .}} \
     {{.}}{{end}}
}
{{end}}

{{/* dispatcher for commands with or without subcommands */}}
{{define "selectCmdTemplate" -}}
{{if .SubCmds}}{{template "argumentsC" .}}{{else}}{{template "arguments" .}}{{end}}
{{- end}}

{{/* template entry point */}}
{{define "Main" -}}
#compdef _{{.Name}} {{.Name}}

{{template "selectCmdTemplate" .}}
compdef _{{.Name}} {{.Name}}
{{end}}
`
)

func zshCompGenFuncName(c *Cmd) string {
	if c.ParentCmd != nil {
		return zshCompGenFuncName(c.ParentCmd) + "_" + c.Name
	}
	return "_" + c.Name
}

func zshCompExtractFlag(c *Cmd) []structarg.Argument {
	return c.GetArguments()
}

func zshCompGenFlagEntryForArguments(f structarg.Argument) string {
	// not process positional argument and single command
	if f.IsPositional() {
		return ""
	}
	if f.ShortToken() == "" {
		return zshCompGenFlagEntryForSingleOptionFlag(f)
	}
	return zshCompGenFlagEntryForMultiOptionFlag(f)
}

func zshCompGenFlagEntryForSingleOptionFlag(f structarg.Argument) string {
	var option, multiMark, extras string
	if f.IsMulti() {
		multiMark = "*"
	}

	option = "--" + f.Token()
	extras = zshCompGenFlagEntryExtras(f)
	return fmt.Sprintf(`'%s%s[%s]%s'`, multiMark, option, zshCompQuoteFlagDescription(f.HelpString("")), extras)
}

func zshCompGenFlagEntryForMultiOptionFlag(f structarg.Argument) string {
	var options, parenMultiMark, curlyMultiMark, extras string
	if f.IsMulti() {
		parenMultiMark = "*"
		curlyMultiMark = "\\*"
	}

	options = fmt.Sprintf(`'(%s-%s %s--%s)'{%s-%s,%s--%s}`,
		parenMultiMark, f.ShortToken(), parenMultiMark, f.Token(), curlyMultiMark, f.ShortToken(), curlyMultiMark, f.Token())
	extras = zshCompGenFlagEntryExtras(f)

	return fmt.Sprintf(`%s'[%s]%s'`, options, zshCompQuoteFlagDescription(f.HelpString("")), extras)
}

func zshCompGenFlagEntryExtras(f structarg.Argument) string {
	if !f.NeedData() {
		return ""
	}
	// allow options for flag
	extras := ":"

	type iChoices interface {
		Choices() []string
	}
	if hasChoices, ok := f.(iChoices); ok {
		// process choices
		var words []string
		for _, w := range hasChoices.Choices() {
			words = append(words, fmt.Sprintf("%q", w))
		}
		if len(words) != 0 {
			extras = fmt.Sprintf("%s :(%s)", extras, strings.Join(words, " "))
		}
	}
	return extras
}

func zshCompQuoteFlagDescription(s string) string {
	s = strings.Replace(s, "'", `'\''`, -1)
	s = strings.Replace(s, "[", `\[`, -1)
	s = strings.Replace(s, "]", `\]`, -1)
	return s
}

func zshCompExtractArgumentCompletionHintsForRendering(c *Cmd) []string {
	return nil
}

func (c *Cmd) GenZshCompletion(w io.Writer) error {
	tmpl, err := template.New("Main").Funcs(zshCompFuncMap).Parse(zshCompletionText)
	if err != nil {
		return fmt.Errorf("error creating zsh completion template: %v", err)
	}
	return tmpl.Execute(w, c.Root())
}
