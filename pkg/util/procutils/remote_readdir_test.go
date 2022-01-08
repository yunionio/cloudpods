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

package procutils

import (
	"testing"
)

func TestReadDir(t *testing.T) {
	files, err := RemoteReadDir(".")
	if err != nil {
		t.Errorf("RemoteReadDIr %s", err)
	} else {
		for _, f := range files {
			t.Logf("%s %d %v %s", f.Name(), f.Size(), f.IsDir(), f.ModTime())
		}
	}
}

func TestParseLsLine(t *testing.T) {
	cases := []string{
		"dr-x------. 45    0    0      16384 2022-04-26 19:48:03.811985235 +0800 .",
		"drwxr-xr-x. 26    0    0       4096 2022-04-20 10:06:16.339991488 +0800 ..",
		"drwxr-xr-x.  2    0    0       4096 2022-04-01 14:20:53.293839273 +0800 0401",
		"drwxr-xr-x.  2    0    0       4096 2022-04-05 15:39:23.634908120 +0800 0405image",
		"-rw-r--r--.  1    0    0        335 2022-01-10 09:12:34.924898855 +0800 1.c",
		"-rw-r--r--.  1    0    0        304 2022-03-04 11:10:00.803987595 +0800 2.c",
		"drwx------.  4    0    0       4096 2021-12-08 11:08:04.304950170 +0800 .ansible",
		"-rw-r--r--.  1    0    0   26115746 2022-03-22 19:00:45.368976212 +0800 apigateway-ee-200200321.tgz",
		"lrwxrwxrwx.  1    0    0          7 2021-12-16 10:19:58.924982587 +0800 .bash_profile -> .bashrc",
		"-rw-------.  1    0    0       4107 2022-04-18 17:26:56.108984702 +0800 .bashrc",
		"drwx------.  2    0    0       4096 2021-12-20 20:02:30.483709077 +0800 build",
	}
	for _, c := range cases {
		f, err := parseLsLine(c)
		if err != nil {
			t.Errorf("parseLsLine %s fail %s", c, err)
		} else {
			t.Logf("%s %d %v %s", f.Name(), f.Size(), f.IsDir(), f.ModTime())
		}
	}
}
