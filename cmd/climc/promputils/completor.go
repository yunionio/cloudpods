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
	"bytes"
	"fmt"
	"regexp"
	"strings"

	prompt "github.com/c-bata/go-prompt"

	"yunion.io/x/structarg"
)

var (
	subcmds = make(map[string]*Cmd)
	rootCmd *Cmd
)

var optionHelp = []prompt.Suggest{
	{Text: "-h"},
	{Text: "--help"},
}

type Cmd struct {
	Name      string
	Desc      string
	optArgs   []CmdArgument
	posArgs   []CmdArgument
	ParentCmd *Cmd
	SubCmds   []*Cmd
}

func InitRootCmd(name, desc string, optArgs, posArgs []structarg.Argument) *Cmd {
	rootCmd = NewCmd(name, desc)
	for _, optA := range optArgs {
		rootCmd.AddOptArgument(optA, optA.Token(), optA.HelpString(""))
	}
	for _, posA := range posArgs {
		rootCmd.AddPosArgument(posA, posA.Token(), posA.HelpString(""))
	}
	return rootCmd
}

func GetRootCmd() *Cmd {
	return rootCmd
}

func NewCmd(name, desc string) *Cmd {
	return &Cmd{
		Name:    name,
		Desc:    desc,
		optArgs: make([]CmdArgument, 0),
		posArgs: make([]CmdArgument, 0),
		SubCmds: make([]*Cmd, 0),
	}
}

func (c Cmd) getPromptSuggests(args []CmdArgument) []prompt.Suggest {
	ret := make([]prompt.Suggest, 0)
	for _, arg := range args {
		ret = append(ret, arg.Suggest)
	}
	return ret
}

func (c *Cmd) Root() *Cmd {
	if c.ParentCmd == nil {
		return c
	}
	return c.ParentCmd.Root()
}

func (c Cmd) GetName() string {
	return c.Name
}

func (c *Cmd) AddCmd(cmd *Cmd) {
	c.SubCmds = append(c.SubCmds, cmd)
}

func (c Cmd) GetPromptOptSuggests() []prompt.Suggest {
	return c.getPromptSuggests(c.optArgs)
}

func (c Cmd) GetPromptPosSuggests() []prompt.Suggest {
	return c.getPromptSuggests(c.posArgs)
}

func (c Cmd) GetOptArguments() []structarg.Argument {
	return c.getArguments(c.optArgs)
}

func (c Cmd) GetArguments() []structarg.Argument {
	ret := c.GetOptArguments()
	ret = append(ret, c.GetPosArguments()...)
	return ret
}

func (c Cmd) GetPosArguments() []structarg.Argument {
	return c.getArguments(c.posArgs)
}

func (c Cmd) getArguments(args []CmdArgument) []structarg.Argument {
	ret := make([]structarg.Argument, 0)
	for _, a := range args {
		ret = append(ret, a.Argument)
	}
	return ret
}

type CmdArgument struct {
	Suggest  prompt.Suggest
	Argument structarg.Argument
}

func optionCompleter(args []string, long bool) []prompt.Suggest {
	l := len(args)
	if l == 0 {
		return []prompt.Suggest{}
	}
	if l <= 1 {
		if long {
			return prompt.FilterHasPrefix(optionHelp, "--", false)
		}
		return optionHelp
	}

	if subcmds[args[0]] == nil {
		return []prompt.Suggest{}
	}

	_cmd := subcmds[args[0]].GetPromptOptSuggests()
	return prompt.FilterContains(_cmd, strings.TrimLeft(args[l-1], "-"), true)
}

func Completer(d prompt.Document) []prompt.Suggest {
	if d.TextBeforeCursor() == "" {
		return []prompt.Suggest{}
	}
	var re = regexp.MustCompile(`(?m)^(?:-d|--debug)\s+`)
	s := re.ReplaceAllString(d.TextBeforeCursor(), "")
	s = strings.TrimSpace(s)
	args := strings.Split(s, " ")
	w := d.GetWordBeforeCursor()

	// If PIPE is in text before the cursor, returns empty suggestions.
	for i := range args {
		if args[i] == "|" {
			return []prompt.Suggest{}
		}
	}

	// If word before the cursor starts with "-", returns CLI flag options.
	if strings.HasPrefix(w, "-") {
		return optionCompleter(args, strings.HasPrefix(w, "--"))
	}

	// Return suggestions for option
	if suggests, found := completeOptionArguments(d); found {
		return suggests
	}

	return argumentsCompleter(excludeOptions(args))
}

var commands = []prompt.Suggest{

	// Custom command.
	{Text: "exit", Description: "Exit this program"},
	{Text: "quit", Description: "Exit this program"},
}

func argumentsCompleter(args []string) []prompt.Suggest {
	if len(args) == 0 {
		return []prompt.Suggest{}
	}

	if len(args) == 1 {
		return prompt.FilterHasPrefix(commands, args[0], true)
	}
	_cmd, ok := subcmds[args[0]]
	if !ok {
		return []prompt.Suggest{}
	}

	if len(args)-1 > len(_cmd.posArgs) {
		return []prompt.Suggest{}
	}

	prm := _cmd.posArgs[len(args)-2]
	subcommands := []prompt.Suggest{
		prm.Suggest,
	}

	return prompt.FilterHasPrefix(subcommands, args[len(args)-1], true)
}

func AppendCommand(parentCmd *Cmd, text, desc string) {
	commands = append(commands, prompt.Suggest{Text: text, Description: desc})
	cmd := &Cmd{
		Name:      text,
		Desc:      desc,
		posArgs:   make([]CmdArgument, 0),
		optArgs:   make([]CmdArgument, 0),
		ParentCmd: parentCmd,
	}
	subcmds[text] = cmd
	parentCmd.AddCmd(cmd)
}

func (c *Cmd) addArgument(target *[]CmdArgument, arg structarg.Argument, argStr string, desc string) {
	*target = append(*target, CmdArgument{
		Suggest: prompt.Suggest{
			Text:        argStr,
			Description: desc,
		},
		Argument: arg,
	})
}

func (c *Cmd) AddPosArgument(arg structarg.Argument, argStr string, desc string) {
	c.addArgument(&c.posArgs, arg, argStr, desc)
}

func (c *Cmd) AddOptArgument(arg structarg.Argument, argStr string, desc string) {
	c.addArgument(&c.optArgs, arg, argStr, desc)
}

func AppendPos(text, cmd, desc string, arg structarg.Argument) {
	cmdObj := subcmds[text]
	cmdObj.AddPosArgument(arg, cmd, desc)
}

func AppendOpt(text, cmd, desc string, arg structarg.Argument) {
	cmdObj := subcmds[text]
	cmdObj.AddOptArgument(arg, cmd, desc)
}

func GenerateAutoCompleteCmds(rootCmd *Cmd, shell string) string {
	var ret = []string{}
	var i = 0
	if strings.ToLower(shell) == "zsh" {
		out := bytes.NewBufferString("")
		if err := rootCmd.GenZshCompletion(out); err != nil {
			panic(err)
		}
		return out.String()
	}
	for _, cmd := range subcmds {
		var (
			strPosArgs = []string{}
			strOptArgs = []string{}
		)
		for _, posArg := range cmd.GetPromptPosSuggests() {
			strPosArgs = append(strPosArgs, posArg.Text)
		}
		for _, optArg := range cmd.GetPromptOptSuggests() {
			strOptArgs = append(strOptArgs, strings.Split(optArg.Text, " ")[0])
		}
		ret = append(ret, fmt.Sprintf(`arr[%d]="%s# %s %s"`, i, cmd.Name,
			strings.Join(strPosArgs, " "), strings.Join(strOptArgs, " ")))
		i += 1
	}
	options := strings.Join(ret, "\n")
	if strings.ToLower(shell) == "bash" {
		return fmt.Sprintf(BASH_COMPLETE_SCRIPT_1, options, BASH_COMPLETE_SCRIPT_2)
	} else {
		return ""
	}
}
