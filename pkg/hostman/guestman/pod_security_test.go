package guestman

import (
	"reflect"
	"testing"
)

func Test_parseGroupEntLine(t *testing.T) {
	gid, err := parseGroupEntLine("video:x:44:")
	if err != nil {
		t.Fatalf("parseGroupEntLine: %v", err)
	}
	if gid != 44 {
		t.Fatalf("gid = %d, want 44", gid)
	}
}

func Test_resolveSupplementalGroups(t *testing.T) {
	out, err := resolveSupplementalGroups([]int64{44, 100}, []string{})
	if err != nil {
		t.Fatalf("resolveSupplementalGroups: %v", err)
	}
	want := []int64{44, 100}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("got %v, want %v", out, want)
	}

	out, err = resolveSupplementalGroups([]int64{44}, nil)
	if err != nil {
		t.Fatalf("resolveSupplementalGroups single gid: %v", err)
	}
	want = []int64{44}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("single gid got %v, want %v", out, want)
	}
}
