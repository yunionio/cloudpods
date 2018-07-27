package uuid

import (
	"github.com/golang-plus/uuid/internal/namebased"
)

// Standard Namespaces.
const (
	NamespaceDNS  = namebased.NamespaceDNS  // "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
	NamespaceURL  = namebased.NamespaceURL  // "6ba7b811-9dad-11d1-80b4-00c04fd430c8"
	NamespaceOID  = namebased.NamespaceOID  // "6ba7b812-9dad-11d1-80b4-00c04fd430c8"
	NamespaceX500 = namebased.NamespaceX500 // "6ba7b814-9dad-11d1-80b4-00c04fd430c8"
)
