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
	"archive/zip"
	"context"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

// rdpDriveRootPrefix is the host directory Guacamole maps as the Cloudpods drive.
// Overridable in tests.
var rdpDriveRootPrefix = "/opt/cloudpods"

// guacd runs as uid/gid 1000 (see Dockerfile.guacd). Files created by webconsole
// (often root) must be owned by guacd, otherwise Windows RDP drive redirection
// maps them to S-1-1-0 (Everyone) and guests cannot delete/modify them.
var (
	rdpDriveFSUid = 1000
	rdpDriveFSGid = 1000
)

type rdpDriveEntry struct {
	ownerId string
}

var (
	rdpDriveMux      = sync.Mutex{}
	rdpDriveSessions = make(map[string]*rdpDriveEntry)
)

func addRdpDriveSession(sId, ownerId string) {
	rdpDriveMux.Lock()
	defer rdpDriveMux.Unlock()
	rdpDriveSessions[sId] = &rdpDriveEntry{ownerId: ownerId}
}

func delRdpDriveSession(sId string) {
	rdpDriveMux.Lock()
	defer rdpDriveMux.Unlock()
	delete(rdpDriveSessions, sId)
}

func getRdpDriveOwner(sId string, userCred mcclient.TokenCredential) (string, error) {
	rdpDriveMux.Lock()
	defer rdpDriveMux.Unlock()
	entry, ok := rdpDriveSessions[sId]
	if !ok {
		return "", errors.Wrapf(cloudprovider.ErrNotFound, "%s", sId)
	}
	if userCred == nil || entry.ownerId != userCred.GetUserId() {
		return "", httperrors.NewForbiddenError("rdp session %s does not belong to current user", sId)
	}
	return entry.ownerId, nil
}

func rdpDriveRoot(ownerId string) string {
	return filepath.Join(rdpDriveRootPrefix, ownerId)
}

func fixRdpDrivePerms(path string, dir bool) {
	mode := os.FileMode(0o666)
	if dir {
		mode = 0o777
	}
	_ = os.Chmod(path, mode)
	_ = os.Chown(path, rdpDriveFSUid, rdpDriveFSGid)
}

// fixRdpDriveTreePerms chowns/chmods path and every ancestor under rootAbs
// so Windows guests can create/delete via the redirected drive.
func fixRdpDriveTreePerms(rootAbs, path string) {
	rootAbs = filepath.Clean(rootAbs)
	cur := filepath.Clean(path)
	for {
		info, err := os.Stat(cur)
		if err == nil {
			fixRdpDrivePerms(cur, info.IsDir())
		}
		if cur == rootAbs {
			break
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
}

// resolveRdpDrivePath joins relPath under the user's drive root and ensures the
// result stays inside the jail (no ".." escape).
func resolveRdpDrivePath(ownerId, relPath string) (string, error) {
	root := rdpDriveRoot(ownerId)
	// 0777 so guacd (non-root) can read files written by webconsole.
	if err := os.MkdirAll(root, 0o777); err != nil {
		return "", errors.Wrapf(err, "mkdir drive root %s", root)
	}
	fixRdpDrivePerms(root, true)
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", errors.Wrapf(err, "abs root %s", root)
	}
	if relPath == "" {
		relPath = "/"
	}
	// Treat path as POSIX-style relative to drive root.
	clean := path.Clean("/" + strings.TrimPrefix(relPath, "/"))
	rel := strings.TrimPrefix(clean, "/")
	full := rootAbs
	if rel != "" {
		full = filepath.Join(rootAbs, filepath.FromSlash(rel))
	}
	fullAbs, err := filepath.Abs(full)
	if err != nil {
		return "", errors.Wrapf(err, "abs %s", full)
	}
	if fullAbs != rootAbs && !strings.HasPrefix(fullAbs, rootAbs+string(os.PathSeparator)) {
		return "", httperrors.NewBadRequestError("path %s escapes drive root", relPath)
	}
	return fullAbs, nil
}

func logicalRdpPath(ownerId, absPath string) string {
	root := rdpDriveRoot(ownerId)
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "/"
	}
	if absPath == rootAbs {
		return "/"
	}
	rel, err := filepath.Rel(rootAbs, absPath)
	if err != nil {
		return "/"
	}
	return "/" + filepath.ToSlash(rel)
}

func HandleRdpList(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, query, _ := appsrv.FetchEnv(ctx, w, r)
	dir := "/"
	if query != nil && query.Contains("path") {
		dir, _ = query.GetString("path")
	}
	sId := params[SESSION_ID]
	userCred := auth.FetchUserCredential(ctx, nil)

	files, err := func() (Files, error) {
		ownerId, err := getRdpDriveOwner(sId, userCred)
		if err != nil {
			return nil, errors.Wrapf(err, "getRdpDriveOwner")
		}
		absDir, err := resolveRdpDrivePath(ownerId, dir)
		if err != nil {
			return nil, err
		}
		entries, err := os.ReadDir(absDir)
		if err != nil {
			return nil, errors.Wrapf(httperrors.FsErrorNormalize(err), "ReadDir %s", dir)
		}
		ret := Files{}
		for _, e := range entries {
			info, err := e.Info()
			if err != nil {
				continue
			}
			name := e.Name()
			absPath := filepath.Join(absDir, name)
			mode := info.Mode()
			vv := sFileList{
				Name:      name,
				Mode:      mode.String(),
				ModeNum:   mode.Perm(),
				IsRegular: mode.IsRegular(),
				Size:      info.Size(),
				ModTime:   info.ModTime(),
				IsDir:     e.IsDir(),
				Path:      logicalRdpPath(ownerId, absPath),
			}
			if mode.Type() == fs.ModeSymlink {
				if link, err := os.Readlink(absPath); err == nil {
					vv.LinkFile = &sLinkFile{Name: link}
					if stat, err := os.Stat(absPath); err == nil {
						vv.LinkFile.IsDir = stat.IsDir()
						vv.LinkFile.Path = path.Join(dir, link)
						vv.LinkFile.Size = stat.Size()
						vv.LinkFile.Mode = stat.Mode().String()
						vv.LinkFile.ModeNum = stat.Mode().Perm()
						vv.LinkFile.IsRegular = stat.Mode().IsRegular()
					}
				}
			}
			ret = append(ret, vv)
		}
		return ret, nil
	}()
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	sort.Sort(files)
	appsrv.SendJSON(w, jsonutils.Marshal(files))
}

func HandleRdpUpload(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, query, _ := appsrv.FetchEnv(ctx, w, r)
	dir := "/"
	if query != nil && query.Contains("path") {
		dir, _ = query.GetString("path")
	}
	sId := params[SESSION_ID]
	userCred := auth.FetchUserCredential(ctx, nil)

	err := func() error {
		ownerId, err := getRdpDriveOwner(sId, userCred)
		if err != nil {
			return errors.Wrapf(err, "getRdpDriveOwner")
		}
		absDir, err := resolveRdpDrivePath(ownerId, dir)
		if err != nil {
			return err
		}
		rootAbs, err := filepath.Abs(rdpDriveRoot(ownerId))
		if err != nil {
			return errors.Wrapf(err, "abs drive root")
		}
		// Create nested dirs for recursive folder upload (e.g. /foo/bar).
		if err := os.MkdirAll(absDir, 0o777); err != nil {
			return errors.Wrapf(httperrors.FsErrorNormalize(err), "mkdir %s", dir)
		}
		fixRdpDriveTreePerms(rootAbs, absDir)
		st, err := os.Stat(absDir)
		if err != nil {
			return errors.Wrapf(httperrors.FsErrorNormalize(err), "stat %s", dir)
		}
		if !st.IsDir() {
			return httperrors.NewBadRequestError("path %s is not a directory", dir)
		}

		if err := r.ParseMultipartForm(32 << 20); err != nil {
			return errors.Wrapf(err, "ParseMultipartForm")
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			return errors.Wrapf(err, "FormFile")
		}
		defer file.Close()

		destPath := filepath.Join(absDir, filepath.Base(header.Filename))
		// Re-check jail after joining filename.
		if _, err := resolveRdpDrivePath(ownerId, logicalRdpPath(ownerId, destPath)); err != nil {
			return err
		}

		newFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o666)
		if err != nil {
			return errors.Wrapf(httperrors.FsErrorNormalize(err), "create file")
		}

		if _, err := io.Copy(newFile, file); err != nil {
			newFile.Close()
			return errors.Wrapf(httperrors.FsErrorNormalize(err), "Copy")
		}
		// Close before chown so guacd/Windows see final ownership (avoids S-1-1-0 delete errors).
		if err := newFile.Close(); err != nil {
			return errors.Wrapf(httperrors.FsErrorNormalize(err), "close")
		}
		fixRdpDrivePerms(destPath, false)
		fixRdpDriveTreePerms(rootAbs, absDir)
		return nil
	}()
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, jsonutils.Marshal(map[string]string{"status": "success"}))
}

func contentDispositionAttachment(name string) string {
	return "attachment;filename*=utf-8''" + strings.ReplaceAll(url.QueryEscape(name), "+", "%20")
}

// flushWriter periodically flushes the HTTP response so proxies/browsers receive
// zip bytes as they are produced (chunked), instead of buffering the archive.
type flushWriter struct {
	w       http.ResponseWriter
	f       http.Flusher
	written int
}

func (fw *flushWriter) Write(p []byte) (int, error) {
	n, err := fw.w.Write(p)
	fw.written += n
	// Flush ~every 256KiB to balance latency vs syscall overhead.
	if fw.f != nil && fw.written >= 256<<10 {
		fw.f.Flush()
		fw.written = 0
	}
	return n, err
}

// writeRdpDirZip streams a zip archive of dirAbs. Entries are rooted at zipRootName/.
// Does not buffer the whole archive in memory; files are compressed and written as walked.
func writeRdpDirZip(dirAbs, zipRootName string, w io.Writer) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	return filepath.Walk(dirAbs, func(p string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(dirAbs, p)
		if err != nil {
			return err
		}
		if rel == "." {
			// Represent the folder itself as an empty dir entry when it has no children later.
			_, err := zw.Create(zipRootName + "/")
			return err
		}
		name := filepath.ToSlash(filepath.Join(zipRootName, rel))
		if info.IsDir() {
			_, err := zw.Create(name + "/")
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		hdr, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		hdr.Name = name
		hdr.Method = zip.Deflate
		fw, err := zw.CreateHeader(hdr)
		if err != nil {
			return err
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(fw, f)
		f.Close()
		return copyErr
	})
}

func HandleRdpDownload(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, query, _ := appsrv.FetchEnv(ctx, w, r)
	if query == nil || !query.Contains("path") {
		httperrors.GeneralServerError(ctx, w, httperrors.NewMissingParameterError("path"))
		return
	}
	filePath, _ := query.GetString("path")
	sId := params[SESSION_ID]
	userCred := auth.FetchUserCredential(ctx, nil)

	headersSent := false
	err := func() error {
		ownerId, err := getRdpDriveOwner(sId, userCred)
		if err != nil {
			return errors.Wrapf(err, "getRdpDriveOwner")
		}
		absPath, err := resolveRdpDrivePath(ownerId, filePath)
		if err != nil {
			return err
		}
		info, err := os.Stat(absPath)
		if err != nil {
			return errors.Wrapf(httperrors.FsErrorNormalize(err), "stat %s", filePath)
		}
		if info.IsDir() {
			zipRoot := info.Name()
			if zipRoot == "" || zipRoot == "." || zipRoot == "/" {
				zipRoot = "download"
			}
			zipName := zipRoot + ".zip"
			w.Header().Set("Content-Disposition", contentDispositionAttachment(zipName))
			w.Header().Set("Content-Type", "application/zip")
			// Disable nginx/proxy response buffering for progressive zip download.
			w.Header().Set("X-Accel-Buffering", "no")
			headersSent = true
			out := io.Writer(w)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
				out = &flushWriter{w: w, f: f}
			}
			if err := writeRdpDirZip(absPath, zipRoot, out); err != nil {
				return errors.Wrapf(httperrors.FsErrorNormalize(err), "zip dir %s", filePath)
			}
			return nil
		}
		reader, err := os.Open(absPath)
		if err != nil {
			return errors.Wrapf(httperrors.FsErrorNormalize(err), "open file")
		}
		defer reader.Close()

		w.Header().Set("Content-Disposition", contentDispositionAttachment(info.Name()))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("X-Accel-Buffering", "no")
		headersSent = true
		if _, err := io.Copy(w, reader); err != nil {
			return errors.Wrap(httperrors.FsErrorNormalize(err), "Copy")
		}
		return nil
	}()
	if err != nil {
		if headersSent {
			// Body already streaming; cannot rewrite as JSON error.
			log.Errorf("rdp download stream error: %v", err)
			return
		}
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
}
