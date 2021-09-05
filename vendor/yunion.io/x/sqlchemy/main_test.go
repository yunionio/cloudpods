package sqlchemy

import (
	"testing"
)

type testQueryTable struct {
	Col0 string
	Col1 int
}

var (
	testTableSpec = NewTableSpecFromStruct(testQueryTable{}, "test")
	testTable     = testTableSpec.Instance()
)

func testReset() {
	tableIDLock.Lock()
	defer tableIDLock.Unlock()

	// next alias index must be 2 because 1 has already been taken by
	// testTable
	tableID = 1
}

func testGotWant(t *testing.T, got, want string) {
	if got != want {
		t.Fatalf("\ngot:\n%s\nwant:\n%s\n", got, want)
	}
}
