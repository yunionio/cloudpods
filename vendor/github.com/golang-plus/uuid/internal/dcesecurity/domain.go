package dcesecurity

// Domain represents the identifier for a local domain.
type Domain byte

const (
	// User represents POSIX UID domain.
	User Domain = iota + 1
	// Group represents POSIX GID domain.
	Group
)
