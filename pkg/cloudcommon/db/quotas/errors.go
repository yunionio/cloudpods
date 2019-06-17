package quotas

import (
	"fmt"
	"strings"
)

type SOutOfQuotaError struct {
	name  string
	limit int
	used  int
}

type SOutOfQuotaErrors struct {
	errors []SOutOfQuotaError
}

func (e *SOutOfQuotaError) Error() string {
	return fmt.Sprintf("%s limit %d used %d", e.name, e.limit, e.used)
}

func (es *SOutOfQuotaErrors) Error() string {
	qs := make([]string, len(es.errors))
	for i := range es.errors {
		e := es.errors[i]
		qs = append(qs, e.Error())
	}
	return fmt.Sprintf("Out of quota: %s", strings.Join(qs, ", "))
}

func (es *SOutOfQuotaErrors) IsError() bool {
	if len(es.errors) == 0 {
		return false
	} else {
		return true
	}
}

func NewOutOfQuotaError() *SOutOfQuotaErrors {
	return &SOutOfQuotaErrors{
		errors: make([]SOutOfQuotaError, 0),
	}
}

func (es *SOutOfQuotaErrors) Add(name string, limit int, used int) {
	e := SOutOfQuotaError{
		name:  name,
		limit: limit,
		used:  used,
	}
	es.errors = append(es.errors, e)
}
