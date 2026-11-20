package models

import (
	"testing"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestServerHostMappedPort(t *testing.T) {
	hostPort := 23456
	dpPort := 23457
	server := &computeapi.ServerDetails{}
	server.Nics = []computeapi.GuestnetworkShortDesc{
		{
			IpAddr: "192.168.100.10",
			PortMappings: computeapi.GuestPortMappings{
				{Port: 8000, HostPort: &hostPort},
				{Port: api.LLM_DIST_DP_RPC_PORT, HostPort: &dpPort},
				{Port: api.LLM_DIST_MASTER_PORT},
			},
		},
	}
	if got, ok := ServerHostMappedPort(server, 8000); !ok || got != 23456 {
		t.Fatalf("port 8000 => %d,%v", got, ok)
	}
	if got, ok := ServerHostMappedPort(server, api.LLM_DIST_DP_RPC_PORT); !ok || got != dpPort {
		t.Fatalf("dp rpc port => %d,%v", got, ok)
	}
	if _, ok := ServerHostMappedPort(server, api.LLM_DIST_MASTER_PORT); ok {
		t.Fatal("unmapped port must not resolve")
	}
	if _, ok := ServerHostMappedPort(nil, 8000); ok {
		t.Fatal("nil server must not resolve")
	}
}

func TestIsDistInternalPort(t *testing.T) {
	if !IsDistInternalPort(2, api.LLM_DIST_DP_RPC_PORT+2*api.LLM_DIST_PORT_STRIDE) {
		t.Fatal("group dp rpc port must be internal")
	}
	if !IsDistInternalPort(0, api.LLM_DIST_MASTER_PORT) {
		t.Fatal("group 0 master port must be internal")
	}
	if IsDistInternalPort(0, 8000) {
		t.Fatal("service port must not be internal")
	}
}

func TestIsHostLocalLLM(t *testing.T) {
	if !IsHostLocalLLM(&SLLM{SLLMBase: SLLMBase{NetworkType: string(computeapi.NETWORK_TYPE_HOSTLOCAL)}}) {
		t.Fatal("expected hostlocal")
	}
	if IsHostLocalLLM(&SLLM{SLLMBase: SLLMBase{NetworkType: string(computeapi.NETWORK_TYPE_GUEST)}}) {
		t.Fatal("guest is not hostlocal")
	}
	if IsHostLocalLLM(nil) {
		t.Fatal("nil is not hostlocal")
	}
}
