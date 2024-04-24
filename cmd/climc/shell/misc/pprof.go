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
	"io"
	"io/ioutil"
	"os"
	"syscall"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/signalutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func init() {
	type TraceOptions struct {
		Second  int    `help:"pprof seconds" short-token:"s" default:"15"`
		Service string `help:"Service type"`
		Address string `help:"Service listen address"`
		Gc      bool   `help:"run GC before taking the heap sample"`
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
			src    io.Reader
			err    error
			svcUrl string
		)
		if len(opts.Service) > 0 {
			svcUrl, err = s.GetServiceURL(opts.Service, "")
			if err != nil {
				return errors.Wrapf(err, "get service %s url", opts.Service)
			}
		} else if len(opts.Address) > 0 {
			svcUrl = opts.Address
		} else {
			return fmt.Errorf("no service address provide")
		}

		params := jsonutils.NewDict()
		params.Add(jsonutils.NewInt(int64(opts.Second)), "seconds")
		if pType == "heap" && opts.Gc {
			params.Add(jsonutils.JSONTrue, "gc")
		}
		src, err = modules.GetNamedAddressPProfByType(s, svcUrl, pType, params)
		if err != nil {
			return err
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
		// A trace of execution of the current program
		return pprofRun(s, args, "trace", "trace")
	})

	for _, kind := range []string{
		// A sampling of all past memory allocations
		"allocs",
		// Stack traces that led to blocking on synchronization primitives
		"block",
		// The command line invocation of the current program
		"cmdline",
		// Stack traces of all current goroutines
		"goroutine",
		// A sampling of memory allocations of live objects
		"heap",
		// Stack straces of holders of contended mutexes
		"mutex",
		// CPU profile
		"profile",
		// Stack traces that led to the creation of new OS threads
		"threadcreate",
	} {
		pType := kind
		R(&TraceOptions{}, fmt.Sprintf("pprof-%s", pType), fmt.Sprintf("pprof %s of backend service", pType), func(s *mcclient.ClientSession, args *TraceOptions) error {
			port, err := netutils.GetFreePort()
			if err != nil {
				return err
			}
			return pprofRun(s, args, pType, "pprof", fmt.Sprintf("-http=:%d", port))
		})
	}
}
