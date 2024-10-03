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

package compute

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/pod/remotecommand"
	"yunion.io/x/onecloud/pkg/util/pod/stream"
	"yunion.io/x/onecloud/pkg/util/pod/term"
)

type ContainerManager struct {
	modulebase.ResourceManager
}

func (man ContainerManager) SetupTTY(in io.Reader, out io.Writer, errOut io.Writer, raw bool) term.TTY {
	/*t := term.TTY{
		Out: out,
	}
	if in == nil {
		t.In = nil
		return t
	}*/
	return term.TTY{
		In:     in,
		Out:    out,
		Raw:    raw,
		TryDev: false,
		Parent: nil,
	}
}

func (man ContainerManager) Exec(s *mcclient.ClientSession, id string, opt *api.ContainerExecInput) error {
	//baseUrl, err := man.GetBaseUrl(s)
	//if err != nil {
	//	return errors.Wrapf(err, "GetBaseUrl")
	//}
	info, err := man.GetSpecific(s, id, "exec-info", nil)
	if err != nil {
		return errors.Wrap(err, "get exec info")
	}
	infoOut := new(api.ContainerExecInfoOutput)
	info.Unmarshal(infoOut)
	qs := jsonutils.Marshal(opt).QueryString()
	// urlLoc := fmt.Sprintf("%s/%s/%s/exec?%s", baseUrl, man.URLPath(), url.PathEscape(id), qs)
	urlLoc := fmt.Sprintf("%s/pods/%s/containers/%s/exec?%s", infoOut.HostUri, infoOut.PodId, infoOut.ContainerId, qs)
	url, err := url.Parse(urlLoc)
	if err != nil {
		return errors.Wrapf(err, "parse url: %s", urlLoc)
	}
	exec, err := remotecommand.NewSPDYExecutor("POST", url)
	if err != nil {
		return errors.Wrap(err, "NewSPDYExecutor")
	}
	headers := mcclient.GetTokenHeaders(s.GetToken())

	t := man.SetupTTY(os.Stdin, os.Stdout, os.Stderr, opt.Tty)
	sizeQueue := t.MonitorSize(t.GetSize())
	fn := func() error {
		return exec.Stream(remotecommand.StreamOptions{
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
			// Tty:               opt.Tty,
			Tty:               true,
			TerminalSizeQueue: sizeQueue,
			Header:            headers,
		})
	}
	return t.Safe(fn)
}

func (man ContainerManager) Log(s *mcclient.ClientSession, id string, opt *api.PodLogOptions) (io.ReadCloser, error) {
	info, err := man.GetSpecific(s, id, "exec-info", nil)
	if err != nil {
		return nil, errors.Wrap(err, "get exec info")
	}
	infoOut := new(api.ContainerExecInfoOutput)
	if err := info.Unmarshal(infoOut); err != nil {
		return nil, errors.Wrap(err, "unmarshal exec info")
	}

	qs := jsonutils.Marshal(opt).QueryString()
	urlLoc := fmt.Sprintf("%s/pods/%s/containers/%s/log?%s", infoOut.HostUri, infoOut.PodId, infoOut.ContainerId, qs)

	headers := mcclient.GetTokenHeaders(s.GetToken())
	req := stream.NewRequest(httputils.GetTimeoutClient(1*time.Hour), nil, headers)
	reader, err := req.Stream(context.Background(), "GET", urlLoc)
	if err != nil {
		return nil, errors.Wrap(err, "stream request")
	}
	return reader, nil
}

func (man ContainerManager) LogToWriter(s *mcclient.ClientSession, id string, opt *api.PodLogOptions, out io.Writer) error {
	reader, err := man.Log(s, id, opt)
	if err != nil {
		return errors.Wrap(err, "get container log")
	}
	defer reader.Close()

	r := bufio.NewReader(reader)
	for {
		bytes, err := r.ReadBytes('\n')
		if _, err := out.Write(bytes); err != nil {
			return errors.Wrap(err, "write container log to stdout")
		}
		if err != nil {
			if err != io.EOF {
				return errors.Wrap(err, "read container log")
			}
			return nil
		}
	}
	return nil
}

var (
	Containers ContainerManager
)

func init() {
	Containers = ContainerManager{
		modules.NewComputeManager("container", "containers",
			[]string{"ID", "Name", "Guest_ID", "Status", "Started_At", "Last_Finished_At", "Restart_Count", "Spec"},
			[]string{}),
	}
	modules.RegisterCompute(&Containers)
}
