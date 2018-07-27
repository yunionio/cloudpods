package uuid

import (
	"github.com/golang-plus/uuid/internal"
)

// Layout represents the layout of UUID. See page 5 in RFC 4122.
type Layout byte

const (
	// LayoutInvalid represents invalid layout.
	LayoutInvalid = Layout(internal.LayoutInvalid)
	// LayoutNCS represents the layout: Reserved, NCS backward compatibility.
	LayoutNCS = Layout(internal.LayoutNCS)
	// LayoutRFC4122 represents the layout: The variant specified in RFC 4122.
	LayoutRFC4122 = Layout(internal.LayoutRFC4122)
	// LayoutMicrosoft represents the layout: Reserved, Microsoft Corporation backward compatibility.
	LayoutMicrosoft = Layout(internal.LayoutMicrosoft)
	// LayoutFuture represents the layout: Reserved for future definition.
	LayoutFuture = Layout(internal.LayoutFuture)
)

// String returns English description of layout.
func (l Layout) String() string {
	switch l {
	case LayoutNCS:
		return "Reserved For NCS"
	case LayoutRFC4122:
		return "RFC 4122"
	case LayoutMicrosoft:
		return "Reserved For Microsoft"
	case LayoutFuture:
		return "Reserved For Future"
	default:
		return "Invalid"
	}
}
