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

package main

import (
	"fmt"
	"os"
	"path"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/structarg"
)

func ParseOptions(optStruct interface{}, args []string, configFileName string) {
	parser, err := structarg.NewArgumentParser(optStruct,
		"dingtalk-sender", "", "")
	if err != nil {
		log.Fatalf("Error define argument parser: %v", err)
	}

	err = parser.ParseArgs2(args[1:], false, false)
	if err != nil {
		log.Fatalf("Parse arguments error: %v", err)
	}

	var optionsRef *structarg.BaseOptions

	err = reflectutils.FindAnonymouStructPointer(optStruct, &optionsRef)
	if err != nil {
		log.Fatalf("Find common options fail %s", err)
	}

	if optionsRef.Help {
		fmt.Println(parser.HelpString())
		os.Exit(0)
	}

	if len(optionsRef.Config) == 0 {
		for _, p := range []string{"./etc", "/etc/yunion/notify"} {
			confTmp := path.Join(p, configFileName)
			if _, err := os.Stat(confTmp); err == nil {
				optionsRef.Config = confTmp
				break
			}
		}
	}

	if len(optionsRef.Config) > 0 {
		log.Infof("Use configuration file: %s", optionsRef.Config)
		err = parser.ParseFile(optionsRef.Config)
		if err != nil {
			log.Fatalf("Parse configuration file: %v", err)
		}
	}
}

type SRegularConfig struct {
	SockFileDir string `help:"socket file directory" default:"/etc/yunion/notify"`

	SenderNum int `help:"number of sender" default:5`

	TemplateDir string `help:"template directroy"`

	structarg.BaseOptions
}
