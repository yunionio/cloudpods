// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package metadata

import (
	"testing"
)

func TestRewriteInfluxLineTenant(t *testing.T) {
	resolve := func(vmId string) (string, bool) {
		if vmId == "vm-1" {
			return "tenant-correct", true
		}
		return "", false
	}
	line := `agent_cpu,vm_id=vm-1,tenant_id=wrong u=1i`
	out, ch, err := rewriteInfluxLineTenant(line, resolve)
	if err != nil {
		t.Fatal(err)
	}
	if !ch {
		t.Fatal("expected change")
	}
	want := `agent_cpu,vm_id=vm-1,tenant_id=tenant-correct u=1i`
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

func TestRewriteInfluxLineTenantAddMissing(t *testing.T) {
	resolve := func(vmId string) (string, bool) {
		if vmId == "vm-1" {
			return "t1", true
		}
		return "", false
	}
	line := `agent_mem,vm_id=vm-1 used=1i`
	out, ch, err := rewriteInfluxLineTenant(line, resolve)
	if err != nil {
		t.Fatal(err)
	}
	if !ch {
		t.Fatal("expected change")
	}
	want := `agent_mem,vm_id=vm-1,tenant_id=t1 used=1i`
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}
