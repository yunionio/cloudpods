package validators

import (
	"strings"
)

type Empty struct{}
type Choices map[string]Empty

func NewChoices(choices ...string) Choices {
	cs := Choices{}
	for _, choice := range choices {
		cs[choice] = Empty{}
	}
	return cs
}

func (cs Choices) Has(choice string) bool {
	_, ok := cs[choice]
	return ok
}

func (cs Choices) String() string {
	choices := make([]string, len(cs))
	i := 0
	for choice := range cs {
		choices[i] = choice
		i++
	}
	return strings.Join(choices, "|")
}
