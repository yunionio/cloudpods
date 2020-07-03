package jwe

import "github.com/lestrrat-go/jwx/internal/option"

// WithPrettyJSONFormat specifies if the `jwe.JSON` serialization tool
// should generate pretty-formatted output
func WithPrettyJSONFormat(b bool) Option {
	return option.New(optkeyPrettyJSONFormat, b)
}
