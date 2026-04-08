// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

//go:build go1.19
// +build go1.19

package instrumentation

import "net/netip"

// NetaddrIP wraps a netip.Addr value
type NetaddrIP = netip.Addr

// NetaddrIPPrefix wraps a netip.Prefix value
type NetaddrIPPrefix = netip.Prefix

var (
	// NetaddrParseIP wraps the netip.ParseAddr function
	NetaddrParseIP = netip.ParseAddr
	// NetaddrParseIPPrefix wraps the netip.ParsePrefix function
	NetaddrParseIPPrefix = netip.ParsePrefix
	// NetaddrMustParseIP wraps the netip.MustParseAddr function
	NetaddrMustParseIP = netip.MustParseAddr
	// NetaddrIPv6Raw wraps the netIP.AddrFrom16 function
	NetaddrIPv6Raw = netip.AddrFrom16
)

// NetaddrIPv4 wraps the netip.AddrFrom4 function
func NetaddrIPv4(a, b, c, d byte) NetaddrIP {
	e := [4]byte{a, b, c, d}
	return netip.AddrFrom4(e)
}
