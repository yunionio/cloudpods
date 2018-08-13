package promputils

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

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
	args := strings.Split(s, " ")
	e := parser.ParseArgs(strings.Split(s, " "), false)
	subcmd := parser.GetSubcommand()
	subparser := subcmd.GetSubParser()
	if args[0] == "--debug" {
		session.GetClient().SetDebug(true)
	} else {
		session.GetClient().SetDebug(false)
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
		if args[0] == "help" {
			e = subcmd.Invoke(suboptions)
		} else {
			e = subcmd.Invoke(session, suboptions)
		}
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
