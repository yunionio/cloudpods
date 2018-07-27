package uuid

// Style represents the style of UUID string.
type Style byte

const (
	// StyleStandard represents the standard style of UUID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (8-4-4-4-12, length: 36).
	StyleStandard Style = iota + 1
	// StyleWithoutDash represents the style without dash of UUID: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx (length: 32).
	StyleWithoutDash
)

// String returns English description of style.
func (s Style) String() string {
	switch s {
	case StyleStandard:
		return "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (8-4-4-4-12)"
	case StyleWithoutDash:
		return "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	default:
		return "Unknown"
	}
}
