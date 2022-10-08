package core

import (
	"testing"
	"time"
)

func TestPredicateAnalysor(t *testing.T) {
	a := newPredicateAnalysor("for test")
	a.Start("a")
	a.End("a", time.Now().Add(time.Second*2))
	a.Start("b")
	a.End("b", time.Now().Add(time.Second*3))
	a.Start("c")
	a.End("c", time.Now().Add(time.Second*5))

	a.ShowResult()
}
