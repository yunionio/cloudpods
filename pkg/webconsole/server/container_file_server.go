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

package server

import (
	"bytes"
	"context"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	compute_modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	o "yunion.io/x/onecloud/pkg/webconsole/options"
)

const containerIDParam = "<container-id>"

// ls -la long listing: mode, nlink, owner, group, size, date, name.
var lsLongLine = regexp.MustCompile(`^(\S+)\s+\d+\s+\S+\s+\S+\s+(\d+)\s+([A-Z][a-z]{2}\s+\d{1,2}\s+(?:\d{2}:\d{2}|\d{4}))\s+(.*)$`)

func containerClientSession(ctx context.Context, params map[string]string) (*mcclient.ClientSession, string, error) {
	userCred := auth.FetchUserCredential(ctx, nil)
	if userCred == nil || userCred.GetTokenString() == "" {
		return nil, "", httperrors.NewUnauthorizedError("No token found")
	}
	ctrId := ""
	if params != nil {
		ctrId = params[containerIDParam]
	}
	if ctrId == "" {
		return nil, "", httperrors.NewMissingParameterError("container_id")
	}
	log.Infof("containerClientSession: %s", ctrId)
	s := auth.GetSession(ctx, userCred, o.Options.Region)
	// if ctrId is not valid UUID, it should be in the format of "pod_name/container_name"
	// try to resolve it to a valid UUID
	if !regutils.MatchUUIDExact(ctrId) {
		podName, containerName, ok := strings.Cut(ctrId, "/")
		if !ok {
			return nil, "", errors.Wrapf(errors.ErrInvalidFormat, "invalid container_id format: %s", ctrId)
		}
		pod, err := compute_modules.Servers.Get(s, podName, nil)
		if err != nil {
			return nil, "", errors.Wrapf(err, "get pod %s", podName)
		}
		podId, _ := pod.GetString("id")
		params := map[string]string{
			"guest_id": podId,
		}
		container, err := compute_modules.Containers.Get(s, containerName, jsonutils.Marshal(params))
		if err != nil {
			return nil, "", errors.Wrapf(err, "get container %s", containerName)
		}
		ctrId, _ = container.GetString("id")
	} else {
		container, err := compute_modules.Containers.Get(s, ctrId, nil)
		if err != nil {
			return nil, "", errors.Wrapf(err, "get container %s", ctrId)
		}
		ctrId, _ = container.GetString("id")
	}
	if ctrId == "" {
		return nil, "", httperrors.NewMissingParameterError("container_id")
	}
	return s, ctrId, nil
}

func queryPath(query jsonutils.JSONObject) string {
	dir := "/"
	if query != nil && query.Contains("path") {
		dir, _ = query.GetString("path")
	}
	if dir == "" {
		dir = "/"
	}
	return dir
}

func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func containerLsCommand(dir string) []string {
	// -la fills the same name/size/mode/mtime fields as the sftp file list.
	return []string{"sh", "-c", "LC_ALL=C ls -la " + shellSingleQuote(dir)}
}

func HandleContainerList(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, query, _ := appsrv.FetchEnv(ctx, w, r)
	dir := queryPath(query)
	files, err := func() (Files, error) {
		s, ctrId, err := containerClientSession(ctx, params)
		if err != nil {
			return nil, err
		}
		var stdout, stderr bytes.Buffer
		err = compute_modules.Containers.Exec(s, ctrId, &compute_modules.ContainerExecInput{
			Command: containerLsCommand(dir),
			Tty:     false,
			Stdout:  &stdout,
			Stderr:  &stderr,
		})
		if err != nil {
			msg := strings.TrimSpace(stderr.String())
			if msg != "" {
				return nil, errors.Wrapf(err, "ls %s: %s", dir, msg)
			}
			return nil, errors.Wrapf(err, "ls %s", dir)
		}
		return parseContainerLs(dir, stdout.String()), nil
	}()
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	sort.Sort(files)
	appsrv.SendJSON(w, jsonutils.Marshal(files))
}

func HandleContainerDownload(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, query, _ := appsrv.FetchEnv(ctx, w, r)
	if query == nil || !query.Contains("path") {
		httperrors.GeneralServerError(ctx, w, httperrors.NewMissingParameterError("path"))
		return
	}
	filePath, _ := query.GetString("path")
	err := func() error {
		s, ctrId, err := containerClientSession(ctx, params)
		if err != nil {
			return err
		}
		if err := containerTestRegularFile(s, ctrId, filePath); err != nil {
			return err
		}
		w.Header().Add("Content-Disposition", "attachment;filename*=utf-8''"+strings.ReplaceAll(url.QueryEscape(path.Base(filePath)), "+", "%20"))
		w.Header().Add("Content-Type", "application/octet-stream")
		if err := compute_modules.Containers.CopyFrom(s, ctrId, filePath, w); err != nil {
			return errors.Wrapf(err, "copy from %s", filePath)
		}
		return nil
	}()
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
}

func HandleContainerUpload(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, query, _ := appsrv.FetchEnv(ctx, w, r)
	dir := queryPath(query)
	err := func() error {
		s, ctrId, err := containerClientSession(ctx, params)
		if err != nil {
			return err
		}
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			return errors.Wrap(err, "ParseMultipartForm")
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			return errors.Wrapf(err, "FormFile")
		}
		defer file.Close()

		name := path.Base(header.Filename)
		if name == "." || name == "/" || name == "" {
			return httperrors.NewInputParameterError("invalid filename")
		}
		dest := path.Join(dir, name)
		if err := compute_modules.Containers.CopyTo(s, ctrId, dest, file); err != nil {
			return errors.Wrapf(err, "copy to %s", dest)
		}
		return nil
	}()
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, jsonutils.Marshal(map[string]string{"status": "success"}))
}

func containerTestRegularFile(s *mcclient.ClientSession, ctrId, filePath string) error {
	var stderr bytes.Buffer
	err := compute_modules.Containers.Exec(s, ctrId, &compute_modules.ContainerExecInput{
		Command: []string{"sh", "-c", "test -f " + shellSingleQuote(filePath)},
		Tty:     false,
		Stderr:  &stderr,
	})
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return errors.Wrapf(err, "stat %s: %s", filePath, msg)
		}
		return errors.Wrapf(err, "stat %s", filePath)
	}
	return nil
}

func parseContainerLs(dir, output string) Files {
	ret := Files{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(line, "total ") {
			continue
		}
		m := lsLongLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		mode, sizeStr, dateStr, name := m[1], m[2], m[3], m[4]
		var link *sLinkFile
		if strings.HasPrefix(mode, "l") {
			if n, target, ok := strings.Cut(name, " -> "); ok {
				name = n
				linkPath := target
				if !path.IsAbs(target) {
					linkPath = path.Join(dir, target)
				}
				link = &sLinkFile{
					Name: target,
					Path: linkPath,
				}
			}
		}
		if name == "." || name == ".." {
			continue
		}
		size, _ := strconv.ParseInt(sizeStr, 10, 64)
		ret = append(ret, sFileList{
			Name:      name,
			Path:      path.Join(dir, name),
			Size:      size,
			ModTime:   parseLsModTime(dateStr),
			IsDir:     strings.HasPrefix(mode, "d"),
			Mode:      mode,
			ModeNum:   parseLsPerm(mode),
			IsRegular: strings.HasPrefix(mode, "-"),
			LinkFile:  link,
		})
	}
	return ret
}

func parseLsPerm(mode string) fs.FileMode {
	if len(mode) < 10 {
		return 0
	}
	bits := mode[1:10]
	var perm fs.FileMode
	for i, c := range bits {
		if c == '-' || c == 'S' || c == 'T' {
			continue
		}
		perm |= 1 << (8 - uint(i))
	}
	return perm
}

func parseLsModTime(dateStr string) time.Time {
	dateStr = strings.Join(strings.Fields(dateStr), " ")
	if t, err := time.Parse("Jan 2 15:04", dateStr); err == nil {
		now := time.Now()
		t = time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, time.Local)
		if t.After(now.Add(24 * time.Hour)) {
			t = t.AddDate(-1, 0, 0)
		}
		return t
	}
	if t, err := time.ParseInLocation("Jan 2 2006", dateStr, time.Local); err == nil {
		return t
	}
	return time.Time{}
}
