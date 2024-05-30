package errors

const (
	ErrServer       = Error("ServerError")
	ErrClient       = Error("ClientError")
	ErrUnclassified = Error("UnclassifiedError")

	// network error
	ErrDNS            = Error("DNSError")
	ErrEOF            = Error("EOFError")
	ErrNetwork        = Error("NetworkError")
	ErrConnectRefused = Error("ConnectRefusedError")
	ErrConnectReset   = Error("ConnectResetError")
	ErrTimeout        = Error("TimeoutError")

	ErrNotFound        = Error("NotFoundError")
	ErrNotEmpty        = Error("NotEmptyError")
	ErrEmpty           = Error("EmptyError")
	ErrDuplicateId     = Error("DuplicateIdError")
	ErrInvalidStatus   = Error("InvalidStatusError")
	ErrNotImplemented  = Error("NotImplementedError")
	ErrNotSupported    = Error("NotSupportedError")
	ErrAccountReadOnly = Error("AccountReadOnlyError")

	ErrAggregate = Error("AggregateError")

	ErrInvalidFormat = Error("InvalidFormatError")

	ErrUnsupportedProtocol = Error("UnsupportedProtocol")
)
