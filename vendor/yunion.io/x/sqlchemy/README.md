# sqlchemy

[![CircleCI](https://circleci.com/gh/yunionio/sqlchemy.svg?style=svg)](https://circleci.com/gh/yunionio/sqlchemy)
[![codecov](https://codecov.io/gh/yunionio/sqlchemy/branch/master/graph/badge.svg?token=K8cSYZzLbc)](https://codecov.io/gh/yunionio/sqlchemy)
[![Go Report Card](https://goreportcard.com/badge/github.com/yunionio/sqlchemy)](https://goreportcard.com/report/github.com/yunionio/sqlchemy)

A lightweight golang ORM library inspired by python sqlalchemy.

Features
----------------

* Automatic creation and synchronization of table schema based on golang struct
* Query syntax inpired by sqlalchemy
* Support MySQL/MariaDB with InnoDB engine ONLY
* Support select, insert, update and insert or update

Quick Examples
----------------

## Table Schema

Table schema is defined by struct field tags

```go
type TestTable struct {
    Id        string               `primary:"true" width:"128" charset:"ascii" nullable:"false"`
    Name      string               `width:"64" charset:"utf8" index:"true"`
    Gender    string               `width:"10" charset:"ascii"`
    Age       uint8                `default:"18"`
    Info      jsonutils.JSONObject `nullable:"false"`
    Compond   *SCompondStruct      `width:1024`
    CreatedAt time.Time            `nullable:"false" created_at:"true"`
    UpdatedAt time.Time            `nullable:"false" updated_at:"true"`
    Version   int                  `default:"0" nullable:"false" auto_version:"true"`
    DeletedAt time.Time            ``
    Deleted   bool                 `nullable:"false" default:"false"`
    Notes     string               `default:"default notes"`
}
````

## Table initialization

Create a table from a struct schema

```go
tablespec := sqlchemy.NewTableSpecFromStruct(TestTable{}, "testtable")
```

Check whether table schema definition is consistent with schema in database.

```go
if !tablespec.CheckSync() {
    log.Fatalf("table not in sync")
}
```

Synchronize database table schema and make it consistent with the struct defintion.

```go
err := tablespec.Sync()
if err != nil {
    log.Fataf("synchronize table schema failed: %s", er)
}
```

## Query

### Construct query

```go
ti := tablespec.Instance()

// select * from testtable
q := ti.Query()

// select * from testtable where id = '981b10ed-b6f9-4120-8a77-a3b03e343143'
// query by field name, in which the name is unique in the query
q := ti.Query().Equals("id", "981b10ed-b6f9-4120-8a77-a3b03e343143")

// query by field instance, in which the field name might be ambiguous
q := ti.Query().Filter(sqlchemy.Equals(ti.Field("id"), "981b10ed-b6f9-4120-8a77-a3b03e343143"))

// joint query

// select * from t1 join t2 on t1.id=t2.testtable_id where t2.created_at > '2019-11-02'
q := ti.Query("name").Join(t2, sqlchemy.Equals(ti.Field("id"), t2.Field("testtable_id"))).Filter(sqlchermy.GT(t2.Field("created_at"), '2019-11-02')

// union query
// select id, name from testtable where id = '981b10ed-b6f9-4120-8a77-a3b03e343143' union select id, name from testtable where id='6fcc87ca-c1da-40ab-849a-305ff2663901'
q1 := t1.Query("id", "name").Equals("id", "981b10ed-b6f9-4120-8a77-a3b03e343143")
q2 := t1.Query("id", "name").Equals("id", "6fcc87ca-c1da-40ab-849a-305ff2663901")
qu := sqlchemy.Union(q1, q2)
```

### Fetch data

```go
q := ti.Query().Equals("id", "e2bc9b659cec407590dc2f3fcb009acb")

// fetch single row into object
row := TestTable{}
err := q.First(&row)
if err != nil {
    log.Fatalf("fetch object error %s", err)
}

// fetch single row into a string map, where strMap is map[string]string
strMap, err := q.FirstStringMap()
if err != nil {
    log.Fatalf("fetch object error %s", err)
}

q := ti.Query().Startswith("id", "abc")
// fetch rows
rows := make([]TestTable, 0)
err := q.All(&rows)
if err != nil {
    log.Fatalf("query failure: %s", err)
}

// fetch rows into string maps, where maps is []map[string]string
maps, err := q.AllStringMap()
if err != nil {
    log.Fatalf("query failure: %s", err)
}
```

### SubQuery

Query can be used as a subquery in other queries.

```go
// derive a subquery from an ordinary query
subq := t1.Query("id").Equals("version", "v2.0").SubQuery()
// use subquery
q := t1.Query().In("id", subq)
```

## Insert

```go
// hook to initialize data field before insert
func (t *TestTable) BeforeInsert() {
    t.Id = uuid4()
}
// initialize data struct
dt1 := TestTable{
    Name: "Test",
}
// insert the data, primary key fields must be populated
// the primary key has been populated by the BeforeInsert hook
err = tablespec.Insert(&dt1)

// insert or update
// insert the object if no primary key conflict, otherwise, update the record
err = tablespec.InsertOrUpdate(&dt1)
```

## Update

```go
// update the field
_, err = tablespec.Update(&dt3, func() error {
    dt3.Name = "New name 4"
    dt3.Compond = &SCompondStruct{Id: "998822333", Age: 80}
    return nil
})
```

Please refer to sqltest/main.go for more examples.

