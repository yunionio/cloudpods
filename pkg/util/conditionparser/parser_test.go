package conditionparser

import (
	"testing"

	"go/parser"
	"yunion.io/x/jsonutils"
)

func TestAst(t *testing.T) {
	input := jsonutils.NewDict()
	input.Add(jsonutils.NewString("windows"), "server", "os_type")
	disk := jsonutils.NewDict()
	disk.Add(jsonutils.NewString("ssd"), "medium_type")
	disks := jsonutils.NewArray(disk)
	input.Add(disks, "server", "disks")
	input.Add(disk, "server", "disk.0")

	cases := []struct {
		in   string
		want bool
	}{
		{`server.os_type == "windows"`, true},
		{`server.os_type.startswith("window")`, true},
		{`server.disks[0].medium_type == "ssd"`, true},
		{`server.disks[0].medium_type == "hdd"`, false},
		{`server.os_type == "windows" && server.disks[0].medium_type == "ssd"`, true},
		{`server.contains("os_type")`, true},
		{`server.disks[0].contains("medium_type")`, true},
		{`server.disk[0].contains("medium_type")`, true},
		{`server.disks[0].contains("backend")`, false},
	}

	for _, c := range cases {
		result, err := Eval(c.in, input)
		if err != nil {
			t.Errorf("eval expr %s error %s", c.in, err)
			return
		}

		if result != c.want {
			t.Errorf("expect %v got %v", c.want, result)
			return
		}
	}

}

func TestIsValid(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{`server.os_type == "windows"`, true},
		{`server.os_type ==`, false},
		{`dfadsfsdf ==`, false},
		{`dfadsfsdf +`, false},
	}
	for _, c := range cases {
		got := IsValid(c.in)
		if got != c.want {
			t.Errorf("%s is valid %v got %v", c.in, c.want, got)
		}
	}
}

func TestEval2(t *testing.T) {
	inputStr := `{"server": {"disable_delete":false,
"disk.0":{"backend":"local","format":"qcow2","image_id":"4b9fa54c-858c-4c2b-8719-27aee120b3cb","image_properties":{"os_arch":"x86_64","os_distribution":"CentOS","os_type":"Linux","os_version":"7.5.1804"},"medium":"hybrid","size":40960},
"hypervisor":"kvm","keypair_id":"None","name":"testsched","os_type":"Linux","owner_tenant_id":"5d65667d112e47249ae66dbd7bc07030",
"sched_tag.0":"ssd","secgrp_id":"default","vcpu_count":1,"vmem_size":1024}}`
	input, err := jsonutils.ParseString(inputStr)
	if err != nil {
		t.Errorf("fail to parse server json")
		return
	}

	cases := []struct {
		in   string
		want bool
	}{
		{`server.os_type == "Linux"`, true},
		{`server["os_type"] == "Linux"`, true},
		{`server.vmem_size > 2048`, false},
		{`server.hypervisor.in("kvm", "aliyun")`, true},
		{`server.disable_delete`, false},
		{`server.disk[0].backend == "local"`, true},
		{`server.sched_tag[0] != "ssd"`, false},
		{`server.sched_tag[0] == "ssd"`, true},
		// {`server.os_type == "windows" && server.disks[0].medium_type == "ssd"`, true},
	}

	for _, c := range cases {
		result, err := Eval(c.in, input)
		if err != nil {
			t.Errorf("eval expr %s error %s", c.in, err)
			return
		}

		if result != c.want {
			t.Errorf("expect %v got %v", c.want, result)
			return
		}
	}
}

func TestEval3(t *testing.T) {
	exprStr := `server.disk.backend == "local"`
	expr, err := parser.ParseExpr(exprStr)
	if err != nil {
		t.Errorf("parse exprStr fail %s %s", exprStr, err)
		return
	}

	t.Logf("%s", jsonutils.Marshal(expr))

	inputStr := `{"server":{
	"disk.0":{"backend": "local", "medium": "hdd"},
	"disk.1":{"backend": "local", "medium": "hdd"},
	"disk.2":{"backend": "rbd", "medium": "ssd"},
	"disk.3":{"backend": "rbd", "medium": "ssd"}}
}`
	input, err := jsonutils.ParseString(inputStr)
	if err != nil {
		t.Errorf("fail to parse server json %s", err)
		return
	}

	cases := []struct {
		in   string
		want bool
	}{
		{`server.disk.backend == "local"`, true},
		{`server.disk.medium == "ssd"`, true},
		{`server.disk.medium == "hdd"`, true},
		{`server.disk.medium == "hybrid"`, false},
		{`server.disk.medium.contains("ssd")`, true},
		{`server.disk[0]["medium"] == "hdd"`, true},
		{`server["disk"][0]["medium"] == "hdd"`, true},
	}

	for _, c := range cases {
		result, err := Eval(c.in, input)
		if err != nil {
			t.Errorf("eval expr %s error %s", c.in, err)
			return
		}

		if result != c.want {
			t.Errorf("expect %v got %v", c.want, result)
			return
		}
	}
}
