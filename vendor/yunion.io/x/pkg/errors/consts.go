package errors

const (
	ErrServer       = Error("ServerError")
	ErrClient       = Error("ClientError")
	ErrUnclassified = Error("UnclassifiedError")

	ErrNotFound       = Error("NotFoundError")
	ErrDuplicateId    = Error("DuplicateIdError")
	ErrInvalidStatus  = Error("InvalidStatusError")
	ErrTimeout        = Error("TimeoutError")
	ErrNotImplemented = Error("NotImplementedError")
	ErrNotSupported   = Error("NotSupportedError")

	ErrAggregate = Error("AggregateError")
)
