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
	"fmt"
	"io"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

func FilePutContents(filename string, content string) error {
	cmd := NewRemoteCommandAsFarAsPossible("cp", "/dev/stdin", filename)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return errors.Wrap(err, "StdinPipe")
	}
	exitSignal := make(chan error)
	go func() {
		defer stdin.Close()
		_, err := stdin.Write([]byte(content))
		if err != nil {
			exitSignal <- err
		}
		log.Debugf("write content %d to %s", len(content), filename)
		exitSignal <- nil
	}()
	err = cmd.Start()
	if err != nil {
		return errors.Wrap(err, "Run")
	}
	err = <-exitSignal
	if err != nil {
		return errors.Wrap(err, "Write")
	}
	killTimer := time.NewTimer(time.Second)
	go func() {
		<-killTimer.C
		fmt.Println("killTimer fired")
		cmd.Kill()
	}()
	err = cmd.Wait()
	if err != nil && !strings.Contains(err.Error(), "killed") {
		// ignore killed signal error
		return errors.Wrap(err, "Wait")
	}
	if !killTimer.Stop() {
		log.Debugf("killTimer has been expired")
	}
	return nil
}

func FileGetContents(filename string) ([]byte, error) {
	cmd := NewRemoteCommandAsFarAsPossible("cat", filename)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "StdoutPipe")
	}
	err = cmd.Start()
	if err != nil {
		return nil, errors.Wrap(err, "Run")
	}
	contentChan := make(chan []byte)
	go func() {
		defer stdout.Close()
		content, err := io.ReadAll(stdout)
		if err != nil {
			log.Errorf("ReadAll: %v", err)
		}
		contentChan <- content
	}()
	content := <-contentChan
	return content, nil
}
