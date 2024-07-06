package stock

import "github.com/jmoiron/sqlx"

func initializeDatabaseSchema(db *sqlx.DB) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS stock (
			product_id UUID PRIMARY KEY,
			quantity INT NOT NULL
		);
	`)
	if err != nil {
		panic(err)
	}
}

type ProductStock struct {
	ProductID string `db:"product_id" json:"product_id"`
	Quantity  int    `db:"quantity" json:"quantity"`
}
