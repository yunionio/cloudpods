//go:build linux
// +build linux

package snapshot_service

import (
	"context"
	"net"
	"os"
	"path"
	"strings"

	snapshotsapi "github.com/containerd/containerd/api/services/snapshots/v1"
	"github.com/containerd/containerd/contrib/snapshotservice"
	"github.com/containerd/containerd/mount"
	"github.com/containerd/containerd/snapshots"
	"github.com/containerd/containerd/snapshots/overlay"
	"google.golang.org/grpc"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

func StartService(guestMan IGuestManager, root string) error {
	sn, err := NewSnapshotter(guestMan, root)
	if err != nil {
		return errors.Wrap(err, "NewSnapshotter")
	}
	svc := snapshotservice.FromSnapshotter(sn)
	rpc := grpc.NewServer()
	snapshotsapi.RegisterSnapshotsServer(rpc, svc)
	socksPath := path.Join(root, SocksFileName)
	if err := os.RemoveAll(socksPath); err != nil {
		return errors.Wrapf(err, "RemoveAll %s", socksPath)
	}
	listener, err := net.Listen("unix", socksPath)
	if err != nil {
		return errors.Wrapf(err, "Listen %s", socksPath)
	}
	return rpc.Serve(listener)
}

func NewSnapshotter(guestMan IGuestManager, root string, opts ...overlay.Opt) (snapshots.Snapshotter, error) {
	sn, err := overlay.NewSnapshotter(root, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "NewSnapshotter")
	}
	return &overlayRootFsUpperSnapshotter{
		Snapshotter: sn,
		guestMan:    guestMan,
	}, nil
}

type overlayRootFsUpperSnapshotter struct {
	snapshots.Snapshotter
	guestMan IGuestManager
}

func (s *overlayRootFsUpperSnapshotter) Mounts(ctx context.Context, key string) ([]mount.Mount, error) {
	info, err := s.Snapshotter.Stat(ctx, key)
	if err != nil {
		return nil, errors.Wrapf(err, "Stat with %s", key)
	}
	infoJson := jsonutils.Marshal(info)
	log.Infof("mount request info: %s", infoJson.String())
	mounts, err := s.Snapshotter.Mounts(ctx, key)
	if err != nil {
		return nil, errors.Wrapf(err, "Mounts with %s", key)
	}
	mounts, err = s.changeUpper(ctx, key, mounts)
	if err != nil {
		return nil, errors.Wrapf(err, "Mounts.changeUpper with %s", key)
	}
	return mounts, nil
}

func (s *overlayRootFsUpperSnapshotter) Prepare(ctx context.Context, key string, parent string, opts ...snapshots.Opt) ([]mount.Mount, error) {
	mounts, err := s.Snapshotter.Prepare(ctx, key, parent, opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "Prepare with %s", key)
	}
	infoJson := jsonutils.Marshal(mounts)
	log.Debugf("prepare request key: %s, parent: %s, mounts: %s", key, parent, infoJson.String())
	mounts, err = s.changeUpper(ctx, key, mounts)
	if err != nil {
		return nil, errors.Wrapf(err, "Prepare.changeUpper with %s", key)
	}
	return mounts, nil
}

func (s *overlayRootFsUpperSnapshotter) View(ctx context.Context, key string, parent string, opts ...snapshots.Opt) ([]mount.Mount, error) {
	mounts, err := s.Snapshotter.View(ctx, key, parent, opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "View with %s", key)
	}
	infoJson := jsonutils.Marshal(mounts)
	log.Debugf("view request key: %s, parent: %s, info: %s", key, parent, infoJson.String())
	mounts, err = s.changeUpper(ctx, key, mounts)
	if err != nil {
		return nil, errors.Wrapf(err, "View.changeUpper with %s", key)
	}
	return mounts, nil
}

func (s *overlayRootFsUpperSnapshotter) changeUpper(ctx context.Context, key string, mounts []mount.Mount) ([]mount.Mount, error) {
	if len(mounts) != 1 || mounts[0].Type != "overlay" {
		return mounts, nil
	}
	info, err := s.Snapshotter.Stat(ctx, key)
	if err != nil {
		return nil, errors.Wrapf(err, "Stat with %s", key)
	}
	log.Debugf("change upper key: %s, info: %s", key, jsonutils.Marshal(info))
	serverId, ok := info.Labels[LabelServerId]
	if !ok || serverId == "" {
		return mounts, nil
	}
	containerId, ok := info.Labels[LabelContainerId]
	if !ok || containerId == "" {
		return mounts, nil
	}
	ctrMan, err := s.guestMan.GetContainerManager(serverId)
	if err != nil {
		return mounts, errors.Wrapf(err, "GetContainerManager with %s", serverId)
	}
	rootFsPath, err := ctrMan.GetRootFsMountPath(containerId)
	if err != nil {
		return mounts, errors.Wrapf(err, "GetRootFsMountPath with %s, %s", serverId, containerId)
	}
	log.Debugf("changeUpper: rootFsPath: %s , container: %s", rootFsPath, containerId)
	upperPath := path.Join(rootFsPath, "upper")
	workPath := path.Join(rootFsPath, "work")
	for _, dir := range []string{upperPath, workPath} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return mounts, errors.Wrapf(err, "MkdirAll %s", dir)
		}
	}

	upperDirKey := "upperdir="
	workDirKey := "workdir="
	for i, o := range mounts[0].Options {
		if strings.HasPrefix(o, upperDirKey) {
			mounts[0].Options[i] = upperDirKey + upperPath
		}
		if strings.HasPrefix(o, workDirKey) {
			mounts[0].Options[i] = workDirKey + workPath
		}
	}
	return mounts, nil
}
