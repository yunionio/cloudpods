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

package shellutils

import (
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"yunion.io/x/pkg/errors"
)

var editors = []string{
	"vim",
	"vi",
	"nvim",
	"nano",
}

func findEditor() string {
	for _, prog := range editors {
		cmd := exec.Command(prog, "--version")
		if err := cmd.Run(); err == nil {
			return prog
		}
	}
	return ""
}

func Edit(yaml string) (string, error) {
	tmpfile, err := ioutil.TempFile("", "policy-blob")
	if err != nil {
		return "", errors.Wrap(err, "ioutil.TempFile")
	}
	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write([]byte(yaml)); err != nil {
		return "", errors.Wrap(err, "tmpfile.Write")
	}
	if err := tmpfile.Close(); err != nil {
		return "", errors.Wrap(err, "tmpfile.Close")
	}

	editor := findEditor()
	if len(editor) == 0 {
		return "", errors.Wrapf(errors.ErrNotFound, "no editor found, supported editors are: %s", strings.Join(editors, ","))
	}
	cmd := exec.Command(editor, tmpfile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		return "", errors.Wrap(err, "cmd.Run")
	}

	policyBytes, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		return "", errors.Wrap(err, "ioutil.ReadFile")
	}

	if yaml == string(policyBytes) {
		return "", errors.Error("no change")
	}

	return string(policyBytes), nil
}
