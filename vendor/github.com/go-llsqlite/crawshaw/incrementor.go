package sqlite

// BindIndexStart is the index of the first parameter when using the Stmt.Bind*
// functions.
const BindIndexStart = 1

// BindIncrementor returns an Incrementor that starts on 1, the first index
// used in Stmt.Bind* functions. This is provided as syntactic sugar for
// binding parameter values to a Stmt. It allows for easily changing query
// parameters without manually fixing up the bind indexes, which can be error
// prone. For example,
//
//	stmt := conn.Prep(`INSERT INTO test (a, b, c) VALUES (?, ?, ?);`)
//	i := BindIncrementor()
//	stmt.BindInt64(i(), a)          // i() == 1
//	if b > 0 {
//	        stmt.BindInt64(i(), b)  // i() == 2
//	} else {
//	        // Remember to increment the index even if a param is NULL
//	        stmt.BindNull(i())      // i() == 2
//	}
//	stmt.BindText(i(), c)           // i() == 3
func BindIncrementor() Incrementor {
	return NewIncrementor(BindIndexStart)
}

// ColumnIndexStart is the index of the first column when using the
// Stmt.Column* functions.
const ColumnIndexStart = 0

// ColumnIncrementor returns an Incrementor that starts on 0, the first index
// used in Stmt.Column* functions. This is provided as syntactic sugar for
// parsing column values from a Stmt. It allows for easily changing queried
// columns without manually fixing up the column indexes, which can be error
// prone. For example,
//
//	stmt := conn.Prep(`SELECT a, b, c FROM test;`)
//	stmt.Step()
//	i := ColumnIncrementor()
//	a := stmt.ColumnInt64(i())      // i() == 1
//	b := stmt.ColumnInt64(i())      // i() == 2
//	c := stmt.ColumnText(i())       // i() == 3
func ColumnIncrementor() Incrementor {
	return NewIncrementor(ColumnIndexStart)
}

// NewIncrementor returns an Incrementor that starts on start.
func NewIncrementor(start int) Incrementor {
	return func() int {
		start++
		return start - 1
	}
}

// Incrementor is a closure around a value that returns and increments the
// value on each call. For example, the boolean statments in the following code
// snippet would all be true.
//
//	i := NewIncrementor(3)
//	i() == 3
//	i() == 4
//	i() == 5
//
// This is provided as syntactic sugar for dealing with bind param and column
// indexes. See BindIncrementor and ColumnIncrementor for small examples.
type Incrementor func() int
