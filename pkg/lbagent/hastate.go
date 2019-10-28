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

package lbagent

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"github.com/fsnotify/fsnotify"

	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	agentutils "yunion.io/x/onecloud/pkg/lbagent/utils"
)

const (
	HA_STATE_SCRIPT_NAME    = "ha_state.sh"
	HA_STATE_SCRIPT_CONTENT = `
#!/bin/bash
echo "$@" >%s
`
	HA_STATE_FILENAME = "ha_state"
)

type HaStateProvider interface {
	StateChannel() <-chan string
	StateScript() string
}

type HaStateWatcher struct {
	HaStateScriptPath string
	HaStatePath       string
	CurrentState      string // TODO hide it

	opts *Options
	w    *fsnotify.Watcher
	C    chan string
}

func (hsw *HaStateWatcher) StateChannel() <-chan string {
	return hsw.C
}

func (hsw *HaStateWatcher) StateScript() string {
	return hsw.HaStateScriptPath
}

func (hsw *HaStateWatcher) Run(ctx context.Context) {
	defer func() {
		log.Infof("ha state watcher bye")
		wg := ctx.Value("wg").(*sync.WaitGroup)
		wg.Done()
	}()

	hsw.loadHaState()
	hsw.C <- hsw.CurrentState

	statePending := false
	tick := time.NewTicker(3 * time.Second)
	defer tick.Stop()
	for {
		select {
		case ev := <-hsw.w.Events:
			switch {
			case ev.Name == hsw.opts.haproxyRunDir:
				switch ev.Op {
				case fsnotify.Remove, fsnotify.Rename:
					log.Errorf("run dir %s", ev.Op.String())
					hsw.sayBye()
				default:
					log.Debugf("ignored %s", ev)
				}
			case ev.Name == hsw.HaStatePath:
				log.Infof("hastate file: %s", ev)
				switch ev.Op {
				case fsnotify.Create, fsnotify.Write:
					err := hsw.loadHaState()
					if err != nil {
						log.Errorf("load state: %s", err)
						hsw.sayBye()
					}
					select {
					case hsw.C <- hsw.CurrentState:
						statePending = false
					default:
						statePending = true
					}
				}
			}
		case err := <-hsw.w.Errors:
			log.Errorf("watcher err: %s", err)
			hsw.sayBye()
		case <-tick.C:
			if statePending {
				select {
				case hsw.C <- hsw.CurrentState:
					statePending = false
				default:
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (hsw *HaStateWatcher) loadHaState() (err error) {
	defer func() {
		if err != nil {
			hsw.CurrentState = api.LB_HA_STATE_UNKNOWN
		}
	}()
	data, err := ioutil.ReadFile(hsw.HaStatePath)
	if err != nil {
		return err
	}
	log.Infof("got state: %s", data)
	// Sample content
	//
	// 	INSTANCE YunionLB BACKUP 110
	//
	fields := strings.Fields(string(data))
	if len(fields) >= 3 {
		hsw.CurrentState = fields[2]
		return
	}
	err = fmt.Errorf("ha state file contains too little info")
	return
}

func (hsw *HaStateWatcher) sayBye() {
	syscall.Kill(os.Getpid(), syscall.SIGTERM)
}

func NewHaStateWatcher(opts *Options) (hsw *HaStateWatcher, err error) {
	var (
		w *fsnotify.Watcher
	)
	defer func() {
		if err != nil {
			if w != nil {
				w.Close()
			}
		}
	}()

	haStateScriptPath := path.Join(opts.haproxyShareDir, HA_STATE_SCRIPT_NAME)
	haStatePath := path.Join(opts.haproxyRunDir, HA_STATE_FILENAME)
	{
		content := fmt.Sprintf(HA_STATE_SCRIPT_CONTENT, haStatePath)
		content = strings.TrimLeftFunc(content, unicode.IsSpace)
		mode := agentutils.FileModeFileExec
		err = ioutil.WriteFile(haStateScriptPath, []byte(content), mode)
		if err != nil {
			return
		}
		var fi os.FileInfo
		fi, err = os.Stat(haStateScriptPath)
		if err != nil {
			return
		}
		if fi.Mode() != mode {
			err = os.Chmod(haStateScriptPath, mode)
			if err != nil {
				return
			}
		}
	}

	w, err = fsnotify.NewWatcher()
	if err != nil {
		return
	}
	err = w.Add(opts.haproxyRunDir)
	if err != nil {
		return
	}

	hsw = &HaStateWatcher{
		HaStateScriptPath: haStateScriptPath,
		HaStatePath:       haStatePath,
		CurrentState:      api.LB_HA_STATE_UNKNOWN,

		opts: opts,
		w:    w,
		C:    make(chan string),
	}
	return
}
