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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"yunion.io/x/pkg/utils"
	"yunion.io/x/structarg"

	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	parser  *structarg.ArgumentParser
	session *mcclient.ClientSession
)

func InitEnv(_parser *structarg.ArgumentParser, _session *mcclient.ClientSession) {
	parser = _parser
	session = _session
}

func escaper(str string) string {
	var re = regexp.MustCompile(`(?i)--filter\s+[a-z0-9]+[^\s]+( |$)`)
	for _, match := range re.FindAllString(str, -1) {
		rep := strings.Replace(match, "--filter", "", -1)
		rep = strings.TrimSpace(rep)
		rep = strings.Replace(rep, `\'`, "efWpvXpY6lH5", -1)
		rep = strings.Replace(rep, "'", "", -1)
		rep = strings.Replace(rep, `\"`, "GsVHUhkj68Be", -1)
		rep = strings.Replace(rep, `"`, "", -1)
		rep = strings.Replace(rep, `GsVHUhkj68Be`, `\"`, -1)
		rep = strings.Replace(rep, `efWpvXpY6lH5`, `\'`, -1)
		rep = `--filter ` + rep
		str = strings.Replace(str, match, rep, -1)
	}
	return str
}

func Executor(s string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return
	} else if s == "quit" || s == "exit" || s == "x" || s == "q" {
		fmt.Println("Bye!")
		os.Exit(0)
		return
	}
	s = escaper(s)
	args := utils.ArgsStringToArray(s)
	e := parser.ParseArgs(utils.ArgsStringToArray(s), false)
	subcmd := parser.GetSubcommand()
	subparser := subcmd.GetSubParser()
	if args[0] == "--debug" {
		session.GetClient().SetDebug(true)
	} else {
		session.GetClient().SetDebug(false)
	}
	if subparser.IsHelpSet() {
		fmt.Print(subparser.HelpString())
		return
	}
	if e != nil {
		if subparser != nil {
			fmt.Print(subparser.Usage())
		} else {
			fmt.Print(parser.Usage())
		}
		fmt.Println(e)
	} else {
		suboptions := subparser.Options()
		e = subcmd.Invoke(session, suboptions)
		if e != nil {
			fmt.Println(e)
		}
	}
	return
}

func ExecuteAndGetResult(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", errors.New("you need to pass the something arguments")
	}

	out := &bytes.Buffer{}
	cmd := exec.Command("/bin/sh", "-c", "source ~/.RC_ADMIN &&/home/yunion/git/yunioncloud/_output/bin/climc "+s)
	cmd.Stdin = os.Stdin
	cmd.Stdout = out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	r := string(out.Bytes())
	return r, nil
}
