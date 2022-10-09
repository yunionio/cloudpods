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

package misc

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

func init() {
	type SdkGenerateOptions struct {
		SERVICE  string `help:"Service type" choices:"compute|report|meter|cloudid|yunionconf|yunionagent"`
		SDK_PATH string
	}
	R(&SdkGenerateOptions{}, "generate-python-sdk", "generate python sdk mod", func(s *mcclient.ClientSession, args *SdkGenerateOptions) error {
		keywords := []string{"quota", "usage"}
		path := fmt.Sprintf("%s/yunionclient/api", args.SDK_PATH)
		files, err := os.ReadDir(path)
		if err != nil {
			return errors.Wrapf(err, "read sdk path %s", path)
		}
		for i := range files {
			if files[i].IsDir() {
				continue
			}
			data, _ := ioutil.ReadFile(fmt.Sprintf("%s/%s", path, files[i].Name()))
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.Contains(line, "keyword ") {
					info := strings.Split(line, "=")
					if len(info) == 2 {
						keyword := strings.Trim(info[1], " ")
						keyword = strings.ReplaceAll(keyword, "'", "")
						keyword = strings.ReplaceAll(keyword, `"`, "")
						keywords = append(keywords, keyword)
					}
				}
			}
		}
		template := `from yunionclient.common import base

class %s(base.ResourceBase):
    pass


class %sManager(base.%sManager):
    resource_class = %s
    keyword = '%s'
    keyword_plural = '%s'
    _columns = %s`

		shellTemplate := `import yunionclient

from yunionclient.common import utils

@utils.arg('--limit', metavar='<NUMBER>', default=20, help='Page limit')
@utils.arg('--offset', metavar='<OFFSET>', help='Page offset')
@utils.arg('--order-by', metavar='<ORDER_BY>', help='Name of fields order by')
@utils.arg('--order', metavar='<ORDER>', choices=['desc', 'asc'], help='order')
@utils.arg('--details', action='store_true', help='More detailed list')
@utils.arg('--search', metavar='<KEYWORD>', help='Filter result by simple keyword search')
@utils.arg('--meta', action='store_true', help='Piggyback metadata')
@utils.arg('--filter', metavar='<FILTER>', action='append', help='Filters')
@utils.arg('--filter-any', action='store_true', help='If true, match if any of the filters matches; otherwise, match if all of the filters match')
@utils.arg('--admin', action='store_true', help='Is admin call?')
@utils.arg('--tenant', metavar='<TENANT>', help='Tenant ID or Name')
@utils.arg('--field', metavar='<FIELD>', action='append', help='Show only specified fields')
def do_%s_list(client, args):
    """ List all %s"""
    page_info = utils.get_paging_info(args)
    %s = client.%s.list(**page_info)
    utils.print_list(%s, client.%s.columns)`
		mods, _ := modulebase.GetRegisterdModules()
		invalidMods := []string{}
		imports, clients, shells := []string{}, []string{}, []string{}
		for i := range mods {
			mod, err := modulebase.GetModule(s, mods[i])
			if err != nil {
				invalidMods = append(invalidMods, mods[i])
				continue
			}
			if mod.ServiceType() != args.SERVICE {
				continue
			}
			if strings.Contains(mod.GetKeyword(), "-") {
				continue
			}
			manager := "Standalone"
			if args.SERVICE != "compute" {
				manager = utils.Capitalize(args.SERVICE)
			}
			if !utils.IsInStringArray(mod.GetKeyword(), keywords) {
				cls := utils.Kebab2Camel(mod.GetKeyword(), "_")
				columes := mod.GetColumns(s)
				data := fmt.Sprintf(template, cls, cls, manager, cls, mod.GetKeyword(), mod.KeyString(), jsonutils.Marshal(columes))
				filename := strings.ReplaceAll(mod.KeyString(), "_", "")
				filePath := fmt.Sprintf("%s/%s.py", path, filename)
				err := ioutil.WriteFile(filePath, []byte(data), 0644)
				if err != nil {
					log.Errorf("write %s error: %v", filePath, err)
					continue
				}
				log.Infof("mod %s generage...", mod.GetKeyword())
				imports = append(imports, fmt.Sprintf("from yunionclient.api import %s", filename))
				clients = append(clients, fmt.Sprintf("        self.%s = %s.%sManager(self)", filename, filename, cls))
				shell := fmt.Sprintf(shellTemplate, mod.GetKeyword(), strings.ReplaceAll(mod.KeyString(), "_", ""), filename, filename, filename, filename)
				filePath = fmt.Sprintf("%s/yunionclient/shells/%s.py", args.SDK_PATH, filename)
				err = ioutil.WriteFile(filePath, []byte(shell), 0644)
				if err != nil {
					log.Errorf("write %s error: %v", filePath, err)
					continue
				}
				shells = append(shells, fmt.Sprintf("from .%s import *", filename))
			}
		}
		clientPath := fmt.Sprintf("%s/client.py", path)
		data, err := ioutil.ReadFile(clientPath)
		if err != nil {
			return errors.Wrapf(err, "read client")
		}
		newLines := []string{}
		lines := strings.Split(string(data), "\n")
		isImport, isClient := false, false
		for i := len(lines) - 1; i >= 0; i-- {
			if !isImport && strings.HasPrefix(lines[i], "from") {
				newLines = append(imports, newLines...)
				isImport = true
			}
			if !isClient && strings.HasSuffix(lines[i], "Manager(self)") {
				isClient = true
				newLines = append(clients, newLines...)
			}
			newLines = append([]string{lines[i]}, newLines...)
		}
		err = ioutil.WriteFile(clientPath, []byte(strings.Join(newLines, "\n")), 0755)
		if err != nil {
			return errors.Wrapf(err, "write client")
		}
		shellPath := fmt.Sprintf("%s/yunionclient/shells/__init__.py", args.SDK_PATH)
		data, err = ioutil.ReadFile(shellPath)
		if err != nil {
			return errors.Wrapf(err, "read shell")
		}
		lines = strings.Split(string(data), "\n")
		lines = append(lines, shells...)
		return ioutil.WriteFile(shellPath, []byte(strings.Join(lines, "\n")), 0755)
	})

}
