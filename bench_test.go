package worksheets

import (
	"database/sql"
	"math/rand"
	"strings"
	"testing"

	"github.com/helloeave/dat/sqlx-runner"
	_ "github.com/lib/pq"
)

type bencher struct {
	*testing.B
	db    *runner.DB
	defs  *Definitions
	store *DbStore
}

func start(b *testing.B) *bencher {
	// db
	dbUrl := "postgres://ws_user:@localhost/ws_test?sslmode=disable"
	sqlDb, err := sql.Open("postgres", dbUrl)
	if err != nil {
		panic(err)
	}
	db := runner.NewDB(sqlDb, "postgres")

	// defs
	defs, err := NewDefinitions(strings.NewReader(`
	type parent worksheet {
		1:f1 text
		2:f2 text
		3:f3 text
		4:f4 text
		5:f5 text
		6:f6 []child
	}

	type child worksheet {
		1:f1 number[1]
		2:f2 number[2]
		3:f3 number[3]
		4:f4 number[4]
		5:f5 number[5]
	}
	`))
	if err != nil {
		panic(err)
	}

	return &bencher{
		B:     b,
		db:    db,
		defs:  defs,
		store: NewStore(defs),
	}
}

func (b *bencher) prime(count int) {
	for i := 0; i < count; i++ {
		if err := RunTransaction(b.db, func(tx *runner.Tx) error {
			parent := b.parentWs(rand.Intn(5))
			session := b.store.Open(tx)
			_, err := session.Save(parent)
			return err
		}); err != nil {
			panic(err)
		}
	}
}

func (b *bencher) childWs() *Worksheet {
	child := b.defs.MustNewWorksheet("child")

	child.MustSet("f1", NewNumberFromInt(1697))
	child.MustSet("f2", NewNumberFromInt(1759))
	child.MustSet("f3", NewNumberFromInt(2153))
	child.MustSet("f4", NewNumberFromInt(2161))
	child.MustSet("f5", NewNumberFromInt(4229))

	return child
}

func (b *bencher) parentWs(numChildren int) *Worksheet {
	parent := b.defs.MustNewWorksheet("parent")

	parent.MustSet("f1", NewText("Lorem ipsum dolor sit amet"))
	parent.MustSet("f2", NewText("consectetur adipiscing elit."))
	parent.MustSet("f3", NewText("Proin nisi ex,"))
	parent.MustSet("f4", NewText("fringilla nec imperdiet sit amet,"))
	parent.MustSet("f5", NewText("vehicula at ipsum."))

	for 0 < numChildren {
		child := b.childWs()
		parent.MustAppend("f6", child)
		numChildren--
	}

	return parent
}

func BenchmarkLoad(_b *testing.B) {
	b := start(_b)
	// b.prime(100000)

	// choose a random parent ws to load
	var parentId string
	if err := b.db.SQL(`
		select id
		from worksheets
		where name = 'parent'
		order by random()
		limit 1
		`).QueryScalar(&parentId); err != nil {
		panic(err)
	}

	// reset, and get ready to do the benchmark
	b.ResetTimer()

	// lots of loads
	for i := 0; i < b.N; i++ {
		err := RunTransaction(b.db, func(tx *runner.Tx) error {
			session := b.store.Open(tx)
			_, err := session.Load(parentId)
			return err
		})
		if err != nil {
			panic(err)
		}
	}
}
