package promputils

import (
	"fmt"
	"regexp"
	"strings"

	prompt "github.com/c-bata/go-prompt"
)

type Cmd struct {
	desc    string
	optArgs []prompt.Suggest
	posArgs []prompt.Suggest
}

var cmds = make(map[string]*Cmd)

var optionHelp = []prompt.Suggest{
	{Text: "-h"},
	{Text: "--help"},
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

	if cmds[args[0]] == nil {
		return []prompt.Suggest{}
	}

	_cmd := cmds[args[0]].optArgs
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
	_cmd, ok := cmds[args[0]]
	if !ok {
		return []prompt.Suggest{}
	}

	if len(args)-1 > len(_cmd.posArgs) {
		return []prompt.Suggest{}
	}

	prm := _cmd.posArgs[len(args)-2]
	subcommands := []prompt.Suggest{
		prm,
	}

	return prompt.FilterHasPrefix(subcommands, args[len(args)-1], true)
}

func AppendCommand(text, desc string) {
	commands = append(commands, prompt.Suggest{Text: text, Description: desc})
	cmds[text] = &Cmd{}
	cmds[text].desc = desc
	cmds[text].posArgs = []prompt.Suggest{}
	cmds[text].optArgs = []prompt.Suggest{}
}

func AppendPos(text, cmd, desc string) {
	var v = []prompt.Suggest{{Text: cmd, Description: desc}}
	cmds[text].posArgs = append(cmds[text].posArgs, v...)
}

func AppendOpt(text, cmd, desc string) {
	var v = []prompt.Suggest{{Text: cmd, Description: desc}}
	cmds[text].optArgs = append(cmds[text].optArgs, v...)
}

func GenerateAutoCompleteCmds(shell string) string {
	var ret = []string{}
	var i = 0
	if strings.ToLower(shell) == "zsh" {
		i = 1
	}
	for cmd, options := range cmds {
		var (
			strPosArgs = []string{}
			strOptArgs = []string{}
		)
		for _, posArg := range options.posArgs {
			strPosArgs = append(strPosArgs, posArg.Text)
		}
		for _, optArg := range options.optArgs {
			strOptArgs = append(strOptArgs, strings.Split(optArg.Text, " ")[0])
		}
		ret = append(ret, fmt.Sprintf(`arr[%d]="%s# %s %s"`, i, cmd,
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
