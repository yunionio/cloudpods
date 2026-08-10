package storageman

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type snapshotImageDriverMock struct {
	backing map[string]string
	errPath string
}

func (m *snapshotImageDriverMock) GetBackingFile(imgPath string) (string, error) {
	if imgPath == m.errPath {
		return "", errors.New("mock probe error")
	}
	return m.backing[filepath.Clean(imgPath)], nil
}

func makeSnapshotFiles(t *testing.T, paths ...string) {
	t.Helper()
	for _, p := range paths {
		if err := os.WriteFile(p, []byte("snapshot"), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoadLocalSnapshotGraph(t *testing.T) {
	dir := t.TempDir()
	disk := filepath.Join(dir, "disk")
	s1 := filepath.Join(dir, "snap_s1")
	s2 := filepath.Join(dir, "snap_s2")
	s3 := filepath.Join(dir, "snap_s3")
	orphan := filepath.Join(dir, "snap_orphan")
	makeSnapshotFiles(t, disk, s1, s2, s3, orphan)
	driver := &snapshotImageDriverMock{backing: map[string]string{
		disk: s3, s3: s2, s2: s1, s1: "", orphan: "",
	}}
	graph, err := loadLocalSnapshotGraph(dir, disk, []string{"snap_s1", "snap_s2", "snap_s3", "snap_orphan"}, driver)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := graph.parents[disk], s3; got != want {
		t.Fatalf("disk parent: got %q, want %q", got, want)
	}
	if len(graph.chains) != 2 || len(graph.chains[0]) != 4 || graph.chains[0][0] != disk {
		t.Fatalf("unexpected chains: %#v", graph.chains)
	}
	if graph.chains[1][0] != orphan {
		t.Fatalf("unexpected orphan chain: %#v", graph.chains)
	}
}

func TestLoadLocalSnapshotGraphErrors(t *testing.T) {
	dir := t.TempDir()
	disk := filepath.Join(dir, "disk")
	s1 := filepath.Join(dir, "snap_s1")
	makeSnapshotFiles(t, disk, s1)
	tests := []struct {
		name string
		back map[string]string
		err  string
	}{
		{"missing backing", map[string]string{disk: filepath.Join(dir, "missing")}, "missing"},
		{"cycle", map[string]string{disk: s1, s1: disk}, "cycle"},
		{"driver error", map[string]string{disk: ""}, "mock probe error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &snapshotImageDriverMock{backing: tt.back}
			if tt.name == "driver error" {
				mock.errPath = disk
			}
			_, err := loadLocalSnapshotGraph(dir, disk, []string{"snap_s1"}, mock)
			if err == nil || !strings.Contains(err.Error(), tt.err) {
				t.Fatalf("expected %q error, got %v", tt.err, err)
			}
		})
	}
}

func TestResolveLocalSnapshotDeletePlanWithImageDriver(t *testing.T) {
	dir := t.TempDir()
	disk := filepath.Join(dir, "disk")
	base := filepath.Join(dir, "disk_snap_base")
	target := filepath.Join(dir, "snap_target")
	child := filepath.Join(dir, "snap_child")
	makeSnapshotFiles(t, disk, base, target, child)

	plan, err := ResolveLocalSnapshotDeletePlan(dir, "snap_target", []string{"disk_snap_base", "snap_target", "snap_child"}, disk,
		&snapshotImageDriverMock{backing: map[string]string{disk: child, child: target, target: base, base: ""}})
	if err != nil || plan.Action != LocalSnapshotCommit || plan.Parent != base || len(plan.Children) != 1 || plan.Children[0] != child {
		t.Fatalf("unexpected commit plan: %#v, %v", plan, err)
	}

	if err := os.Remove(child); err != nil {
		t.Fatal(err)
	}
	plan, err = ResolveLocalSnapshotDeletePlan(dir, "snap_target", []string{"disk_snap_base", "snap_target"}, disk,
		&snapshotImageDriverMock{backing: map[string]string{disk: base, target: base, base: ""}})
	if err != nil || plan.Action != LocalSnapshotRemove {
		t.Fatalf("unexpected remove plan: %#v, %v", plan, err)
	}

	missing, err := ResolveLocalSnapshotDeletePlan(dir, "does-not-exist", []string{"disk_snap_base"}, disk,
		&snapshotImageDriverMock{backing: map[string]string{disk: base, base: ""}})
	if err != nil || missing.Action != LocalSnapshotRemove {
		t.Fatalf("unexpected missing-target plan: %#v, %v", missing, err)
	}
}

func TestResolveLocalSnapshotDeletePlanMultipleChains(t *testing.T) {
	dir := t.TempDir()
	disk := filepath.Join(dir, "disk")
	base := filepath.Join(dir, "disk_snap_base")
	diskSnapshot := filepath.Join(dir, "snap_disk")
	target := filepath.Join(dir, "snap_target")
	child1 := filepath.Join(dir, "snap_child1")
	child2 := filepath.Join(dir, "snap_child2")
	makeSnapshotFiles(t, disk, base, diskSnapshot, target, child1, child2)

	plan, err := ResolveLocalSnapshotDeletePlan(dir, "snap_target",
		[]string{"disk_snap_base", "snap_disk", "snap_target", "snap_child1", "snap_child2"}, disk,
		&snapshotImageDriverMock{backing: map[string]string{
			disk: diskSnapshot, diskSnapshot: base, base: "",
			target: base, child1: target, child2: target,
		}})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Action != LocalSnapshotRebase || plan.Parent != base || len(plan.Children) != 2 {
		t.Fatalf("unexpected disconnected multi-child plan: %#v", plan)
	}
	children := map[string]bool{}
	for _, child := range plan.Children {
		children[child] = true
	}
	if !children[child1] || !children[child2] {
		t.Fatalf("missing children in plan: %#v", plan)
	}
}

func TestResolveLocalSnapshotDeletePlanConvertsRootedNonDiskChain(t *testing.T) {
	dir := t.TempDir()
	disk := filepath.Join(dir, "disk")
	diskSnapshot := filepath.Join(dir, "snap_disk")
	target := filepath.Join(dir, "snap_target")
	child := filepath.Join(dir, "snap_child")
	makeSnapshotFiles(t, disk, diskSnapshot, target, child)

	plan, err := ResolveLocalSnapshotDeletePlan(dir, "snap_target", []string{"snap_disk", "snap_target", "snap_child"}, disk,
		&snapshotImageDriverMock{backing: map[string]string{
			disk: diskSnapshot, diskSnapshot: "", target: "", child: target,
		}})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Action != LocalSnapshotConvert || len(plan.Children) != 1 || plan.Children[0] != child {
		t.Fatalf("unexpected convert plan: %#v", plan)
	}
}

func TestSnapshotBasePath(t *testing.T) {
	dir := "/storage/snapshots/disk-id_snapshots"
	disk := "/storage/disks/disk-id"
	base := filepath.Join(dir, "disk-id_snap_base")

	if got := snapshotBasePath(dir, disk, base); got != base {
		t.Fatalf("expected base %q, got %q", base, got)
	}
	if got := snapshotBasePath(dir, disk, filepath.Join(dir, "other_snap_base")); got != "" {
		t.Fatalf("must not accept another disk's base: %q", got)
	}
}

func TestPrefixSnapshotIds(t *testing.T) {
	ids := prefixSnapshotIds([]string{"s1", "snap_base", "disk_snap_base"})
	want := []string{"snap_s1", "snap_snap_base", "disk_snap_base"}
	if len(ids) != len(want) {
		t.Fatalf("expected %v, got %v", want, ids)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, ids)
		}
	}
}

func TestResolveLocalSnapshotDeleteEdges(t *testing.T) {
	dir := "/storage/snapshots/disk_snap"
	target := filepath.Join(dir, "s2")
	parent := filepath.Join(dir, "s1")
	child := filepath.Join(dir, "s3")
	base := filepath.Join(dir, "disk_snap_base")

	plan := resolveLocalSnapshotDeleteEdges(target, parent, base, []string{child}, false)
	if plan.Action != LocalSnapshotRebase || plan.Parent != parent || len(plan.Children) != 1 || plan.Children[0] != child {
		t.Fatalf("unexpected disconnected-chain plan: %#v", plan)
	}

	plan = resolveLocalSnapshotDeleteEdges(target, base, base, []string{child}, true)
	if plan.Action != LocalSnapshotCommit || plan.Base != base {
		t.Fatalf("expected base commit, got %#v", plan)
	}

	otherBase := filepath.Join(dir, "other-disk_snap_base")
	plan = resolveLocalSnapshotDeleteEdges(target, otherBase, base, []string{child}, true)
	if plan.Action == LocalSnapshotCommit {
		t.Fatalf("must not commit into another disk's base: %#v", plan)
	}

	plan = resolveLocalSnapshotDeleteEdges(target, "/storage/imagecache_image", base, []string{child}, true)
	if plan.Action != LocalSnapshotPromote {
		t.Fatalf("expected image-cache promotion, got %#v", plan)
	}

	plan = resolveLocalSnapshotDeleteEdges(target, "", base, []string{child}, false)
	if plan.Action != LocalSnapshotConvert {
		t.Fatalf("expected convert without parent, got %#v", plan)
	}

	plan = resolveLocalSnapshotDeleteEdges(target, parent, base, []string{"child-1", "child-2"}, true)
	if plan.Action != LocalSnapshotRebase || len(plan.Children) != 2 {
		t.Fatalf("expected multi-child rebase, got %#v", plan)
	}
}
