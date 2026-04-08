// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

//go:build !go1.19
// +build !go1.19

package instrumentation

import "inet.af/netaddr"

// NetaddrIP wraps an netaddr.IP value
type NetaddrIP = netaddr.IP

// NetaddrIPPrefix wraps an netaddr.IPPrefix value
type NetaddrIPPrefix = netaddr.IPPrefix

var (
	// NetaddrParseIP wraps the netaddr.ParseIP function
	NetaddrParseIP = netaddr.ParseIP
	// NetaddrParseIPPrefix wraps the netaddr.ParseIPPrefix function
	NetaddrParseIPPrefix = netaddr.ParseIPPrefix
	// NetaddrMustParseIP wraps the netaddr.MustParseIP function
	NetaddrMustParseIP = netaddr.MustParseIP
	// NetaddrIPv4 wraps the netaddr.IPv4 function
	NetaddrIPv4 = netaddr.IPv4
	// NetaddrIPv6Raw wraps the netaddr.IPv6Raw function
	NetaddrIPv6Raw = netaddr.IPv6Raw
)
