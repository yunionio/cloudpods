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

package ovnutil

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/ovsdb/schema/ovn_nb"
	"yunion.io/x/ovsdb/types"
	"yunion.io/x/pkg/errors"
)

const ovnNbCtlTimeout = 20 * time.Second
const ovnBatchCmd = 75

type CmdResult struct {
	Output string
	Err    error
}

func (res *CmdResult) Error() string {
	return fmt.Sprintf("err: %v, output: %s", res.Err, res.Output)
}

type OvnNbCtl struct {
	db string
}

func NewOvnNbCtl(db string) *OvnNbCtl {
	cli := &OvnNbCtl{
		db: db,
	}
	return cli
}

func (cli *OvnNbCtl) prepArgs(args []string) []string {
	var r []string
	if cli.db != "" {
		r = make([]string, len(args)+1)
		r[0] = "--db=" + cli.db
		copy(r[1:], args)
	} else {
		r = args
	}
	return r
}

func (cli *OvnNbCtl) run(ctx context.Context, args []string) *CmdResult {
	ctx, cancel := context.WithTimeout(ctx, ovnNbCtlTimeout)
	defer cancel()

	args = cli.prepArgs(args)
	cmd := exec.CommandContext(ctx, "ovn-nbctl", args...)
	combined, err := cmd.CombinedOutput()
	res := &CmdResult{
		Output: string(combined),
		Err:    err,
	}
	return res
}

func (cli *OvnNbCtl) splitArgs(args []string) [][]string {
	idx, ret, part := 0, [][]string{}, []string{}
	for _, arg := range args {
		if arg == "--" {
			if idx > 0 && idx%ovnBatchCmd == 0 {
				ret = append(ret, part)
				part = []string{}
			}
			idx++
		}
		part = append(part, arg)
	}
	if len(part) > 0 {
		ret = append(ret, part)
	}
	return ret
}

func (cli *OvnNbCtl) Must(ctx context.Context, msg string, args []string) *CmdResult {
	var res *CmdResult
	for _, _args := range cli.splitArgs(args) {
		res = cli.must(ctx, msg, _args)
		if res.Err != nil {
			return res
		}
	}
	return res
}

func (cli *OvnNbCtl) must(ctx context.Context, msg string, args []string) *CmdResult {
	res := cli.run(ctx, args)
	if res.Err != nil {
		panic(cli.errWrap(res, msg, args))
	}
	if cli.argsHasWrite(args) {
		log.Infof("%s:\n%s", msg, ovnNbctlArgsString(args))
	}
	return res
}

func (cli *OvnNbCtl) errWrap(err error, msg string, args []string) error {
	s := cli.argsString(args)
	return errors.Wrapf(err, "%s:\n%s\n", msg, s)
}

func (cli *OvnNbCtl) argsString(args []string) string {
	args = cli.prepArgs(args)
	s := ovnNbctlArgsString(args)
	return s
}

func (cli *OvnNbCtl) argsHasWrite(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "create", "set", "add", "remove", "destroy", "clear":
			return true
		case "list", "find", "get":
		case "lsp-del", "lrp-del":
			return true
		default:
		}
	}
	return false
}

func ovnNbctlArgsString(args []string) string {
	var (
		s       = ""
		indent  = ""
		indent1 = "\t"
		indent2 = "\t\t"
	)
	s += "ovn-nbctl"
	for _, arg := range args {
		if arg == "--" {
			indent = indent1
			s += ` \` + "\n"
			s += indent
			s += arg
		} else if !strings.HasPrefix(arg, "--") && strings.ContainsRune(arg, '=') {
			if indent == indent1 {
				indent = indent2
			}
			s += ` \` + "\n"
			s += indent
			s += fmt.Sprintf("%q", arg)
		} else {
			s += fmt.Sprintf(" %q", arg)
		}
	}
	return s
}

func OvnNbctlArgsDestroy(irows []types.IRow) []string {
	sort.Slice(irows, func(i, j int) bool {
		ri := irows[i]
		rj := irows[j]
		iri := ri.OvsdbIsRoot()
		irj := rj.OvsdbIsRoot()
		if !iri && irj {
			return true
		}
		return false
	})
	var args []string
	for _, irow := range irows {
		var newArgs []string
		switch irow.(type) {
		case *ovn_nb.LogicalSwitchPort:
			newArgs = []string{"--", "--if-exists", "lsp-del", irow.OvsdbUuid()}
		case *ovn_nb.LogicalRouterPort:
			newArgs = []string{"--", "--if-exists", "lrp-del", irow.OvsdbUuid()}
		case *ovn_nb.LogicalRouterStaticRoute:
		case *ovn_nb.ACL:
		case *ovn_nb.QoS:
		default:
			if !irow.OvsdbIsRoot() {
				panic(irow.OvsdbTableName())
			}
			newArgs = []string{"--", "--if-exists", "destroy", irow.OvsdbTableName(), irow.OvsdbUuid()}
		}
		if len(newArgs) > 0 {
			if irow.OvsdbIsRoot() {
				args = append(args, newArgs...)
			} else {
				args = append(newArgs, args...)
			}
		}
	}
	return args
}
