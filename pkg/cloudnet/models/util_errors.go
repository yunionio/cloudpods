package models

type (
	errNotFound    error
	errMoreThanOne error
)

func IsNotFound(err error) bool {
	_, ok := err.(errNotFound)
	return ok
}

func IsMoreThanOne(err error) bool {
	_, ok := err.(errMoreThanOne)
	return ok
}
