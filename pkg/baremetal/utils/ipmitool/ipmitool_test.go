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

package ipmitool

import (
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
)

func TestIpmitoolOutputAcceptable(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want bool
	}{
		{
			name: "empty",
			out:  "",
			want: false,
		},
		{
			name: "valid lan print with exit 1 style output",
			out: `Set in Progress         : Set Complete
IP Address Source       : Static Address
IP Address              : 10.127.223.102
Subnet Mask             : 255.255.255.0
MAC Address             : aa:bb:cc:dd:ee:ff
Default Gateway IP      : 10.127.223.1`,
			want: true,
		},
		{
			name: "auth failure",
			out:  "Error: Unable to establish IPMI v2 / RMCP+ session\n",
			want: false,
		},
		{
			name: "invalid channel",
			out:  "Get Channel Info command failed\nInvalid channel: 8\n",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ipmitoolOutputAcceptable([]byte(tt.out))
			if got != tt.want {
				t.Errorf("ipmitoolOutputAcceptable() = %v, want %v", got, tt.want)
			}
		})
	}
}

type fakeIPMIResponse struct {
	lines []string
	err   error
}

type fakeIPMIExecutor struct {
	responses map[uint8][]fakeIPMIResponse
	calls     []uint8
	attempts  map[uint8]int
}

func (f *fakeIPMIExecutor) GetMode() string {
	return "fake"
}

func (f *fakeIPMIExecutor) ExecuteCommand(args ...string) ([]string, error) {
	if len(args) != 3 || args[0] != "lan" || args[1] != "print" {
		return nil, fmt.Errorf("unexpected command: %v", args)
	}
	channel, err := strconv.ParseUint(args[2], 10, 8)
	if err != nil {
		return nil, fmt.Errorf("parse channel %q: %w", args[2], err)
	}
	channelID := uint8(channel)
	f.calls = append(f.calls, channelID)
	if f.attempts == nil {
		f.attempts = make(map[uint8]int)
	}
	f.attempts[channelID]++
	responses := f.responses[channelID]
	if len(responses) > 0 {
		response := responses[0]
		f.responses[channelID] = responses[1:]
		return response.lines, response.err
	}
	return nil, fmt.Errorf("channel %d unavailable", channel)
}

func fakeResponses(responses ...fakeIPMIResponse) []fakeIPMIResponse {
	return responses
}

func lanConfigLines(ipAddr, mac string) []string {
	return []string{
		"IP Address Source       : Static Address",
		"IP Address              : " + ipAddr,
		"Subnet Mask             : 255.255.255.0",
		"MAC Address             : " + mac,
		"Default Gateway IP      : 192.0.2.1",
		"802.1q VLAN ID          : 42",
	}
}

func probeChannels(probes []LanConfigProbeResult) []uint8 {
	channels := make([]uint8, len(probes))
	for i := range probes {
		channels[i] = probes[i].Channel
	}
	return channels
}

func TestGetSysInfo(t *testing.T) {
	type args struct {
		exector IPMIExecutor
	}
	tests := []struct {
		name    string
		args    args
		want    *types.SSystemInfo
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSysInfo(tt.args.exector)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSysInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSysInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDiscoverLanConfigRetriesPreferredChannel(t *testing.T) {
	executor := &fakeIPMIExecutor{
		responses: map[uint8][]fakeIPMIResponse{
			8: fakeResponses(
				fakeIPMIResponse{err: fmt.Errorf("transient failure 1")},
				fakeIPMIResponse{err: fmt.Errorf("transient failure 2")},
				fakeIPMIResponse{lines: lanConfigLines("192.0.2.8", "00:11:22:33:44:88")},
			),
		},
	}

	discovery, err := DiscoverLanConfig(executor, []uint8{8}, LanConfigSelectionOptions{
		ConnectedIP:         "192.0.2.8",
		RequireConfiguredIP: true,
	})
	if err != nil {
		t.Fatalf("DiscoverLanConfig() error = %v", err)
	}
	if got, want := executor.calls, []uint8{8, 8, 8}; !reflect.DeepEqual(got, want) {
		t.Fatalf("DiscoverLanConfig() calls = %v, want %v", got, want)
	}
	if discovery.Selected == nil || discovery.Selected.Channel != 8 {
		t.Fatalf("DiscoverLanConfig() selected = %#v, want channel 8", discovery.Selected)
	}
	if got, want := len(discovery.Probes), 1; got != want {
		t.Errorf("DiscoverLanConfig() probe count = %d, want %d", got, want)
	}
	if got, want := executor.attempts[8], 3; got != want {
		t.Errorf("DiscoverLanConfig() channel 8 attempts = %d, want %d", got, want)
	}
}

func TestDiscoverLanConfigPreferredExhaustionUsesSingleFallbackAttempts(t *testing.T) {
	executor := &fakeIPMIExecutor{
		responses: map[uint8][]fakeIPMIResponse{
			8: fakeResponses(
				fakeIPMIResponse{err: fmt.Errorf("preferred failure 1")},
				fakeIPMIResponse{err: fmt.Errorf("preferred failure 2")},
				fakeIPMIResponse{err: fmt.Errorf("preferred failure 3")},
			),
		},
	}

	discovery, err := DiscoverLanConfig(executor, []uint8{8}, LanConfigSelectionOptions{
		RequireConfiguredIP: true,
		AllowFallback:       true,
	})
	if err == nil {
		t.Fatal("DiscoverLanConfig() error = nil, want non-nil")
	}
	wantCalls := []uint8{8, 8, 8, 1, 2, 3, 4, 5, 6, 7, 9, 10, 11}
	if got := executor.calls; !reflect.DeepEqual(got, wantCalls) {
		t.Fatalf("DiscoverLanConfig() calls = %v, want %v", got, wantCalls)
	}
	if got, want := probeChannels(discovery.Probes), []uint8{8, 1, 2, 3, 4, 5, 6, 7, 9, 10, 11}; !reflect.DeepEqual(got, want) {
		t.Errorf("DiscoverLanConfig() probe channels = %v, want %v", got, want)
	}
	for channel := uint8(1); channel <= 11; channel++ {
		wantAttempts := 1
		if channel == 8 {
			wantAttempts = 3
		}
		if got := executor.attempts[channel]; got != wantAttempts {
			t.Errorf("DiscoverLanConfig() channel %d attempts = %d, want %d", channel, got, wantAttempts)
		}
	}
}

func TestDiscoverLanConfigProbeOrderAndDeduplication(t *testing.T) {
	invalidConfig := fakeIPMIResponse{lines: lanConfigLines("192.0.2.1", "not-a-mac")}
	executor := &fakeIPMIExecutor{
		responses: map[uint8][]fakeIPMIResponse{
			1: fakeResponses(invalidConfig),
			2: fakeResponses(invalidConfig),
			8: fakeResponses(invalidConfig),
		},
	}

	discovery, err := DiscoverLanConfig(executor, []uint8{1, 8, 2, 1, 12}, LanConfigSelectionOptions{
		ConnectedIP:         "192.0.2.99",
		PersistedChannel:    8,
		RequireConfiguredIP: true,
		AllowFallback:       true,
	})
	if err == nil {
		t.Fatal("DiscoverLanConfig() error = nil, want non-nil")
	}
	want := []uint8{1, 8, 2, 3, 4, 5, 6, 7, 9, 10, 11}
	if got := executor.calls; !reflect.DeepEqual(got, want) {
		t.Errorf("DiscoverLanConfig() calls = %v, want %v", got, want)
	}
	if got := probeChannels(discovery.Probes); !reflect.DeepEqual(got, want) {
		t.Errorf("DiscoverLanConfig() probe order = %v, want %v", got, want)
	}
	for i, channel := range []uint8{1, 8, 2} {
		probe := discovery.Probes[i]
		if probe.Channel != channel || probe.Config == nil || probe.Err != nil {
			t.Errorf("DiscoverLanConfig() probe %d = %#v, want successful unusable channel %d", i, probe, channel)
		}
	}
}

func TestDiscoverLanConfigSelectionPriority(t *testing.T) {
	t.Run("connected IP beats persisted channel", func(t *testing.T) {
		executor := &fakeIPMIExecutor{
			responses: map[uint8][]fakeIPMIResponse{
				1: fakeResponses(fakeIPMIResponse{lines: lanConfigLines("192.0.2.1", "00:11:22:33:44:11")}),
				4: fakeResponses(fakeIPMIResponse{lines: lanConfigLines("192.0.2.4", "00:11:22:33:44:44")}),
			},
		}

		discovery, err := DiscoverLanConfig(executor, []uint8{4}, LanConfigSelectionOptions{
			ConnectedIP:         "192.0.2.4",
			PersistedChannel:    1,
			RequireConfiguredIP: true,
		})
		if err != nil {
			t.Fatalf("DiscoverLanConfig() error = %v", err)
		}
		if got, want := executor.calls, []uint8{4}; !reflect.DeepEqual(got, want) {
			t.Fatalf("DiscoverLanConfig() calls = %v, want %v", got, want)
		}
		if discovery.Selected == nil || discovery.Selected.Channel != 4 {
			t.Fatalf("DiscoverLanConfig() selected = %#v, want channel 4", discovery.Selected)
		}
	})

	t.Run("equivalent IPv6 text matches connected IP", func(t *testing.T) {
		executor := &fakeIPMIExecutor{
			responses: map[uint8][]fakeIPMIResponse{
				4: fakeResponses(fakeIPMIResponse{lines: lanConfigLines("2001:0db8:0:0:0:0:0:4", "00:11:22:33:44:44")}),
			},
		}

		discovery, err := DiscoverLanConfig(executor, []uint8{4}, LanConfigSelectionOptions{
			ConnectedIP:         "2001:db8::4",
			RequireConfiguredIP: true,
		})
		if err != nil {
			t.Fatalf("DiscoverLanConfig() error = %v", err)
		}
		if discovery.Selected == nil || discovery.Selected.Channel != 4 {
			t.Fatalf("DiscoverLanConfig() selected = %#v, want channel 4", discovery.Selected)
		}
		if got, want := executor.calls, []uint8{4}; !reflect.DeepEqual(got, want) {
			t.Fatalf("DiscoverLanConfig() calls = %v, want %v", got, want)
		}
	})

	t.Run("persisted channel beats multiple usable candidates", func(t *testing.T) {
		executor := &fakeIPMIExecutor{
			responses: map[uint8][]fakeIPMIResponse{
				1: fakeResponses(fakeIPMIResponse{lines: lanConfigLines("192.0.2.1", "00:11:22:33:44:11")}),
				4: fakeResponses(fakeIPMIResponse{lines: lanConfigLines("192.0.2.4", "00:11:22:33:44:44")}),
			},
		}

		discovery, err := DiscoverLanConfig(executor, []uint8{4}, LanConfigSelectionOptions{
			ConnectedIP:         "192.0.2.99",
			PersistedChannel:    1,
			RequireConfiguredIP: true,
			AllowFallback:       true,
		})
		if err != nil {
			t.Fatalf("DiscoverLanConfig() error = %v", err)
		}
		if discovery.Selected == nil || discovery.Selected.Channel != 1 {
			t.Fatalf("DiscoverLanConfig() selected = %#v, want channel 1", discovery.Selected)
		}
		if got, want := len(discovery.Probes), 11; got != want {
			t.Errorf("DiscoverLanConfig() probe count = %d, want %d", got, want)
		}
	})

	t.Run("empty connected IP short-circuits persisted channel", func(t *testing.T) {
		executor := &fakeIPMIExecutor{
			responses: map[uint8][]fakeIPMIResponse{
				8: fakeResponses(fakeIPMIResponse{lines: lanConfigLines("0.0.0.0", "00:11:22:33:44:88")}),
			},
		}

		discovery, err := DiscoverLanConfig(executor, []uint8{1, 2}, LanConfigSelectionOptions{PersistedChannel: 8})
		if err != nil {
			t.Fatalf("DiscoverLanConfig() error = %v", err)
		}
		if got, want := executor.calls, []uint8{1, 1, 1, 2, 2, 2, 8}; !reflect.DeepEqual(got, want) {
			t.Fatalf("DiscoverLanConfig() calls = %v, want %v", got, want)
		}
		if discovery.Selected == nil || discovery.Selected.Channel != 8 {
			t.Fatalf("DiscoverLanConfig() selected = %#v, want channel 8", discovery.Selected)
		}
	})

	t.Run("sole usable candidate", func(t *testing.T) {
		executor := &fakeIPMIExecutor{
			responses: map[uint8][]fakeIPMIResponse{
				1: fakeResponses(fakeIPMIResponse{lines: lanConfigLines("192.0.2.1", "not-a-mac")}),
				4: fakeResponses(fakeIPMIResponse{lines: lanConfigLines("192.0.2.4", "00:11:22:33:44:44")}),
			},
		}

		discovery, err := DiscoverLanConfig(executor, []uint8{1, 4}, LanConfigSelectionOptions{RequireConfiguredIP: true})
		if err != nil {
			t.Fatalf("DiscoverLanConfig() error = %v", err)
		}
		if discovery.Selected == nil || discovery.Selected.Channel != 4 {
			t.Fatalf("DiscoverLanConfig() selected = %#v, want channel 4", discovery.Selected)
		}
	})
}

func TestDiscoverLanConfigAmbiguous(t *testing.T) {
	executor := &fakeIPMIExecutor{
		responses: map[uint8][]fakeIPMIResponse{
			1: fakeResponses(fakeIPMIResponse{lines: lanConfigLines("192.0.2.1", "00:11:22:33:44:11")}),
			4: fakeResponses(fakeIPMIResponse{lines: lanConfigLines("192.0.2.4", "00:11:22:33:44:44")}),
		},
	}

	discovery, err := DiscoverLanConfig(executor, []uint8{1, 4}, LanConfigSelectionOptions{RequireConfiguredIP: true})
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("DiscoverLanConfig() error = %v, want ambiguity error", err)
	}
	if discovery == nil {
		t.Fatal("DiscoverLanConfig() discovery = nil, want probe diagnostics")
	}
	if discovery.Selected != nil {
		t.Errorf("DiscoverLanConfig() selected = %#v, want nil", discovery.Selected)
	}
	if !strings.Contains(err.Error(), "[1 4]") {
		t.Errorf("DiscoverLanConfig() error = %v, want ambiguous channel list", err)
	}
}

func TestDiscoverLanConfigDiagnostics(t *testing.T) {
	executor := &fakeIPMIExecutor{
		responses: map[uint8][]fakeIPMIResponse{
			1: fakeResponses(fakeIPMIResponse{err: fmt.Errorf("probe failed")}),
			2: fakeResponses(fakeIPMIResponse{lines: lanConfigLines("192.0.2.2", "not-a-mac")}),
			3: fakeResponses(fakeIPMIResponse{lines: lanConfigLines("0.0.0.0", "00:11:22:33:44:33")}),
			4: fakeResponses(fakeIPMIResponse{lines: lanConfigLines("::", "00:11:22:33:44:44")}),
		},
	}

	discovery, err := DiscoverLanConfig(executor, nil, LanConfigSelectionOptions{
		RequireConfiguredIP: true,
		AllowFallback:       true,
	})
	if err == nil {
		t.Fatal("DiscoverLanConfig() error = nil, want non-nil")
	}
	if discovery == nil {
		t.Fatal("DiscoverLanConfig() discovery = nil, want probe diagnostics")
	}
	if got, want := len(discovery.Probes), 11; got != want {
		t.Fatalf("DiscoverLanConfig() probe count = %d, want %d", got, want)
	}
	if got := discovery.Probes[0].Err; got == nil || !strings.Contains(got.Error(), "probe failed") {
		t.Errorf("DiscoverLanConfig() channel 1 error = %v, want retained executor error", got)
	}
	for _, probe := range discovery.Probes[1:4] {
		if probe.Config == nil || probe.Err != nil {
			t.Errorf("DiscoverLanConfig() channel %d probe = %#v, want successful unusable config", probe.Channel, probe)
		}
	}
	for _, want := range []string{
		"no usable IPMI LAN configuration",
		"probe IPMI LAN channel 1: probe failed",
		"LAN channel 2 has no valid MAC address",
		"LAN channel 3 has unconfigured IP address \"0.0.0.0\"",
		"LAN channel 4 has unconfigured IP address \"::\"",
		"channel 11 unavailable",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("DiscoverLanConfig() error = %q, want substring %q", err, want)
		}
	}
}

func TestIsUsableLanConfig(t *testing.T) {
	mac := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	tests := []struct {
		name                string
		config              *types.SIPMILanConfig
		requireConfiguredIP bool
		want                bool
	}{
		{name: "nil config"},
		{name: "missing MAC", config: &types.SIPMILanConfig{IPAddr: "192.0.2.1"}},
		{name: "invalid MAC", config: &types.SIPMILanConfig{Mac: net.HardwareAddr{0x01, 0x02}}},
		{name: "MAC only allowed", config: &types.SIPMILanConfig{Mac: mac}, want: true},
		{name: "missing required IP", config: &types.SIPMILanConfig{Mac: mac}, requireConfiguredIP: true},
		{name: "malformed required IP", config: &types.SIPMILanConfig{Mac: mac, IPAddr: "not-an-ip"}, requireConfiguredIP: true},
		{name: "unspecified IPv4", config: &types.SIPMILanConfig{Mac: mac, IPAddr: "0.0.0.0"}, requireConfiguredIP: true},
		{name: "unspecified IPv6", config: &types.SIPMILanConfig{Mac: mac, IPAddr: "::"}, requireConfiguredIP: true},
		{name: "configured IPv4", config: &types.SIPMILanConfig{Mac: mac, IPAddr: "192.0.2.1"}, requireConfiguredIP: true, want: true},
		{name: "configured IPv6", config: &types.SIPMILanConfig{Mac: mac, IPAddr: "2001:db8::1"}, requireConfiguredIP: true, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUsableLanConfig(tt.config, tt.requireConfiguredIP); got != tt.want {
				t.Errorf("isUsableLanConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
