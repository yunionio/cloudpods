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

package models

import (
	"encoding/json"
	"testing"

	compute_models "yunion.io/x/onecloud/pkg/compute/models"
)

func TestGenGobetweenConfigs_UDPHealthcheckUsesProbeSchema(t *testing.T) {
	server := genUDPHealthcheckTestServer(t, "PING", "PONG", "on")
	healthcheck := server["healthcheck"].(map[string]interface{})
	if got := healthcheck["kind"]; got != "probe" {
		t.Fatalf("healthcheck.kind = %v, want probe", got)
	}
	if got := healthcheck["probe_protocol"]; got != "udp" {
		t.Fatalf("healthcheck.probe_protocol = %v, want udp", got)
	}
	if got := healthcheck["probe_strategy"]; got != "starts_with" {
		t.Fatalf("healthcheck.probe_strategy = %v, want starts_with", got)
	}
	if got := healthcheck["probe_send"]; got != "PING" {
		t.Fatalf("healthcheck.probe_send = %v, want PING", got)
	}
	if got := healthcheck["probe_recv"]; got != "PONG" {
		t.Fatalf("healthcheck.probe_recv = %v, want PONG", got)
	}
	if got := healthcheck["interval"]; got != "10s" {
		t.Fatalf("healthcheck.interval = %v, want 10s", got)
	}
	if got := healthcheck["timeout"]; got != "5s" {
		t.Fatalf("healthcheck.timeout = %v, want 5s", got)
	}
	if got := healthcheck["passes"]; got != float64(3) {
		t.Fatalf("healthcheck.passes = %v, want 3", got)
	}
	if got := healthcheck["fails"]; got != float64(2) {
		t.Fatalf("healthcheck.fails = %v, want 2", got)
	}
	if _, ok := healthcheck["Receive"]; ok {
		t.Fatal("healthcheck must not contain Receive")
	}
	if _, ok := healthcheck["Send"]; ok {
		t.Fatal("healthcheck must not contain Send")
	}
}

func TestGenGobetweenConfigs_UDPHealthcheckSkipsIncompleteProbe(t *testing.T) {
	tests := []struct {
		name     string
		request  string
		expected string
	}{
		{name: "empty request", request: "", expected: "PONG"},
		{name: "empty expected response", request: "PING", expected: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := genUDPHealthcheckTestServer(t, test.request, test.expected, "on")
			if healthcheck, ok := server["healthcheck"]; ok && healthcheck != nil {
				t.Fatalf("incomplete UDP healthcheck must not be generated: %v", healthcheck)
			}
		})
	}
}

func TestGenGobetweenConfigs_UDPHealthcheckOffOmitsConfig(t *testing.T) {
	server := genUDPHealthcheckTestServer(t, "PING", "PONG", "off")
	if healthcheck, ok := server["healthcheck"]; ok && healthcheck != nil {
		t.Fatalf("disabled UDP healthcheck must not be generated: %v", healthcheck)
	}
}

func genUDPHealthcheckTestServer(t *testing.T, request, expected, healthCheck string) map[string]interface{} {
	t.Helper()
	corpus, lb := newUDPHealthcheckTestCorpus(request, expected)
	lb.Listeners["listener-id"].HealthCheck = healthCheck
	opts := &GenGobetweenConfigOptions{
		LoadbalancersEnabled: []*Loadbalancer{lb},
	}

	if err := corpus.GenGobetweenConfigs(t.TempDir(), opts); err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(opts.Config)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}

	return config["servers"].(map[string]interface{})["listener-id"].(map[string]interface{})
}

func newUDPHealthcheckTestCorpus(request, expected string) (*LoadbalancerCorpus, *Loadbalancer) {
	lb := &Loadbalancer{
		SLoadbalancer: &compute_models.SLoadbalancer{},
		Listeners:     LoadbalancerListeners{},
		BackendGroups: LoadbalancerBackendGroups{},
	}
	lb.Id = "loadbalancer-id"
	lb.Address = "127.0.0.1"

	listener := &LoadbalancerListener{
		SLoadbalancerListener: &compute_models.SLoadbalancerListener{},
	}
	listener.Id = "listener-id"
	listener.Status = "enabled"
	listener.ListenerType = "udp"
	listener.ListenerPort = 9000
	listener.BackendGroupId = "backend-group-id"
	listener.Scheduler = "rr"
	listener.HealthCheck = "on"
	listener.HealthCheckType = "udp"
	listener.HealthCheckInterval = 10
	listener.HealthCheckTimeout = 5
	listener.HealthCheckRise = 3
	listener.HealthCheckFall = 2
	listener.HealthCheckReq = request
	listener.HealthCheckExp = expected
	lb.Listeners[listener.Id] = listener
	listener.loadbalancer = lb

	backendGroup := &LoadbalancerBackendGroup{
		SLoadbalancerBackendGroup: &compute_models.SLoadbalancerBackendGroup{},
		Backends:                  LoadbalancerBackends{},
	}
	backendGroup.Id = listener.BackendGroupId
	backendGroup.LoadbalancerId = lb.Id
	backendGroup.loadbalancer = lb
	lb.BackendGroups[backendGroup.Id] = backendGroup

	backend := &LoadbalancerBackend{
		SLoadbalancerBackend: &compute_models.SLoadbalancerBackend{},
	}
	backend.Id = "backend-id"
	backend.BackendGroupId = backendGroup.Id
	backend.Address = "127.0.0.1"
	backend.Port = 9001
	backend.Weight = 1
	backend.backendGroup = backendGroup
	backendGroup.Backends[backend.Id] = backend

	corpus := NewEmptyLoadbalancerCorpus()
	corpus.Loadbalancers[lb.Id] = lb
	return corpus, lb
}
