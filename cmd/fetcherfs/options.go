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
	"os"

	"yunion.io/x/log"
	"yunion.io/x/structarg"
)

const BLOCK_SIZE = 8

type Options struct {
	Url        string `help:"destination to fetch content" required:"true"`
	Tmpdir     string `help:"temporary dir save fs content" required:"true"`
	Token      string `help:"authentication information to access given url" required:"true"`
	Blocksize  int    `help:"block size of content file system(MB)"`
	MountPoint string `help:"mount path of fuse fs" required:"true"`
	Debug      bool   `help:"enable debug go fuse"`
	Foreground bool   `help:"run in foreground"`
}

var opt = &Options{}

func init() {
	// structarg.NewArgumentParser(&BaseOptions{}
	parser, err := structarg.NewArgumentParser(opt, "", "", "")
	if err != nil {
		log.Fatalf("Error define argument parser: %v", err)
	}
	err = parser.ParseArgs2(os.Args[1:], true, true)
	if err != nil {
		log.Fatalf("Failed parse args %s", err)
	}
	log.Errorf("%v", opt)
	if opt.Blocksize <= 0 {
		opt.Blocksize = BLOCK_SIZE
	}
}
