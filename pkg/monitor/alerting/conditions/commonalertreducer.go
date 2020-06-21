package conditions

func NewCommonAlertReducer(t string) *queryReducer {
	return &queryReducer{Type: t}
}
