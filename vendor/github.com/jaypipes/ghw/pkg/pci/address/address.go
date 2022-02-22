//
// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.
//

package address

import (
	"regexp"
	"strings"
)

var (
	regexAddress *regexp.Regexp = regexp.MustCompile(
		`^(([0-9a-f]{0,4}):)?([0-9a-f]{2}):([0-9a-f]{2})\.([0-9a-f]{1})$`,
	)
)

type Address struct {
	Domain   string
	Bus      string
	Slot     string
	Function string
}

// String() returns the canonical [D]BSF representation of this Address
func (addr *Address) String() string {
	return addr.Domain + ":" + addr.Bus + ":" + addr.Slot + "." + addr.Function
}

// Given a string address, returns a complete Address struct, filled in with
// domain, bus, slot and function components. The address string may either
// be in $BUS:$SLOT.$FUNCTION (BSF) format or it can be a full PCI address
// that includes the 4-digit $DOMAIN information as well:
// $DOMAIN:$BUS:$SLOT.$FUNCTION.
//
// Returns "" if the address string wasn't a valid PCI address.
func FromString(address string) *Address {
	addrLowered := strings.ToLower(address)
	matches := regexAddress.FindStringSubmatch(addrLowered)
	if len(matches) == 6 {
		dom := "0000"
		if matches[1] != "" {
			dom = matches[2]
		}
		return &Address{
			Domain:   dom,
			Bus:      matches[3],
			Slot:     matches[4],
			Function: matches[5],
		}
	}
	return nil
}
