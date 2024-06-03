package db

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type DB struct {
	Conn *sqlx.DB
}

func NewDBConn(connString string) (DB, error) {
	db, err := sqlx.Open("postgres", connString)
	if err != nil {
		return DB{}, err
	}

	return DB{Conn: db}, nil
}

func (db *DB) Close() error {
	return db.Conn.Close()
}

func (db *DB) MigrateSchema() {
	db.Conn.MustExec(schema)
}
