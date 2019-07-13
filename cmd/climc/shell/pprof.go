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

package shell

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"syscall"

	"yunion.io/x/pkg/util/signalutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func init() {
	type TraceOptions struct {
		Second  int    `help:"pprof seconds" short-token:"s"`
		Service string `help:"Service type"`
		Address string `help:"Service listen address"`
	}

	downloadToTemp := func(input io.Reader, pattern string) (string, error) {
		tmpfile, err := ioutil.TempFile("", pattern)
		if err != nil {
			return "", err
		}
		defer tmpfile.Close()
		if _, err := io.Copy(tmpfile, input); err != nil {
			return "", err
		}
		return tmpfile.Name(), nil
	}

	pprofRun := func(s *mcclient.ClientSession, opts *TraceOptions, pType string, args ...string) error {
		var (
			src io.Reader
			err error
		)
		if len(opts.Service) > 0 {
			src, err = modules.GetPProfByType(s, opts.Service, pType, opts.Second)
			if err != nil {
				return err
			}
		} else if len(opts.Address) > 0 {
			src, err = modules.GetNamedAddressPProfByType(s, opts.Address, pType, opts.Second)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("no service address provide")
		}

		tempfile, err := downloadToTemp(src, pType)
		if err != nil {
			return err
		}
		defer func() { os.Remove(tempfile) }()

		signalutils.RegisterSignal(func() {
			os.Remove(tempfile)
			os.Exit(0)
		}, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
		signalutils.StartTrap()

		argv := []string{"tool"}
		argv = append(argv, args...)
		argv = append(argv, tempfile)
		cmd := procutils.NewCommand("go", argv...)
		if err := cmd.Run(); err != nil {
			return err
		}
		return nil
	}

	R(&TraceOptions{}, "pprof-trace", "pprof trace of backend service", func(s *mcclient.ClientSession, args *TraceOptions) error {
		return pprofRun(s, args, "trace", "trace")
	})

	R(&TraceOptions{}, "pprof-profile", "pprof profile of backend service", func(s *mcclient.ClientSession, args *TraceOptions) error {
		port, err := netutils2.GetFreePort()
		if err != nil {
			return err
		}
		return pprofRun(s, args, "profile", "pprof", fmt.Sprintf("-http=:%d", port))
	})
}
