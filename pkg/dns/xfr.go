package dns

import (
	"context"
	"time"

	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

// Serial implements the Transferer interface
func (r *SRegionDNS) Serial(state request.Request) uint32 {
	return uint32(time.Now().Unix())
}

// MinTTL implements the Transferer interface
func (r *SRegionDNS) MinTTL(state request.Request) uint32 {
	return 30
}

// Transferer implements the Transferer interface
func (r *SRegionDNS) Transfer(ctx context.Context, state request.Request) (int, error) {
	return dns.RcodeServerFailure, nil
}
