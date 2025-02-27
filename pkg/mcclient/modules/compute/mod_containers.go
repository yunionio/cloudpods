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
	"path"
	"runtime/debug"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
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

type ContainerManager struct {
	modulebase.ResourceManager
}

func (man ContainerManager) SetupTTY(in io.Reader, out io.Writer, errOut io.Writer, raw bool) term.TTY {
	t := term.TTY{
		Out: out,
	}
	if in == nil {
		t.In = nil
		t.Raw = false
		return t
	}
	return term.TTY{
		In:     in,
		Out:    out,
		Raw:    raw,
		TryDev: false,
		Parent: nil,
	}
}

type ContainerExecInput struct {
	Command []string
	Tty     bool
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

func (man ContainerManager) Exec(s *mcclient.ClientSession, id string, opt *ContainerExecInput) error {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Panic catched: %s", r)
			debug.PrintStack()
		}
	}()
	return man.exec(s, id, opt)
}

func (man ContainerManager) exec(s *mcclient.ClientSession, id string, opt *ContainerExecInput) error {
	info, err := man.GetSpecific(s, id, "exec-info", nil)
	if err != nil {
		return errors.Wrap(err, "get exec info")
	}
	infoOut := new(api.ContainerExecInfoOutput)
	info.Unmarshal(infoOut)
	apiInput := &api.ContainerExecInput{
		Command: opt.Command,
		Tty:     opt.Tty,
		SetIO:   true,
		Stdin:   opt.Stdin != nil,
		Stdout:  opt.Stdout != nil,
	}
	urlLoc := fmt.Sprintf("%s/pods/%s/containers/%s/exec?%s", infoOut.HostUri, infoOut.PodId, infoOut.ContainerId, jsonutils.Marshal(apiInput).QueryString())
	url, err := url.Parse(urlLoc)
	if err != nil {
		return errors.Wrapf(err, "parse url: %s", urlLoc)
	}
	exec, err := remotecommand.NewSPDYExecutor("POST", url)
	if err != nil {
		return errors.Wrap(err, "NewSPDYExecutor")
	}
	headers := mcclient.GetTokenHeaders(s.GetToken())

	t := man.SetupTTY(opt.Stdin, opt.Stdout, opt.Stderr, true)
	sizeQueue := t.MonitorSize(t.GetSize())
	fn := func() error {
		return exec.Stream(remotecommand.StreamOptions{
			Stdin:             opt.Stdin,
			Stdout:            opt.Stdout,
			Stderr:            opt.Stderr,
			Tty:               opt.Tty,
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

func (man ContainerManager) EnsureDir(s *mcclient.ClientSession, ctrId string, dirName string) error {
	opt := &ContainerExecInput{
		Command: []string{"mkdir", "-p", dirName},
		Tty:     false,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}
	return man.Exec(s, ctrId, opt)
}

func (man ContainerManager) copyTo(s *mcclient.ClientSession, ctrId string, destPath string, in io.Reader, ctrCmd []string) error {
	destDir := path.Dir(destPath)
	if err := man.EnsureDir(s, ctrId, destDir); err != nil {
		return errors.Wrapf(err, "ensure dir %s", destDir)
	}

	reader, writer := io.Pipe()
	go func() {
		defer writer.Close()
		written, err := io.Copy(writer, in)
		if err != nil {
			log.Errorf("copy reader to writer, written %d, error: %v", written, err)
		}
	}()

	opt := &ContainerExecInput{
		Command: ctrCmd,
		Tty:     false,
		Stdin:   reader,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}
	return man.Exec(s, ctrId, opt)
}

func (man ContainerManager) CopyTo(s *mcclient.ClientSession, ctrId string, destPath string, in io.Reader) error {
	ctrCmd := []string{"sh", "-c", fmt.Sprintf("cat - > %s", destPath)}
	return man.copyTo(s, ctrId, destPath, in, ctrCmd)
}

func (man ContainerManager) CopyTarTo(s *mcclient.ClientSession, ctrId string, destDir string, in io.Reader, noSamePermissions bool) error {
	ctrCmd := []string{"tar", "-xmf", "-"}
	if noSamePermissions {
		ctrCmd = []string{"tar", "--no-same-permissions", "--no-same-owner", "-xmf", "-"}
	}
	ctrCmd = append(ctrCmd, "-C", destDir)
	return man.copyTo(s, ctrId, destDir, in, ctrCmd)
}

func (man ContainerManager) CheckDestinationIsDir(s *mcclient.ClientSession, ctrId string, destPath string) error {
	opt := &ContainerExecInput{
		Command: []string{"test", "-d", destPath},
		Tty:     false,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}
	return man.Exec(s, ctrId, opt)
}

func (man ContainerManager) copyFrom(s *mcclient.ClientSession, ctrId string, out io.Writer, cmd []string) error {
	reader, outStream := io.Pipe()
	opts := &ContainerExecInput{
		Command: cmd,
		Tty:     false,
		Stdin:   nil,
		Stdout:  outStream,
		Stderr:  os.Stderr,
	}
	go func() {
		defer outStream.Close()
		if err := man.Exec(s, ctrId, opts); err != nil {
			log.Errorf("compute.Containers.Exec: %v", err)
		}
	}()
	written, err := io.Copy(out, reader)
	if err != nil {
		return errors.Wrapf(err, "copy from reader written: %d", written)
	}
	return nil
}

func (man ContainerManager) CopyFrom(s *mcclient.ClientSession, ctrId string, ctrFile string, out io.Writer) error {
	return man.copyFrom(s, ctrId, out, []string{"cat", ctrFile})
}

func (man ContainerManager) CopyTarFrom(s *mcclient.ClientSession, ctrId string, ctrDir []string, out io.Writer) error {
	cmd := []string{"tar", "cf", "-"}
	cmd = append(cmd, ctrDir...)
	return man.copyFrom(s, ctrId, out, cmd)
}
