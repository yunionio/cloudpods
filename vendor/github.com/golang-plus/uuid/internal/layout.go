package internal

// Layout represents the layout of UUID. See page 5 in RFC 4122.
type Layout byte

const (
	// LayoutInvalid represents invalid layout.
	LayoutInvalid Layout = iota
	// LayoutNCS represents the NCS layout: Reserved, NCS backward compatibility (Values: 0x00-0x07).
	LayoutNCS
	// LayoutRFC4122 represents the RFC4122 layout: The variant specified in RFC 4122 (Values: 0x08-0x0b).
	LayoutRFC4122
	// LayoutMicrosoft represents the Microsoft layout: Reserved, Microsoft Corporation backward compatibility (Values: 0x0c-0x0d).
	LayoutMicrosoft
	// LayoutFuture represents the Future layout: Reserved for future definition. (Values: 0x0e-0x0f).
	LayoutFuture
)

// SetLayout sets the layout for uuid.
// This is intended to be called from the New function in packages that implement uuid generating functions.
func SetLayout(uuid []byte, layout Layout) {
	switch layout {
	case LayoutNCS:
		uuid[8] = (uuid[8] | 0x00) & 0x0f // Msb0=0
	case LayoutRFC4122:
		uuid[8] = (uuid[8] | 0x80) & 0x8f // Msb0=1, Msb1=0
	case LayoutMicrosoft:
		uuid[8] = (uuid[8] | 0xc0) & 0xcf // Msb0=1, Msb1=1, Msb2=0
	case LayoutFuture:
		uuid[8] = (uuid[8] | 0xe0) & 0xef // Msb0=1, Msb1=1, Msb2=1
	default:
		panic("layout is invalid")
	}
}

// GetLayout returns layout of uuid.
func GetLayout(uuid []byte) Layout {
	switch {
	case (uuid[8] & 0x80) == 0x00:
		return LayoutNCS
	case (uuid[8] & 0xc0) == 0x80:
		return LayoutRFC4122
	case (uuid[8] & 0xe0) == 0xc0:
		return LayoutMicrosoft
	case (uuid[8] & 0xe0) == 0xe0:
		return LayoutFuture
	}

	return LayoutInvalid
}
