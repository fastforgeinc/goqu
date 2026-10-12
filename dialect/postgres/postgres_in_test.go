package postgres_test

import (
	"testing"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	"github.com/stretchr/testify/assert"
)

// IN and NOT IN take a list of values, not a Postgres array: a slice on their
// right-hand side renders as (v1, v2), while a slice anywhere else stays an
// array.
func TestInRendersValueList(t *testing.T) {
	ds := goqu.Dialect("postgres").From("entry")

	tests := []struct {
		name     string
		where    goqu.Expression
		sql      string
		prepared string
		args     []interface{}
	}{
		{
			name:     "slice",
			where:    goqu.C("string").In([]string{"a", "b"}),
			sql:      `SELECT * FROM "entry" WHERE ("string" IN ('a', 'b'))`,
			prepared: `SELECT * FROM "entry" WHERE ("string" IN ($1, $2))`,
			args:     []interface{}{"a", "b"},
		},
		{
			name:     "variadic",
			where:    goqu.C("int").In(1, 2),
			sql:      `SELECT * FROM "entry" WHERE ("int" IN (1, 2))`,
			prepared: `SELECT * FROM "entry" WHERE ("int" IN ($1, $2))`,
			args:     []interface{}{int64(1), int64(2)},
		},
		{
			name:     "not in",
			where:    goqu.C("string").NotIn([]string{"a", "b"}),
			sql:      `SELECT * FROM "entry" WHERE ("string" NOT IN ('a', 'b'))`,
			prepared: `SELECT * FROM "entry" WHERE ("string" NOT IN ($1, $2))`,
			args:     []interface{}{"a", "b"},
		},
		{
			name:     "subquery",
			where:    goqu.C("int").In(goqu.From("other").Select("int")),
			// As upstream renders it; Postgres reads ((SELECT ...)) as a subquery.
			sql:      `SELECT * FROM "entry" WHERE ("int" IN ((SELECT "int" FROM "other")))`,
			prepared: `SELECT * FROM "entry" WHERE ("int" IN ((SELECT "int" FROM "other")))`,
			args:     []interface{}{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, _, err := ds.Where(tt.where).ToSQL()
			assert.NoError(t, err)
			assert.Equal(t, tt.sql, sql)

			sql, args, err := ds.Prepared(true).Where(tt.where).ToSQL()
			assert.NoError(t, err)
			assert.Equal(t, tt.prepared, sql)
			assert.Equal(t, tt.args, args)
		})
	}
}

// The fork's reason to exist: a slice value outside IN is a Postgres array.
func TestSliceValueStaysArray(t *testing.T) {
	ds := goqu.Dialect("postgres").Insert("entry").Rows(goqu.Record{"tags": []string{"x", "y"}})

	sql, _, err := ds.ToSQL()
	assert.NoError(t, err)
	assert.Equal(t, `INSERT INTO "entry" ("tags") VALUES ('{"x", "y"}')`, sql)

	sql, args, err := ds.Prepared(true).ToSQL()
	assert.NoError(t, err)
	assert.Equal(t, `INSERT INTO "entry" ("tags") VALUES ($1)`, sql)
	assert.Equal(t, []interface{}{[]string{"x", "y"}}, args)
}
