package esxi

import "testing"

func TestInitValue(t *testing.T) {
	type testStruct struct {
		member1 int
		member2 string
		Member3 int
		Member4 string
	}

	dst := testStruct{}

	dst.member1 = 1
	dst.member2 = "2"
	dst.Member3 = 3
	dst.Member4 = "4"

	pDst := &dst
	t.Logf("%p %#v", pDst, pDst)

	*pDst = testStruct{}

	t.Logf("%p %#v", pDst, pDst)

	new := testStruct{}
	if dst != new {
		t.Errorf("dst != new")
	}
}
